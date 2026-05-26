package agent

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// ClaudeCodeAdapter implements AgentAdapter for Anthropic's Claude Code CLI.
type ClaudeCodeAdapter struct {
	config Config
	mu     sync.Mutex
	status Status
}

// ClaudeResult represents the JSON output from claude -p --output-format json.
type ClaudeResult struct {
	Type        string  `json:"type"`
	Subtype     string  `json:"subtype"`
	Result      string  `json:"result"`
	SessionID   string  `json:"session_id"`
	NumTurns    int     `json:"num_turns"`
	TotalCost   float64 `json:"total_cost_usd"`
	DurationMs  int64   `json:"duration_ms"`
	StopReason  string  `json:"stop_reason"`
}

func NewClaudeCodeAdapter(cfg Config) *ClaudeCodeAdapter {
	return &ClaudeCodeAdapter{
		config: cfg,
		status: StatusIdle,
	}
}

func (c *ClaudeCodeAdapter) Name() string   { return "claude-code" }
func (c *ClaudeCodeAdapter) Status() Status { return c.status }

func (c *ClaudeCodeAdapter) Capabilities() []string {
	return []string{"code_gen", "shell", "file_io", "review", "subagents", "complex_refactoring"}
}

func (c *ClaudeCodeAdapter) Execute(ctx context.Context, req *TaskRequest) (*TaskResponse, error) {
	c.mu.Lock()
	c.status = StatusBusy
	c.mu.Unlock()
	defer func() {
		c.mu.Lock()
		c.status = StatusIdle
		c.mu.Unlock()
	}()

	start := time.Now()

	// Build prompt
	var ctxParts []string
	ctxParts = append(ctxParts, fmt.Sprintf("Task: %s\n%s", req.Title, req.Description))
	for k, v := range req.Context {
		ctxParts = append(ctxParts, fmt.Sprintf("%s: %s", k, v))
	}
	prompt := strings.Join(ctxParts, "\n\n")

	// Build command: claude -p "prompt" --output-format json
	args := []string{"-p", prompt, "--output-format", "json"}
	if req.MaxTurns > 0 {
		args = append(args, "--max-turns", fmt.Sprintf("%d", req.MaxTurns))
	} else {
		args = append(args, "--max-turns", "15")
	}
	if len(req.AllowedTools) > 0 {
		args = append(args, "--allowedTools", strings.Join(req.AllowedTools, ","))
	}
	args = append(args, c.config.Flags...)

	cmd := exec.CommandContext(ctx, c.config.Entrypoint, args...)
	cmd.Dir = req.Workdir

	output, err := cmd.CombinedOutput()
	duration := time.Since(start)

	if err != nil {
		c.status = StatusError
		return &TaskResponse{
			TaskID:    req.TaskID,
			Success:   false,
			Error:     err.Error(),
			RawOutput: string(output),
			Duration:  duration.String(),
		}, nil
	}

	// Try to parse JSON result
	var result ClaudeResult
	if jsonErr := json.Unmarshal(output, &result); jsonErr == nil {
		return &TaskResponse{
			TaskID:       req.TaskID,
			Success:      result.Subtype == "success",
			Summary:      result.Result,
			CostUSD:      result.TotalCost,
			Duration:     duration.String(),
			RawOutput:    string(output),
			Error:        func() string { if result.Subtype != "success" { return result.Subtype }; return "" }(),
		}, nil
	}

	// Fallback: treat raw output as summary
	return &TaskResponse{
		TaskID:    req.TaskID,
		Success:   true,
		Summary:   string(output),
		Duration:  duration.String(),
		RawOutput: string(output),
	}, nil
}

func (c *ClaudeCodeAdapter) ExecuteAsync(ctx context.Context, req *TaskRequest) (<-chan *Event, error) {
	events := make(chan *Event, 100)

	go func() {
		defer close(events)

		c.mu.Lock()
		c.status = StatusBusy
		c.mu.Unlock()
		defer func() {
			c.mu.Lock()
			c.status = StatusIdle
			c.mu.Unlock()
		}()

		var ctxParts []string
		ctxParts = append(ctxParts, fmt.Sprintf("Task: %s\n%s", req.Title, req.Description))
		for k, v := range req.Context {
			ctxParts = append(ctxParts, fmt.Sprintf("%s: %s", k, v))
		}
		prompt := strings.Join(ctxParts, "\n\n")

		args := []string{"-p", prompt, "--output-format", "json", "--max-turns", "15"}
		args = append(args, c.config.Flags...)

		cmd := exec.CommandContext(ctx, c.config.Entrypoint, args...)
		cmd.Dir = req.Workdir

		stdout, _ := cmd.StdoutPipe()
		stderr, _ := cmd.StderrPipe()

		if err := cmd.Start(); err != nil {
			events <- &Event{Type: EventError, Agent: "claude-code", TaskID: req.TaskID, Content: err.Error(), Timestamp: time.Now()}
			return
		}

		go func() {
			scanner := bufio.NewScanner(stdout)
			for scanner.Scan() {
				line := scanner.Text()
				events <- &Event{Type: EventStdout, Agent: "claude-code", TaskID: req.TaskID, Content: line, Timestamp: time.Now()}
			}
		}()

		go func() {
			scanner := bufio.NewScanner(stderr)
			for scanner.Scan() {
				events <- &Event{Type: EventStderr, Agent: "claude-code", TaskID: req.TaskID, Content: scanner.Text(), Timestamp: time.Now()}
			}
		}()

		err := cmd.Wait()
		if err != nil {
			events <- &Event{Type: EventError, Agent: "claude-code", TaskID: req.TaskID, Content: err.Error(), Timestamp: time.Now()}
		} else {
			events <- &Event{Type: EventResult, Agent: "claude-code", TaskID: req.TaskID, Content: "completed", Timestamp: time.Now()}
		}
	}()

	return events, nil
}

func (c *ClaudeCodeAdapter) Cancel(taskID string) error {
	return nil
}

var _ AgentAdapter = (*ClaudeCodeAdapter)(nil)
