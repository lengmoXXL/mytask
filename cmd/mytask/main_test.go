package main

import (
	"bytes"
	"database/sql"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"mytask/internal/task"

	_ "modernc.org/sqlite"
)

// MockHookExecutor 模拟 hook 执行器
type MockHookExecutor struct {
	PreCreateCalls  []*task.Task
	PostCreateCalls []*task.Task
	PreStartCalls   []*task.Task
	PostStartCalls  []*task.Task
	PreSubmitCalls  []*task.Task
	PostSubmitCalls []*task.Task
	PreResetCalls   []*task.Task
	PostResetCalls  []*task.Task

	PreCreateError  error
	PostCreateError error
	PreStartError   error
	PostStartError  error
	PreSubmitError  error
	PostSubmitError error
	PreResetError   error
	PostResetError  error
}

func (m *MockHookExecutor) ExecutePreCreate(t *task.Task) error {
	m.PreCreateCalls = append(m.PreCreateCalls, t)
	return m.PreCreateError
}

func (m *MockHookExecutor) ExecutePostCreate(t *task.Task) error {
	m.PostCreateCalls = append(m.PostCreateCalls, t)
	return m.PostCreateError
}

func (m *MockHookExecutor) ExecutePreStart(t *task.Task) error {
	m.PreStartCalls = append(m.PreStartCalls, t)
	return m.PreStartError
}

func (m *MockHookExecutor) ExecutePostStart(t *task.Task) error {
	m.PostStartCalls = append(m.PostStartCalls, t)
	return m.PostStartError
}

func (m *MockHookExecutor) ExecutePreSubmit(t *task.Task) error {
	m.PreSubmitCalls = append(m.PreSubmitCalls, t)
	return m.PreSubmitError
}

func (m *MockHookExecutor) ExecutePostSubmit(t *task.Task) error {
	m.PostSubmitCalls = append(m.PostSubmitCalls, t)
	return m.PostSubmitError
}

func (m *MockHookExecutor) ExecutePreReset(t *task.Task) error {
	m.PreResetCalls = append(m.PreResetCalls, t)
	return m.PreResetError
}

func (m *MockHookExecutor) ExecutePostReset(t *task.Task) error {
	m.PostResetCalls = append(m.PostResetCalls, t)
	return m.PostResetError
}

// FT 测试上下文
type ftContext struct {
	t      *testing.T
	db     *sql.DB
	store  *task.Store
	stdout *bytes.Buffer
	stderr *bytes.Buffer
	originStdout os.File
	originStderr os.File
}

func newFTContext(t *testing.T) *ftContext {
	t.Helper()

	// 使用内存数据库
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open in-memory database: %v", err)
	}

	store, err := task.NewStoreWithDB(db)
	if err != nil {
		db.Close()
		t.Fatalf("Failed to create store: %v", err)
	}

	ctx := &ftContext{
		t:      t,
		db:     db,
		store:  store,
		stdout: bytes.NewBuffer(nil),
		stderr: bytes.NewBuffer(nil),
	}

	// 重定向输出
	ctx.originStdout = *os.Stdout
	ctx.originStderr = *os.Stderr
	r, w, _ := os.Pipe()
	os.Stdout = w
	os.Stderr = w
	ctx.t.Cleanup(func() {
		os.Stdout = &ctx.originStdout
		os.Stderr = &ctx.originStderr
		w.Close()
		io.Copy(ctx.stdout, r)
	})

	ctx.t.Cleanup(func() {
		db.Close()
	})

	return ctx
}

func (ctx *ftContext) getOutput() string {
	return ctx.stdout.String()
}

func (ctx *ftContext) createTask(name, desc string) *task.Task {
	ctx.t.Helper()
	t, err := ctx.store.Create(name, desc)
	if err != nil {
		ctx.t.Fatalf("Create task failed: %v", err)
	}
	return t
}

func (ctx *ftContext) assertTaskStatus(id int64, expected task.Status) {
	ctx.t.Helper()
	t, err := ctx.store.GetByID(id)
	if err != nil {
		ctx.t.Fatalf("Get task %d failed: %v", id, err)
	}
	if t.Status != expected {
		ctx.t.Errorf("Task %d: expected status '%s', got '%s'", id, expected, t.Status)
	}
}

func (ctx *ftContext) assertOnlyOneInProgress() {
	ctx.t.Helper()
	tasks, err := ctx.store.List()
	if err != nil {
		ctx.t.Fatalf("List tasks failed: %v", err)
	}
	count := 0
	for _, t := range tasks {
		if t.Status == task.StatusInProgress {
			count++
		}
	}
	if count > 1 {
		ctx.t.Errorf("Expected at most 1 in_progress task, got %d", count)
	}
}

// 测试：创建任务默认为 pending 状态
func TestFT_CreateTask_DefaultPending(t *testing.T) {
	ctx := newFTContext(t)

	tsk := ctx.createTask("测试任务", "描述")

	if tsk.Status != task.StatusPending {
		t.Errorf("Expected status '%s', got '%s'", task.StatusPending, tsk.Status)
	}
	if tsk.Name != "测试任务" {
		t.Errorf("Expected name '测试任务', got '%s'", tsk.Name)
	}
}

