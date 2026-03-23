#!/bin/bash
# Example postreset hook
# This hook runs AFTER a task is reset (skipped)
# Task info available: TASK_ID, TASK_NAME, TASK_DESCRIPTION, TASK_STATUS

echo "Task #$TASK_ID has been reset"
echo "Name: $TASK_NAME"

# Example: Clean up working directory
# rm -rf work-dir-$TASK_ID

# Example: Notify stakeholders
# echo "Task $TASK_NAME was skipped" | mail -s "Task Reset" team@example.com