#!/bin/bash
# Example postsubmit hook
# This hook runs AFTER a task is submitted
# Task info available: TASK_ID, TASK_NAME, TASK_DESCRIPTION, TASK_STATUS

echo "Task #$TASK_ID submitted successfully!"
echo "New status: $TASK_STATUS"

# Example: Push changes
# git push origin HEAD

# Example: Notify team
# curl -X POST -d "Task $TASK_NAME submitted" $SLACK_WEBHOOK_URL