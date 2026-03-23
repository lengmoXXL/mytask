package command

import (
	"fmt"
	"io"
	"text/tabwriter"
	"time"

	"mytask/internal/hook"
	"mytask/internal/task"
)

// Dependencies holds the dependencies needed by commands
type Dependencies struct {
	Store    *task.Store
	HooksDir string
	Stdout   io.Writer
	Stderr   io.Writer
}

// Create creates a new task
func Create(deps *Dependencies, name, description string) (*task.Task, error) {
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}

	// Create a temporary task for precreate hook
	t := &task.Task{Name: name, Description: description, Status: task.StatusPending}

	executor := hook.NewExecutor(deps.HooksDir)
	if err := executor.ExecutePreCreate(t); err != nil {
		return nil, fmt.Errorf("precreate hook failed: %w", err)
	}

	t, err := deps.Store.Create(name, description)
	if err != nil {
		return nil, err
	}

	if err := executor.ExecutePostCreate(t); err != nil {
		fmt.Fprintf(deps.Stderr, "Warning: postcreate hook failed: %v\n", err)
	}

	return t, nil
}

// List lists all tasks
func List(deps *Dependencies) error {
	tasks, err := deps.Store.List()
	if err != nil {
		return err
	}

	if len(tasks) == 0 {
		fmt.Fprintln(deps.Stdout, "No tasks found.")
		return nil
	}

	w := tabwriter.NewWriter(deps.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tSTATUS\tCREATED\tDESCRIPTION")
	for _, t := range tasks {
		created := t.CreatedAt.Format("2006-01-02 15:04")
		desc := t.Description
		if len(desc) > 30 {
			desc = desc[:27] + "..."
		}
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\n", t.ID, t.Name, t.Status, created, desc)
	}
	w.Flush()
	return nil
}

// Get retrieves a task by ID and prints its details
func Get(deps *Dependencies, id int64) error {
	t, err := deps.Store.GetByID(id)
	if err != nil {
		return err
	}

	fmt.Fprintf(deps.Stdout, "ID:          %d\n", t.ID)
	fmt.Fprintf(deps.Stdout, "Name:        %s\n", t.Name)
	fmt.Fprintf(deps.Stdout, "Description: %s\n", t.Description)
	fmt.Fprintf(deps.Stdout, "Status:      %s\n", t.Status)
	fmt.Fprintf(deps.Stdout, "Created:     %s\n", t.CreatedAt.Format(time.RFC3339))
	fmt.Fprintf(deps.Stdout, "Updated:     %s\n", t.UpdatedAt.Format(time.RFC3339))
	if t.ResetReason != "" {
		fmt.Fprintf(deps.Stdout, "Reset Reason: %s\n", t.ResetReason)
	}
	return nil
}

// Start starts working on a task
func Start(deps *Dependencies, id int64) (*task.Task, error) {
	t, err := deps.Store.GetByID(id)
	if err != nil {
		return nil, err
	}

	executor := hook.NewExecutor(deps.HooksDir)
	if err := executor.ExecutePreStart(t); err != nil {
		return nil, fmt.Errorf("prestart hook failed: %w", err)
	}

	t, err = deps.Store.Submit(id)
	if err != nil {
		return nil, err
	}

	if err := executor.ExecutePostStart(t); err != nil {
		fmt.Fprintf(deps.Stderr, "Warning: poststart hook failed: %v\n", err)
	}

	return t, nil
}

// Submit submits a task
func Submit(deps *Dependencies, id int64) (*task.Task, error) {
	t, err := deps.Store.GetByID(id)
	if err != nil {
		return nil, err
	}

	executor := hook.NewExecutor(deps.HooksDir)
	if err := executor.ExecutePreSubmit(t); err != nil {
		return nil, fmt.Errorf("presubmit hook failed: %w", err)
	}

	t, err = deps.Store.Submit(id)
	if err != nil {
		return nil, err
	}

	if err := executor.ExecutePostSubmit(t); err != nil {
		fmt.Fprintf(deps.Stderr, "Warning: postsubmit hook failed: %v\n", err)
	}

	return t, nil
}

// Complete completes a task
func Complete(deps *Dependencies, id int64) (*task.Task, error) {
	return deps.Store.Complete(id)
}

// Reset resets (skips) a task with a reason
func Reset(deps *Dependencies, id int64, reason string) (*task.Task, error) {
	if reason == "" {
		return nil, fmt.Errorf("reason is required for reset")
	}

	t, err := deps.Store.GetByID(id)
	if err != nil {
		return nil, err
	}

	executor := hook.NewExecutor(deps.HooksDir)
	if err := executor.ExecutePreReset(t); err != nil {
		return nil, fmt.Errorf("prereset hook failed: %w", err)
	}

	t, err = deps.Store.Reset(id, reason)
	if err != nil {
		return nil, err
	}

	if err := executor.ExecutePostReset(t); err != nil {
		fmt.Fprintf(deps.Stderr, "Warning: postreset hook failed: %v\n", err)
	}

	return t, nil
}