package hook

import (
	"os"
	"path/filepath"
	"testing"

	"mytask/internal/task"
)

func TestExecutor_ExecutePreStart_NotExecutableError(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "hook-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	preStartDir := filepath.Join(tmpDir, "prestart")
	if err := os.MkdirAll(preStartDir, 0755); err != nil {
		t.Fatalf("Failed to create prestart dir: %v", err)
	}

	scriptPath := filepath.Join(preStartDir, "01-test.sh")
	if err := os.WriteFile(scriptPath, []byte("#!/bin/bash\necho 'test'"), 0644); err != nil {
		t.Fatalf("Failed to create script: %v", err)
	}

	executor := NewExecutor(tmpDir)

	tsk := &task.Task{ID: 1, Name: "test", Status: task.StatusPending}
	err = executor.ExecutePreStart(tsk)

	if err == nil {
		t.Error("Expected error for non-executable script")
	}
	if err != nil && !containsString(err.Error(), "not executable") {
		t.Errorf("Expected 'not executable' error, got: %v", err)
	}
}

func TestExecutor_ExecutePreStart_ExecutableScript(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "hook-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	preStartDir := filepath.Join(tmpDir, "prestart")
	if err := os.MkdirAll(preStartDir, 0755); err != nil {
		t.Fatalf("Failed to create prestart dir: %v", err)
	}

	scriptPath := filepath.Join(preStartDir, "01-test.sh")
	if err := os.WriteFile(scriptPath, []byte("#!/bin/bash\nexit 0"), 0755); err != nil {
		t.Fatalf("Failed to create script: %v", err)
	}

	executor := NewExecutor(tmpDir)

	tsk := &task.Task{ID: 1, Name: "test", Status: task.StatusPending}
	err = executor.ExecutePreStart(tsk)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
}

func TestExecutor_ExecutePreStart_ScriptFailure(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "hook-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	preStartDir := filepath.Join(tmpDir, "prestart")
	if err := os.MkdirAll(preStartDir, 0755); err != nil {
		t.Fatalf("Failed to create prestart dir: %v", err)
	}

	scriptPath := filepath.Join(preStartDir, "01-fail.sh")
	if err := os.WriteFile(scriptPath, []byte("#!/bin/bash\nexit 1"), 0755); err != nil {
		t.Fatalf("Failed to create script: %v", err)
	}

	executor := NewExecutor(tmpDir)

	tsk := &task.Task{ID: 1, Name: "test", Status: task.StatusPending}
	err = executor.ExecutePreStart(tsk)

	if err == nil {
		t.Error("Expected error for failing script")
	}
	if err != nil && !containsString(err.Error(), "failed") {
		t.Errorf("Expected 'failed' error, got: %v", err)
	}
}

func TestExecutor_ExecutePreStart_Order(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "hook-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	preStartDir := filepath.Join(tmpDir, "prestart")
	if err := os.MkdirAll(preStartDir, 0755); err != nil {
		t.Fatalf("Failed to create prestart dir: %v", err)
	}

	orderFile := filepath.Join(tmpDir, "order.txt")
	script1 := filepath.Join(preStartDir, "01-first.sh")
	script2 := filepath.Join(preStartDir, "02-second.sh")

	if err := os.WriteFile(script1, []byte("#!/bin/bash\necho 'first' >> "+orderFile), 0755); err != nil {
		t.Fatalf("Failed to create script1: %v", err)
	}
	if err := os.WriteFile(script2, []byte("#!/bin/bash\necho 'second' >> "+orderFile), 0755); err != nil {
		t.Fatalf("Failed to create script2: %v", err)
	}

	executor := NewExecutor(tmpDir)

	tsk := &task.Task{ID: 1, Name: "test", Status: task.StatusPending}
	err = executor.ExecutePreStart(tsk)
	if err != nil {
		t.Fatalf("ExecutePreStart failed: %v", err)
	}

	content, err := os.ReadFile(orderFile)
	if err != nil {
		t.Fatalf("Failed to read order file: %v", err)
	}
	expected := "first\nsecond\n"
	if string(content) != expected {
		t.Errorf("Expected order '%s', got '%s'", expected, string(content))
	}
}

