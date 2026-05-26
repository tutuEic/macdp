package orchestrator

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/tutuEic/macdp/internal/agent"
	"github.com/tutuEic/macdp/internal/task"
)

// Scheduler manages task execution across multiple agents.
type Scheduler struct {
	dag      *task.DAG
	registry *agent.Registry
	mu       sync.Mutex

	// Config
	MaxParallel int
	TaskTimeout time.Duration
	RetryOnFail bool
	MaxRetries  int

	// Callbacks
	OnTaskStart    func(t *task.Task, agentName string)
	OnTaskComplete func(t *task.Task, result *agent.TaskResponse)
	OnTaskFailed   func(t *task.Task, err error)
	OnAllComplete  func(dag *task.DAG)
}

// NewScheduler creates a new task scheduler.
func NewScheduler(dag *task.DAG, registry *agent.Registry) *Scheduler {
	return &Scheduler{
		dag:         dag,
		registry:    registry,
		MaxParallel: 5,
		TaskTimeout: 10 * time.Minute,
		RetryOnFail: true,
		MaxRetries:  2,
	}
}

// Run executes all tasks in the DAG respecting dependencies and parallelism.
func (s *Scheduler) Run(ctx context.Context) error {
	sem := make(chan struct{}, s.MaxParallel)
	var wg sync.WaitGroup

	for {
		// Check if done
		if s.dag.IsComplete() {
			break
		}

		// Get tasks ready to execute
		ready := s.dag.Ready()
		if len(ready) == 0 {
			// All remaining tasks are either running or blocked
			time.Sleep(500 * time.Millisecond)
			continue
		}

		for _, t := range ready {
			wg.Add(1)
			sem <- struct{}{} // acquire slot

			go func(t *task.Task) {
				defer wg.Done()
				defer func() { <-sem }() // release slot

				s.executeTask(ctx, t)
			}(t)
		}

		// Wait a bit before checking again
		time.Sleep(1 * time.Second)
	}

	wg.Wait()

	if s.OnAllComplete != nil {
		s.OnAllComplete(s.dag)
	}
	return nil
}

// executeTask assigns and runs a single task on the best available agent.
func (s *Scheduler) executeTask(ctx context.Context, t *task.Task) {
	// Find best agent
	agentName := s.bestAgentFor(t)
	if agentName == "" {
		s.markFailed(t, fmt.Errorf("no suitable agent available for task %s", t.ID))
		return
	}

	adapter := s.registry.Get(agentName)
	if adapter == nil {
		s.markFailed(t, fmt.Errorf("agent %s not found in registry", agentName))
		return
	}

	// Update task status
	s.mu.Lock()
	t.Status = task.StatusAssigned
	t.Agent = agentName
	now := time.Now()
	t.StartedAt = &now
	s.mu.Unlock()

	if s.OnTaskStart != nil {
		s.OnTaskStart(t, agentName)
	}

	// Build request
	req := &agent.TaskRequest{
		TaskID:      t.ID,
		Title:       t.Title,
		Description: t.Description,
		Workdir:     t.Worktree,
		MaxTurns:    t.MaxTurns,
		Timeout:     s.TaskTimeout,
	}

	// Execute with timeout
	taskCtx, cancel := context.WithTimeout(ctx, s.TaskTimeout)
	defer cancel()

	s.mu.Lock()
	t.Status = task.StatusRunning
	s.mu.Unlock()

	resp, err := adapter.Execute(taskCtx, req)

	if err != nil {
		if s.RetryOnFail && t.RetryCount < s.MaxRetries {
			t.RetryCount++
			t.Status = task.StatusPending
			log.Printf("[scheduler] Task %s failed, retry %d/%d: %v", t.ID, t.RetryCount, s.MaxRetries, err)
			return
		}
		s.markFailed(t, err)
		return
	}

	// Mark complete
	s.mu.Lock()
	completedAt := time.Now()
	t.CompletedAt = &completedAt
	t.CostUSD = resp.CostUSD
	if resp.Success {
		t.Status = task.StatusDone
	} else {
		t.Status = task.StatusFailed
	}
	s.mu.Unlock()

	if s.OnTaskComplete != nil {
		s.OnTaskComplete(t, resp)
	}

	log.Printf("[scheduler] Task %s completed by %s in %s (cost: $%.4f)",
		t.ID, agentName, resp.Duration, resp.CostUSD)
}

// bestAgentFor selects the best agent for a given task based on capabilities.
func (s *Scheduler) bestAgentFor(t *task.Task) string {
	// If task has a pre-assigned agent, use it
	if t.Agent != "" {
		a := s.registry.Get(t.Agent)
		if a != nil && a.Status() == agent.StatusIdle {
			return t.Agent
		}
	}

	// Otherwise, find the best available agent
	available := s.registry.Available()
	if len(available) == 0 {
		return ""
	}

	// Simple strategy: prefer claude-code for complex tasks, codex for simple ones
	// TODO: implement capability-based matching
	priority := []string{"claude-code", "hermes", "codex", "opencode"}
	for _, name := range priority {
		for _, a := range available {
			if a.Name() == name {
				return name
			}
		}
	}

	// Fallback to first available
	return available[0].Name()
}

func (s *Scheduler) markFailed(t *task.Task, err error) {
	s.mu.Lock()
	t.Status = task.StatusFailed
	completedAt := time.Now()
	t.CompletedAt = &completedAt
	s.mu.Unlock()

	log.Printf("[scheduler] Task %s failed: %v", t.ID, err)

	if s.OnTaskFailed != nil {
		s.OnTaskFailed(t, err)
	}
}
