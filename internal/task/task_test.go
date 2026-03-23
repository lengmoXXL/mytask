package task

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestStore_Create(t *testing.T) {
	store := newTestStore(t)
	defer store.Close()

	task, err := store.Create("test task", "description")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if task.ID == 0 {
		t.Error("Task ID should not be 0")
	}
	if task.Name != "test task" {
		t.Errorf("Expected name 'test task', got '%s'", task.Name)
	}
	if task.Status != StatusPending {
		t.Errorf("Expected status '%s', got '%s'", StatusPending, task.Status)
	}
}

func TestStore_GetByID(t *testing.T) {
	store := newTestStore(t)
	defer store.Close()

	// 测试获取不存在的任务
	_, err := store.GetByID(999)
	if err == nil {
		t.Error("Expected error for non-existent task")
	}

	// 创建并获取任务
	created, err := store.Create("test", "desc")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	got, err := store.GetByID(created.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	if got.ID != created.ID {
		t.Errorf("Expected ID %d, got %d", created.ID, got.ID)
	}
}

func TestStore_List(t *testing.T) {
	store := newTestStore(t)
	defer store.Close()

	// 空列表
	tasks, err := store.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(tasks) != 0 {
		t.Errorf("Expected empty list, got %d tasks", len(tasks))
	}

	// 创建多个任务
	store.Create("task1", "")
	store.Create("task2", "")

	tasks, err = store.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(tasks) != 2 {
		t.Errorf("Expected 2 tasks, got %d", len(tasks))
	}
}

func TestStore_Start_OnlyOneInProgress(t *testing.T) {
	store := newTestStore(t)
	defer store.Close()

	// 创建两个任务
	task1, err := store.Create("task1", "")
	if err != nil {
		t.Fatalf("Create task1 failed: %v", err)
	}
	task2, err := store.Create("task2", "")
	if err != nil {
		t.Fatalf("Create task2 failed: %v", err)
	}

	// 启动 task1
	task1, err = store.Submit(task1.ID)
	if err != nil {
		t.Fatalf("Start task1 failed: %v", err)
	}
	if task1.Status != StatusInProgress {
		t.Errorf("Expected task1 status '%s', got '%s'", StatusInProgress, task1.Status)
	}

	// 启动 task2 应该失败，因为 task1 还是 in_progress
	_, err = store.Submit(task2.ID)
	if err == nil {
		t.Error("Expected error when starting task2 while task1 is in_progress")
	}
	if !strings.Contains(err.Error(), "another task is already in progress") {
		t.Errorf("Expected 'another task is already in progress' error, got: %v", err)
	}

	// 验证 task1 仍然是 in_progress
	task1, err = store.GetByID(task1.ID)
	if err != nil {
		t.Fatalf("GetByID task1 failed: %v", err)
	}
	if task1.Status != StatusInProgress {
		t.Errorf("Expected task1 to remain in_progress, got '%s'", task1.Status)
	}

	// 验证 task2 仍然是 pending
	task2, err = store.GetByID(task2.ID)
	if err != nil {
		t.Fatalf("GetByID task2 failed: %v", err)
	}
	if task2.Status != StatusPending {
		t.Errorf("Expected task2 to remain pending, got '%s'", task2.Status)
	}
}

func TestStore_Complete(t *testing.T) {
	store := newTestStore(t)
	defer store.Close()

	task, _ := store.Create("test", "")

	// 不能完成 pending 状态的任务
	_, err := store.Complete(task.ID)
	if err == nil {
		t.Error("Should not be able to complete a pending task")
	}

	// 启动后才能完成
	task, _ = store.Submit(task.ID)
	task, err = store.Complete(task.ID)
	if err != nil {
		t.Fatalf("Complete failed: %v", err)
	}
	if task.Status != StatusCompleted {
		t.Errorf("Expected status '%s', got '%s'", StatusCompleted, task.Status)
	}
}

func TestStore_Reset(t *testing.T) {
	store := newTestStore(t)
	defer store.Close()

	task, _ := store.Create("test", "")

	// 不能重置 pending 状态的任务
	_, err := store.Reset(task.ID, "reason")
	if err == nil {
		t.Error("Should not be able to reset a pending task")
	}

	// 启动后重置
	task, _ = store.Submit(task.ID)
	task, err = store.Reset(task.ID, "test reason")
	if err != nil {
		t.Fatalf("Reset failed: %v", err)
	}
	if task.Status != StatusSkipped {
		t.Errorf("Expected status '%s', got '%s'", StatusSkipped, task.Status)
	}
	if task.ResetReason != "test reason" {
		t.Errorf("Expected reset reason 'test reason', got '%s'", task.ResetReason)
	}
}

func TestStore_Completed_CannotBeModified(t *testing.T) {
	store := newTestStore(t)
	defer store.Close()

	task, _ := store.Create("test", "")
	task, _ = store.Submit(task.ID)
	task, _ = store.Complete(task.ID)

	// 不能再次完成
	_, err := store.Complete(task.ID)
	if err == nil {
		t.Error("Should not be able to complete an already completed task")
	}

	// 不能启动已完成任务
	_, err = store.Submit(task.ID)
	if err == nil {
		t.Error("Should not be able to start a completed task")
	}
}

func TestStore_Skipped_CannotBeModified(t *testing.T) {
	store := newTestStore(t)
	defer store.Close()

	task, _ := store.Create("test", "")
	task, _ = store.Submit(task.ID)
	task, _ = store.Reset(task.ID, "reason")

	// 不能再次重置
	_, err := store.Reset(task.ID, "another reason")
	if err == nil {
		t.Error("Should not be able to reset a skipped task")
	}

	// 不能完成已跳过任务
	_, err = store.Complete(task.ID)
	if err == nil {
		t.Error("Should not be able to complete a skipped task")
	}

	// 不能启动已跳过任务
	_, err = store.Submit(task.ID)
	if err == nil {
		t.Error("Should not be able to start a skipped task")
	}
}

func TestStore_Reset_CompletedTask(t *testing.T) {
	store := newTestStore(t)
	defer store.Close()

	task, _ := store.Create("test", "")
	task, _ = store.Submit(task.ID)
	task, _ = store.Complete(task.ID)

	// 可以重置已完成任务
	task, err := store.Reset(task.ID, "need redo")
	if err != nil {
		t.Fatalf("Reset failed: %v", err)
	}
	if task.Status != StatusSkipped {
		t.Errorf("Expected status '%s', got '%s'", StatusSkipped, task.Status)
	}
}

func TestStore_Timestamps(t *testing.T) {
	store := newTestStore(t)
	defer store.Close()

	before := time.Now().Truncate(time.Second)
	task, err := store.Create("test", "desc")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	after := time.Now().Add(time.Second)

	if task.CreatedAt.Before(before) || task.CreatedAt.After(after) {
		t.Errorf("CreatedAt %v not in expected range [%v, %v]", task.CreatedAt, before, after)
	}
	if task.UpdatedAt.Before(before) || task.UpdatedAt.After(after) {
		t.Errorf("UpdatedAt %v not in expected range [%v, %v]", task.UpdatedAt, before, after)
	}
}

func TestStore_List_Order(t *testing.T) {
	store := newTestStore(t)
	defer store.Close()

	// 按顺序创建任务
	task1, _ := store.Create("first", "")
	task2, _ := store.Create("second", "")
	task3, _ := store.Create("third", "")

	tasks, err := store.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	// 应该按 ID 倒序排列
	if len(tasks) != 3 {
		t.Fatalf("Expected 3 tasks, got %d", len(tasks))
	}
	if tasks[0].ID != task3.ID {
		t.Errorf("Expected first task ID %d, got %d", task3.ID, tasks[0].ID)
	}
	if tasks[1].ID != task2.ID {
		t.Errorf("Expected second task ID %d, got %d", task2.ID, tasks[1].ID)
	}
	if tasks[2].ID != task1.ID {
		t.Errorf("Expected third task ID %d, got %d", task1.ID, tasks[2].ID)
	}
}

func TestStore_Close(t *testing.T) {
	store := newTestStore(t)
	// 不应该报错
	if err := store.Close(); err != nil {
		t.Errorf("Close failed: %v", err)
	}
}

func newTestStore(t *testing.T) *Store {
	t.Helper()
	tmpDir, err := os.MkdirTemp("", "mytask-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(tmpDir) })

	store, err := NewStore(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	return store
}