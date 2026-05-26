package memory

import (
	"fmt"
	"sort"
	"strings"
)

// Retriever fetches relevant memories for context assembly.
// Uses keyword+module matching (embedding-based retrieval is the next upgrade path).
type Retriever struct {
	store *Store
}

// NewRetriever creates a new retriever.
func NewRetriever(store *Store) *Retriever {
	return &Retriever{store: store}
}

// Result holds a retrieved memory with its relevance score.
type Result struct {
	Entry *Entry
	Score float64
}

// Retrieve finds the most relevant memories for a given task context.
// Strategy: multi-pass with progressive narrowing to stay within token budget.
func (r *Retriever) Retrieve(projectID, module, taskDesc string, cfg ContextConfig) []*Entry {
	var allResults []Result

	// Pass 1: Same module summaries (highest relevance)
	moduleEntries, _ := r.store.Find(Query{
		ProjectID: projectID,
		Module:    module,
		Tier:      TierShort,
		Limit:     cfg.MaxRetrievedEntries * 2,
	})
	for _, e := range moduleEntries {
		score := 0.8 // same module: high base score
		score += keywordScore(taskDesc, e.Content) * 0.2
		allResults = append(allResults, Result{Entry: e, Score: score})
	}

	// Pass 2: Other module summaries from same project
	otherEntries, _ := r.store.Find(Query{
		ProjectID: projectID,
		Tier:      TierShort,
		Limit:     cfg.MaxRetrievedEntries * 2,
	})
	for _, e := range otherEntries {
		if e.Module == module {
			continue // already covered in pass 1
		}
		score := 0.3 // different module: lower base
		score += keywordScore(taskDesc, e.Content) * 0.5
		allResults = append(allResults, Result{Entry: e, Score: score})
	}

	// Pass 3: Long-term decisions and conventions
	if cfg.IncludeDecisions || cfg.IncludeConventions {
		category := ""
		if cfg.IncludeDecisions {
			category = "decision"
		}
		longEntries, _ := r.store.Find(Query{
			ProjectID: projectID,
			Category:  category,
			Tier:      TierLong,
			Limit:     cfg.MaxRetrievedEntries,
		})
		for _, e := range longEntries {
			score := 0.5
			if cfg.IncludeConventions && e.Category == "convention" {
				score = 0.6
			}
			allResults = append(allResults, Result{Entry: e, Score: score})
		}
	}

	// Sort by relevance score descending
	sort.Slice(allResults, func(i, j int) bool {
		return allResults[i].Score > allResults[j].Score
	})

	// Apply token budget: only include up to max tokens/entries
	var selected []*Entry
	totalTokens := 0
	for _, r := range allResults {
		if len(selected) >= cfg.MaxRetrievedEntries {
			break
		}
		if totalTokens+r.Entry.Tokens > cfg.MaxRetrievedTokens {
			continue
		}
		selected = append(selected, r.Entry)
		totalTokens += r.Entry.Tokens
	}

	return selected
}

// BuildWorkingContext assembles the working memory block for a task prompt.
// This is what gets injected into every agent call's context.
func (r *Retriever) BuildWorkingContext(taskDeps []DependencyInfo, recentDecisions []*Entry, cfg ContextConfig) string {
	tokenBudget := cfg.MaxWorkingTokens
	if tokenBudget <= 0 {
		tokenBudget = 2000
	}

	var parts []string
	tokensUsed := 0

	// 1. Dependency outputs (most important, include first)
	if len(taskDeps) > 0 {
		parts = append(parts, "## Dependency Outputs")
		for _, dep := range taskDeps {
			// Truncate each dependency output to fit budget
			depText := fmt.Sprintf("### %s (%s)\n%s", dep.Title, dep.Status, truncateToTokens(dep.Output, 400))
			depTokens := estimateTokens(depText)
			if tokensUsed+depTokens > tokenBudget/2 {
				depText = fmt.Sprintf("### %s (%s)\n[summary: %s]", dep.Title, dep.Status, truncateToTokens(dep.Summary, 100))
				depTokens = estimateTokens(depText)
			}
			parts = append(parts, depText)
			tokensUsed += depTokens
		}
	}

	// 2. Recent decisions
	if len(recentDecisions) > 0 {
		parts = append(parts, "## Recent Decisions")
		for _, d := range recentDecisions {
			decText := fmt.Sprintf("- %s", d.Summary)
			decTokens := estimateTokens(decText)
			if tokensUsed+decTokens > tokenBudget {
				break
			}
			parts = append(parts, decText)
			tokensUsed += decTokens
		}
	}

	return strings.Join(parts, "\n\n")
}

// DependencyInfo is a lightweight view of a completed dependency.
type DependencyInfo struct {
	Title   string
	Status  string
	Output  string
	Summary string
	Module  string
}

// keywordScore computes a simple TF-based relevance score.
// Upgrade path: replace with embedding cosine similarity.
func keywordScore(query, document string) float64 {
	queryWords := tokenize(query)
	if len(queryWords) == 0 {
		return 0
	}

	docLower := strings.ToLower(document)
	hits := 0
	for _, w := range queryWords {
		if strings.Contains(docLower, w) {
			hits++
		}
	}
	return float64(hits) / float64(len(queryWords))
}

func tokenize(s string) []string {
	words := strings.Fields(strings.ToLower(s))
	// Filter out short/common words
	var filtered []string
	for _, w := range words {
		if len(w) > 2 {
			filtered = append(filtered, w)
		}
	}
	return filtered
}

func truncateToTokens(s string, maxTokens int) string {
	words := strings.Fields(s)
	target := maxTokens * 3 / 4 // rough: token ≈ 0.75 * word count
	if len(words) <= target {
		return s
	}
	return strings.Join(words[:target], " ") + "..."
}