func TestExecutor_ExecutePreStart_NoHookDir(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "hook-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	executor := NewExecutor(tmpDir)

	tsk := &task.Task{ID: 1, Name: "test", Status: task.StatusPending}
	err = executor.ExecutePreStart(tsk)

	if err != nil {
		t.Errorf("Expected no error when hook dir doesn't exist, got: %v", err)
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && s[0:len(substr)] == substr ||
		len(s) > len(substr) && containsString(s[1:], substr)
}

func TestExecutor_ExecutePreCreate(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "hook-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	preCreateDir := filepath.Join(tmpDir, "precreate")
	if err := os.MkdirAll(preCreateDir, 0755); err != nil {
		t.Fatalf("Failed to create precreate dir: %v", err)
	}

	markerFile := filepath.Join(tmpDir, "marker.txt")
	scriptPath := filepath.Join(preCreateDir, "01-marker.sh")
	if err := os.WriteFile(scriptPath, []byte("#!/bin/bash\necho 'precreate' >> "+markerFile), 0755); err != nil {
		t.Fatalf("Failed to create script: %v", err)
	}

	executor := NewExecutor(tmpDir)
	tsk := &task.Task{Name: "test", Status: task.StatusPending}

	err = executor.ExecutePreCreate(tsk)
	if err != nil {
		t.Fatalf("ExecutePreCreate failed: %v", err)
	}

	content, err := os.ReadFile(markerFile)
	if err != nil {
		t.Fatalf("Failed to read marker file: %v", err)
	}
	if string(content) != "precreate\n" {
		t.Errorf("Expected 'precreate', got '%s'", string(content))
	}
}

func TestExecutor_ExecutePostCreate(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "hook-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	postCreateDir := filepath.Join(tmpDir, "postcreate")
	if err := os.MkdirAll(postCreateDir, 0755); err != nil {
		t.Fatalf("Failed to create postcreate dir: %v", err)
	}

	markerFile := filepath.Join(tmpDir, "marker.txt")
	scriptPath := filepath.Join(postCreateDir, "01-marker.sh")
	if err := os.WriteFile(scriptPath, []byte("#!/bin/bash\necho 'postcreate' >> "+markerFile), 0755); err != nil {
		t.Fatalf("Failed to create script: %v", err)
	}

	executor := NewExecutor(tmpDir)
	tsk := &task.Task{ID: 1, Name: "test", Status: task.StatusPending}

	err = executor.ExecutePostCreate(tsk)
	if err != nil {
		t.Fatalf("ExecutePostCreate failed: %v", err)
	}

	content, err := os.ReadFile(markerFile)
	if err != nil {
		t.Fatalf("Failed to read marker file: %v", err)
	}
	if string(content) != "postcreate\n" {
		t.Errorf("Expected 'postcreate', got '%s'", string(content))
	}
}

func TestExecutor_ExecutePreSubmit(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "hook-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	preSubmitDir := filepath.Join(tmpDir, "presubmit")
	if err := os.MkdirAll(preSubmitDir, 0755); err != nil {
		t.Fatalf("Failed to create presubmit dir: %v", err)
	}

	markerFile := filepath.Join(tmpDir, "marker.txt")
	scriptPath := filepath.Join(preSubmitDir, "01-marker.sh")
	if err := os.WriteFile(scriptPath, []byte("#!/bin/bash\necho 'presubmit' >> "+markerFile), 0755); err != nil {
		t.Fatalf("Failed to create script: %v", err)
	}

	executor := NewExecutor(tmpDir)
	tsk := &task.Task{ID: 1, Name: "test", Status: task.StatusPending}

	err = executor.ExecutePreSubmit(tsk)
	if err != nil {
		t.Fatalf("ExecutePreSubmit failed: %v", err)
	}

	content, err := os.ReadFile(markerFile)
	if err != nil {
		t.Fatalf("Failed to read marker file: %v", err)
	}
	if string(content) != "presubmit\n" {
		t.Errorf("Expected 'presubmit', got '%s'", string(content))
	}
}

func TestExecutor_ExecutePostSubmit(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "hook-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	postSubmitDir := filepath.Join(tmpDir, "postsubmit")
	if err := os.MkdirAll(postSubmitDir, 0755); err != nil {
		t.Fatalf("Failed to create postsubmit dir: %v", err)
	}

	markerFile := filepath.Join(tmpDir, "marker.txt")
	scriptPath := filepath.Join(postSubmitDir, "01-marker.sh")
	if err := os.WriteFile(scriptPath, []byte("#!/bin/bash\necho 'postsubmit' >> "+markerFile), 0755); err != nil {
		t.Fatalf("Failed to create script: %v", err)
	}

	executor := NewExecutor(tmpDir)
	tsk := &task.Task{ID: 1, Name: "test", Status: task.StatusInProgress}

	err = executor.ExecutePostSubmit(tsk)
	if err != nil {
		t.Fatalf("ExecutePostSubmit failed: %v", err)
	}

	content, err := os.ReadFile(markerFile)
	if err != nil {
		t.Fatalf("Failed to read marker file: %v", err)
	}
	if string(content) != "postsubmit\n" {
		t.Errorf("Expected 'postsubmit', got '%s'", string(content))
	}
}

func TestExecutor_ExecutePreStart_EnvVars(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "hook-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	preStartDir := filepath.Join(tmpDir, "prestart")
	if err := os.MkdirAll(preStartDir, 0755); err != nil {
		t.Fatalf("Failed to create prestart dir: %v", err)
	}

	envFile := filepath.Join(tmpDir, "env.txt")
	scriptContent := `#!/bin/bash
echo "ID=$TASK_ID" >> ` + envFile + `
echo "NAME=$TASK_NAME" >> ` + envFile + `
echo "STATUS=$TASK_STATUS" >> ` + envFile + `
echo "DESC=$TASK_DESCRIPTION" >> ` + envFile

	scriptPath := filepath.Join(preStartDir, "01-env.sh")
	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0755); err != nil {
		t.Fatalf("Failed to create script: %v", err)
	}

	executor := NewExecutor(tmpDir)

	tsk := &task.Task{ID: 42, Name: "test task", Status: task.StatusPending, Description: "test desc"}
	err = executor.ExecutePreStart(tsk)
	if err != nil {
		t.Fatalf("ExecutePreStart failed: %v", err)
	}

	content, err := os.ReadFile(envFile)
	if err != nil {
		t.Fatalf("Failed to read env file: %v", err)
	}

	expected := "ID=42\nNAME=test task\nSTATUS=pending\nDESC=test desc\n"
	if string(content) != expected {
		t.Errorf("Expected env vars:\n%s\nGot:\n%s", expected, string(content))
	}
}

