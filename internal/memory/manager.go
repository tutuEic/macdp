package memory

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"time"

	"github.com/tutuEic/macdp/internal/llm"
	"github.com/tutuEic/macdp/internal/store"
)

// Manager is the central memory orchestrator.
// Implements the tiered memory architecture inspired by MemGPT/Letta + Mem0.
//
// Three tiers:
//
//	Working  — injected into every agent prompt (deps + recent decisions, ≤2K tokens)
//	Short    — task summaries, file changes, retrieved by relevance
//	Long     — architectural decisions, project conventions, agent patterns
type Manager struct {
	store      *Store
	summarizer *Summarizer
	retriever  *Retriever
	llm        *llm.Client
	config     ContextConfig
	budget     TokenBudget
}

// NewManager creates a new memory manager.
func NewManager(db *store.DB, llmClient *llm.Client) (*Manager, error) {
	memStore, err := NewStore(db.Conn())
	if err != nil {
		return nil, fmt.Errorf("memory manager: %w", err)
	}

	return &Manager{
		store:      memStore,
		summarizer: NewSummarizer(llmClient),
		retriever:  NewRetriever(memStore),
		llm:        llmClient,
		config: ContextConfig{
			MaxWorkingTokens:    2000,
			MaxRetrievedTokens:  3000,
			MaxRetrievedEntries: 5,
			IncludeDecisions:    true,
			IncludeConventions:  true,
			IncludeFiles:        true,
		},
		budget: TokenBudget{
			Max:       8000,
			Breakdown: make(map[string]int),
		},
	}, nil
}

// OnTaskComplete is called after a task finishes.
// It auto-summarizes the output and extracts decisions.
// This is the core MemGPT pattern: compress on write, decompress on read.
func (m *Manager) OnTaskComplete(ctx context.Context, task *store.Task) {
	var entries []*Entry

	// 1. Summarize task output (compression: 10K → 300 tokens)
	summary, err := m.summarizer.SummarizeTask(ctx, task)
	if err != nil {
		log.Printf("[memory] Task summary failed for %s: %v", task.ID, err)
	} else if summary != nil {
		entries = append(entries, summary)
	}

	// 2. Extract architectural decisions
	if task.Output != "" {
		decision, err := m.summarizer.ExtractDecision(ctx, task)
		if err != nil {
			log.Printf("[memory] Decision extraction failed for %s: %v", task.ID, err)
		} else if decision != nil {
			entries = append(entries, decision)
		}
	}

	// 3. Record file changes
	if len(task.FilesChanged) > 0 {
		entries = append(entries, &Entry{
			ID:        "file-" + task.ID,
			ProjectID: task.ProjectID,
			TaskID:    task.ID,
			Module:    task.Module,
			Tier:      TierShort,
			Category:  "file_change",
			Content:   fmt.Sprintf("Task %s modified: %v", task.ID, task.FilesChanged),
			Summary:   fmt.Sprintf("%s changed %d files", task.ID, len(task.FilesChanged)),
			Tokens:    estimateTokens(fmt.Sprintf("%v", task.FilesChanged)),
			Metadata: Metadata{
				Files: task.FilesChanged,
				Tags:  []string{task.Module},
			},
			CreatedAt: time.Now(),
		})
	}

	// 4. Save all entries
	if len(entries) > 0 {
		if err := m.store.SaveBatch(entries); err != nil {
			log.Printf("[memory] Save batch failed: %v", err)
		} else {
			log.Printf("[memory] Saved %d entries for task %s (total tokens: ~%d)",
				len(entries), task.ID, sumTokens(entries))
		}
	}

	// 5. Prune old entries (keep last 20 per category)
	if err := m.store.Prune(task.ProjectID, 20); err != nil {
		log.Printf("[memory] Prune failed: %v", err)
	}
}

