package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

// Message represents a chat message.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Choice represents a completion choice.
type Choice struct {
	Message Message `json:"message"`
}

// ChatResponse is the API response.
type ChatResponse struct {
	Choices []Choice `json:"choices"`
	Usage   struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// Client is an OpenAI-compatible LLM client.
type Client struct {
	baseURL string
	apiKey  string
	model   string
	http    *http.Client
}

// New creates a new LLM client with optimized HTTP transport.
func New(baseURL, apiKey, model string) *Client {
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
		MaxConnsPerHost:       20,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		ForceAttemptHTTP2:     true,
	}

	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		model:   model,
		http:    &http.Client{Timeout: 120 * time.Second, Transport: transport},
	}
}

// Generate sends messages and returns the response.
func (c *Client) Generate(ctx context.Context, messages []Message, opts ...Option) (*ChatResponse, error) {
	cfg := &options{}
	for _, o := range opts {
		o(cfg)
	}

	body := map[string]any{
		"model":    c.model,
		"messages": messages,
	}
	if cfg.temperature > 0 {
		body["temperature"] = cfg.temperature
	}
	if cfg.maxTokens > 0 {
		body["max_tokens"] = cfg.maxTokens
	}
	if cfg.jsonMode {
		body["response_format"] = map[string]string{"type": "json_object"}
	}

	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/chat/completions", bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("llm request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("llm error %d: %s", resp.StatusCode, string(respBody))
	}

	var result ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("llm decode error: %w", err)
	}

	return &result, nil
}

// GenerateText is a convenience method that returns just the text.
func (c *Client) GenerateText(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	messages := []Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}
	resp, err := c.Generate(ctx, messages)
	if err != nil {
		return "", err
	}
	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no choices returned")
	}
	return resp.Choices[0].Message.Content, nil
}

// GenerateJSON sends a request expecting JSON output.
func (c *Client) GenerateJSON(ctx context.Context, systemPrompt, userPrompt string, target any) error {
	messages := []Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}
	resp, err := c.Generate(ctx, messages, WithJSONMode())
	if err != nil {
		return err
	}
	if len(resp.Choices) == 0 {
		return fmt.Errorf("no choices returned")
	}
	content := resp.Choices[0].Message.Content
	// Strip markdown code fences if present
	content = stripCodeFences(content)
	return json.Unmarshal([]byte(content), target)
}

// StreamGenerate sends messages and streams the response.
func (c *Client) StreamGenerate(ctx context.Context, messages []Message) (<-chan string, error) {
	body := map[string]any{
		"model":    c.model,
		"messages": messages,
		"stream":   true,
	}
	data, _ := json.Marshal(body)

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/chat/completions", bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		resp.Body.Close()
		return nil, fmt.Errorf("stream error %d", resp.StatusCode)
	}

	ch := make(chan string, 100)
	go func() {
		defer close(ch)
		defer resp.Body.Close()
		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				break
			}
			var chunk struct {
				Choices []struct {
					Delta struct {
						Content string `json:"content"`
					} `json:"delta"`
				} `json:"choices"`
			}
			if json.Unmarshal([]byte(data), &chunk) == nil && len(chunk.Choices) > 0 {
				if chunk.Choices[0].Delta.Content != "" {
					ch <- chunk.Choices[0].Delta.Content
				}
			}
		}
	}()
	return ch, nil
}

func stripCodeFences(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```json") {
		s = strings.TrimPrefix(s, "```json")
	} else if strings.HasPrefix(s, "```") {
		s = strings.TrimPrefix(s, "```")
	}
	if strings.HasSuffix(s, "```") {
		s = strings.TrimSuffix(s, "```")
	}
	return strings.TrimSpace(s)
}

// Option is a generation option.
type Option func(*options)

type options struct {
	temperature float64
	maxTokens   int
	jsonMode    bool
}

func WithTemperature(t float64) Option  { return func(o *options) { o.temperature = t } }
func WithMaxTokens(n int) Option        { return func(o *options) { o.maxTokens = n } }
func WithJSONMode() Option              { return func(o *options) { o.jsonMode = true } }
