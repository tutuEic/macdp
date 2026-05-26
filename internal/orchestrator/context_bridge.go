package orchestrator

import (
	"context"

	"github.com/tutuEic/macdp/internal/memory"
	"github.com/tutuEic/macdp/internal/store"
)

// ContextBridge builds task context using the memory manager.
// Replaces the old raw-dump approach with tiered memory retrieval.
type ContextBridge struct {
	store  *store.DB
	memory *memory.Manager
}

// NewContextBridge creates a context bridge backed by the memory manager.
func NewContextBridge(db *store.DB, mem *memory.Manager) *ContextBridge {
	return &ContextBridge{store: db, memory: mem}
}

// BuildContext assembles context for a task using the memory manager's tiered retrieval.
func (cb *ContextBridge) BuildContext(ctx context.Context, task *store.Task) string {
	// Load dependency tasks
	var deps []*store.Task
	for _, depID := range task.DependsOn {
		dep, err := cb.store.GetTask(depID)
		if err != nil {
			continue
		}
		deps = append(deps, dep)
	}

	// Use memory manager to build context with token budget
	return cb.memory.BuildContext(ctx, task, deps)
}
