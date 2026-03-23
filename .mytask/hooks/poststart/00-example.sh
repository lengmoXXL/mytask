#!/bin/bash
# Example poststart hook
# This hook runs AFTER a task is started
# Task info available: TASK_ID, TASK_NAME, TASK_DESCRIPTION, TASK_STATUS

echo "Task #$TASK_ID started!"
echo "Current status: $TASK_STATUS"

# Example: Log to file
# echo "$(date): Started task #$TASK_ID - $TASK_NAME" >> ~/task-log.txt

# Example: Create working branch
# git checkout -b "task-$TASK_ID-$TASK_NAME" 2>/dev/null || true