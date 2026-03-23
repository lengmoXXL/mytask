#!/bin/bash
# Example prestart hook
# This hook runs BEFORE a task is started
# Task info available: TASK_ID, TASK_NAME, TASK_DESCRIPTION, TASK_STATUS

echo "Preparing to start task #$TASK_ID: $TASK_NAME"

# Example: Check if dependencies are ready
# if ! command -v required-tool &> /dev/null; then
#     echo "Error: required-tool is not installed"
#     exit 1
# fi

# Example: Prepare environment
# git pull origin main