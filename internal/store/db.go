package store

import (
	"database/sql"
	"encoding/json"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// DB wraps the SQLite database.
type DB struct {
	conn *sql.DB
}

// New opens (or creates) the SQLite database with performance tuning.
func New(path string) (*DB, error) {
	// Use WAL mode with optimized settings for concurrent reads
	conn, err := sql.Open("sqlite3", path+
		"?_journal_mode=WAL"+
		"&_busy_timeout=5000"+
		"&_synchronous=NORMAL"+
		"&_cache_size=-8000"+ // 8MB cache
		"&_foreign_keys=ON")
	if err != nil {
		return nil, err
	}

	// Connection pool tuning: SQLite is single-writer, but WAL allows concurrent readers
	conn.SetMaxOpenConns(1)          // single writer for SQLite
	conn.SetMaxIdleConns(1)
	conn.SetConnMaxLifetime(0)       // connections never expire

	db := &DB{conn: conn}
	if err := db.migrate(); err != nil {
		return nil, err
	}
	return db, nil
}

func (db *DB) Close() error {
	return db.conn.Close()
}

func (db *DB) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS projects (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		description TEXT,
		repo_path TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS tasks (
		id TEXT PRIMARY KEY,
		project_id TEXT NOT NULL,
		title TEXT NOT NULL,
		description TEXT,
		module TEXT,
		status TEXT DEFAULT 'pending',
		priority INTEGER DEFAULT 0,
		assigned_agent TEXT,
		reviewer TEXT,
		depends_on TEXT,
		branch TEXT,
		worktree TEXT,
		progress INTEGER DEFAULT 0,
		output TEXT,
		files_changed TEXT,
		cost_usd REAL DEFAULT 0,
		max_turns INTEGER DEFAULT 15,
		started_at DATETIME,
		completed_at DATETIME,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (project_id) REFERENCES projects(id)
	);

	CREATE INDEX IF NOT EXISTS idx_tasks_project ON tasks(project_id);
	CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);
	CREATE INDEX IF NOT EXISTS idx_tasks_agent ON tasks(assigned_agent);

	CREATE TABLE IF NOT EXISTS agents (
		name TEXT PRIMARY KEY,
		type TEXT NOT NULL,
		endpoint TEXT,
		status TEXT DEFAULT 'offline',
		last_ping DATETIME,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS chat_messages (
		id TEXT PRIMARY KEY,
		project_id TEXT NOT NULL,
		task_id TEXT,
		agent_name TEXT NOT NULL,
		role TEXT NOT NULL,
		content TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (project_id) REFERENCES projects(id)
	);

	CREATE INDEX IF NOT EXISTS idx_chat_project ON chat_messages(project_id);
	CREATE INDEX IF NOT EXISTS idx_chat_agent ON chat_messages(agent_name);

	CREATE TABLE IF NOT EXISTS plans (
		id TEXT PRIMARY KEY,
		project_id TEXT NOT NULL,
		tasks TEXT NOT NULL,
		summary TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (project_id) REFERENCES projects(id)
	);
	`
	_, err := db.conn.Exec(schema)
	return err
}

// --- Project CRUD ---

func (db *DB) CreateProject(p *Project) error {
	_, err := db.conn.Exec(
		"INSERT INTO projects (id, name, description, repo_path) VALUES (?, ?, ?, ?)",
		p.ID, p.Name, p.Description, p.RepoPath,
	)
	return err
}

func (db *DB) GetProject(id string) (*Project, error) {
	p := &Project{}
	err := db.conn.QueryRow(
		"SELECT id, name, description, repo_path, created_at, updated_at FROM projects WHERE id = ?", id,
	).Scan(&p.ID, &p.Name, &p.Description, &p.RepoPath, &p.CreatedAt, &p.UpdatedAt)
	return p, err
}

func (db *DB) ListProjects() ([]*Project, error) {
	rows, err := db.conn.Query("SELECT id, name, description, repo_path, created_at, updated_at FROM projects ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []*Project
	for rows.Next() {
		p := &Project{}
		rows.Scan(&p.ID, &p.Name, &p.Description, &p.RepoPath, &p.CreatedAt, &p.UpdatedAt)
		projects = append(projects, p)
	}
	return projects, nil
}

// --- Task CRUD ---

func (db *DB) CreateTask(t *Task) error {
	deps, _ := json.Marshal(t.DependsOn)
	files, _ := json.Marshal(t.FilesChanged)
	_, err := db.conn.Exec(
		`INSERT INTO tasks (id, project_id, title, description, module, status, priority, 
		assigned_agent, reviewer, depends_on, branch, worktree, progress, max_turns, files_changed) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		t.ID, t.ProjectID, t.Title, t.Description, t.Module, t.Status, t.Priority,
		t.AssignedAgent, t.Reviewer, string(deps), t.Branch, t.Worktree, t.Progress, t.MaxTurns, string(files),
	)
	return err
}

