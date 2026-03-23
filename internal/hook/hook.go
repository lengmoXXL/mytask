package hook

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"

	"mytask/internal/task"
)

// Executor interface for dependency injection
type Executor interface {
	ExecutePreCreate(t *task.Task) error
	ExecutePostCreate(t *task.Task) error
	ExecutePreStart(t *task.Task) error
	ExecutePostStart(t *task.Task) error
	ExecutePreSubmit(t *task.Task) error
	ExecutePostSubmit(t *task.Task) error
	ExecutePreReset(t *task.Task) error
	ExecutePostReset(t *task.Task) error
}

type executor struct {
	hooksDir string
}

func NewExecutor(hooksDir string) Executor {
	return &executor{hooksDir: hooksDir}
}

func (e *executor) ExecutePreCreate(t *task.Task) error {
	return e.executeHooks(filepath.Join(e.hooksDir, "precreate"), t)
}

func (e *executor) ExecutePostCreate(t *task.Task) error {
	return e.executeHooks(filepath.Join(e.hooksDir, "postcreate"), t)
}

func (e *executor) ExecutePreStart(t *task.Task) error {
	return e.executeHooks(filepath.Join(e.hooksDir, "prestart"), t)
}

func (e *executor) ExecutePostStart(t *task.Task) error {
	return e.executeHooks(filepath.Join(e.hooksDir, "poststart"), t)
}

func (e *executor) ExecutePreSubmit(t *task.Task) error {
	return e.executeHooks(filepath.Join(e.hooksDir, "presubmit"), t)
}

func (e *executor) ExecutePostSubmit(t *task.Task) error {
	return e.executeHooks(filepath.Join(e.hooksDir, "postsubmit"), t)
}

func (e *executor) ExecutePreReset(t *task.Task) error {
	return e.executeHooks(filepath.Join(e.hooksDir, "prereset"), t)
}

func (e *executor) ExecutePostReset(t *task.Task) error {
	return e.executeHooks(filepath.Join(e.hooksDir, "postreset"), t)
}

func (e *executor) executeHooks(dir string, t *task.Task) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read hooks directory: %w", err)
	}

	type scriptInfo struct {
		name string
		path string
	}
	var scripts []scriptInfo
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return fmt.Errorf("failed to get file info for %s: %w", entry.Name(), err)
		}
		if info.Mode().Perm()&0111 == 0 {
			return fmt.Errorf("hook script '%s' is not executable (no execute permission)", entry.Name())
		}
		scripts = append(scripts, scriptInfo{
			name: entry.Name(),
			path: filepath.Join(dir, entry.Name()),
		})
	}
	sort.Slice(scripts, func(i, j int) bool {
		return scripts[i].name < scripts[j].name
	})

	env := os.Environ()
	env = append(env,
		fmt.Sprintf("TASK_ID=%d", t.ID),
		fmt.Sprintf("TASK_NAME=%s", t.Name),
		fmt.Sprintf("TASK_STATUS=%s", t.Status),
		fmt.Sprintf("TASK_DESCRIPTION=%s", t.Description),
	)

	for _, script := range scripts {
		cmd := exec.Command(script.path)
		cmd.Env = env
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("hook %s failed: %w", script.name, err)
		}
	}

	return nil
}