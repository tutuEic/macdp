package agent

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"time"
)

type CodexConnector struct {
	binary  string
	status  AgentStatus
}

func NewCodexConnector(binary string) *CodexConnector {
	if binary == "" {
		binary = "codex"
	}
	return &CodexConnector{
		binary: binary,
		status: AgentStatus{Online: false, State: "offline"},
	}
}

func (c *CodexConnector) Name() string     { return "codex" }
func (c *CodexConnector) Type() AgentType   { return Codex }
func (c *CodexConnector) Status() AgentStatus { return c.status }

func (c *CodexConnector) Connect(ctx context.Context, config map[string]string) error {
	if b, ok := config["binary"]; ok {
		c.binary = b
	}
	c.status.Online = true
	c.status.State = "idle"
	return nil
}

func (c *CodexConnector) Disconnect() error {
	c.status.Online = false
	c.status.State = "offline"
	return nil
}

func (c *CodexConnector) Ping(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, c.binary, "--version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("codex not available: %w", err)
	}
	c.status.LastPing = time.Now()
	return nil
}

func (c *CodexConnector) ExecuteTask(ctx context.Context, req *TaskRequest) (<-chan *TaskEvent, error) {
	c.status.State = "busy"
	c.status.CurrentTask = req.TaskID
	events := make(chan *TaskEvent, 100)

	go func() {
		defer func() {
			c.status.State = "idle"
			c.status.CurrentTask = ""
			close(events)
		}()

		start := time.Now()
		prompt := fmt.Sprintf("Task: %s\n%s", req.Title, req.Description)
		if req.Context != "" {
			prompt += "\n\nContext:\n" + req.Context
		}

		taskCtx, cancel := context.WithCancel(ctx)
		_ = cancel

		cmd := exec.CommandContext(taskCtx, c.binary, "exec", prompt, "--full-auto")
		if req.Workdir != "" {
			cmd.Dir = req.Workdir
		}

		stdout, _ := cmd.StdoutPipe()
		stderr, _ := cmd.StderrPipe()

		if err := cmd.Start(); err != nil {
			events <- &TaskEvent{Type: "error", TaskID: req.TaskID, Agent: "codex", Content: err.Error()}
			return
		}

		go func() {
			scanner := bufio.NewScanner(stdout)
			for scanner.Scan() {
				events <- &TaskEvent{Type: "output", TaskID: req.TaskID, Agent: "codex", Content: scanner.Text()}
			}
		}()
		go func() {
			scanner := bufio.NewScanner(stderr)
			for scanner.Scan() {
				events <- &TaskEvent{Type: "output", TaskID: req.TaskID, Agent: "codex", Content: "[err] " + scanner.Text()}
			}
		}()

		err := cmd.Wait()
		if err != nil {
			events <- &TaskEvent{Type: "error", TaskID: req.TaskID, Agent: "codex", Content: err.Error()}
		} else {
			events <- &TaskEvent{Type: "complete", TaskID: req.TaskID, Agent: "codex", Content: fmt.Sprintf("completed in %s", time.Since(start)), Progress: 100}
		}
	}()

	return events, nil
}

func (c *CodexConnector) CancelTask(taskID string) error { return nil }

func (c *CodexConnector) SendMessage(ctx context.Context, msg string) (<-chan *ChatMessage, error) {
	ch := make(chan *ChatMessage, 100)
	go func() {
		defer close(ch)
		cmd := exec.CommandContext(ctx, c.binary, "exec", msg, "--full-auto")
		output, err := cmd.CombinedOutput()
		if err != nil {
			ch <- &ChatMessage{Role: "system", Content: fmt.Sprintf("Error: %v", err)}
			return
		}
		ch <- &ChatMessage{Role: "agent", Content: string(output)}
	}()
	return ch, nil
}

var _ AgentConnector = (*CodexConnector)(nil)
