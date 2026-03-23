package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"mytask/internal/config"
	"mytask/internal/hook"
	"mytask/internal/task"

	"github.com/spf13/cobra"
)

var (
	store *task.Store
	cfg   *config.Config
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "mytask",
	Short: "A task management tool",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		var err error
		cfg, err = config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		store, err = task.NewStore(cfg.DBPath)
		if err != nil {
			return fmt.Errorf("failed to initialize store: %w", err)
		}
		return nil
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		if store != nil {
			store.Close()
		}
	},
}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new task",
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		desc, _ := cmd.Flags().GetString("description")

		if name == "" {
			return fmt.Errorf("name is required")
		}

		// Create a temporary task for precreate hook
		t := &task.Task{Name: name, Description: desc, Status: task.StatusPending}

		executor := hook.NewExecutor(cfg.HooksDir)
		if err := executor.ExecutePreCreate(t); err != nil {
			return fmt.Errorf("precreate hook failed: %w", err)
		}

		t, err := store.Create(name, desc)
		if err != nil {
			return err
		}

		if err := executor.ExecutePostCreate(t); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: postcreate hook failed: %v\n", err)
		}

		fmt.Printf("Task created: ID=%d, Name=%s\n", t.ID, t.Name)
		return nil
	},
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all tasks",
	RunE: func(cmd *cobra.Command, args []string) error {
		tasks, err := store.List()
		if err != nil {
			return err
		}

		if len(tasks) == 0 {
			fmt.Println("No tasks found.")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
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
	},
}

var getCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get task details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := parseID(args[0])
		if err != nil {
			return err
		}

		t, err := store.GetByID(id)
		if err != nil {
			return err
		}

		fmt.Printf("ID:          %d\n", t.ID)
		fmt.Printf("Name:        %s\n", t.Name)
		fmt.Printf("Description: %s\n", t.Description)
		fmt.Printf("Status:      %s\n", t.Status)
		fmt.Printf("Created:     %s\n", t.CreatedAt.Format(time.RFC3339))
		fmt.Printf("Updated:     %s\n", t.UpdatedAt.Format(time.RFC3339))
		if t.ResetReason != "" {
			fmt.Printf("Reset Reason: %s\n", t.ResetReason)
		}
		return nil
	},
}

var startCmd = &cobra.Command{
	Use:   "start <id>",
	Short: "Start working on a task",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := parseID(args[0])
		if err != nil {
			return err
		}

		t, err := store.GetByID(id)
		if err != nil {
			return err
		}

		executor := hook.NewExecutor(cfg.HooksDir)
		if err := executor.ExecutePreStart(t); err != nil {
			return fmt.Errorf("prestart hook failed: %w", err)
		}

		t, err = store.Submit(id)
		if err != nil {
			return err
		}

		if err := executor.ExecutePostStart(t); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: poststart hook failed: %v\n", err)
		}

		fmt.Printf("Task %d started, status: %s\n", t.ID, t.Status)
		return nil
	},
}

var submitCmd = &cobra.Command{
	Use:   "submit <id>",
	Short: "Submit a task",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := parseID(args[0])
		if err != nil {
			return err
		}

		t, err := store.GetByID(id)
		if err != nil {
			return err
		}

		executor := hook.NewExecutor(cfg.HooksDir)
		if err := executor.ExecutePreSubmit(t); err != nil {
			return fmt.Errorf("presubmit hook failed: %w", err)
		}

		t, err = store.Submit(id)
		if err != nil {
			return err
		}

		if err := executor.ExecutePostSubmit(t); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: postsubmit hook failed: %v\n", err)
		}

		fmt.Printf("Task %d submitted, status: %s\n", t.ID, t.Status)
		return nil
	},
}

var completeCmd = &cobra.Command{
	Use:   "complete <id>",
	Short: "Complete a task",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := parseID(args[0])
		if err != nil {
			return err
		}

		t, err := store.Complete(id)
		if err != nil {
			return err
		}

		fmt.Printf("Task %d completed\n", t.ID)
		return nil
	},
}

