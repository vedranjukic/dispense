#!/bin/bash

# Test script for remote sandbox creation with GitHub issue
cd /workspaces/dispense

# Input for the CLI prompts:
# 1. Branch name (empty = auto-generated)
# 2. Task description (GitHub issue URL)
echo -e "\n\nhttps://github.com/anthropics/claude-code/issues/2564\n" | timeout 300s yarn nx serve cli --remote

echo "Exit code: $?"