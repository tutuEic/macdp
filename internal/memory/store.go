package memory

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
)

// Store manages memory persistence in SQLite.
type Store struct {
	db *sql.DB
}

// NewStore creates a memory store using an existing DB connection.
func NewStore(db *sql.DB) (*Store, error) {
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("memory store migrate: %w", err)
	}
	return s, nil
}

func (s *Store) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS memories (
		id TEXT PRIMARY KEY,
		project_id TEXT NOT NULL,
		task_id TEXT,
		module TEXT,
		tier TEXT NOT NULL DEFAULT 'short',
		category TEXT NOT NULL DEFAULT 'summary',
		content TEXT NOT NULL,
		summary TEXT,
		tokens INTEGER DEFAULT 0,
		metadata TEXT DEFAULT '{}',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_memories_project ON memories(project_id);
	CREATE INDEX IF NOT EXISTS idx_memories_tier ON memories(tier);
	CREATE INDEX IF NOT EXISTS idx_memories_category ON memories(category);
	CREATE INDEX IF NOT EXISTS idx_memories_module ON memories(module);
	`
	_, err := s.db.Exec(schema)
	return err
}

// Save persists a memory entry.
func (s *Store) Save(e *Entry) error {
	metaJSON, _ := json.Marshal(e.Metadata)
	_, err := s.db.Exec(
		`INSERT OR REPLACE INTO memories (id, project_id, task_id, module, tier, category, content, summary, tokens, metadata)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		e.ID, e.ProjectID, e.TaskID, e.Module, string(e.Tier), e.Category,
		e.Content, e.Summary, e.Tokens, string(metaJSON),
	)
	return err
}

// SaveBatch persists multiple entries in a transaction.
func (s *Store) SaveBatch(entries []*Entry) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(
		`INSERT OR REPLACE INTO memories (id, project_id, task_id, module, tier, category, content, summary, tokens, metadata)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, e := range entries {
		metaJSON, _ := json.Marshal(e.Metadata)
		_, err = stmt.Exec(
			e.ID, e.ProjectID, e.TaskID, e.Module, string(e.Tier), e.Category,
			e.Content, e.Summary, e.Tokens, string(metaJSON),
		)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

// Query filters memories with limits.
type Query struct {
	ProjectID string
	Module    string
	Category  string
	Tier      Tier
	TaskID    string
	Limit     int
}

// Find retrieves memory entries matching the query.
func (s *Store) Find(q Query) ([]*Entry, error) {
	var clauses []string
	var args []any

	if q.ProjectID != "" {
		clauses = append(clauses, "project_id = ?")
		args = append(args, q.ProjectID)
	}
	if q.Module != "" {
		clauses = append(clauses, "module = ?")
		args = append(args, q.Module)
	}
	if q.Category != "" {
		clauses = append(clauses, "category = ?")
		args = append(args, q.Category)
	}
	if q.Tier != "" {
		clauses = append(clauses, "tier = ?")
		args = append(args, string(q.Tier))
	}
	if q.TaskID != "" {
		clauses = append(clauses, "task_id = ?")
		args = append(args, q.TaskID)
	}

	query := "SELECT id, project_id, task_id, module, tier, category, content, summary, tokens, metadata, created_at FROM memories"
	if len(clauses) > 0 {
		query += " WHERE " + strings.Join(clauses, " AND ")
	}
	query += " ORDER BY created_at DESC"

	if q.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", q.Limit)
		// Can't use ? for LIMIT in SQLite in some drivers, but we control Limit so no injection risk
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []*Entry
	for rows.Next() {
		e := &Entry{}
		var metaStr string
		err := rows.Scan(&e.ID, &e.ProjectID, &e.TaskID, &e.Module, &e.Tier, &e.Category,
			&e.Content, &e.Summary, &e.Tokens, &metaStr, &e.CreatedAt)
		if err != nil {
			return nil, err
		}
		json.Unmarshal([]byte(metaStr), &e.Metadata)
		entries = append(entries, e)
	}
	return entries, nil
}

// GetByID retrieves a single memory entry.
func (s *Store) GetByID(id string) (*Entry, error) {
	rows, err := s.db.Query(
		"SELECT id, project_id, task_id, module, tier, category, content, summary, tokens, metadata, created_at FROM memories WHERE id = ?",
		id,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, fmt.Errorf("memory %s not found", id)
	}

	e := &Entry{}
	var metaStr string
	err = rows.Scan(&e.ID, &e.ProjectID, &e.TaskID, &e.Module, &e.Tier, &e.Category,
		&e.Content, &e.Summary, &e.Tokens, &metaStr, &e.CreatedAt)
	if err != nil {
		return nil, err
	}
	json.Unmarshal([]byte(metaStr), &e.Metadata)
	return e, nil
}

// Delete removes a memory entry.
func (s *Store) Delete(id string) error {
	_, err := s.db.Exec("DELETE FROM memories WHERE id = ?", id)
	return err
}

// Prune removes old entries beyond a certain count per project.
func (s *Store) Prune(projectID string, keepPerCategory int) error {
	categories := []string{"summary", "decision", "convention", "file_change", "pattern"}
	for _, cat := range categories {
		// Delete oldest entries beyond keep count
		_, err := s.db.Exec(
			`DELETE FROM memories WHERE id IN (
				SELECT id FROM memories WHERE project_id = ? AND category = ?
				ORDER BY created_at ASC
				LIMIT -1 OFFSET ?
			)`,
			projectID, cat, keepPerCategory,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

// Stats returns aggregated memory statistics.
func (s *Store) Stats(projectID string) (*Stats, error) {
	stats := &Stats{
		ByTier:     make(map[Tier]int),
		ByCategory: make(map[string]int),
	}

	rows, err := s.db.Query(
		"SELECT tier, category, COUNT(*), SUM(tokens) FROM memories WHERE project_id = ? GROUP BY tier, category",
		projectID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var tier, category string
		var count, tokens int
		rows.Scan(&tier, &category, &count, &tokens)
		stats.ByTier[Tier(tier)] += count
		stats.ByCategory[category] += count
		stats.TotalEntries += count
		stats.TotalTokens += tokens
	}
	return stats, nil
}