var resetCmd = &cobra.Command{
	Use:   "reset <id>",
	Short: "Skip a task",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := parseID(args[0])
		if err != nil {
			return err
		}

		reason, _ := cmd.Flags().GetString("reason")
		if reason == "" {
			return fmt.Errorf("reason is required for reset")
		}

		t, err := store.GetByID(id)
		if err != nil {
			return err
		}

		executor := hook.NewExecutor(cfg.HooksDir)
		if err := executor.ExecutePreReset(t); err != nil {
			return fmt.Errorf("prereset hook failed: %w", err)
		}

		t, err = store.Reset(id, reason)
		if err != nil {
			return err
		}

		if err := executor.ExecutePostReset(t); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: postreset hook failed: %v\n", err)
		}

		fmt.Printf("Task %d skipped, reason: %s\n", t.ID, reason)
		return nil
	},
}

func parseID(s string) (int64, error) {
	var id int64
	_, err := fmt.Sscanf(s, "%d", &id)
	if err != nil {
		return 0, fmt.Errorf("invalid task id: %s", s)
	}
	return id, nil
}

func init() {
	createCmd.Flags().StringP("name", "n", "", "task name")
	createCmd.Flags().StringP("description", "d", "", "task description")
	resetCmd.Flags().StringP("reason", "r", "", "reset reason")
	skillCmd.Flags().StringP("name", "n", "mytask", "skill name")
	installHooksCmd.Flags().String("hook-type", "", "hook type (precreate, postcreate, prestart, poststart, presubmit, postsubmit, prereset, postreset)")
	installHooksCmd.Flags().BoolP("force", "f", false, "overwrite existing hook")

	rootCmd.AddCommand(createCmd, listCmd, getCmd, startCmd, submitCmd, completeCmd, resetCmd, skillCmd, installHooksCmd)
}

// Predefined hook templates
var predefinedHooks = map[string]struct {
	filename string
	content  string
}{
	"git-reset": {
		filename: "git-reset.sh",
		content: `#!/bin/bash
# Git reset hook - resets git state before task operation
# Can be customized by editing this file

set -e

# Reset to clean state
git reset --hard HEAD
git clean -fd

echo "Git repository reset to clean state"
`,
	},
	"git-commit": {
		filename: "git-commit.sh",
		content: `#!/bin/bash
# Git commit hook - commits changes after task operation
# Uses task info from environment variables

set -e

# Check if there are changes to commit
if git diff --quiet && git diff --cached --quiet; then
    echo "No changes to commit"
    exit 0
fi

# Generate commit message based on hook type and task info
COMMIT_MSG="[$TASK_STATUS] $TASK_NAME"

if [ -n "$TASK_DESCRIPTION" ]; then
    COMMIT_MSG="$COMMIT_MSG

$TASK_DESCRIPTION"
fi

git add -A
git commit -m "$COMMIT_MSG"

echo "Changes committed: $COMMIT_MSG"
`,
	},
	"notify": {
		filename: "notify.sh",
		content: `#!/bin/bash
# Notification hook - sends notification about task operation
# Requires notify-send (Linux) or terminal-notifier (macOS)

MESSAGE="Task: $TASK_NAME (ID: $TASK_ID)
Status: $TASK_STATUS"

if command -v notify-send &> /dev/null; then
    notify-send "MyTask" "$MESSAGE"
elif command -v terminal-notifier &> /dev/null; then
    terminal-notifier -title "MyTask" -message "$MESSAGE"
else
    echo "No notification system found"
fi
`,
	},
	"log": {
		filename: "log.sh",
		content: `#!/bin/bash
# Logging hook - logs task operations to a file

LOG_FILE="${MYTASK_LOG_FILE:-$HOME/.mytask/task.log}"
TIMESTAMP=$(date '+%Y-%m-%d %H:%M:%S')

echo "[$TIMESTAMP] Task #$TASK_ID: $TASK_NAME | Status: $TASK_STATUS | Desc: $TASK_DESCRIPTION" >> "$LOG_FILE"
`,
	},
}

