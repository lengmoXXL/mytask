#!/bin/bash
# Example precreate hook
# This hook runs BEFORE a task is created
# Task info available: TASK_NAME, TASK_DESCRIPTION, TASK_STATUS

echo "About to create task: $TASK_NAME"
echo "Description: $TASK_DESCRIPTION"

# Example: Validate task name format
# if [[ ! "$TASK_NAME" =~ ^[A-Z] ]]; then
#     echo "Error: Task name must start with uppercase letter"
#     exit 1
# fi