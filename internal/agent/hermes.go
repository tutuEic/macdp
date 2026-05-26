package agent

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// HermesConnector connects to Hermes Agent via CLI.
type HermesConnector struct {
	binary   string
	mu       sync.Mutex
	status   AgentStatus
	cancelFn context.CancelFunc
}

// NewHermesConnector creates a Hermes connector.
func NewHermesConnector(binary string) *HermesConnector {
	if binary == "" {
		binary = "hermes"
	}
	return &HermesConnector{
		binary: binary,
		status: AgentStatus{Online: false, State: "offline"},
	}
}

func (h *HermesConnector) Name() string     { return "hermes" }
func (h *HermesConnector) Type() AgentType   { return Hermes }
func (h *HermesConnector) Status() AgentStatus { return h.status }

func (h *HermesConnector) Connect(ctx context.Context, config map[string]string) error {
	if b, ok := config["binary"]; ok {
		h.binary = b
	}
	if err := h.Ping(ctx); err != nil {
		return fmt.Errorf("hermes not available: %w", err)
	}
	h.status.Online = true
	h.status.State = "idle"
	h.status.LastPing = time.Now()
	return nil
}

func (h *HermesConnector) Disconnect() error {
	h.status.Online = false
	h.status.State = "offline"
	return nil
}

func (h *HermesConnector) Ping(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, h.binary, "chat", "-q", "ping", "--quiet")
	cmd.Env = append(cmd.Environ(), "HERMES_NO_BANNER=1")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("hermes ping failed: %s: %w", string(output), err)
	}
	h.status.LastPing = time.Now()
	return nil
}

func (h *HermesConnector) ExecuteTask(ctx context.Context, req *TaskRequest) (<-chan *TaskEvent, error) {
	h.mu.Lock()
	h.status.State = "busy"
	h.status.CurrentTask = req.TaskID
	h.mu.Unlock()

	events := make(chan *TaskEvent, 100)

	go func() {
		defer func() {
			h.mu.Lock()
			h.status.State = "idle"
			h.status.CurrentTask = ""
			h.mu.Unlock()
			close(events)
		}()

		start := time.Now()

		// Build prompt
		prompt := fmt.Sprintf("Task: %s\n%s", req.Title, req.Description)
		if req.Context != "" {
			prompt += "\n\nContext:\n" + req.Context
		}

		args := []string{"chat", "-q", prompt, "--quiet"}
		if req.MaxTurns > 0 {
			args = append(args, "--max-turns", fmt.Sprintf("%d", req.MaxTurns))
		}

		taskCtx, cancel := context.WithCancel(ctx)
		h.cancelFn = cancel

		cmd := exec.CommandContext(taskCtx, h.binary, args...)
		if req.Workdir != "" {
			cmd.Dir = req.Workdir
		}

		stdout, _ := cmd.StdoutPipe()
		stderr, _ := cmd.StderrPipe()

		if err := cmd.Start(); err != nil {
			events <- &TaskEvent{Type: "error", TaskID: req.TaskID, Agent: "hermes", Content: err.Error()}
			return
		}

		// Stream stdout
		go func() {
			scanner := bufio.NewScanner(stdout)
			for scanner.Scan() {
				line := scanner.Text()
				events <- &TaskEvent{Type: "output", TaskID: req.TaskID, Agent: "hermes", Content: line}
			}
		}()

		// Stream stderr
		go func() {
			scanner := bufio.NewScanner(stderr)
			for scanner.Scan() {
				events <- &TaskEvent{Type: "output", TaskID: req.TaskID, Agent: "hermes", Content: "[err] " + scanner.Text()}
			}
		}()

		err := cmd.Wait()
		duration := time.Since(start)

		if err != nil {
			events <- &TaskEvent{Type: "error", TaskID: req.TaskID, Agent: "hermes", Content: fmt.Sprintf("failed after %s: %v", duration, err)}
		} else {
			events <- &TaskEvent{Type: "complete", TaskID: req.TaskID, Agent: "hermes", Content: fmt.Sprintf("completed in %s", duration), Progress: 100}
		}
	}()

	return events, nil
}

func (h *HermesConnector) CancelTask(taskID string) error {
	if h.cancelFn != nil {
		h.cancelFn()
	}
	return nil
}

func (h *HermesConnector) SendMessage(ctx context.Context, msg string) (<-chan *ChatMessage, error) {
	ch := make(chan *ChatMessage, 100)
	go func() {
		defer close(ch)
		cmd := exec.CommandContext(ctx, h.binary, "chat", "-q", msg, "--quiet")
		output, err := cmd.CombinedOutput()
		if err != nil {
			ch <- &ChatMessage{Role: "system", Content: fmt.Sprintf("Error: %v", err)}
			return
		}
		ch <- &ChatMessage{Role: "agent", Content: strings.TrimSpace(string(output))}
	}()
	return ch, nil
}

// compile-time check
var _ AgentConnector = (*HermesConnector)(nil)
