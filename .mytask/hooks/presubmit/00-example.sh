#!/bin/bash
# Example presubmit hook
# This hook runs BEFORE a task is submitted
# Task info available: TASK_ID, TASK_NAME, TASK_DESCRIPTION, TASK_STATUS

echo "Validating task #$TASK_ID before submit..."

# Example: Run tests
# if ! go test ./...; then
#     echo "Error: Tests failed, cannot submit"
#     exit 1
# fi

# Example: Check for uncommitted changes
# if ! git diff --quiet; then
#     echo "Error: You have uncommitted changes"
#     exit 1
# fi