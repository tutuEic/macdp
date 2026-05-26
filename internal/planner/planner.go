package planner

import (
	"context"
	"fmt"
	"time"

	"github.com/tutuEic/macdp/internal/llm"
	"github.com/tutuEic/macdp/internal/store"
)

type Planner struct {
	llm *llm.Client
}

func New(llmClient *llm.Client) *Planner {
	return &Planner{llm: llmClient}
}

type PlannedTask struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Module      string   `json:"module"`
	Agent       string   `json:"agent"`
	DependsOn   []string `json:"depends_on"`
	Priority    int      `json:"priority"`
	MaxTurns    int      `json:"max_turns"`
}

type PlanResult struct {
	Tasks   []PlannedTask `json:"tasks"`
	Summary string        `json:"summary"`
}

var systemPrompt = "You are a software project task decomposition expert. Given a user requirement, break it down into concrete, actionable sub-tasks.\n\nRules:\n1. Each task should be independently executable by an AI coding agent\n2. Define clear dependency relationships (which tasks must complete before others)\n3. Assign each task to the best-fit agent:\n   - \"hermes\": project setup, testing, debugging, general tasks, database setup\n   - \"claude-code\": complex coding, API implementation, code review, refactoring\n   - \"codex\": frontend components, simple scripts, quick prototypes\n4. Estimate complexity (max_turns: 5-25)\n5. Group tasks by module (backend, frontend, database, testing, etc.)\n\nOutput STRICT JSON format:\n{\"tasks\": [{\"id\": \"T1\", \"title\": \"short title\", \"description\": \"detailed description\", \"module\": \"backend|frontend|database|testing\", \"agent\": \"hermes|claude-code|codex\", \"depends_on\": [], \"priority\": 1, \"max_turns\": 10}], \"summary\": \"brief summary of the plan\"}"

func (p *Planner) Plan(ctx context.Context, requirement string, projectCtx string) (*PlanResult, error) {
	userPrompt := requirement
	if projectCtx != "" {
		userPrompt = fmt.Sprintf("Project context:\n%s\n\nRequirement: %s", projectCtx, requirement)
	}

	var result PlanResult
	err := p.llm.GenerateJSON(ctx, systemPrompt, userPrompt, &result)
	if err != nil {
		return nil, fmt.Errorf("planner LLM error: %w", err)
	}

	ids := make(map[string]bool)
	for _, t := range result.Tasks {
		ids[t.ID] = true
	}
	for _, t := range result.Tasks {
		for _, dep := range t.DependsOn {
			if !ids[dep] {
				return nil, fmt.Errorf("task %s depends on non-existent task %s", t.ID, dep)
			}
		}
	}

	for i := range result.Tasks {
		if result.Tasks[i].MaxTurns == 0 {
			result.Tasks[i].MaxTurns = 15
		}
		if result.Tasks[i].Agent == "" {
			result.Tasks[i].Agent = "hermes"
		}
	}

	return &result, nil
}

func PlanToTasks(plan *PlanResult, projectID string) []*store.Task {
	tasks := make([]*store.Task, 0, len(plan.Tasks))
	now := time.Now()
	for _, pt := range plan.Tasks {
		tasks = append(tasks, &store.Task{
			ID:          pt.ID,
			ProjectID:   projectID,
			Title:       pt.Title,
			Description: pt.Description,
			Module:      pt.Module,
			Status:      store.TaskPending,
			Priority:    pt.Priority,
			DependsOn:   pt.DependsOn,
			MaxTurns:    pt.MaxTurns,
			CreatedAt:   now,
		})
	}
	return tasks
}