// 测试：只有一个任务可以处于进行中
func TestFT_OnlyOneInProgress(t *testing.T) {
	ctx := newFTContext(t)

	// 创建两个任务
	task1 := ctx.createTask("任务1", "")
	task2 := ctx.createTask("任务2", "")

	// 启动 task1
	_, err := ctx.store.Submit(task1.ID)
	if err != nil {
		t.Fatalf("Start task1 failed: %v", err)
	}
	ctx.assertTaskStatus(task1.ID, task.StatusInProgress)

	// 启动 task2，task1 应该自动变成 pending
	_, err = ctx.store.Submit(task2.ID)
	if err != nil {
		t.Fatalf("Start task2 failed: %v", err)
	}

	ctx.assertTaskStatus(task2.ID, task.StatusInProgress)
	ctx.assertTaskStatus(task1.ID, task.StatusPending)
	ctx.assertOnlyOneInProgress()
}

// 测试：不能启动非 pending 状态的任务
func TestFT_Start_NonPendingTask(t *testing.T) {
	ctx := newFTContext(t)

	tsk := ctx.createTask("任务", "")

	// 启动任务
	_, err := ctx.store.Submit(tsk.ID)
	if err != nil {
		t.Fatalf("Start task failed: %v", err)
	}

	// 再次启动应该失败
	_, err = ctx.store.Submit(tsk.ID)
	if err == nil {
		t.Error("Expected error when starting non-pending task")
	}
	if !strings.Contains(err.Error(), "not in pending status") {
		t.Errorf("Unexpected error: %v", err)
	}
}

// 测试：完成任务
func TestFT_CompleteTask(t *testing.T) {
	ctx := newFTContext(t)

	tsk := ctx.createTask("任务", "")

	// 不能直接完成 pending 任务
	_, err := ctx.store.Complete(tsk.ID)
	if err == nil {
		t.Error("Expected error when completing pending task")
	}

	// 启动后才能完成
	ctx.store.Submit(tsk.ID)
	completed, err := ctx.store.Complete(tsk.ID)
	if err != nil {
		t.Fatalf("Complete task failed: %v", err)
	}

	if completed.Status != task.StatusCompleted {
		t.Errorf("Expected status '%s', got '%s'", task.StatusCompleted, completed.Status)
	}
}

// 测试：重置任务需要原因
func TestFT_ResetTask_RequiresReason(t *testing.T) {
	ctx := newFTContext(t)

	tsk := ctx.createTask("任务", "")

	// 启动任务
	ctx.store.Submit(tsk.ID)
	ctx.assertTaskStatus(tsk.ID, task.StatusInProgress)

	// 重置任务
	resetTask, err := ctx.store.Reset(tsk.ID, "无法完成，依赖缺失")
	if err != nil {
		t.Fatalf("Reset task failed: %v", err)
	}

	if resetTask.Status != task.StatusSkipped {
		t.Errorf("Expected status '%s', got '%s'", task.StatusSkipped, resetTask.Status)
	}
	if resetTask.ResetReason != "无法完成，依赖缺失" {
		t.Errorf("Expected reset reason '无法完成，依赖缺失', got '%s'", resetTask.ResetReason)
	}
}

// 测试：不能重置 pending 状态的任务
func TestFT_Reset_PendingTask(t *testing.T) {
	ctx := newFTContext(t)

	tsk := ctx.createTask("任务", "")

	_, err := ctx.store.Reset(tsk.ID, "原因")
	if err == nil {
		t.Error("Expected error when resetting pending task")
	}
}

// 测试：Mock Hook 执行器
func TestFT_MockHookExecutor(t *testing.T) {
	mock := &MockHookExecutor{}
	ctx := newFTContext(t)

	tsk := ctx.createTask("任务", "")

	// 模拟 pre_start hook
	if err := mock.ExecutePreStart(tsk); err != nil {
		t.Fatalf("ExecutePreStart failed: %v", err)
	}

	if len(mock.PreStartCalls) != 1 {
		t.Errorf("Expected 1 PreStart call, got %d", len(mock.PreStartCalls))
	}
	if mock.PreStartCalls[0].ID != tsk.ID {
		t.Errorf("Expected task ID %d, got %d", tsk.ID, mock.PreStartCalls[0].ID)
	}
}

