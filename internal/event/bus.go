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

// maxHistory caps the number of events retained in the ring buffer.
const maxHistory = 1000

// EventBus is the central event distribution system.
type EventBus struct {
	mu          sync.RWMutex
	subscribers map[string][]Handler
	ring        []*Event    // ring buffer for history
	head        int         // write position
	size        int         // current number of entries
	wsCallback  func(*Event) // push to WebSocket
}

// New creates a new EventBus.
func New() *EventBus {
	return &EventBus{
		subscribers: make(map[string][]Handler),
		ring:        make([]*Event, maxHistory),
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

	// Store in ring buffer (lock only for the write)
	eb.mu.Lock()
	eb.ring[eb.head] = event
	eb.head = (eb.head + 1) % maxHistory
	if eb.size < maxHistory {
		eb.size++
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

// GetHistory returns recent events of a given type (newest first).
func (eb *EventBus) GetHistory(eventType string, limit int) []*Event {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	if limit > eb.size {
		limit = eb.size
	}

	result := make([]*Event, 0, limit)
	for i := 0; i < eb.size && len(result) < limit; i++ {
		// Walk backwards from head-1 (newest)
		idx := (eb.head - 1 - i + maxHistory) % maxHistory
		if eventType == "" || eb.ring[idx].Type == eventType {
			result = append(result, eb.ring[idx])
		}
	}
	return result
}
