package presets

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"mytask/internal/hook"
)

// ValidHookTypes contains all valid hook type names
var ValidHookTypes = []string{
	"precreate", "postcreate",
	"prestart", "poststart",
	"presubmit", "postsubmit",
	"prereset", "postreset",
}

// IsValidHookType checks if the given hook type is valid
func IsValidHookType(hookType string) bool {
	for _, t := range ValidHookTypes {
		if t == hookType {
			return true
		}
	}
	return false
}

// HookTemplate represents a predefined hook template
type HookTemplate struct {
	Filename string
	Content  string
}

// PredefinedHooks contains all predefined hook templates
var PredefinedHooks = map[string]HookTemplate{
	"git-reset": {
		Filename: "git-reset.sh",
		Content: `#!/bin/bash
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
		Filename: "git-commit.sh",
		Content: `#!/bin/bash
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
		Filename: "notify.sh",
		Content: `#!/bin/bash
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
		Filename: "log.sh",
		Content: `#!/bin/bash
# Logging hook - logs task operations to a file

LOG_FILE="${MYTASK_LOG_FILE:-$HOME/.mytask/task.log}"
TIMESTAMP=$(date '+%Y-%m-%d %H:%M:%S')

echo "[$TIMESTAMP] Task #$TASK_ID: $TASK_NAME | Status: $TASK_STATUS | Desc: $TASK_DESCRIPTION" >> "$LOG_FILE"
`,
	},
}

// GetHookNames returns a list of available predefined hook names
func GetHookNames() []string {
	names := make([]string, 0, len(PredefinedHooks))
	for name := range PredefinedHooks {
		names = append(names, name)
	}
	return names
}

// InstallHook installs a predefined hook to the specified hooks directory
func InstallHook(hooksDir, hookName, hookType string, force bool) (string, error) {
	if hookType == "" {
		return "", fmt.Errorf("--hook-type is required (%s)", strings.Join(ValidHookTypes, ", "))
	}

	if !IsValidHookType(hookType) {
		return "", fmt.Errorf("invalid hook type '%s', must be one of: %s", hookType, strings.Join(ValidHookTypes, ", "))
	}

	template, ok := PredefinedHooks[hookName]
	if !ok {
		return "", fmt.Errorf("unknown hook '%s', available: %s", hookName, strings.Join(GetHookNames(), ", "))
	}

	// Create hook directory
	hookDir := filepath.Join(hooksDir, hookType)
	if err := os.MkdirAll(hookDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create hook directory: %w", err)
	}

	// Create hook script
	scriptPath := filepath.Join(hookDir, template.Filename)
	if _, err := os.Stat(scriptPath); err == nil && !force {
		return "", fmt.Errorf("hook already exists: %s (use --force to overwrite)", scriptPath)
	}

	if err := os.WriteFile(scriptPath, []byte(template.Content), 0755); err != nil {
		return "", fmt.Errorf("failed to write hook script: %w", err)
	}

	return scriptPath, nil
}

// GenerateSkillFile generates a skill file for the given directory
func GenerateSkillFile(baseDir, skillName string) (string, error) {
	skillsDir := filepath.Join(baseDir, "skills")
	skillPkgDir := filepath.Join(skillsDir, skillName)
	skillPath := filepath.Join(skillPkgDir, "SKILL.md")

	if err := os.MkdirAll(skillPkgDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create skill directory: %w", err)
	}

	if _, err := os.Stat(skillPath); err == nil {
		return "", fmt.Errorf("skill file already exists: %s", skillPath)
	}

	content := generateSkillContent(skillName)

	if err := os.WriteFile(skillPath, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("failed to write skill file: %w", err)
	}

	return skillPath, nil
}

func generateSkillContent(skillName string) string {
	return `---
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
}

// GetHookExecutor returns a hook executor for the given hooks directory
func GetHookExecutor(hooksDir string) hook.Executor {
	return hook.NewExecutor(hooksDir)
}