var installHooksCmd = &cobra.Command{
	Use:   "install-hooks <hook-name>",
	Short: "Install a predefined hook",
	Long: `Install a predefined hook to the hooks directory.

Available hooks:
  git-reset   - Reset git state before operation
  git-commit  - Commit changes after operation
  notify      - Send desktop notification
  log         - Log operations to a file

Examples:
  mytask install-hooks git-commit --hook-type postcreate
  mytask install-hooks git-reset --hook-type prestart
  mytask install-hooks notify --hook-type poststart
  mytask install-hooks log --hook-type postsubmit`,
	Args: cobra.ExactArgs(1),
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Skip store initialization, only need config
		var err error
		cfg, err = config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		hookName := args[0]
		hookType, _ := cmd.Flags().GetString("hook-type")
		force, _ := cmd.Flags().GetBool("force")

		if hookType == "" {
			return fmt.Errorf("--hook-type is required (precreate, postcreate, prestart, poststart, presubmit, postsubmit, prereset, postreset)")
		}

		// Validate hook type
		validTypes := []string{"precreate", "postcreate", "prestart", "poststart", "presubmit", "postsubmit", "prereset", "postreset"}
		valid := false
		for _, t := range validTypes {
			if t == hookType {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("invalid hook type '%s', must be one of: %s", hookType, strings.Join(validTypes, ", "))
		}

		// Get predefined hook
		hook, ok := predefinedHooks[hookName]
		if !ok {
			available := make([]string, 0, len(predefinedHooks))
			for name := range predefinedHooks {
				available = append(available, name)
			}
			return fmt.Errorf("unknown hook '%s', available: %s", hookName, strings.Join(available, ", "))
		}

		// Create hook directory
		hookDir := filepath.Join(cfg.HooksDir, hookType)
		if err := os.MkdirAll(hookDir, 0755); err != nil {
			return fmt.Errorf("failed to create hook directory: %w", err)
		}

		// Create hook script
		scriptPath := filepath.Join(hookDir, hook.filename)
		if _, err := os.Stat(scriptPath); err == nil && !force {
			return fmt.Errorf("hook already exists: %s (use --force to overwrite)", scriptPath)
		}

		if err := os.WriteFile(scriptPath, []byte(hook.content), 0755); err != nil {
			return fmt.Errorf("failed to write hook script: %w", err)
		}

		fmt.Printf("Hook installed: %s -> %s\n", hookName, scriptPath)
		return nil
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		// Skip store cleanup
	},
}

