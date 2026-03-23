package command

import (
	"bytes"
	"database/sql"
	"testing"

	taskpkg "mytask/internal/task"

	_ "modernc.org/sqlite"
)

// MockHookExecutor implements hook.Executor for testing
type MockHookExecutor struct {
	PreCreateCalls  []*taskpkg.Task
	PostCreateCalls []*taskpkg.Task
	PreStartCalls   []*taskpkg.Task
	PostStartCalls  []*taskpkg.Task
	PreSubmitCalls  []*taskpkg.Task
	PostSubmitCalls []*taskpkg.Task
	PreResetCalls   []*taskpkg.Task
	PostResetCalls  []*taskpkg.Task

	PreCreateError  error
	PostCreateError error
	PreStartError   error
	PostStartError  error
	PreSubmitError  error
	PostSubmitError error
	PreResetError   error
	PostResetError  error
}

func (m *MockHookExecutor) ExecutePreCreate(t *taskpkg.Task) error {
	m.PreCreateCalls = append(m.PreCreateCalls, t)
	return m.PreCreateError
}

func (m *MockHookExecutor) ExecutePostCreate(t *taskpkg.Task) error {
	m.PostCreateCalls = append(m.PostCreateCalls, t)
	return m.PostCreateError
}

func (m *MockHookExecutor) ExecutePreStart(t *taskpkg.Task) error {
	m.PreStartCalls = append(m.PreStartCalls, t)
	return m.PreStartError
}

func (m *MockHookExecutor) ExecutePostStart(t *taskpkg.Task) error {
	m.PostStartCalls = append(m.PostStartCalls, t)
	return m.PostStartError
}

func (m *MockHookExecutor) ExecutePreSubmit(t *taskpkg.Task) error {
	m.PreSubmitCalls = append(m.PreSubmitCalls, t)
	return m.PreSubmitError
}

func (m *MockHookExecutor) ExecutePostSubmit(t *taskpkg.Task) error {
	m.PostSubmitCalls = append(m.PostSubmitCalls, t)
	return m.PostSubmitError
}

func (m *MockHookExecutor) ExecutePreReset(t *taskpkg.Task) error {
	m.PreResetCalls = append(m.PreResetCalls, t)
	return m.PreResetError
}

func (m *MockHookExecutor) ExecutePostReset(t *taskpkg.Task) error {
	m.PostResetCalls = append(m.PostResetCalls, t)
	return m.PostResetError
}

func setupTestDeps(t *testing.T) (*Dependencies, *sql.DB) {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	store, err := taskpkg.NewStoreWithDB(db)
	if err != nil {
		db.Close()
		t.Fatalf("Failed to create store: %v", err)
	}

	deps := &Dependencies{
		Store:    store,
		HooksDir: "", // No hooks for basic tests
		Stdout:   &bytes.Buffer{},
		Stderr:   &bytes.Buffer{},
	}

	t.Cleanup(func() {
		db.Close()
	})

	return deps, db
}

func TestCreate(t *testing.T) {
	deps, _ := setupTestDeps(t)

	// Test: name is required
	_, err := Create(deps, "", "description")
	if err == nil {
		t.Error("Expected error when name is empty")
	}

	// Test: successful create
	tsk, err := Create(deps, "test task", "test description")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if tsk.Name != "test task" {
		t.Errorf("Expected name 'test task', got '%s'", tsk.Name)
	}
	if tsk.Description != "test description" {
		t.Errorf("Expected description 'test description', got '%s'", tsk.Description)
	}
	if tsk.Status != taskpkg.StatusPending {
		t.Errorf("Expected status '%s', got '%s'", taskpkg.StatusPending, tsk.Status)
	}
}

func TestList(t *testing.T) {
	deps, _ := setupTestDeps(t)

	// Test: empty list
	if err := List(deps); err != nil {
		t.Fatalf("List failed: %v", err)
	}

	// Test: list with tasks
	deps.Store.Create("task1", "")
	deps.Store.Create("task2", "")
	if err := List(deps); err != nil {
		t.Fatalf("List failed: %v", err)
	}
}