// BuildContext assembles the full context for a task execution.
// This is what replaces the old ContextBridge.BuildContext.
// Returns: (working_context, retrieved_context, combined_prompt_block, token_breakdown)
func (m *Manager) BuildContext(ctx context.Context, task *store.Task, deps []*store.Task) string {
	// Reset budget for this call
	m.budget = TokenBudget{
		Max:       8000,
		Breakdown: make(map[string]int),
	}

	var parts []string

	// 1. Working memory: dependency outputs (≤2K tokens)
	depInfos := make([]DependencyInfo, 0, len(deps))
	for _, dep := range deps {
		summary := ""
		if dep.Output != "" {
			summary = firstLine(dep.Output)
		}
		depInfos = append(depInfos, DependencyInfo{
			Title:   dep.Title,
			Status:  string(dep.Status),
			Output:  dep.Output,
			Summary: summary,
			Module:  dep.Module,
		})
	}

	// Get recent decisions
	recentDecisions, _ := m.store.Find(Query{
		ProjectID: task.ProjectID,
		Category:  "decision",
		Tier:      TierLong,
		Limit:     3,
	})

	workingBlock := m.retriever.BuildWorkingContext(depInfos, recentDecisions, m.config)
	workingTokens := estimateTokens(workingBlock)
	m.budget.Breakdown["working"] = workingTokens
	m.budget.Used += workingTokens

	if workingBlock != "" {
		parts = append(parts, workingBlock)
	}

	// 2. Short-term memory: retrieve relevant past task summaries (≤3K tokens)
	retrieved := m.retriever.Retrieve(task.ProjectID, task.Module, task.Description, m.config)
	if len(retrieved) > 0 {
		parts = append(parts, "## Relevant Past Work")
		retrievedTokens := 0
		for _, e := range retrieved {
			if e == nil {
				continue
			}
			text := fmt.Sprintf("### %s [%s]\n%s", e.TaskID, e.Category, e.Content)
			t := estimateTokens(text)
			if retrievedTokens+t > m.config.MaxRetrievedTokens {
				// Truncated version
				text = fmt.Sprintf("### %s [%s]\n%s", e.TaskID, e.Category, e.Summary)
				t = estimateTokens(text)
			}
			if m.budget.Used+t > m.budget.Max {
				break
			}
			parts = append(parts, text)
			retrievedTokens += t
			m.budget.Used += t
		}
		m.budget.Breakdown["retrieved"] = retrievedTokens
	}

	// 3. Project conventions (long-term memory)
	if m.config.IncludeConventions {
		conventions, _ := m.store.Find(Query{
			ProjectID: task.ProjectID,
			Category:  "convention",
			Tier:      TierLong,
			Limit:     3,
		})
		if len(conventions) > 0 {
			parts = append(parts, "## Project Conventions")
			convTokens := 0
			for _, c := range conventions {
				text := fmt.Sprintf("- %s", c.Summary)
				t := estimateTokens(text)
				if convTokens+t > 500 {
					break
				}
				parts = append(parts, text)
				convTokens += t
				m.budget.Used += t
			}
			m.budget.Breakdown["conventions"] = convTokens
		}
	}

	result := ""
	for i, p := range parts {
		if i > 0 {
			result += "\n\n"
		}
		result += p
	}

	log.Printf("[memory] Context built for %s: %d tokens (budget: %d/%d)",
		task.ID, m.budget.Used, m.budget.Used, m.budget.Max)

	return result
}

// SetConvention adds a long-term project convention.
func (m *Manager) SetConvention(projectID, content string) error {
	entry := &Entry{
		ID:        "conv-" + genMemID(),
		ProjectID: projectID,
		Tier:      TierLong,
		Category:  "convention",
		Content:   content,
		Summary:   firstLine(content),
		Tokens:    estimateTokens(content),
		CreatedAt: time.Now(),
	}
	return m.store.Save(entry)
}

// GetStats returns memory statistics for a project.
func (m *Manager) GetStats(projectID string) (*Stats, error) {
	return m.store.Stats(projectID)
}

// TokenBudget returns the current token budget state.
func (m *Manager) TokenBudget() *TokenBudget {
	return &m.budget
}

// Clean removes all memory for a project.
func (m *Manager) Clean(projectID string) error {
	entries, err := m.store.Find(Query{ProjectID: projectID, Limit: 1000})
	if err != nil {
		return err
	}
	for _, e := range entries {
		m.store.Delete(e.ID)
	}
	return nil
}

// Find retrieves memory entries matching the given filters.
func (m *Manager) Find(projectID, module, category string, limit int) ([]*Entry, error) {
	return m.store.Find(Query{
		ProjectID: projectID,
		Module:    module,
		Category:  category,
		Limit:     limit,
	})
}

func genMemID() string {
	b := make([]byte, 6)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func sumTokens(entries []*Entry) int {
	total := 0
	for _, e := range entries {
		total += e.Tokens
	}
	return total
}