func TestExecutor_ExecutePostStart(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "hook-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	postStartDir := filepath.Join(tmpDir, "poststart")
	if err := os.MkdirAll(postStartDir, 0755); err != nil {
		t.Fatalf("Failed to create poststart dir: %v", err)
	}

	markerFile := filepath.Join(tmpDir, "marker.txt")
	scriptPath := filepath.Join(postStartDir, "01-marker.sh")
	if err := os.WriteFile(scriptPath, []byte("#!/bin/bash\necho 'poststart' >> "+markerFile), 0755); err != nil {
		t.Fatalf("Failed to create script: %v", err)
	}

	executor := NewExecutor(tmpDir)
	tsk := &task.Task{ID: 1, Name: "test", Status: task.StatusInProgress}

	err = executor.ExecutePostStart(tsk)
	if err != nil {
		t.Fatalf("ExecutePostStart failed: %v", err)
	}

	content, err := os.ReadFile(markerFile)
	if err != nil {
		t.Fatalf("Failed to read marker file: %v", err)
	}
	if string(content) != "poststart\n" {
		t.Errorf("Expected 'poststart', got '%s'", string(content))
	}
}

func TestExecutor_ExecutePreReset(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "hook-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	preResetDir := filepath.Join(tmpDir, "prereset")
	if err := os.MkdirAll(preResetDir, 0755); err != nil {
		t.Fatalf("Failed to create prereset dir: %v", err)
	}

	markerFile := filepath.Join(tmpDir, "marker.txt")
	scriptPath := filepath.Join(preResetDir, "01-marker.sh")
	if err := os.WriteFile(scriptPath, []byte("#!/bin/bash\necho 'prereset' >> "+markerFile), 0755); err != nil {
		t.Fatalf("Failed to create script: %v", err)
	}

	executor := NewExecutor(tmpDir)
	tsk := &task.Task{ID: 1, Name: "test", Status: task.StatusInProgress}

	err = executor.ExecutePreReset(tsk)
	if err != nil {
		t.Fatalf("ExecutePreReset failed: %v", err)
	}

	content, err := os.ReadFile(markerFile)
	if err != nil {
		t.Fatalf("Failed to read marker file: %v", err)
	}
	if string(content) != "prereset\n" {
		t.Errorf("Expected 'prereset', got '%s'", string(content))
	}
}