func TestGet(t *testing.T) {
	deps, _ := setupTestDeps(t)

	// Test: get non-existent task
	err := Get(deps, 999)
	if err == nil {
		t.Error("Expected error for non-existent task")
	}

	// Test: get existing task
	created, _ := deps.Store.Create("test", "desc")
	if err := Get(deps, created.ID); err != nil {
		t.Fatalf("Get failed: %v", err)
	}
}

func TestStart(t *testing.T) {
	deps, _ := setupTestDeps(t)

	// Test: start non-existent task
	_, err := Start(deps, 999)
	if err == nil {
		t.Error("Expected error for non-existent task")
	}

	// Test: successful start
	created, _ := deps.Store.Create("test", "")
	started, err := Start(deps, created.ID)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	if started.Status != taskpkg.StatusInProgress {
		t.Errorf("Expected status '%s', got '%s'", taskpkg.StatusInProgress, started.Status)
	}
}

func TestStart_OnlyOneInProgress(t *testing.T) {
	deps, _ := setupTestDeps(t)

	task1, _ := deps.Store.Create("task1", "")
	task2, _ := deps.Store.Create("task2", "")

	// Start task1
	_, err := Start(deps, task1.ID)
	if err != nil {
		t.Fatalf("Start task1 failed: %v", err)
	}

	// Start task2 should fail
	_, err = Start(deps, task2.ID)
	if err == nil {
		t.Error("Expected error when starting second task")
	}
}

func TestSubmit(t *testing.T) {
	deps, _ := setupTestDeps(t)

	// Test: submit non-existent task
	_, err := Submit(deps, 999)
	if err == nil {
		t.Error("Expected error for non-existent task")
	}

	// Test: successful submit
	created, _ := deps.Store.Create("test", "")
	submitted, err := Submit(deps, created.ID)
	if err != nil {
		t.Fatalf("Submit failed: %v", err)
	}
	if submitted.Status != taskpkg.StatusInProgress {
		t.Errorf("Expected status '%s', got '%s'", taskpkg.StatusInProgress, submitted.Status)
	}
}

func TestComplete(t *testing.T) {
	deps, _ := setupTestDeps(t)

	// Test: complete non-existent task
	_, err := Complete(deps, 999)
	if err == nil {
		t.Error("Expected error for non-existent task")
	}

	// Test: complete pending task should fail
	created, _ := deps.Store.Create("test", "")
	_, err = Complete(deps, created.ID)
	if err == nil {
		t.Error("Expected error when completing pending task")
	}

	// Test: successful complete
	deps.Store.Submit(created.ID)
	completed, err := Complete(deps, created.ID)
	if err != nil {
		t.Fatalf("Complete failed: %v", err)
	}
	if completed.Status != taskpkg.StatusCompleted {
		t.Errorf("Expected status '%s', got '%s'", taskpkg.StatusCompleted, completed.Status)
	}
}

func TestReset(t *testing.T) {
	deps, _ := setupTestDeps(t)

	// Test: reset requires reason
	created, _ := deps.Store.Create("test", "")
	deps.Store.Submit(created.ID)
	_, err := Reset(deps, created.ID, "")
	if err == nil {
		t.Error("Expected error when reason is empty")
	}

	// Test: successful reset
	reset, err := Reset(deps, created.ID, "test reason")
	if err != nil {
		t.Fatalf("Reset failed: %v", err)
	}
	if reset.Status != taskpkg.StatusSkipped {
		t.Errorf("Expected status '%s', got '%s'", taskpkg.StatusSkipped, reset.Status)
	}
	if reset.ResetReason != "test reason" {
		t.Errorf("Expected reset reason 'test reason', got '%s'", reset.ResetReason)
	}
}