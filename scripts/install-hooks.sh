#!/bin/bash
# Script to install Git hooks for the project

# Set colors for output
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${YELLOW}Installing Git hooks for GitHub Pokemon repository...${NC}"

# Get directory of this script
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
REPO_ROOT="$(dirname "$SCRIPT_DIR")"
HOOKS_DIR="$REPO_ROOT/.githooks"
GIT_HOOKS_DIR="$REPO_ROOT/.git/hooks"

# Check if .githooks directory exists
if [ ! -d "$HOOKS_DIR" ]; then
  echo -e "${RED}Error: .githooks directory not found at $HOOKS_DIR${NC}"
  exit 1
fi

# Check if Git repo exists
if [ ! -d "$REPO_ROOT/.git" ]; then
  echo -e "${RED}Error: .git directory not found. Are you in a Git repository?${NC}"
  exit 1
fi

# Make all hook scripts executable
chmod +x "$HOOKS_DIR"/*
echo -e "${GREEN}Made hook scripts executable${NC}"

# Configure Git to use our hooks directory
echo -e "${YELLOW}Configuring Git to use hooks from: ${HOOKS_DIR}${NC}"
git config core.hooksPath .githooks

echo -e "${GREEN}Success! Git hooks installed and configured.${NC}"
echo -e "${YELLOW}The following hooks are now active:${NC}"

# List installed hooks
ls -la "$HOOKS_DIR" | grep -v "total" | grep -v "^\." | awk '{print "  - " $9}'

echo -e "\n${YELLOW}To test the pre-push hook, try pushing to main:${NC}"
echo -e "  git push origin main\n"
echo -e "${YELLOW}Note: You can bypass hooks with the --no-verify flag:${NC}"
echo -e "  git push --no-verify origin main\n"

exit 0