func TestExecutor_ExecutePostReset(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "hook-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	postResetDir := filepath.Join(tmpDir, "postreset")
	if err := os.MkdirAll(postResetDir, 0755); err != nil {
		t.Fatalf("Failed to create postreset dir: %v", err)
	}

	markerFile := filepath.Join(tmpDir, "marker.txt")
	scriptPath := filepath.Join(postResetDir, "01-marker.sh")
	if err := os.WriteFile(scriptPath, []byte("#!/bin/bash\necho 'postreset' >> "+markerFile), 0755); err != nil {
		t.Fatalf("Failed to create script: %v", err)
	}

	executor := NewExecutor(tmpDir)
	tsk := &task.Task{ID: 1, Name: "test", Status: task.StatusSkipped}

	err = executor.ExecutePostReset(tsk)
	if err != nil {
		t.Fatalf("ExecutePostReset failed: %v", err)
	}

	content, err := os.ReadFile(markerFile)
	if err != nil {
		t.Fatalf("Failed to read marker file: %v", err)
	}
	if string(content) != "postreset\n" {
		t.Errorf("Expected 'postreset', got '%s'", string(content))
	}
}

func TestExecutor_ExecutePreStart_EmptyDir(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "hook-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 创建空的 prestart 目录
	preStartDir := filepath.Join(tmpDir, "prestart")
	if err := os.MkdirAll(preStartDir, 0755); err != nil {
		t.Fatalf("Failed to create prestart dir: %v", err)
	}

	executor := NewExecutor(tmpDir)
	tsk := &task.Task{ID: 1, Name: "test", Status: task.StatusPending}

	// 空目录不应该报错
	err = executor.ExecutePreStart(tsk)
	if err != nil {
		t.Errorf("Expected no error for empty hook dir, got: %v", err)
	}
}

func TestExecutor_ExecutePreStart_SkipSubdirectories(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "hook-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	preStartDir := filepath.Join(tmpDir, "prestart")
	if err := os.MkdirAll(preStartDir, 0755); err != nil {
		t.Fatalf("Failed to create prestart dir: %v", err)
	}

	// 创建子目录（应该被跳过）
	subDir := filepath.Join(preStartDir, "subdir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdir: %v", err)
	}

	// 创建一个脚本
	markerFile := filepath.Join(tmpDir, "marker.txt")
	scriptPath := filepath.Join(preStartDir, "01-script.sh")
	if err := os.WriteFile(scriptPath, []byte("#!/bin/bash\necho 'executed' >> "+markerFile), 0755); err != nil {
		t.Fatalf("Failed to create script: %v", err)
	}

	executor := NewExecutor(tmpDir)
	tsk := &task.Task{ID: 1, Name: "test", Status: task.StatusPending}

	err = executor.ExecutePreStart(tsk)
	if err != nil {
		t.Fatalf("ExecutePreStart failed: %v", err)
	}

	// 脚本应该被执行
	content, err := os.ReadFile(markerFile)
	if err != nil {
		t.Fatalf("Failed to read marker file: %v", err)
	}
	if string(content) != "executed\n" {
		t.Errorf("Expected 'executed', got '%s'", string(content))
	}
}