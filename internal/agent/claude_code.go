package agent

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"time"
)

type ClaudeCodeConnector struct {
	binary string
	status AgentStatus
}

func NewClaudeCodeConnector(binary string) *ClaudeCodeConnector {
	if binary == "" {
		binary = "claude"
	}
	return &ClaudeCodeConnector{
		binary: binary,
		status: AgentStatus{Online: false, State: "offline"},
	}
}

func (cc *ClaudeCodeConnector) Name() string     { return "claude-code" }
func (cc *ClaudeCodeConnector) Type() AgentType   { return ClaudeCode }
func (cc *ClaudeCodeConnector) Status() AgentStatus { return cc.status }

func (cc *ClaudeCodeConnector) Connect(ctx context.Context, config map[string]string) error {
	if b, ok := config["binary"]; ok {
		cc.binary = b
	}
	cc.status.Online = true
	cc.status.State = "idle"
	return nil
}

func (cc *ClaudeCodeConnector) Disconnect() error {
	cc.status.Online = false
	cc.status.State = "offline"
	return nil
}

func (cc *ClaudeCodeConnector) Ping(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, cc.binary, "--version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("claude-code not available: %w", err)
	}
	cc.status.LastPing = time.Now()
	return nil
}

func (cc *ClaudeCodeConnector) ExecuteTask(ctx context.Context, req *TaskRequest) (<-chan *TaskEvent, error) {
	cc.status.State = "busy"
	cc.status.CurrentTask = req.TaskID
	events := make(chan *TaskEvent, 100)

	go func() {
		defer func() {
			cc.status.State = "idle"
			cc.status.CurrentTask = ""
			close(events)
		}()

		start := time.Now()
		prompt := fmt.Sprintf("Task: %s\n%s", req.Title, req.Description)
		if req.Context != "" {
			prompt += "\n\nContext:\n" + req.Context
		}

		args := []string{"-p", prompt, "--output-format", "stream-json", "--verbose"}
		if req.MaxTurns > 0 {
			args = append(args, "--max-turns", fmt.Sprintf("%d", req.MaxTurns))
		} else {
			args = append(args, "--max-turns", "15")
		}

		taskCtx, cancel := context.WithCancel(ctx)
		_ = cancel

		cmd := exec.CommandContext(taskCtx, cc.binary, args...)
		if req.Workdir != "" {
			cmd.Dir = req.Workdir
		}

		stdout, _ := cmd.StdoutPipe()
		stderr, _ := cmd.StderrPipe()

		if err := cmd.Start(); err != nil {
			events <- &TaskEvent{Type: "error", TaskID: req.TaskID, Agent: "claude-code", Content: err.Error()}
			return
		}

		go func() {
			scanner := bufio.NewScanner(stdout)
			for scanner.Scan() {
				events <- &TaskEvent{Type: "output", TaskID: req.TaskID, Agent: "claude-code", Content: scanner.Text()}
			}
		}()
		go func() {
			scanner := bufio.NewScanner(stderr)
			for scanner.Scan() {
				events <- &TaskEvent{Type: "output", TaskID: req.TaskID, Agent: "claude-code", Content: "[err] " + scanner.Text()}
			}
		}()

		err := cmd.Wait()
		if err != nil {
			events <- &TaskEvent{Type: "error", TaskID: req.TaskID, Agent: "claude-code", Content: err.Error()}
		} else {
			events <- &TaskEvent{Type: "complete", TaskID: req.TaskID, Agent: "claude-code", Content: fmt.Sprintf("completed in %s", time.Since(start)), Progress: 100}
		}
	}()

	return events, nil
}

func (cc *ClaudeCodeConnector) CancelTask(taskID string) error { return nil }

func (cc *ClaudeCodeConnector) SendMessage(ctx context.Context, msg string) (<-chan *ChatMessage, error) {
	ch := make(chan *ChatMessage, 100)
	go func() {
		defer close(ch)
		cmd := exec.CommandContext(ctx, cc.binary, "-p", msg, "--output-format", "text", "--max-turns", "3")
		output, err := cmd.CombinedOutput()
		if err != nil {
			ch <- &ChatMessage{Role: "system", Content: fmt.Sprintf("Error: %v", err)}
			return
		}
		ch <- &ChatMessage{Role: "agent", Content: string(output)}
	}()
	return ch, nil
}

var _ AgentConnector = (*ClaudeCodeConnector)(nil)
