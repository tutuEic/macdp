package agent

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"sync"
	"time"
)

// CodexAdapter implements AgentAdapter for OpenAI Codex CLI.
type CodexAdapter struct {
	config Config
	mu     sync.Mutex
	status Status
}

func NewCodexAdapter(cfg Config) *CodexAdapter {
	return &CodexAdapter{
		config: cfg,
		status: StatusIdle,
	}
}

func (cd *CodexAdapter) Name() string   { return "codex" }
func (cd *CodexAdapter) Status() Status { return cd.status }

func (cd *CodexAdapter) Capabilities() []string {
	return []string{"code_gen", "shell", "file_io", "fast_prototyping"}
}

func (cd *CodexAdapter) Execute(ctx context.Context, req *TaskRequest) (*TaskResponse, error) {
	cd.mu.Lock()
	cd.status = StatusBusy
	cd.mu.Unlock()
	defer func() {
		cd.mu.Lock()
		cd.status = StatusIdle
		cd.mu.Unlock()
	}()

	start := time.Now()

	prompt := fmt.Sprintf("Task: %s\n%s", req.Title, req.Description)
	for k, v := range req.Context {
		prompt += fmt.Sprintf("\n%s: %s", k, v)
	}

	// codex exec "prompt" --full-auto
	args := []string{"exec", prompt}
	args = append(args, cd.config.Flags...)
	// Ensure --full-auto is present
	hasAuto := false
	for _, f := range cd.config.Flags {
		if f == "--full-auto" || f == "--yolo" {
			hasAuto = true
			break
		}
	}
	if !hasAuto {
		args = append(args, "--full-auto")
	}

	cmd := exec.CommandContext(ctx, cd.config.Entrypoint, args...)
	cmd.Dir = req.Workdir

	output, err := cmd.CombinedOutput()
	duration := time.Since(start)

	if err != nil {
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
		Duration:  duration.String(),
		RawOutput: string(output),
	}, nil
}

func (cd *CodexAdapter) ExecuteAsync(ctx context.Context, req *TaskRequest) (<-chan *Event, error) {
	events := make(chan *Event, 100)

	go func() {
		defer close(events)

		cd.mu.Lock()
		cd.status = StatusBusy
		cd.mu.Unlock()
		defer func() {
			cd.mu.Lock()
			cd.status = StatusIdle
			cd.mu.Unlock()
		}()

		prompt := fmt.Sprintf("Task: %s\n%s", req.Title, req.Description)
		for k, v := range req.Context {
			prompt += fmt.Sprintf("\n%s: %s", k, v)
		}

		args := []string{"exec", prompt, "--full-auto"}
		args = append(args, cd.config.Flags...)

		cmd := exec.CommandContext(ctx, cd.config.Entrypoint, args...)
		cmd.Dir = req.Workdir

		stdout, _ := cmd.StdoutPipe()
		stderr, _ := cmd.StderrPipe()

		if err := cmd.Start(); err != nil {
			events <- &Event{Type: EventError, Agent: "codex", TaskID: req.TaskID, Content: err.Error(), Timestamp: time.Now()}
			return
		}

		go func() {
			scanner := bufio.NewScanner(stdout)
			for scanner.Scan() {
				events <- &Event{Type: EventStdout, Agent: "codex", TaskID: req.TaskID, Content: scanner.Text(), Timestamp: time.Now()}
			}
		}()

		go func() {
			scanner := bufio.NewScanner(stderr)
			for scanner.Scan() {
				events <- &Event{Type: EventStderr, Agent: "codex", TaskID: req.TaskID, Content: scanner.Text(), Timestamp: time.Now()}
			}
		}()

		err := cmd.Wait()
		if err != nil {
			events <- &Event{Type: EventError, Agent: "codex", TaskID: req.TaskID, Content: err.Error(), Timestamp: time.Now()}
		} else {
			events <- &Event{Type: EventResult, Agent: "codex", TaskID: req.TaskID, Content: "completed", Timestamp: time.Now()}
		}
	}()

	return events, nil
}

func (cd *CodexAdapter) Cancel(taskID string) error {
	return nil
}

var _ AgentAdapter = (*CodexAdapter)(nil)
