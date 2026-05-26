package api

import (
	"log"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/tutuEic/macdp/internal/agent"
	"github.com/tutuEic/macdp/internal/event"
	"github.com/tutuEic/macdp/internal/store"
	"time"
	"fmt"
)

// Server is the HTTP + WebSocket API server.
type Server struct {
	store    *store.DB
	agents   *agent.Registry
	bus      *event.EventBus
	wsHub    *WSHub
}

// NewServer creates a new API server.
func NewServer(db *store.DB, agents *agent.Registry, bus *event.EventBus) *Server {
	s := &Server{
		store:  db,
		agents: agents,
		bus:    bus,
		wsHub:  NewWSHub(),
	}
	// Wire EventBus to WebSocket
	bus.SetWSCallback(func(e *event.Event) {
		s.wsHub.Broadcast(e)
	})
	return s
}

// Handler returns the Gin engine with all routes.
func (s *Server) Handler() *gin.Engine {
	r := gin.Default()

	// CORS
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	api := r.Group("/api")
	{
		api.GET("/health", s.health)

		// Projects
		api.POST("/projects", s.createProject)
		api.GET("/projects", s.listProjects)
		api.GET("/projects/:id", s.getProject)

		// Tasks
		api.POST("/projects/:id/tasks", s.createTask)
		api.GET("/projects/:id/tasks", s.listTasks)
		api.GET("/tasks/:id", s.getTask)
		api.PUT("/tasks/:id/assign/:agent", s.assignTask)
		api.PUT("/tasks/:id/status", s.updateTaskStatus)

		// Agents
		api.GET("/agents", s.listAgents)
		api.GET("/agents/:name/status", s.agentStatus)
		api.POST("/agents/:name/message", s.sendAgentMessage)

		// Chat
		api.GET("/projects/:id/chat", s.getChatHistory)
		api.POST("/projects/:id/chat", s.sendChatMessage)

		// Events
		api.GET("/events", s.getEvents)
	}

	// WebSocket
	r.GET("/ws", s.handleWS)

	return r
}

func (s *Server) health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// --- Project handlers ---

func (s *Server) createProject(c *gin.Context) {
	var p store.Project
	if err := c.ShouldBindJSON(&p); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := s.store.CreateProject(&p); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, p)
}

func (s *Server) listProjects(c *gin.Context) {
	projects, err := s.store.ListProjects()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, projects)
}

func (s *Server) getProject(c *gin.Context) {
	p, err := s.store.GetProject(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "project not found"})
		return
	}
	c.JSON(http.StatusOK, p)
}

// --- Task handlers ---

func (s *Server) createTask(c *gin.Context) {
	var t store.Task
	if err := c.ShouldBindJSON(&t); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	t.ProjectID = c.Param("id")
	t.Status = store.TaskPending
	if err := s.store.CreateTask(&t); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	s.bus.Emit(event.TaskCreated, "api", t)
	c.JSON(http.StatusCreated, t)
}

func (s *Server) listTasks(c *gin.Context) {
	tasks, err := s.store.ListTasks(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, tasks)
}

func (s *Server) getTask(c *gin.Context) {
	t, err := s.store.GetTask(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}
	c.JSON(http.StatusOK, t)
}

func (s *Server) assignTask(c *gin.Context) {
	taskID := c.Param("id")
	agentName := c.Param("agent")
	if err := s.store.AssignTask(taskID, agentName); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	s.bus.Emit(event.TaskAssigned, "api", gin.H{"task_id": taskID, "agent": agentName})
	c.JSON(http.StatusOK, gin.H{"status": "assigned"})
}

func (s *Server) updateTaskStatus(c *gin.Context) {
	var body struct {
		Status store.TaskStatus `json:"status"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := s.store.UpdateTaskStatus(c.Param("id"), body.Status); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "updated"})
}

// --- Agent handlers ---

func (s *Server) listAgents(c *gin.Context) {
	names := s.agents.List()
	var result []gin.H
	for _, name := range names {
		conn := s.agents.Get(name)
		st := conn.Status()
		result = append(result, gin.H{
			"name":   name,
			"type":   conn.Type(),
			"online": st.Online,
			"state":  st.State,
			"task":   st.CurrentTask,
		})
	}
	c.JSON(http.StatusOK, result)
}

func (s *Server) agentStatus(c *gin.Context) {
	conn := s.agents.Get(c.Param("name"))
	if conn == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
		return
	}
	c.JSON(http.StatusOK, conn.Status())
}

func (s *Server) sendAgentMessage(c *gin.Context) {
	conn := s.agents.Get(c.Param("name"))
	if conn == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
		return
	}
	var body struct {
		Message string `json:"message"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ch, err := conn.SendMessage(c.Request.Context(), body.Message)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	msg := <-ch
	c.JSON(http.StatusOK, msg)
}

// --- Chat handlers ---

func (s *Server) getChatHistory(c *gin.Context) {
	agentName := c.Query("agent")
	msgs, err := s.store.GetMessages(c.Param("id"), agentName, 50)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, msgs)
}

func (s *Server) sendChatMessage(c *gin.Context) {
	var body struct {
		AgentName string `json:"agent_name"`
		Content   string `json:"content"`
		TaskID    string `json:"task_id,omitempty"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Save user message
	msg := &store.ChatMessage{
		ID:        genID(),
		ProjectID: c.Param("id"),
		TaskID:    body.TaskID,
		AgentName: body.AgentName,
		Role:      "user",
		Content:   body.Content,
	}
	s.store.SaveMessage(msg)
	s.bus.Emit(event.UserCommand, "user", msg)

	// Get agent response
	conn := s.agents.Get(body.AgentName)
	if conn == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
		return
	}

	ch, err := conn.SendMessage(c.Request.Context(), body.Content)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	resp := <-ch

	// Save agent response
	agentMsg := &store.ChatMessage{
		ID:        genID(),
		ProjectID: c.Param("id"),
		TaskID:    body.TaskID,
		AgentName: body.AgentName,
		Role:      "agent",
		Content:   resp.Content,
	}
	s.store.SaveMessage(agentMsg)
	s.bus.Emit(event.AgentMessage, body.AgentName, agentMsg)

	c.JSON(http.StatusOK, agentMsg)
}

func (s *Server) getEvents(c *gin.Context) {
	eventType := c.Query("type")
	events := s.bus.GetHistory(eventType, 50)
	c.JSON(http.StatusOK, events)
}

// --- WebSocket ---

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func (s *Server) handleWS(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("[ws] upgrade failed: %v", err)
		return
	}
	s.wsHub.Add(conn)
}

// WSHub manages WebSocket connections.
type WSHub struct {
	mu    sync.RWMutex
	conns map[*websocket.Conn]bool
}

func NewWSHub() *WSHub {
	return &WSHub{conns: make(map[*websocket.Conn]bool)}
}

func (h *WSHub) Add(conn *websocket.Conn) {
	h.mu.Lock()
	h.conns[conn] = true
	h.mu.Unlock()
}

func (h *WSHub) Broadcast(data any) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for conn := range h.conns {
		if err := conn.WriteJSON(data); err != nil {
			conn.Close()
			delete(h.conns, conn)
		}
	}
}

func genID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