// CreateTasksBatch inserts multiple tasks in a single transaction.
func (db *DB) CreateTasksBatch(tasks []*Task) error {
	tx, err := db.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(
		`INSERT INTO tasks (id, project_id, title, description, module, status, priority, 
		assigned_agent, reviewer, depends_on, branch, worktree, progress, max_turns, files_changed) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, t := range tasks {
		deps, _ := json.Marshal(t.DependsOn)
		files, _ := json.Marshal(t.FilesChanged)
		_, err = stmt.Exec(
			t.ID, t.ProjectID, t.Title, t.Description, t.Module, t.Status, t.Priority,
			t.AssignedAgent, t.Reviewer, string(deps), t.Branch, t.Worktree, t.Progress, t.MaxTurns, string(files),
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (db *DB) GetTask(id string) (*Task, error) {
	t := &Task{}
	var deps, files string
	var startedAt, completedAt sql.NullTime
	err := db.conn.QueryRow(
		`SELECT id, project_id, title, description, module, status, priority, 
		assigned_agent, reviewer, depends_on, branch, worktree, progress, output,
		files_changed, cost_usd, max_turns, started_at, completed_at, created_at 
		FROM tasks WHERE id = ?`, id,
	).Scan(&t.ID, &t.ProjectID, &t.Title, &t.Description, &t.Module, &t.Status, &t.Priority,
		&t.AssignedAgent, &t.Reviewer, &deps, &t.Branch, &t.Worktree, &t.Progress, &t.Output,
		&files, &t.CostUSD, &t.MaxTurns, &startedAt, &completedAt, &t.CreatedAt)
	if err != nil {
		return nil, err
	}
	json.Unmarshal([]byte(deps), &t.DependsOn)
	json.Unmarshal([]byte(files), &t.FilesChanged)
	if startedAt.Valid {
		t.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		t.CompletedAt = &completedAt.Time
	}
	return t, nil
}

func (db *DB) ListTasks(projectID string) ([]*Task, error) {
	rows, err := db.conn.Query(
		`SELECT id, project_id, title, description, module, status, priority, 
		assigned_agent, reviewer, depends_on, branch, worktree, progress, output,
		files_changed, cost_usd, max_turns, started_at, completed_at, created_at 
		FROM tasks WHERE project_id = ? ORDER BY created_at`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []*Task
	for rows.Next() {
		t := &Task{}
		var deps, files string
		var startedAt, completedAt sql.NullTime
		rows.Scan(&t.ID, &t.ProjectID, &t.Title, &t.Description, &t.Module, &t.Status, &t.Priority,
			&t.AssignedAgent, &t.Reviewer, &deps, &t.Branch, &t.Worktree, &t.Progress, &t.Output,
			&files, &t.CostUSD, &t.MaxTurns, &startedAt, &completedAt, &t.CreatedAt)
		json.Unmarshal([]byte(deps), &t.DependsOn)
		json.Unmarshal([]byte(files), &t.FilesChanged)
		if startedAt.Valid {
			t.StartedAt = &startedAt.Time
		}
		if completedAt.Valid {
			t.CompletedAt = &completedAt.Time
		}
		tasks = append(tasks, t)
	}
	return tasks, nil
}

func (db *DB) UpdateTaskStatus(id string, status TaskStatus) error {
	now := time.Now()
	if status == TaskRunning {
		_, err := db.conn.Exec("UPDATE tasks SET status = ?, started_at = ? WHERE id = ?", status, now, id)
		return err
	}
	if status == TaskDone || status == TaskFailed {
		_, err := db.conn.Exec("UPDATE tasks SET status = ?, completed_at = ? WHERE id = ?", status, now, id)
		return err
	}
	_, err := db.conn.Exec("UPDATE tasks SET status = ? WHERE id = ?", status, id)
	return err
}

func (db *DB) UpdateTaskProgress(id string, progress int, output string) error {
	_, err := db.conn.Exec("UPDATE tasks SET progress = ?, output = ? WHERE id = ?", progress, output, id)
	return err
}

func (db *DB) AssignTask(id string, agent string) error {
	_, err := db.conn.Exec("UPDATE tasks SET assigned_agent = ?, status = ? WHERE id = ?", agent, TaskAssigned, id)
	return err
}

// --- Agent CRUD ---

func (db *DB) UpsertAgent(a *AgentInfo) error {
	_, err := db.conn.Exec(
		`INSERT INTO agents (name, type, endpoint, status, last_ping) VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(name) DO UPDATE SET status = ?, last_ping = ?`,
		a.Name, a.Type, a.Endpoint, a.Status, a.LastPing, a.Status, a.LastPing,
	)
	return err
}

func (db *DB) ListAgents() ([]*AgentInfo, error) {
	rows, err := db.conn.Query("SELECT name, type, endpoint, status, last_ping, created_at FROM agents")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var agents []*AgentInfo
	for rows.Next() {
		a := &AgentInfo{}
		rows.Scan(&a.Name, &a.Type, &a.Endpoint, &a.Status, &a.LastPing, &a.CreatedAt)
		agents = append(agents, a)
	}
	return agents, nil
}

// --- Chat Messages ---

func (db *DB) SaveMessage(m *ChatMessage) error {
	_, err := db.conn.Exec(
		"INSERT INTO chat_messages (id, project_id, task_id, agent_name, role, content) VALUES (?, ?, ?, ?, ?, ?)",
		m.ID, m.ProjectID, m.TaskID, m.AgentName, m.Role, m.Content,
	)
	return err
}

func (db *DB) GetMessages(projectID, agentName string, limit int) ([]*ChatMessage, error) {
	var query strings.Builder
	query.WriteString("SELECT id, project_id, task_id, agent_name, role, content, created_at FROM chat_messages WHERE project_id = ?")
	args := []any{projectID}
	if agentName != "" {
		query.WriteString(" AND agent_name = ?")
		args = append(args, agentName)
	}
	query.WriteString(" ORDER BY created_at DESC LIMIT ?")
	args = append(args, limit)

	rows, err := db.conn.Query(query.String(), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var msgs []*ChatMessage
	for rows.Next() {
		m := &ChatMessage{}
		rows.Scan(&m.ID, &m.ProjectID, &m.TaskID, &m.AgentName, &m.Role, &m.Content, &m.CreatedAt)
		msgs = append(msgs, m)
	}
	return msgs, nil
}
