package orchestrator

import (
	"fmt"
	"sync"

	"github.com/tutuEic/macdp/internal/store"
)

// DAG represents a directed acyclic graph of tasks.
type DAG struct {
	Tasks    map[string]*store.Task
	Order    []string // topological order
	mu       sync.RWMutex
}

// NewDAG builds a DAG from tasks.
func NewDAG(tasks []*store.Task) (*DAG, error) {
	d := &DAG{Tasks: make(map[string]*store.Task, len(tasks))}
	for _, t := range tasks {
		d.Tasks[t.ID] = t
	}
	if err := d.validate(); err != nil {
		return nil, err
	}
	d.Order = d.topologicalSort()
	return d, nil
}

// validate checks for missing deps and cycles.
func (d *DAG) validate() error {
	// Check all deps exist
	for id, t := range d.Tasks {
		for _, dep := range t.DependsOn {
			if _, ok := d.Tasks[dep]; !ok {
				return fmt.Errorf("task %s depends on non-existent task %s", id, dep)
			}
		}
	}
	// Check for cycles via topological sort
	order := d.topologicalSort()
	if len(order) != len(d.Tasks) {
		return fmt.Errorf("cycle detected: only %d of %d tasks in order", len(order), len(d.Tasks))
	}
	return nil
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

// GetReady returns tasks whose dependencies are all done.
func (d *DAG) GetReady() []*store.Task {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var ready []*store.Task
	for _, id := range d.Order {
		t := d.Tasks[id]
		if t.Status != store.TaskPending {
			continue
		}
		allDone := true
		for _, depID := range t.DependsOn {
			dep := d.Tasks[depID]
			if dep.Status != store.TaskDone {
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
	d.mu.RLock()
	defer d.mu.RUnlock()
	for _, t := range d.Tasks {
		if t.Status != store.TaskDone && t.Status != store.TaskFailed {
			return false
		}
	}
	return true
}

// UpdateStatus updates a task status.
func (d *DAG) UpdateStatus(id string, status store.TaskStatus) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if t, ok := d.Tasks[id]; ok {
		t.Status = status
	}
}

// GetTask returns a task by ID.
func (d *DAG) GetTask(id string) *store.Task {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.Tasks[id]
}

// Layers returns tasks grouped by execution layer (for parallel execution).
func (d *DAG) Layers() [][]string {
	visited := make(map[string]bool)
	var layers [][]string

	for len(visited) < len(d.Tasks) {
		var layer []string
		for _, id := range d.Order {
			if visited[id] {
				continue
			}
			t := d.Tasks[id]
			allDepsVisited := true
			for _, dep := range t.DependsOn {
				if !visited[dep] {
					allDepsVisited = false
					break
				}
			}
			if allDepsVisited {
				layer = append(layer, id)
			}
		}
		if len(layer) == 0 {
			break // shouldn't happen if DAG is valid
		}
		for _, id := range layer {
			visited[id] = true
		}
		layers = append(layers, layer)
	}
	return layers
}
