// Package task defines the core data structures for MACDP tasks.
package task

import (
	"encoding/json"
	"time"
)

// Status represents the current state of a task.
type Status string

const (
	StatusPending  Status = "pending"
	StatusAssigned Status = "assigned"
	StatusRunning  Status = "running"
	StatusReview   Status = "review"
	StatusFailed   Status = "failed"
	StatusDone     Status = "done"
	StatusBlocked  Status = "blocked"
)

// Task represents a unit of work to be executed by an agent.
type Task struct {
	ID          string          `json:"id"`
	ProjectID   string          `json:"project_id"`
	ParentID    string          `json:"parent_id,omitempty"`
	Title       string          `json:"title"`
	Description string          `json:"description"`
	Status      Status          `json:"status"`
	Agent       string          `json:"agent,omitempty"`        // assigned agent name
	Worktree    string          `json:"worktree,omitempty"`     // worktree path
	Branch      string          `json:"branch,omitempty"`       // git branch name
	DependsOn   []string        `json:"depends_on,omitempty"`   // task IDs this depends on
	Result      json.RawMessage `json:"result,omitempty"`       // agent output
	CostUSD     float64         `json:"cost_usd,omitempty"`
	MaxTurns    int             `json:"max_turns,omitempty"`
	Timeout     time.Duration   `json:"timeout,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
	StartedAt   *time.Time      `json:"started_at,omitempty"`
	CompletedAt *time.Time      `json:"completed_at,omitempty"`
	RetryCount  int             `json:"retry_count,omitempty"`
	MaxRetries  int             `json:"max_retries,omitempty"`
}

// TaskResult is what an agent returns after executing a task.
type TaskResult struct {
	TaskID        string   `json:"task_id"`
	Status        Status   `json:"status"` // success or failed
	Summary       string   `json:"summary"`
	FilesChanged  []string `json:"files_changed,omitempty"`
	GitBranch     string   `json:"git_branch,omitempty"`
	CostUSD       float64  `json:"cost_usd,omitempty"`
	Duration      string   `json:"duration,omitempty"`
	Error         string   `json:"error,omitempty"`
	RetrySuggested bool    `json:"retry_suggested,omitempty"`
}

// DAG represents a directed acyclic graph of tasks.
type DAG struct {
	Tasks map[string]*Task `json:"tasks"`
	Order []string         `json:"order"` // topological order
}

// NewDAG creates a DAG from a slice of tasks.
func NewDAG(tasks []*Task) *DAG {
	d := &DAG{
		Tasks: make(map[string]*Task, len(tasks)),
	}
	for _, t := range tasks {
		d.Tasks[t.ID] = t
	}
	d.Order = d.topologicalSort()
	return d
}

// Ready returns tasks whose dependencies are all met.
func (d *DAG) Ready() []*Task {
	var ready []*Task
	for _, id := range d.Order {
		t := d.Tasks[id]
		if t.Status != StatusPending {
			continue
		}
		allDone := true
		for _, depID := range t.DependsOn {
			dep, ok := d.Tasks[depID]
			if !ok || dep.Status != StatusDone {
				allDone = false
				break
			}
		}
		if allDone {
			ready = append(ready, t)
		}
	}
	return ready
}

// IsComplete returns true if all tasks are done or failed.
func (d *DAG) IsComplete() bool {
	for _, t := range d.Tasks {
		if t.Status != StatusDone && t.Status != StatusFailed {
			return false
		}
	}
	return true
}

// MarkComplete updates a task's status and records the result.
func (d *DAG) MarkComplete(taskID string, result *TaskResult) {
	t, ok := d.Tasks[taskID]
	if !ok {
		return
	}
	now := time.Now()
	t.CompletedAt = &now
	if result.Status == StatusDone {
		t.Status = StatusDone
	} else {
		t.Status = StatusFailed
	}
	t.CostUSD = result.CostUSD
	raw, _ := json.Marshal(result)
	t.Result = raw
}

// topologicalSort returns task IDs in dependency order.
func (d *DAG) topologicalSort() []string {
	inDegree := make(map[string]int)
	adj := make(map[string][]string)

	for id := range d.Tasks {
		if _, ok := inDegree[id]; !ok {
			inDegree[id] = 0
		}
		for _, dep := range d.Tasks[id].DependsOn {
			adj[dep] = append(adj[dep], id)
			inDegree[id]++
		}
	}

	var queue []string
	for id, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, id)
		}
	}

	var order []string
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		order = append(order, node)
		for _, next := range adj[node] {
			inDegree[next]--
			if inDegree[next] == 0 {
				queue = append(queue, next)
			}
		}
	}
	return order
}
