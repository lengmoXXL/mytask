#!/bin/bash
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
