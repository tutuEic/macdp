// Package agent defines the agent adapter interface and implementations
// for integrating various AI coding agents into MACDP.
package agent

import (
	"context"
	"time"
)

// Event represents a real-time event from an agent during execution.
type Event struct {
	Type      EventType `json:"type"`
	Agent     string    `json:"agent"`
	TaskID    string    `json:"task_id"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

// EventType categorizes agent events.
type EventType string

const (
	EventStdout   EventType = "stdout"
	EventStderr   EventType = "stderr"
	EventToolCall EventType = "tool_call"
	EventResult   EventType = "result"
	EventError    EventType = "error"
	EventProgress EventType = "progress"
)

// Status represents the current state of an agent.
type Status string

const (
	StatusIdle     Status = "idle"
	StatusBusy     Status = "busy"
	StatusError    Status = "error"
	StatusDisabled Status = "disabled"
)

// Config holds configuration for a specific agent.
type Config struct {
	Name         string   `yaml:"name"`
	Entrypoint   string   `yaml:"entrypoint"`
	Flags        []string `yaml:"flags"`
	MaxConcurrent int     `yaml:"max_concurrent"`
	Strengths    []string `yaml:"strengths"`
	Enabled      bool     `yaml:"enabled"`
}

// TaskRequest is what the orchestrator sends to an agent.
type TaskRequest struct {
	TaskID      string            `json:"task_id"`
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Context     map[string]string `json:"context,omitempty"` // extra context (schema, conventions, etc.)
	Workdir     string            `json:"workdir"`           // working directory (worktree path)
	MaxTurns    int               `json:"max_turns,omitempty"`
	Timeout     time.Duration     `json:"timeout,omitempty"`
	AllowedTools []string         `json:"allowed_tools,omitempty"`
}

// TaskResponse is what an agent returns after execution.
type TaskResponse struct {
	TaskID        string   `json:"task_id"`
	Success       bool     `json:"success"`
	Summary       string   `json:"summary"`
	FilesChanged  []string `json:"files_changed,omitempty"`
	GitBranch     string   `json:"git_branch,omitempty"`
	CostUSD       float64  `json:"cost_usd,omitempty"`
	Duration      string   `json:"duration,omitempty"`
	Error         string   `json:"error,omitempty"`
	RawOutput     string   `json:"raw_output,omitempty"`
}

// AgentAdapter is the interface that all agent implementations must satisfy.
type AgentAdapter interface {
	// Name returns the agent's identifier (e.g., "hermes", "codex", "claude-code").
	Name() string

	// Status returns the agent's current availability status.
	Status() Status

	// Execute runs a task and returns the result synchronously.
	Execute(ctx context.Context, req *TaskRequest) (*TaskResponse, error)

	// ExecuteAsync starts a task and returns a channel of events.
	// The channel is closed when the task completes or fails.
	ExecuteAsync(ctx context.Context, req *TaskRequest) (<-chan *Event, error)

	// Cancel terminates a running task.
	Cancel(taskID string) error

	// Capabilities returns what this agent can do.
	Capabilities() []string
}

// Registry holds all registered agent adapters.
type Registry struct {
	agents map[string]AgentAdapter
}

// NewRegistry creates an empty agent registry.
func NewRegistry() *Registry {
	return &Registry{
		agents: make(map[string]AgentAdapter),
	}
}

// Register adds an agent adapter to the registry.
func (r *Registry) Register(a AgentAdapter) {
	r.agents[a.Name()] = a
}

// Get returns an agent by name, or nil if not found.
func (r *Registry) Get(name string) AgentAdapter {
	return r.agents[name]
}

// List returns all registered agent names.
func (r *Registry) List() []string {
	names := make([]string, 0, len(r.agents))
	for name := range r.agents {
		names = append(names, name)
	}
	return names
}

// Available returns agents that are currently idle.
func (r *Registry) Available() []AgentAdapter {
	var available []AgentAdapter
	for _, a := range r.agents {
		if a.Status() == StatusIdle {
			available = append(available, a)
		}
	}
	return available
}