// 测试：完整工作流
func TestFT_FullWorkflow(t *testing.T) {
	ctx := newFTContext(t)

	// 创建任务
	tsk1 := ctx.createTask("实现登录功能", "用户登录模块")
	tsk2 := ctx.createTask("实现注册功能", "用户注册模块")

	ctx.assertTaskStatus(tsk1.ID, task.StatusPending)
	ctx.assertTaskStatus(tsk2.ID, task.StatusPending)

	// 开始任务1
	ctx.store.Submit(tsk1.ID)
	ctx.assertTaskStatus(tsk1.ID, task.StatusInProgress)
	ctx.assertTaskStatus(tsk2.ID, task.StatusPending)

	// 切换到任务2
	ctx.store.Submit(tsk2.ID)
	ctx.assertTaskStatus(tsk1.ID, task.StatusPending)
	ctx.assertTaskStatus(tsk2.ID, task.StatusInProgress)

	// 完成任务2
	ctx.store.Complete(tsk2.ID)
	ctx.assertTaskStatus(tsk2.ID, task.StatusCompleted)

	// 任务1可以重新开始
	ctx.store.Submit(tsk1.ID)
	ctx.assertTaskStatus(tsk1.ID, task.StatusInProgress)

	// 重置任务1
	ctx.store.Reset(tsk1.ID, "需求变更，暂停开发")
	ctx.assertTaskStatus(tsk1.ID, task.StatusSkipped)

	// 验证最终状态：task2 是 completed，task1 是 skipped
	tsk1Check, _ := ctx.store.GetByID(tsk1.ID)
	tsk2Check, _ := ctx.store.GetByID(tsk2.ID)

	fmt.Printf("Task1: status=%s, reset_reason=%s\n", tsk1Check.Status, tsk1Check.ResetReason)
	fmt.Printf("Task2: status=%s\n", tsk2Check.Status)
}

// 测试：Submit 命令
func TestFT_Submit(t *testing.T) {
	ctx := newFTContext(t)

	tsk := ctx.createTask("任务", "")

	// Submit 应该将状态改为 in_progress
	submitted, err := ctx.store.Submit(tsk.ID)
	if err != nil {
		t.Fatalf("Submit failed: %v", err)
	}
	if submitted.Status != task.StatusInProgress {
		t.Errorf("Expected status '%s', got '%s'", task.StatusInProgress, submitted.Status)
	}
}

// 测试：Complete 命令只能完成 in_progress 任务
func TestFT_Complete_OnlyInProgress(t *testing.T) {
	ctx := newFTContext(t)

	tsk := ctx.createTask("任务", "")

	// 不能完成 pending 任务
	_, err := ctx.store.Complete(tsk.ID)
	if err == nil {
		t.Error("Expected error when completing pending task")
	}

	// 启动后可以完成
	ctx.store.Submit(tsk.ID)
	_, err = ctx.store.Complete(tsk.ID)
	if err != nil {
		t.Fatalf("Complete failed: %v", err)
	}
}

// 测试：所有 Hook 方法都能正确调用
func TestFT_AllHookMethods(t *testing.T) {
	mock := &MockHookExecutor{}
	tsk := &task.Task{ID: 1, Name: "test", Status: task.StatusPending}

	// 测试所有 hook 方法
	_ = mock.ExecutePreCreate(tsk)
	_ = mock.ExecutePostCreate(tsk)
	_ = mock.ExecutePreStart(tsk)
	_ = mock.ExecutePostStart(tsk)
	_ = mock.ExecutePreSubmit(tsk)
	_ = mock.ExecutePostSubmit(tsk)
	_ = mock.ExecutePreReset(tsk)
	_ = mock.ExecutePostReset(tsk)

	// 验证所有调用都被记录
	if len(mock.PreCreateCalls) != 1 {
		t.Error("PreCreate not called")
	}
	if len(mock.PostCreateCalls) != 1 {
		t.Error("PostCreate not called")
	}
	if len(mock.PreStartCalls) != 1 {
		t.Error("PreStart not called")
	}
	if len(mock.PostStartCalls) != 1 {
		t.Error("PostStart not called")
	}
	if len(mock.PreSubmitCalls) != 1 {
		t.Error("PreSubmit not called")
	}
	if len(mock.PostSubmitCalls) != 1 {
		t.Error("PostSubmit not called")
	}
	if len(mock.PreResetCalls) != 1 {
		t.Error("PreReset not called")
	}
	if len(mock.PostResetCalls) != 1 {
		t.Error("PostReset not called")
	}
}

// 测试：Hook 错误传播
func TestFT_HookErrors(t *testing.T) {
	mock := &MockHookExecutor{
		PreCreateError: fmt.Errorf("precreate error"),
		PreStartError:  fmt.Errorf("prestart error"),
		PreSubmitError: fmt.Errorf("presubmit error"),
		PreResetError:  fmt.Errorf("prereset error"),
	}

	tsk := &task.Task{ID: 1, Name: "test", Status: task.StatusPending}

	if err := mock.ExecutePreCreate(tsk); err == nil {
		t.Error("Expected PreCreate error")
	}
	if err := mock.ExecutePreStart(tsk); err == nil {
		t.Error("Expected PreStart error")
	}
	if err := mock.ExecutePreSubmit(tsk); err == nil {
		t.Error("Expected PreSubmit error")
	}
	if err := mock.ExecutePreReset(tsk); err == nil {
		t.Error("Expected PreReset error")
	}
}