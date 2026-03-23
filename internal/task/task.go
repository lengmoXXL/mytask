package task

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

type Status string

const (
	StatusPending    Status = "pending"
	StatusInProgress Status = "in_progress"
	StatusCompleted  Status = "completed"
	StatusSkipped    Status = "skipped"
)

type Task struct {
	ID          int64
	Name        string
	Description string
	Status      Status
	ResetReason string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type Store struct {
	db *sql.DB
}

func NewStore(dbPath string) (*Store, error) {
	dir := filepath.Dir(dbPath)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create database directory: %w", err)
		}
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	return NewStoreWithDB(db)
}

// NewStoreWithDB creates a Store with an existing database connection (for testing)
func NewStoreWithDB(db *sql.DB) (*Store, error) {
	store := &Store{db: db}
	if err := store.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}
	return store, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) initSchema() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS tasks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			description TEXT,
			status TEXT NOT NULL DEFAULT 'pending',
			reset_reason TEXT,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`)
	return err
}

func (s *Store) Create(name, description string) (*Task, error) {
	now := time.Now()
	result, err := s.db.Exec(
		`INSERT INTO tasks (name, description, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
		name, description, StatusPending, now, now,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create task: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get last insert id: %w", err)
	}

	return s.GetByID(id)
}

func (s *Store) GetByID(id int64) (*Task, error) {
	task := &Task{}
	var resetReason sql.NullString
	err := s.db.QueryRow(
		`SELECT id, name, description, status, reset_reason, created_at, updated_at FROM tasks WHERE id = ?`,
		id,
	).Scan(&task.ID, &task.Name, &task.Description, &task.Status, &resetReason, &task.CreatedAt, &task.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("task not found: %d", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get task: %w", err)
	}
	task.ResetReason = resetReason.String
	return task, nil
}

func (s *Store) List() ([]*Task, error) {
	rows, err := s.db.Query(
		`SELECT id, name, description, status, reset_reason, created_at, updated_at FROM tasks ORDER BY id DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list tasks: %w", err)
	}
	defer rows.Close()

	var tasks []*Task
	for rows.Next() {
		task := &Task{}
		var resetReason sql.NullString
		if err := rows.Scan(&task.ID, &task.Name, &task.Description, &task.Status, &resetReason, &task.CreatedAt, &task.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan task: %w", err)
		}
		task.ResetReason = resetReason.String
		tasks = append(tasks, task)
	}
	return tasks, rows.Err()
}

func (s *Store) Submit(id int64) (*Task, error) {
	// 检查任务是否存在且状态为 pending
	task, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}

	if task.Status != StatusPending {
		return nil, fmt.Errorf("task %d is not in pending status, current status: %s", id, task.Status)
	}

	// 原子操作：仅在无 in_progress 任务时更新状态
	result, err := s.db.Exec(
		`UPDATE tasks SET status = ?, updated_at = ?, reset_reason = NULL
		 WHERE id = ? AND NOT EXISTS (SELECT 1 FROM tasks WHERE status = ?)`,
		StatusInProgress, time.Now(), id, StatusInProgress,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to submit task: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rowsAffected == 0 {
		return nil, fmt.Errorf("cannot start task: another task is already in progress")
	}

	return s.GetByID(id)
}

func (s *Store) Complete(id int64) (*Task, error) {
	task, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}

	if task.Status != StatusInProgress {
		return nil, fmt.Errorf("task %d is not in in_progress status, current status: %s", id, task.Status)
	}

	_, err = s.db.Exec(
		`UPDATE tasks SET status = ?, updated_at = ? WHERE id = ?`,
		StatusCompleted, time.Now(), id,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to complete task: %w", err)
	}

	return s.GetByID(id)
}

func (s *Store) Reset(id int64, reason string) (*Task, error) {
	task, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}

	if task.Status == StatusPending || task.Status == StatusSkipped {
		return nil, fmt.Errorf("task %d cannot be reset, current status: %s", id, task.Status)
	}

	_, err = s.db.Exec(
		`UPDATE tasks SET status = ?, reset_reason = ?, updated_at = ? WHERE id = ?`,
		StatusSkipped, reason, time.Now(), id,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to reset task: %w", err)
	}

	return s.GetByID(id)
}