#!/bin/bash
# Hook: WorktreeCreate
# Copies Taskfile.local.yml from the main repo into the newly created worktree.

input=$(cat)

worktree_path=$(echo "$input" | python3 -c "
import sys, json
data = json.load(sys.stdin)
print(data.get('worktree_path', ''))
" 2>/dev/null)

src="${CLAUDE_PROJECT_DIR}/Taskfile.local.yml"

if [ -z "$worktree_path" ]; then
  exit 0
fi

if [ ! -f "$src" ]; then
  exit 0
fi

cp "$src" "${worktree_path}/Taskfile.local.yml"
