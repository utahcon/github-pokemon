#!/bin/bash
# Install git hooks for the project.
# Sets git to use .githooks/ as the hooks directory.

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(dirname "$SCRIPT_DIR")"
HOOKS_DIR="$REPO_ROOT/.githooks"

if [ ! -d "$HOOKS_DIR" ]; then
  echo "Error: .githooks directory not found at $HOOKS_DIR"
  exit 1
fi

chmod +x "$HOOKS_DIR"/*
git config core.hooksPath .githooks

echo "Git hooks installed. Active hooks:"
ls -1 "$HOOKS_DIR"
