package event

import (
	"sync"
	"time"
)

// Event represents a system event.
type Event struct {
	Type      string    `json:"type"`
	Source    string    `json:"source"`
	Target    string    `json:"target"`
	Payload   any       `json:"payload"`
	Timestamp time.Time `json:"timestamp"`
}

// Handler processes an event.
type Handler func(event *Event)

// Event types
const (
	TaskCreated   = "task.created"
	TaskAssigned  = "task.assigned"
	TaskStarted   = "task.started"
	TaskProgress  = "task.progress"
	TaskCompleted = "task.completed"
	TaskFailed    = "task.failed"

	AgentConnected = "agent.connected"
	AgentStatus    = "agent.status_changed"
	AgentMessage   = "agent.message"
	AgentOutput    = "agent.output"

	FileChanged = "file.changed"
	FileShared  = "file.shared"

	ReviewResult = "review.result"

	PlanCreated = "plan.created"
	UserCommand = "user.command"
)

// EventBus is the central event distribution system.
type EventBus struct {
	mu          sync.RWMutex
	subscribers map[string][]Handler
	history     []*Event
	maxHistory  int
	wsCallback  func(*Event) // push to WebSocket
}

// New creates a new EventBus.
func New() *EventBus {
	return &EventBus{
		subscribers: make(map[string][]Handler),
		maxHistory:  1000,
	}
}

// Subscribe registers a handler for an event type.
func (eb *EventBus) Subscribe(eventType string, handler Handler) {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	eb.subscribers[eventType] = append(eb.subscribers[eventType], handler)
}

// SetWSCallback sets a callback to push events to WebSocket clients.
func (eb *EventBus) SetWSCallback(fn func(*Event)) {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	eb.wsCallback = fn
}

// Emit publishes an event to all subscribers.
func (eb *EventBus) Emit(eventType string, source string, payload any) {
	event := &Event{
		Type:      eventType,
		Source:    source,
		Payload:   payload,
		Timestamp: time.Now(),
	}

	// Store in history
	eb.mu.Lock()
	eb.history = append(eb.history, event)
	if len(eb.history) > eb.maxHistory {
		eb.history = eb.history[len(eb.history)-eb.maxHistory:]
	}
	wsCb := eb.wsCallback
	eb.mu.Unlock()

	// Notify subscribers (async)
	eb.mu.RLock()
	handlers := eb.subscribers[eventType]
	eb.mu.RUnlock()

	for _, h := range handlers {
		go h(event)
	}

	// Push to WebSocket
	if wsCb != nil {
		go wsCb(event)
	}
}

// GetHistory returns recent events of a given type.
func (eb *EventBus) GetHistory(eventType string, limit int) []*Event {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	var result []*Event
	for i := len(eb.history) - 1; i >= 0 && len(result) < limit; i-- {
		if eventType == "" || eb.history[i].Type == eventType {
			result = append(result, eb.history[i])
		}
	}
	return result
}
