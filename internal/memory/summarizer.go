package memory

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/tutuEic/macdp/internal/llm"
	"github.com/tutuEic/macdp/internal/store"
)

// Summarizer compresses task outputs and builds hierarchical summaries.
// Core technique from MemGPT/Letta: auto-compress to save tokens.
type Summarizer struct {
	llm *llm.Client
}

// NewSummarizer creates a new summarizer.
func NewSummarizer(llmClient *llm.Client) *Summarizer {
	return &Summarizer{llm: llmClient}
}

// summarizePrompt is the system prompt for task output compression.
// Optimized to produce dense, structured summaries in ≤300 tokens.
const summarizePrompt = `You are a code task summarizer. Compress the agent output into a dense technical summary.

Rules:
- MAX 300 tokens total
- Focus on: WHAT changed (files), KEY decisions made, ERRORS encountered and how fixed
- Use bullet points, no prose
- Skip filler, greetings, and obvious setup steps
- Format:

FILES: file1, file2, ...
CHANGES: brief description of what was implemented
DECISIONS: key design choices made
ISSUES: any problems encountered and resolutions
RESULT: success/failure + brief outcome

If the output contains ERROR or FAILED, note it in RESULT.`

// SummarizeTask compresses a completed task's output into a 200-300 token summary.
// This is the primary token-saving mechanism: 10K output → 300 token summary.
func (s *Summarizer) SummarizeTask(ctx context.Context, task *store.Task) (*Entry, error) {
	if task.Output == "" {
		return &Entry{
			ID:        "sum-" + task.ID,
			ProjectID: task.ProjectID,
			TaskID:    task.ID,
			Module:    task.Module,
			Tier:      TierShort,
			Category:  "summary",
			Content:   fmt.Sprintf("Task %s completed with no output.", task.Title),
			Summary:   fmt.Sprintf("Task %s: no output", task.Title),
			Tokens:    estimateTokens(fmt.Sprintf("Task %s completed.", task.Title)),
			CreatedAt: time.Now(),
		}, nil
	}

	// Truncate input if it's huge (save on the summarization call itself)
	input := task.Output
	if estimateTokens(input) > 8000 {
		input = input[:len(input)*8000/estimateTokens(input)] + "\n[...truncated]"
	}

	userPrompt := fmt.Sprintf("Task: %s\nModule: %s\nAgent: %s\n\nOutput:\n%s",
		task.Title, task.Module, task.AssignedAgent, input)

	summary, err := s.llm.GenerateText(ctx, summarizePrompt, userPrompt)
	if err != nil {
		log.Printf("[memory] Summarize failed for %s: %v, using fallback", task.ID, err)
		summary = fallbackSummary(task)
	}

	// Build memory entry
	entry := &Entry{
		ID:        "sum-" + task.ID,
		ProjectID: task.ProjectID,
		TaskID:    task.ID,
		Module:    task.Module,
		Tier:      TierShort,
		Category:  "summary",
		Content:   summary,
		Summary:   firstLine(summary),
		Tokens:    estimateTokens(summary),
		Metadata: Metadata{
			Tags:  []string{task.Module, task.Status.String(), task.AssignedAgent},
			Files: task.FilesChanged,
		},
		CreatedAt: time.Now(),
	}

	return entry, nil
}

// SummarizeModule compresses multiple task summaries into a module-level summary.
// Hierarchical compression: tasks → module (from MemGPT/Letta).
func (s *Summarizer) SummarizeModule(ctx context.Context, moduleName string, taskSummaries []*Entry) (*Entry, error) {
	if len(taskSummaries) == 0 {
		return nil, nil
	}

	var buf strings.Builder
	for _, e := range taskSummaries {
		buf.WriteString(fmt.Sprintf("- [%s] %s\n", e.TaskID, e.Content))
	}

	userPrompt := fmt.Sprintf("Module: %s\n\nTask summaries:\n%s", moduleName, buf.String())
	prompt := "Summarize the following task list into a module-level summary (max 200 tokens). Focus on: overall progress, architecture decisions, and patterns across tasks."

	summary, err := s.llm.GenerateText(ctx, prompt, userPrompt)
	if err != nil {
		log.Printf("[memory] Module summarize failed: %v", err)
		summary = fmt.Sprintf("Module %s: %d tasks completed.", moduleName, len(taskSummaries))
	}

	return &Entry{
		ID:        "mod-" + moduleName,
		ProjectID: taskSummaries[0].ProjectID,
		Module:    moduleName,
		Tier:      TierShort,
		Category:  "summary",
		Content:   summary,
		Summary:   firstLine(summary),
		Tokens:    estimateTokens(summary),
		CreatedAt: time.Now(),
	}, nil
}

// ExtractDecision creates a long-term memory entry for key architectural decisions.
func (s *Summarizer) ExtractDecision(ctx context.Context, task *store.Task) (*Entry, error) {
	if task.Output == "" {
		return nil, nil
	}

	prompt := `Extract any ARCHITECTURAL DECISIONS from this task output. If no significant decisions, return "NONE".

A decision is: choice of library/framework, API design pattern, data model change, or architectural trade-off.

Respond in JSON:
{"decisions": ["decision 1", "decision 2"]} or {"decisions": []}`

	var result struct {
		Decisions []string `json:"decisions"`
	}
	err := s.llm.GenerateJSON(ctx, prompt, task.Output, &result)
	if err != nil || len(result.Decisions) == 0 {
		return nil, nil
	}

	content := strings.Join(result.Decisions, "\n")
	return &Entry{
		ID:        "dec-" + task.ID,
		ProjectID: task.ProjectID,
		TaskID:    task.ID,
		Module:    task.Module,
		Tier:      TierLong,
		Category:  "decision",
		Content:   content,
		Summary:   firstLine(content),
		Tokens:    estimateTokens(content),
		Metadata: Metadata{
			Decisions: result.Decisions,
		},
		CreatedAt: time.Now(),
	}, nil
}

// fallbackSummary creates a basic summary without LLM call.
func fallbackSummary(task *store.Task) string {
	lines := strings.Split(task.Output, "\n")
	// Take first meaningful lines (max 5)
	var kept []string
	count := 0
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "```") {
			continue
		}
		if len(line) > 200 {
			line = line[:200] + "..."
		}
		kept = append(kept, line)
		count++
		if count >= 5 {
			break
		}
	}
	result := fmt.Sprintf("FILES: %s\nRESULT: %s", strings.Join(task.FilesChanged, ", "), task.Status)
	if len(kept) > 0 {
		result += "\nOUTPUT:\n" + strings.Join(kept, "\n")
	}
	return result
}

// firstLine returns the first non-empty line of text.
func firstLine(s string) string {
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			if len(line) > 100 {
				return line[:97] + "..."
			}
			return line
		}
	}
	return s[:min(len(s), 100)]
}

// estimateTokens gives a rough token count (≈ words × 1.3).
func estimateTokens(s string) int {
	words := len(strings.Fields(s))
	if words == 0 {
		return 0
	}
	return int(float64(words) * 1.3)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
