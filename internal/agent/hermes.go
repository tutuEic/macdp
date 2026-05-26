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

// HermesAdapter implements AgentAdapter for the Hermes Agent CLI.
type HermesAdapter struct {
	config Config
	mu     sync.Mutex
	status Status
}

// NewHermesAdapter creates a new Hermes agent adapter.
func NewHermesAdapter(cfg Config) *HermesAdapter {
	return &HermesAdapter{
		config: cfg,
		status: StatusIdle,
	}
}

func (h *HermesAdapter) Name() string   { return "hermes" }
func (h *HermesAdapter) Status() Status { return h.status }

func (h *HermesAdapter) Capabilities() []string {
	return []string{"shell", "file_io", "web", "delegation", "testing", "debugging"}
}

func (h *HermesAdapter) Execute(ctx context.Context, req *TaskRequest) (*TaskResponse, error) {
	h.mu.Lock()
	h.status = StatusBusy
	h.mu.Unlock()
	defer func() {
		h.mu.Lock()
		h.status = StatusIdle
		h.mu.Unlock()
	}()

	start := time.Now()

	// Build context string
	var ctxParts []string
	ctxParts = append(ctxParts, fmt.Sprintf("Task: %s\n%s", req.Title, req.Description))
	for k, v := range req.Context {
		ctxParts = append(ctxParts, fmt.Sprintf("%s: %s", k, v))
	}
	prompt := strings.Join(ctxParts, "\n\n")

	// Build command: hermes chat -q "prompt"
	args := []string{"chat", "-q", prompt}
	args = append(args, h.config.Flags...)

	if req.MaxTurns > 0 {
		args = append(args, "--max-turns", fmt.Sprintf("%d", req.MaxTurns))
	}

	cmd := exec.CommandContext(ctx, h.config.Entrypoint, args...)
	cmd.Dir = req.Workdir

	output, err := cmd.CombinedOutput()
	duration := time.Since(start)

	if err != nil {
		h.status = StatusError
		return &TaskResponse{
			TaskID:    req.TaskID,
			Success:   false,
			Error:     err.Error(),
			RawOutput: string(output),
			Duration:  duration.String(),
		}, nil
	}

	return &TaskResponse{
		TaskID:    req.TaskID,
		Success:   true,
		Summary:   string(output),
		RawOutput: string(output),
		Duration:  duration.String(),
	}, nil
}

func (h *HermesAdapter) ExecuteAsync(ctx context.Context, req *TaskRequest) (<-chan *Event, error) {
	events := make(chan *Event, 100)

	go func() {
		defer close(events)

		h.mu.Lock()
		h.status = StatusBusy
		h.mu.Unlock()
		defer func() {
			h.mu.Lock()
			h.status = StatusIdle
			h.mu.Unlock()
		}()

		start := time.Now()

		var ctxParts []string
		ctxParts = append(ctxParts, fmt.Sprintf("Task: %s\n%s", req.Title, req.Description))
		for k, v := range req.Context {
			ctxParts = append(ctxParts, fmt.Sprintf("%s: %s", k, v))
		}
		prompt := strings.Join(ctxParts, "\n\n")

		args := []string{"chat", "-q", prompt}
		args = append(args, h.config.Flags...)
		if req.MaxTurns > 0 {
			args = append(args, "--max-turns", fmt.Sprintf("%d", req.MaxTurns))
		}

		cmd := exec.CommandContext(ctx, h.config.Entrypoint, args...)
		cmd.Dir = req.Workdir

		stdout, _ := cmd.StdoutPipe()
		stderr, _ := cmd.StderrPipe()

		if err := cmd.Start(); err != nil {
			events <- &Event{Type: EventError, Agent: "hermes", TaskID: req.TaskID, Content: err.Error(), Timestamp: time.Now()}
			return
		}

		// Stream stdout
		go func() {
			scanner := bufio.NewScanner(stdout)
			for scanner.Scan() {
				events <- &Event{Type: EventStdout, Agent: "hermes", TaskID: req.TaskID, Content: scanner.Text(), Timestamp: time.Now()}
			}
		}()

		// Stream stderr
		go func() {
			scanner := bufio.NewScanner(stderr)
			for scanner.Scan() {
				events <- &Event{Type: EventStderr, Agent: "hermes", TaskID: req.TaskID, Content: scanner.Text(), Timestamp: time.Now()}
			}
		}()

		err := cmd.Wait()
		duration := time.Since(start)

		if err != nil {
			events <- &Event{Type: EventError, Agent: "hermes", TaskID: req.TaskID, Content: err.Error(), Timestamp: time.Now()}
		} else {
			events <- &Event{Type: EventResult, Agent: "hermes", TaskID: req.TaskID, Content: fmt.Sprintf("Completed in %s", duration), Timestamp: time.Now()}
		}
	}()

	return events, nil
}

func (h *HermesAdapter) Cancel(taskID string) error {
	// Kill the hermes process — handled by context cancellation in Execute
	return nil
}

// compile-time check
var _ AgentAdapter = (*HermesAdapter)(nil)
