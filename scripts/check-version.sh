#!/bin/bash
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}Running version check...${NC}"

# Get the current branch
CURRENT_BRANCH=$(git rev-parse --abbrev-ref HEAD)
if [ "$CURRENT_BRANCH" == "main" ]; then
    echo -e "${YELLOW}Warning: You are on the main branch${NC}"
    exit 0
fi

# Determine if we're pushing to main (strict version check)
# Default to development mode (loose check)
STRICT_CHECK=false

# Check arguments for --strict flag
for arg in "$@"; do
    if [ "$arg" == "--strict" ]; then
        STRICT_CHECK=true
        echo -e "${YELLOW}Running in strict mode (for PRs to main)${NC}"
        break
    fi
done

# Fetch main branch to ensure we have the latest
git fetch origin main --quiet

# Extract versions
if [ -f "VERSION" ]; then
    PR_VERSION=$(cat VERSION | tr -d '[:space:]')
else
    PR_VERSION="0.0.0-dev"
    echo -e "${RED}VERSION file not found, using $PR_VERSION as fallback${NC}"
fi

if git show origin/main:VERSION &>/dev/null; then
    MAIN_VERSION=$(git show origin/main:VERSION | tr -d '[:space:]')
else
    MAIN_VERSION="0.0.0"
    echo -e "${YELLOW}No VERSION file found in main branch, using $MAIN_VERSION as base version${NC}"
fi

echo -e "Main branch version: ${YELLOW}$MAIN_VERSION${NC}"
echo -e "Current branch version: ${YELLOW}$PR_VERSION${NC}"

# If not in strict mode, allow branch-based versioning
if [ "$STRICT_CHECK" = false ]; then
    # Check if version is branch-based
    if [[ "$PR_VERSION" == *"-$CURRENT_BRANCH"* || "$PR_VERSION" == *"-dev"* ]]; then
        echo -e "${GREEN}✓ Using development version format: $PR_VERSION${NC}"
        echo -e "${YELLOW}Note: Semantic version rules will only be enforced when creating a PR to main${NC}"
        exit 0
    fi
fi

# Strict check mode - validate semver format
if ! [[ $PR_VERSION =~ ^[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9\.]+)?(\+[a-zA-Z0-9\.]+)?$ ]]; then
    echo -e "${RED}Error: Version '$PR_VERSION' does not follow semantic versioning (semver) format${NC}"
    echo -e "${YELLOW}Expected format: MAJOR.MINOR.PATCH (e.g., 1.2.3)${NC}"
    if [ "$STRICT_CHECK" = true ]; then
        exit 1
    else
        echo -e "${YELLOW}Warning: This will fail when creating a PR to main${NC}"
    fi
fi

# Convert to semver for comparison (only in strict mode)
if [ "$STRICT_CHECK" = true ]; then
    MAIN_MAJOR=$(echo $MAIN_VERSION | cut -d. -f1)
    MAIN_MINOR=$(echo $MAIN_VERSION | cut -d. -f2)
    MAIN_PATCH=$(echo $MAIN_VERSION | cut -d. -f3 | sed 's/[^0-9].*$//')
    
    PR_MAJOR=$(echo $PR_VERSION | cut -d. -f1)
    PR_MINOR=$(echo $PR_VERSION | cut -d. -f2)
    PR_PATCH=$(echo $PR_VERSION | cut -d. -f3 | sed 's/[^0-9].*$//')
    
    # Check if version incremented
    VERSION_INCREMENTED=false
    if [[ $PR_MAJOR -gt $MAIN_MAJOR ]]; then
        echo -e "${GREEN}✓ Major version increment detected${NC}"
        VERSION_INCREMENTED=true
    elif [[ $PR_MAJOR -eq $MAIN_MAJOR && $PR_MINOR -gt $MAIN_MINOR ]]; then
        echo -e "${GREEN}✓ Minor version increment detected${NC}"
        VERSION_INCREMENTED=true
    elif [[ $PR_MAJOR -eq $MAIN_MAJOR && $PR_MINOR -eq $MAIN_MINOR && $PR_PATCH -gt $MAIN_PATCH ]]; then
        echo -e "${GREEN}✓ Patch version increment detected${NC}"
        VERSION_INCREMENTED=true
    else
        echo -e "${RED}✗ Version must be incremented for PR to main.${NC}"
        VERSION_INCREMENTED=false
        if [ "$STRICT_CHECK" = true ]; then
            exit 1
        fi
    fi
else
    # In development mode, we don't care about version increments
    VERSION_INCREMENTED=true
    echo -e "${GREEN}✓ Development mode: version increment check skipped${NC}"
fi

# Check CHANGELOG update
CHANGELOG_UPDATED=false
if [ -f "CHANGELOG.md" ]; then
    if grep -q "\[$PR_VERSION\]" CHANGELOG.md; then
        if grep -q "## \[$PR_VERSION\]" CHANGELOG.md; then
            echo -e "${GREEN}✓ CHANGELOG.md has entry for version $PR_VERSION${NC}"
            CHANGELOG_UPDATED=true
            
            if [[ $(grep -c "\[$PR_VERSION\]" CHANGELOG.md) -gt 1 ]]; then
                echo -e "${YELLOW}⚠ Warning: Multiple entries for version $PR_VERSION found in CHANGELOG.md${NC}"
            fi
        else
            echo -e "${RED}✗ CHANGELOG.md does not contain a proper heading (## [$PR_VERSION]) for this version${NC}"
            if [ "$STRICT_CHECK" = true ]; then
                exit 1
            fi
        fi
    else
        if [ "$STRICT_CHECK" = true ]; then
            echo -e "${RED}✗ CHANGELOG.md does not contain any entry for version $PR_VERSION${NC}"
            exit 1
        else
            echo -e "${YELLOW}⚠ Warning: CHANGELOG.md does not contain any entry for version $PR_VERSION${NC}"
            echo -e "${YELLOW}  This will be required when creating a PR to main${NC}"
        fi
    fi
else
    echo -e "${RED}✗ CHANGELOG.md file not found${NC}"
    if [ "$STRICT_CHECK" = true ]; then
        exit 1
    fi
fi

# Final result
if [ "$STRICT_CHECK" = true ]; then
    if [ "$VERSION_INCREMENTED" = true ] && [ "$CHANGELOG_UPDATED" = true ]; then
        echo -e "\n${GREEN}✓ All checks passed! Version is properly incremented and CHANGELOG is updated.${NC}"
        echo -e "${GREEN}  Safe to create a pull request to main.${NC}"
        exit 0
    else
        echo -e "\n${RED}✗ Some checks failed. Please update the version and/or CHANGELOG before creating a PR to main.${NC}"
        exit 1
    fi
else
    echo -e "\n${GREEN}✓ Development checks passed!${NC}"
    if [ "$VERSION_INCREMENTED" = true ] && [ "$CHANGELOG_UPDATED" = true ]; then
        echo -e "${GREEN}✓ Version would pass strict checks for a PR to main.${NC}"
    else
        echo -e "${YELLOW}⚠ Warning: Current version would not pass checks for a PR to main.${NC}"
        echo -e "${YELLOW}  Remember to update VERSION and CHANGELOG.md before creating a PR to main.${NC}"
    fi
    exit 0
fi