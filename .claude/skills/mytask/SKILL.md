---
name: mytask
description: "Use this skill whenever the user wants to manage tasks. Triggers include: creating tasks, listing tasks, starting tasks, completing tasks, skipping tasks. Use when the user mentions 'task', 'todo', 'work item', or needs to track work progress. Do NOT use for general programming tasks unrelated to task management."
---

# mytask - Task Management Tool

A CLI tool for managing tasks with SQLite storage and hook support.

## Quick Reference

| Command | Description | Example |
|---------|-------------|---------|
| create | Create a new task | `mytask create -n "Task name" -d "Description"` |
| list | List all tasks | `mytask list` |
| get | Get task details | `mytask get 1` |
| start | Start a task | `mytask start 1` |
| complete | Complete a task | `mytask complete 1` |
| reset | Skip a task | `mytask reset 1 -r "Reason"` |

## Task Status

| Status | Description |
|--------|-------------|
| pending | Task is waiting to be started |
| in_progress | Task is currently being worked on (only one at a time) |
| completed | Task is finished |
| skipped | Task was abandoned/skipped |

## Status Flow

```
create → pending
         ↓ start
      in_progress
      ↓        ↓
complete    reset
      ↓        ↓
 completed  skipped
```

## Configuration

Set `MYTASK_CONFIG_DIR` environment variable to specify config directory. Default: `~/.mytask/`

```
<config-dir>/
├── tasks.db             # SQLite database
└── hooks/
    ├── pre_start/       # Scripts run before start
    └── post_start/      # Scripts run after start
```

## Hooks

Hooks are executable scripts in the config directory:

- **pre_start/**: Run before `mytask start`. Failure blocks the task from starting.
- **post_start/**: Run after `mytask start`. Failure shows a warning.

Scripts run in alphabetical order by filename.

### Environment Variables

Hooks receive task info via environment variables:

- `TASK_ID` - Task ID
- `TASK_NAME` - Task name
- `TASK_STATUS` - Current status
- `TASK_DESCRIPTION` - Task description

### Example Hook

```bash
#!/bin/bash
# ~/.mytask/hooks/pre_start/01-notify.sh
echo "Starting task: $TASK_NAME" | mail -s "Task Started" user@example.com
```

## Key Constraints

1. **Only one task can be in_progress** - Starting a new task automatically sets any other in_progress task to pending
2. **reset requires a reason** - Must provide -r flag with explanation
3. **Hooks must be executable** - Non-executable scripts cause errors
