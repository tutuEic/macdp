package agent

import (
	"context"
	"time"
)

// AgentType enumerates supported agent types.
type AgentType string

const (
	Hermes    AgentType = "hermes"
	OpenClaw  AgentType = "openclaw"
	Codex     AgentType = "codex"
	ClaudeCode AgentType = "claude-code"
)

// AgentStatus represents current agent state.
type AgentStatus struct {
	Online     bool      `json:"online"`
	State      string    `json:"state"` // idle, busy, error
	CurrentTask string   `json:"current_task,omitempty"`
	LastPing   time.Time `json:"last_ping"`
}

// TaskEvent is emitted during task execution.
type TaskEvent struct {
	Type     string `json:"type"` // progress, output, file_change, complete, error
	TaskID   string `json:"task_id"`
	Agent    string `json:"agent"`
	Content  string `json:"content"`
	Progress int    `json:"progress"`
	File     string `json:"file,omitempty"`
}

// ChatMessage is a message in agent conversation.
type ChatMessage struct {
	Role    string `json:"role"` // user, agent, system
	Content string `json:"content"`
}

// TaskRequest is what we send to an agent.
type TaskRequest struct {
	TaskID      string `json:"task_id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Context     string `json:"context"` // upstream task outputs, project conventions
	Workdir     string `json:"workdir"`
	MaxTurns    int    `json:"max_turns"`
}

// AgentConnector is the interface for connecting to external agent services.
type AgentConnector interface {
	// Name returns the agent identifier.
	Name() string

	// Type returns the agent type.
	Type() AgentType

	// Status returns current agent status.
	Status() AgentStatus

	// Connect establishes connection to the agent service.
	Connect(ctx context.Context, config map[string]string) error

	// Disconnect closes the connection.
	Disconnect() error

	// Ping checks if the agent is alive.
	Ping(ctx context.Context) error

	// ExecuteTask sends a task to the agent and returns an event stream.
	ExecuteTask(ctx context.Context, req *TaskRequest) (<-chan *TaskEvent, error)

	// CancelTask cancels a running task.
	CancelTask(taskID string) error

	// SendMessage sends a chat message and gets a response.
	SendMessage(ctx context.Context, msg string) (<-chan *ChatMessage, error)
}

// Registry holds all registered agent connectors.
type Registry struct {
	agents map[string]AgentConnector
}

// NewRegistry creates an empty registry.
func NewRegistry() *Registry {
	return &Registry{agents: make(map[string]AgentConnector)}
}

// Register adds a connector.
func (r *Registry) Register(c AgentConnector) {
	r.agents[c.Name()] = c
}

// Get returns a connector by name.
func (r *Registry) Get(name string) AgentConnector {
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
