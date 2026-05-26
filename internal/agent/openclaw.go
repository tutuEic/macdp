package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// OpenClawConnector connects to OpenClaw Gateway via HTTP API.
type OpenClawConnector struct {
	baseURL string
	client  *http.Client
	status  AgentStatus
}

func NewOpenClawConnector(baseURL string) *OpenClawConnector {
	if baseURL == "" {
		baseURL = "http://localhost:18789"
	}
	return &OpenClawConnector{
		baseURL: baseURL,
		client:  &http.Client{Timeout: 30 * time.Second},
		status:  AgentStatus{Online: false, State: "offline"},
	}
}

func (oc *OpenClawConnector) Name() string     { return "openclaw" }
func (oc *OpenClawConnector) Type() AgentType   { return OpenClaw }
func (oc *OpenClawConnector) Status() AgentStatus { return oc.status }

func (oc *OpenClawConnector) Connect(ctx context.Context, config map[string]string) error {
	if u, ok := config["base_url"]; ok {
		oc.baseURL = u
	}
	if err := oc.Ping(ctx); err != nil {
		return err
	}
	oc.status.Online = true
	oc.status.State = "idle"
	return nil
}

func (oc *OpenClawConnector) Disconnect() error {
	oc.status.Online = false
	oc.status.State = "offline"
	return nil
}

func (oc *OpenClawConnector) Ping(ctx context.Context) error {
	req, _ := http.NewRequestWithContext(ctx, "GET", oc.baseURL+"/api/health", nil)
	resp, err := oc.client.Do(req)
	if err != nil {
		return fmt.Errorf("openclaw ping failed: %w", err)
	}
	resp.Body.Close()
	oc.status.LastPing = time.Now()
	return nil
}

func (oc *OpenClawConnector) ExecuteTask(ctx context.Context, req *TaskRequest) (<-chan *TaskEvent, error) {
	oc.status.State = "busy"
	oc.status.CurrentTask = req.TaskID
	events := make(chan *TaskEvent, 100)

	go func() {
		defer func() {
			oc.status.State = "idle"
			oc.status.CurrentTask = ""
			close(events)
		}()

		start := time.Now()
		prompt := fmt.Sprintf("Task: %s\n%s", req.Title, req.Description)
		if req.Context != "" {
			prompt += "\n\nContext:\n" + req.Context
		}

		body, _ := json.Marshal(map[string]any{
			"prompt":   prompt,
			"workdir":  req.Workdir,
			"max_turns": req.MaxTurns,
		})

		httpReq, _ := http.NewRequestWithContext(ctx, "POST", oc.baseURL+"/api/agent", bytes.NewReader(body))
		httpReq.Header.Set("Content-Type", "application/json")

		resp, err := oc.client.Do(httpReq)
		if err != nil {
			events <- &TaskEvent{Type: "error", TaskID: req.TaskID, Agent: "openclaw", Content: err.Error()}
			return
		}
		defer resp.Body.Close()

		// Read response
		data, _ := io.ReadAll(resp.Body)
		duration := time.Since(start)
		events <- &TaskEvent{Type: "complete", TaskID: req.TaskID, Agent: "openclaw", Content: fmt.Sprintf("completed in %s: %s", duration, string(data)), Progress: 100}
	}()

	return events, nil
}

func (oc *OpenClawConnector) CancelTask(taskID string) error { return nil }

func (oc *OpenClawConnector) SendMessage(ctx context.Context, msg string) (<-chan *ChatMessage, error) {
	ch := make(chan *ChatMessage, 100)
	go func() {
		defer close(ch)
		body, _ := json.Marshal(map[string]string{"message": msg})
		req, _ := http.NewRequestWithContext(ctx, "POST", oc.baseURL+"/api/chat", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := oc.client.Do(req)
		if err != nil {
			ch <- &ChatMessage{Role: "system", Content: fmt.Sprintf("Error: %v", err)}
			return
		}
		defer resp.Body.Close()
		data, _ := io.ReadAll(resp.Body)
		ch <- &ChatMessage{Role: "agent", Content: string(data)}
	}()
	return ch, nil
}

var _ AgentConnector = (*OpenClawConnector)(nil)
