#!/bin/bash

# GitHub Setup Script for Workie
# Run this script after creating the repository on GitHub

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}🚀 GitHub Setup Script for Workie${NC}"
echo "============================================="

# Check if username is provided
if [ -z "$1" ]; then
    echo -e "${RED}❌ Error: GitHub username is required${NC}"
    echo "Usage: $0 <github-username>"
    echo "Example: $0 johndoe"
    exit 1
fi

USERNAME="$1"
REPO_URL="git@github.com:$USERNAME/workie.git"

echo -e "${YELLOW}📋 Pre-flight checks...${NC}"

# Check if we're in the right directory
if [ ! -f "go.mod" ] || [ ! -d ".git" ]; then
    echo -e "${RED}❌ Error: This script must be run from the workie project root directory${NC}"
    exit 1
fi

# Check if git is configured
if [ -z "$(git config user.name)" ] || [ -z "$(git config user.email)" ]; then
    echo -e "${RED}❌ Error: Git is not configured. Please run:${NC}"
    echo "git config --global user.name 'Your Name'"
    echo "git config --global user.email 'your.email@example.com'"
    exit 1
fi

echo -e "${GREEN}✅ Pre-flight checks passed${NC}"

# Check if remote already exists
if git remote get-url origin >/dev/null 2>&1; then
    echo -e "${YELLOW}⚠️  Origin remote already exists. Removing it...${NC}"
    git remote remove origin
fi

# Add GitHub remote
echo -e "${YELLOW}🔗 Adding GitHub remote: $REPO_URL${NC}"
git remote add origin "$REPO_URL"

# Verify remote was added
git remote -v

# Push main branch
echo -e "${YELLOW}📤 Pushing main branch to GitHub...${NC}"
git push -u origin main

# Push tags
echo -e "${YELLOW}🏷️  Pushing tags to GitHub...${NC}"
git push --tags

echo ""
echo -e "${GREEN}✅ GitHub setup completed successfully!${NC}"
echo ""
echo -e "${YELLOW}🎯 Next steps:${NC}"
echo "1. Visit: https://github.com/$USERNAME/workie"
echo "2. Add repository description: 'A flexible worker queue system for distributed task processing'"
echo "3. Add topics: 'go', 'queue', 'worker', 'distributed', 'task-processing', 'microservices'"
echo "4. Set repository website (if applicable)"
echo "5. Consider enabling GitHub Pages for documentation"
echo ""
echo -e "${GREEN}🎉 Your Workie project is now live on GitHub!${NC}"
