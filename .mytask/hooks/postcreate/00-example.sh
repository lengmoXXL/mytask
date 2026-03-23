#!/bin/bash
# Example postcreate hook
# This hook runs AFTER a task is created
# Task info available: TASK_ID, TASK_NAME, TASK_DESCRIPTION, TASK_STATUS

echo "Task #$TASK_ID created successfully!"
echo "Name: $TASK_NAME"
echo "Status: $TASK_STATUS"

# Example: Send notification
# notify-send "New Task" "$TASK_NAME (ID: $TASK_ID)"