package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/tutuEic/macdp/internal/agent"
	"github.com/tutuEic/macdp/internal/task"
)

// Server is the HTTP + WebSocket API server.
type Server struct {
	registry *agent.Registry
	dag      *task.DAG
	mu       sync.RWMutex
	clients  map[chan *SSEEvent]bool // SSE clients
}

// SSEEvent is a server-sent event for real-time updates.
type SSEEvent struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

// NewServer creates a new API server.
func NewServer(registry *agent.Registry) *Server {
	return &Server{
		registry: registry,
		clients:  make(map[chan *SSEEvent]bool),
	}
}

// SetDAG sets the current task DAG.
func (s *Server) SetDAG(dag *task.DAG) {
	s.mu.Lock()
	s.dag = dag
	s.mu.Unlock()
}

// Broadcast sends an event to all connected SSE clients.
func (s *Server) Broadcast(event *SSEEvent) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for ch := range s.clients {
		select {
		case ch <- event:
		default:
			// client too slow, skip
		}
	}
}

// Handler returns the HTTP handler with all routes.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	// API routes
	mux.HandleFunc("GET /api/status", s.handleStatus)
	mux.HandleFunc("GET /api/tasks", s.handleGetTasks)
	mux.HandleFunc("GET /api/agents", s.handleGetAgents)
	mux.HandleFunc("GET /api/events", s.handleSSE)

	// Health check
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	return mux
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	status := map[string]interface{}{
		"agents_available": len(s.registry.Available()),
		"agents_total":     len(s.registry.List()),
	}
	if s.dag != nil {
		status["tasks_total"] = len(s.dag.Tasks)
		status["is_complete"] = s.dag.IsComplete()
	}
	json.NewEncoder(w).Encode(status)
}

func (s *Server) handleGetTasks(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.dag == nil {
		json.NewEncoder(w).Encode([]*task.Task{})
		return
	}

	tasks := make([]*task.Task, 0, len(s.dag.Tasks))
	for _, t := range s.dag.Tasks {
		tasks = append(tasks, t)
	}
	json.NewEncoder(w).Encode(tasks)
}

func (s *Server) handleGetAgents(w http.ResponseWriter, r *http.Request) {
	type agentInfo struct {
		Name         string   `json:"name"`
		Status       string   `json:"status"`
		Capabilities []string `json:"capabilities"`
	}

	var agents []agentInfo
	for _, name := range s.registry.List() {
		a := s.registry.Get(name)
		agents = append(agents, agentInfo{
			Name:         a.Name(),
			Status:       string(a.Status()),
			Capabilities: a.Capabilities(),
		})
	}
	json.NewEncoder(w).Encode(agents)
}

func (s *Server) handleSSE(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "SSE not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ch := make(chan *SSEEvent, 100)
	s.mu.Lock()
	s.clients[ch] = true
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.clients, ch)
		s.mu.Unlock()
	}()

	for event := range ch {
		data, _ := json.Marshal(event)
		fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.Type, string(data))
		flusher.Flush()
	}
}

// Start starts the HTTP server.
func (s *Server) Start(host string, port int) error {
	addr := fmt.Sprintf("%s:%d", host, port)
	log.Printf("[api] Server starting on %s", addr)
	return http.ListenAndServe(addr, s.Handler())
}