var skillCmd = &cobra.Command{
	Use:   "skill <directory>",
	Short: "Generate a skill file for mytask",
	Args:  cobra.ExactArgs(1),
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return nil // Skip config and store initialization
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		skillName, _ := cmd.Flags().GetString("name")
		baseDir := args[0]

		// Create skills/<name>/SKILL.md structure
		skillsDir := filepath.Join(baseDir, "skills")
		skillPkgDir := filepath.Join(skillsDir, skillName)
		skillPath := filepath.Join(skillPkgDir, "SKILL.md")

		if err := os.MkdirAll(skillPkgDir, 0755); err != nil {
			return fmt.Errorf("failed to create skill directory: %w", err)
		}

		if _, err := os.Stat(skillPath); err == nil {
			return fmt.Errorf("skill file already exists: %s", skillPath)
		}

		content := `---
name: ` + skillName + `
description: "Use this skill whenever the user wants to manage tasks. Triggers include: creating tasks, listing tasks, starting tasks, completing tasks, skipping tasks. Use when the user mentions 'task', 'todo', 'work item', or needs to track work progress. Do NOT use for general programming tasks unrelated to task management."
---

# ` + skillName + ` - Task Management Tool

A CLI tool for managing tasks with SQLite storage and hook support.

## Quick Reference

| Command | Description | Example |
|---------|-------------|---------|
| create | Create a new task | ` + "`mytask create -n \"Task name\" -d \"Description\"`" + ` |
| list | List all tasks | ` + "`mytask list`" + ` |
| get | Get task details | ` + "`mytask get 1`" + ` |
| start | Start a task | ` + "`mytask start 1`" + ` |
| submit | Submit a task | ` + "`mytask submit 1`" + ` |
| complete | Complete a task | ` + "`mytask complete 1`" + ` |
| reset | Skip a task | ` + "`mytask reset 1 -r \"Reason\"`" + ` |

## Task Status

| Status | Description |
|--------|-------------|
| pending | Task is waiting to be started |
| in_progress | Task is currently being worked on (only one at a time) |
| completed | Task is finished |
| skipped | Task was abandoned/skipped |

## Status Flow

` + "```\ncreate → pending\n         ↓ start\n      in_progress\n      ↓        ↓\ncomplete    reset\n      ↓        ↓\n completed  skipped\n```" + `

## Configuration

Set ` + "`MYTASK_CONFIG_DIR`" + ` environment variable to specify config directory. Default: ` + "`./.mytask/`" + ` (current directory)

` + "```\n<config-dir>/\n├── tasks.db             # SQLite database\n└── hooks/\n    ├── precreate/       # Scripts run before create\n    ├── postcreate/      # Scripts run after create\n    ├── prestart/        # Scripts run before start\n    ├── poststart/       # Scripts run after start\n    ├── presubmit/       # Scripts run before submit\n    ├── postsubmit/      # Scripts run after submit\n    ├── prereset/        # Scripts run before reset\n    └── postreset/       # Scripts run after reset\n```" + `

## Hooks

Hooks are executable scripts in the config directory:

### Create Hooks
- **precreate/**: Run before ` + "`mytask create`" + `. Failure blocks the task from being created.
- **postcreate/**: Run after ` + "`mytask create`" + `. Failure shows a warning.

### Start Hooks
- **prestart/**: Run before ` + "`mytask start`" + `. Failure blocks the task from starting.
- **poststart/**: Run after ` + "`mytask start`" + `. Failure shows a warning.

### Submit Hooks
- **presubmit/**: Run before ` + "`mytask submit`" + `. Failure blocks the submit.
- **postsubmit/**: Run after ` + "`mytask submit`" + `. Failure shows a warning.

### Reset Hooks
- **prereset/**: Run before ` + "`mytask reset`" + `. Failure blocks the task from being reset.
- **postreset/**: Run after ` + "`mytask reset`" + `. Failure shows a warning.

Scripts run in alphabetical order by filename.

### Environment Variables

Hooks receive task info via environment variables:

- ` + "`TASK_ID`" + ` - Task ID
- ` + "`TASK_NAME`" + ` - Task name
- ` + "`TASK_STATUS`" + ` - Current status
- ` + "`TASK_DESCRIPTION`" + ` - Task description

### Example Hook

` + "```bash\n#!/bin/bash\n# .mytask/hooks/prestart/01-notify.sh\necho \"Starting task: $TASK_NAME\" | mail -s \"Task Started\" user@example.com\n```" + `

## Predefined Hooks

Use ` + "`mytask install-hooks`" + ` to install predefined hooks:

| Hook | Description | Recommended Type |
|------|-------------|------------------|
| git-reset | Reset git state before operation | prestart |
| git-commit | Commit changes after operation | postcreate, poststart |
| notify | Send desktop notification | poststart, postreset |
| log | Log operations to a file | any |

Example:
` + "```bash\nmytask install-hooks git-commit --hook-type postcreate\nmytask install-hooks git-reset --hook-type prestart\n```" + `

## Key Constraints

1. **Only one task can be in_progress** - Cannot start a new task while another is in_progress (complete or reset the current task first)
2. **reset requires a reason** - Must provide -r flag with explanation
3. **Hooks must be executable** - Non-executable scripts cause errors
`

		if err := os.WriteFile(skillPath, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write skill file: %w", err)
		}

		fmt.Printf("Skill file created: %s\n", skillPath)
		return nil
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		// Skip store cleanup
	},
}