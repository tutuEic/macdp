package review

import (
	"context"
	"fmt"
	"log"

	"github.com/tutuEic/macdp/internal/agent"
	"github.com/tutuEic/macdp/internal/event"
	"github.com/tutuEic/macdp/internal/llm"
	"github.com/tutuEic/macdp/internal/store"
)

type Verdict string

const (
	VerdictApprove Verdict = "approve"
	VerdictChanges Verdict = "request_changes"
	VerdictDiscuss Verdict = "needs_discussion"
)

type Result struct {
	TaskID   string  `json:"task_id"`
	Reviewer string  `json:"reviewer"`
	Verdict  Verdict `json:"verdict"`
	Score    int     `json:"score"`
	Comments string  `json:"comments"`
}

type Pipeline struct {
	llm    *llm.Client
	agents *agent.Registry
	bus    *event.EventBus
}

func NewPipeline(llmClient *llm.Client, agents *agent.Registry, bus *event.EventBus) *Pipeline {
	return &Pipeline{llm: llmClient, agents: agents, bus: bus}
}

var reviewPrompt = "You are a code reviewer. Analyze the following task output and provide a review.\n\nTask: %s\nDescription: %s\nOutput:\n%s\n\nReview criteria:\n1. Code correctness and completeness\n2. Error handling\n3. Code style and readability\n4. Test coverage (if applicable)\n\nRespond in JSON:\n{\"verdict\": \"approve|request_changes|needs_discussion\", \"score\": 0-100, \"comments\": \"detailed review comments\"}"

func (p *Pipeline) Run(ctx context.Context, task *store.Task) *Result {
	if task.Output == "" {
		return &Result{TaskID: task.ID, Verdict: VerdictApprove, Score: 100, Comments: "No output to review"}
	}

	prompt := fmt.Sprintf(reviewPrompt, task.Title, task.Description, task.Output)

	var review struct {
		Verdict  string `json:"verdict"`
		Score    int    `json:"score"`
		Comments string `json:"comments"`
	}

	err := p.llm.GenerateJSON(ctx, "You are a senior code reviewer.", prompt, &review)
	if err != nil {
		log.Printf("[review] LLM review failed for %s: %v", task.ID, err)
		return &Result{TaskID: task.ID, Verdict: VerdictApprove, Score: 50, Comments: "LLM review failed: " + err.Error()}
	}

	result := &Result{
		TaskID:   task.ID,
		Reviewer: "llm",
		Verdict:  Verdict(review.Verdict),
		Score:    review.Score,
		Comments: review.Comments,
	}

	p.bus.Emit(event.ReviewResult, "reviewer", result)
	log.Printf("[review] Task %s: %s (score: %d)", task.ID, result.Verdict, result.Score)
	return result
}

func (p *Pipeline) RunCrossReview(ctx context.Context, task *store.Task, reviewerName string) *Result {
	reviewer := p.agents.Get(reviewerName)
	if reviewer == nil {
		return &Result{TaskID: task.ID, Verdict: VerdictApprove, Score: 50, Comments: "Reviewer not available"}
	}

	prompt := fmt.Sprintf("Review this code output from task '%s':\n\n%s", task.Title, task.Output)
	ch, err := reviewer.SendMessage(ctx, prompt)
	if err != nil {
		return &Result{TaskID: task.ID, Verdict: VerdictApprove, Score: 50, Comments: err.Error()}
	}

	msg := <-ch
	result := &Result{
		TaskID:   task.ID,
		Reviewer: reviewerName,
		Verdict:  VerdictDiscuss,
		Score:    70,
		Comments: msg.Content,
	}

	p.bus.Emit(event.ReviewResult, reviewerName, result)
	return result
}
