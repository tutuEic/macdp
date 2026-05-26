package store

import (
	"encoding/json"
	"time"
)

// Project represents a development project.
type Project struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	RepoPath    string    `json:"repo_path"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// TaskStatus enumeration.
type TaskStatus string

const (
	TaskPending  TaskStatus = "pending"
	TaskAssigned TaskStatus = "assigned"
	TaskRunning  TaskStatus = "running"
	TaskReview   TaskStatus = "review"
	TaskDone     TaskStatus = "done"
	TaskFailed   TaskStatus = "failed"
	TaskBlocked  TaskStatus = "blocked"
)

// Task represents a unit of work.
type Task struct {
	ID            string     `json:"id"`
	ProjectID     string     `json:"project_id"`
	Title         string     `json:"title"`
	Description   string     `json:"description"`
	Module        string     `json:"module"`
	Status        TaskStatus `json:"status"`
	Priority      int        `json:"priority"`
	AssignedAgent string     `json:"assigned_agent"`
	Reviewer      string     `json:"reviewer"`
	DependsOn     []string   `json:"depends_on"`
	Branch        string     `json:"branch"`
	Worktree      string     `json:"worktree"`
	Progress      int        `json:"progress"`
	Output        string     `json:"output"`
	FilesChanged  []string   `json:"files_changed"`
	CostUSD       float64    `json:"cost_usd"`
	MaxTurns      int        `json:"max_turns"`
	StartedAt     *time.Time `json:"started_at,omitempty"`
	CompletedAt   *time.Time `json:"completed_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

// AgentInfo stores agent connection info.
type AgentInfo struct {
	Name      string    `json:"name"`
	Type      string    `json:"type"` // hermes, openclaw, codex, claude-code
	Endpoint  string    `json:"endpoint"`
	Status    string    `json:"status"` // online, offline, busy, idle
	LastPing  time.Time `json:"last_ping"`
	CreatedAt time.Time `json:"created_at"`
}

// ChatMessage represents a message in agent conversation.
type ChatMessage struct {
	ID        string    `json:"id"`
	ProjectID string    `json:"project_id"`
	TaskID    string    `json:"task_id,omitempty"`
	AgentName string    `json:"agent_name"`
	Role      string    `json:"role"` // user, agent, system
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

// PlanResult stores a task plan from the LLM planner.
type PlanResult struct {
	ID        string          `json:"id"`
	ProjectID string          `json:"project_id"`
	Tasks     json.RawMessage `json:"tasks"`
	Summary   string          `json:"summary"`
	CreatedAt time.Time       `json:"created_at"`
}
