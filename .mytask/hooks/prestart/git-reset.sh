#!/bin/bash
# Git reset hook - resets git state before task operation
# Can be customized by editing this file

set -e

# Reset to clean state
git reset --hard HEAD
git clean -fd

echo "Git repository reset to clean state"
