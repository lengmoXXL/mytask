#!/bin/bash
# Example prereset hook
# This hook runs BEFORE a task is reset (skipped)
# Task info available: TASK_ID, TASK_NAME, TASK_DESCRIPTION, TASK_STATUS

echo "Preparing to reset task #$TASK_ID: $TASK_NAME"

# Example: Archive current work
# git stash push -m "task-$TASK_ID-$TASK_NAME"

# Example: Check for important files before reset
# if [ -f "important-config.yaml" ]; then
#     echo "Warning: important-config.yaml exists, please review"
# fi