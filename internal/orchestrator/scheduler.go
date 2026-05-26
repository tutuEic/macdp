package orchestrator

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/tutuEic/macdp/internal/agent"
	"github.com/tutuEic/macdp/internal/event"
	"github.com/tutuEic/macdp/internal/memory"
	"github.com/tutuEic/macdp/internal/store"
)

// Scheduler executes tasks on the DAG using available agents.
type Scheduler struct {
	dag         *DAG
	agents      *agent.Registry
	bus         *event.EventBus
	store       *store.DB
	bridge      *ContextBridge
	memory      *memory.Manager
	maxParallel int
	taskTimeout time.Duration
}

// NewScheduler creates a new scheduler.
func NewScheduler(dag *DAG, agents *agent.Registry, bus *event.EventBus, db *store.DB, mem *memory.Manager) *Scheduler {
	return &Scheduler{
		dag:         dag,
		agents:      agents,
		bus:         bus,
		store:       db,
		bridge:      NewContextBridge(db, mem),
		memory:      mem,
		maxParallel: 5,
		taskTimeout: 10 * time.Minute,
	}
}

// RunLayers executes tasks layer by layer, parallel within each layer.
// Uses semaphore to cap concurrent goroutines per layer.
func (s *Scheduler) RunLayers(ctx context.Context) error {
	layers := s.dag.Layers()

	for i, layer := range layers {
		log.Printf("[scheduler] Executing layer %d/%d: %v", i+1, len(layers), layer)

		sem := make(chan struct{}, s.maxParallel)
		var wg sync.WaitGroup
		var mu sync.Mutex
		var failedTasks []string

		for _, taskID := range layer {
			t := s.dag.GetTask(taskID)
			if t == nil {
				continue
			}

			wg.Add(1)
			sem <- struct{}{} // acquire slot

			go func(task *store.Task) {
				defer wg.Done()
				defer func() { <-sem }() // release slot

				s.executeTask(ctx, task)

				mu.Lock()
				if task.Status == store.TaskFailed {
					failedTasks = append(failedTasks, task.ID)
				}
				mu.Unlock()
			}(t)
		}
		wg.Wait()

		if len(failedTasks) > 0 {
			log.Printf("[scheduler] Layer %d: %d tasks failed: %v", i+1, len(failedTasks), failedTasks)
		}
	}

	return nil
}

func (s *Scheduler) executeTask(ctx context.Context, t *store.Task) {
	// Apply task-level timeout
	taskCtx, cancel := context.WithTimeout(ctx, s.taskTimeout)
	defer cancel()

	// 1. Select agent
	agentName := s.selectAgent(t)
	if agentName == "" {
		s.markFailed(t, "no suitable agent available")
		return
	}

	conn := s.agents.Get(agentName)
	if conn == nil {
		s.markFailed(t, fmt.Sprintf("agent %s not found", agentName))
		return
	}

	// 2. Build context using memory manager (tiered + token-budgeted)
	taskContext := s.bridge.BuildContext(taskCtx, t)

	// 3. Update status
	t.Status = store.TaskRunning
	t.AssignedAgent = agentName
	s.store.UpdateTaskStatus(t.ID, store.TaskRunning)
	s.store.AssignTask(t.ID, agentName)
	s.bus.Emit(event.TaskStarted, "scheduler", map[string]string{
		"task_id": t.ID, "agent": agentName, "title": t.Title,
	})

	// 4. Execute
	req := &agent.TaskRequest{
		TaskID:      t.ID,
		Title:       t.Title,
		Description: t.Description,
		Context:     taskContext,
		Workdir:     t.Worktree,
		MaxTurns:    t.MaxTurns,
	}

	events, err := conn.ExecuteTask(taskCtx, req)
	if err != nil {
		s.markFailed(t, err.Error())
		return
	}

	// 5. Process events (with timeout-awareness)
	var output string
loop:
	for {
		select {
		case ev, ok := <-events:
			if !ok {
				break loop
			}
			switch ev.Type {
			case "output":
				output += ev.Content + "\n"
				s.bus.Emit(event.AgentOutput, agentName, ev)
			case "progress":
				t.Progress = ev.Progress
				s.store.UpdateTaskProgress(t.ID, ev.Progress, output)
				s.bus.Emit(event.TaskProgress, "scheduler", ev)
			case "file_change":
				t.FilesChanged = append(t.FilesChanged, ev.File)
				s.bus.Emit(event.FileChanged, agentName, ev)
			case "complete":
				t.Status = store.TaskDone
				t.Output = output
				t.Progress = 100
				s.store.UpdateTaskStatus(t.ID, store.TaskDone)
				s.store.UpdateTaskProgress(t.ID, 100, output)
				s.bus.Emit(event.TaskCompleted, "scheduler", map[string]string{
					"task_id": t.ID, "agent": agentName, "title": t.Title,
				})
				log.Printf("[scheduler] ✓ Task %s completed by %s", t.ID, agentName)
				break loop
			case "error":
				s.markFailed(t, ev.Content)
				// Still save memory for failed tasks
				s.memory.OnTaskComplete(context.Background(), t)
				break loop
			}
		case <-taskCtx.Done():
			s.markFailed(t, fmt.Sprintf("timeout after %v: %v", s.taskTimeout, taskCtx.Err()))
			break loop
		}
	}

	// If no complete/error event was received, mark as done
	if t.Status == store.TaskRunning {
		t.Status = store.TaskDone
		t.Output = output
		s.store.UpdateTaskStatus(t.ID, store.TaskDone)
		s.bus.Emit(event.TaskCompleted, "scheduler", map[string]string{
			"task_id": t.ID, "agent": agentName,
		})
	}

	// 6. Update memory: auto-summarize output, extract decisions, log file changes
	if output != "" || len(t.FilesChanged) > 0 {
		s.memory.OnTaskComplete(context.Background(), t)
	}
}

func (s *Scheduler) selectAgent(t *store.Task) string {
	// If task has a pre-assigned agent, try it first
	if t.AssignedAgent != "" {
		conn := s.agents.Get(t.AssignedAgent)
		if conn != nil {
			st := conn.Status()
			if st.Online && st.State == "idle" {
				return t.AssignedAgent
			}
		}
	}

	// Module-based routing
	moduleAgentMap := map[string]string{
		"frontend": "codex",
		"backend":  "claude-code",
		"database": "hermes",
		"testing":  "hermes",
		"devops":   "hermes",
	}

	if preferred, ok := moduleAgentMap[t.Module]; ok {
		conn := s.agents.Get(preferred)
		if conn != nil {
			st := conn.Status()
			if st.Online && st.State == "idle" {
				return preferred
			}
		}
	}

	// Fallback: any idle agent
	priority := []string{"claude-code", "hermes", "codex"}
	for _, name := range priority {
		conn := s.agents.Get(name)
		if conn != nil {
			st := conn.Status()
			if st.Online && st.State == "idle" {
				return name
			}
		}
	}

	return ""
}

func (s *Scheduler) markFailed(t *store.Task, reason string) {
	t.Status = store.TaskFailed
	t.Output = "FAILED: " + reason
	s.store.UpdateTaskStatus(t.ID, store.TaskFailed)
	s.bus.Emit(event.TaskFailed, "scheduler", map[string]string{
		"task_id": t.ID, "error": reason,
	})
	log.Printf("[scheduler] ✗ Task %s failed: %s", t.ID, reason)
}
