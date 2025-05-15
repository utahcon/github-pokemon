#!/bin/bash
# set-dev-version.sh - Set a development version based on branch name
set -e

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Get the current branch
CURRENT_BRANCH=$(git rev-parse --abbrev-ref HEAD)

# Skip if on main
if [ "$CURRENT_BRANCH" = "main" ]; then
    echo -e "${RED}Error: Cannot set development version on main branch${NC}"
    exit 1
fi

# Get base version from VERSION file if it exists
if [ -f "VERSION" ]; then
    BASE_VERSION=$(cat VERSION | tr -d '[:space:]')
    
    # Strip any existing -branch suffix
    BASE_VERSION=$(echo $BASE_VERSION | sed 's/-[a-zA-Z0-9_\-]*$//')
    
    # Ensure we have a semantic version base
    if ! [[ $BASE_VERSION =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
        echo -e "${RED}Current version '$BASE_VERSION' is not a valid base semantic version${NC}"
        echo -e "${YELLOW}Defaulting to 0.0.0 as base version${NC}"
        BASE_VERSION="0.0.0"
    fi
else
    # Default to 0.0.0 if VERSION doesn't exist
    BASE_VERSION="0.0.0"
    echo -e "${YELLOW}VERSION file not found, using $BASE_VERSION as base version${NC}"
fi

# Sanitize branch name for version use (remove special chars, convert / to _, etc)
CLEAN_BRANCH=$(echo $CURRENT_BRANCH | sed 's/[^a-zA-Z0-9]/_/g')

# Create development version
DEV_VERSION="${BASE_VERSION}-${CLEAN_BRANCH}"

# Check if a specific version was requested
if [ ! -z "$1" ]; then
    if [[ $1 =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
        # If valid semver provided, use it as base
        DEV_VERSION="${1}-${CLEAN_BRANCH}"
    else
        echo -e "${RED}Provided version '$1' is not a valid semantic version${NC}"
        echo -e "${YELLOW}Using $BASE_VERSION as base instead${NC}"
    fi
fi

# Set the version
echo "$DEV_VERSION" > VERSION
echo -e "${GREEN}Development version set: ${YELLOW}$DEV_VERSION${NC}"
echo -e "${YELLOW}This version will not pass strict semantic version checks${NC}"
echo -e "${YELLOW}but will be allowed during development on branch: $CURRENT_BRANCH${NC}"

# Check if CHANGELOG.md has corresponding entry
if [ -f "CHANGELOG.md" ]; then
    if ! grep -q "\[$DEV_VERSION\]" CHANGELOG.md; then
        echo -e "\n${YELLOW}Warning: CHANGELOG.md does not contain an entry for version $DEV_VERSION${NC}"
        echo -e "${YELLOW}Consider adding this for comprehensive version tracking${NC}"
    fi
fi

echo -e "\n${GREEN}✓ Development version set successfully${NC}"
exit 0