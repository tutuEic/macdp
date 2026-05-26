package memory

import "time"

// Tier represents the memory storage tier.
type Tier string

const (
	TierWorking  Tier = "working"  // Always in context (≤2K tokens)
	TierShort    Tier = "short"    // Retrieved on demand (summaries, file changes)
	TierLong     Tier = "long"     // Persistent patterns, conventions, decisions
)

// Entry is a single memory record.
type Entry struct {
	ID        string    `json:"id"`
	ProjectID string    `json:"project_id"`
	TaskID    string    `json:"task_id,omitempty"`
	Module    string    `json:"module,omitempty"`
	Tier      Tier      `json:"tier"`
	Category  string    `json:"category"`  // summary, decision, convention, file_change, pattern
	Content   string    `json:"content"`   // The actual memory text
	Summary   string    `json:"summary"`   // Compressed version (for listing)
	Tokens    int       `json:"tokens"`    // Estimated token count
	Metadata  Metadata  `json:"metadata,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// Metadata stores structured tags for retrieval.
type Metadata struct {
	Tags       []string          `json:"tags,omitempty"`
	Files      []string          `json:"files,omitempty"`
	Decisions  []string          `json:"decisions,omitempty"`
	Custom     map[string]string `json:"custom,omitempty"`
}

// TokenBudget tracks context window usage.
type TokenBudget struct {
	Max       int    `json:"max"`
	Used      int    `json:"used"`
	Breakdown map[string]int `json:"breakdown"` // "system": 500, "working": 1200, "retrieved": 800
}

// Stats holds aggregated memory statistics.
type Stats struct {
	TotalEntries  int            `json:"total_entries"`
	TotalTokens   int            `json:"total_tokens"`
	ByTier        map[Tier]int   `json:"by_tier"`
	ByCategory    map[string]int `json:"by_category"`
}

// ContextConfig controls how context is assembled for a task.
type ContextConfig struct {
	MaxWorkingTokens    int  `json:"max_working_tokens"`    // Default: 2000
	MaxRetrievedTokens  int  `json:"max_retrieved_tokens"`  // Default: 3000
	MaxRetrievedEntries int  `json:"max_retrieved_entries"` // Default: 5
	IncludeFiles        bool `json:"include_files"`         // Include file change logs
	IncludeDecisions    bool `json:"include_decisions"`     // Include past decisions
	IncludeConventions  bool `json:"include_conventions"`   // Include project conventions
}
