#!/bin/bash
# sync-versions.sh - Synchronize version references across project files
set -e

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Check if a version was provided
if [ -z "$1" ]; then
    # Extract version from VERSION file
    if [ -f "VERSION" ]; then
        VERSION=$(cat VERSION | tr -d '[:space:]')
        echo -e "${YELLOW}Detected version: ${GREEN}v${VERSION}${NC}"
    else
        echo -e "${RED}Error: VERSION file not found and no version provided${NC}"
        echo -e "Usage: $0 [version]"
        exit 1
    fi
else
    # Use provided version
    VERSION=$1
    echo -e "${YELLOW}Using provided version: ${GREEN}v${VERSION}${NC}"
fi

# Validate version follows semver
if ! [[ $VERSION =~ ^[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9\.]+)?(\+[a-zA-Z0-9\.]+)?$ ]]; then
    echo -e "${RED}Error: Version '$VERSION' does not follow semantic versioning (semver) format${NC}"
    exit 1
fi

# Initialize counters
FILES_CHECKED=0
FILES_UPDATED=0

# Function to update version in a file
update_version() {
    local file=$1
    local pattern=$2
    local replacement=$3
    
    if [ -f "$file" ]; then
        FILES_CHECKED=$((FILES_CHECKED + 1))
        
        if grep -q "$pattern" "$file"; then
            # Create a backup
            cp "$file" "${file}.bak"
            
            # Replace the pattern
            sed -i "s/$pattern/$replacement/g" "$file"
            
            # Check if file was modified
            if diff -q "$file" "${file}.bak" > /dev/null; then
                echo -e "  ${YELLOW}No changes needed in ${file}${NC}"
                rm "${file}.bak"
            else
                echo -e "  ${GREEN}✓ Updated version in ${file}${NC}"
                FILES_UPDATED=$((FILES_UPDATED + 1))
                rm "${file}.bak"
            fi
        else
            echo -e "  ${YELLOW}Pattern not found in ${file}${NC}"
        fi
    else
        echo -e "  ${RED}File not found: ${file}${NC}"
    fi
}

echo -e "\n${YELLOW}Synchronizing version references to ${GREEN}v${VERSION}${NC}\n"

# Update version in README.md
echo -e "${YELLOW}Checking README.md...${NC}"
update_version "README.md" "Repository Manager (v[0-9]\+\.[0-9]\+\.[0-9]\+)" "Repository Manager (v${VERSION})"

# Update version in CHANGELOG.md if necessary (only if Unreleased is used)
echo -e "\n${YELLOW}Checking CHANGELOG.md...${NC}"
if grep -q "## \[Unreleased\]" "CHANGELOG.md"; then
    update_version "CHANGELOG.md" "## \[Unreleased\]" "## [${VERSION}] - $(date +%Y-%m-%d)"
fi

# Update version in VERSION file
echo -e "\n${YELLOW}Updating VERSION file...${NC}"
echo "${VERSION}" > VERSION
echo -e "  ${GREEN}✓ Updated VERSION file${NC}"

# Search for other files containing version references
echo -e "\n${YELLOW}Searching for other files with version references...${NC}"
for file in $(grep -r --include="*.md" --include="*.go" --include="*.html" --include="*.txt" -l "v[0-9]\+\.[0-9]\+\.[0-9]\+" . | grep -v "CHANGELOG.md" | grep -v "VERSION" | grep -v "README.md"); do
    echo -e "${YELLOW}Checking ${file}...${NC}"
    
    # Update explicit version references (customize patterns as needed)
    update_version "$file" "v[0-9]\+\.[0-9]\+\.[0-9]\+ release" "v${VERSION} release"
    update_version "$file" "version [0-9]\+\.[0-9]\+\.[0-9]\+" "version ${VERSION}"
    update_version "$file" "\"version\": \"[0-9]\+\.[0-9]\+\.[0-9]\+\"" "\"version\": \"${VERSION}\""
    update_version "$file" "version = \"[0-9]\+\.[0-9]\+\.[0-9]\+\"" "version = \"${VERSION}\""
done

echo -e "\n${GREEN}Version synchronization complete!${NC}"
echo -e "${YELLOW}Files checked: ${GREEN}${FILES_CHECKED}${NC}"
echo -e "${YELLOW}Files updated: ${GREEN}${FILES_UPDATED}${NC}"

if [ $FILES_UPDATED -gt 0 ]; then
    echo -e "\n${YELLOW}Next steps:${NC}"
    echo -e "1. Review the changes"
    echo -e "2. Commit the changes:"
    echo -e "   ${GREEN}git commit -am \"docs: sync version references to v${VERSION}\"${NC}"
fi

exit 0