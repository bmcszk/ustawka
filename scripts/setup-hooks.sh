#!/bin/bash

# Setup script for installing git hooks
# Run this after cloning the repository to set up development environment

set -e

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}🔧 Setting up Ustawka development environment...${NC}"

# Check if we're in the right directory
if [[ ! -f "Makefile" ]] || [[ ! -d ".githooks" ]]; then
    echo -e "${RED}❌ ERROR: Please run this script from the project root directory.${NC}"
    exit 1
fi

# Create .git/hooks directory if it doesn't exist
mkdir -p .git/hooks

# Install pre-commit hook
if [[ -f ".githooks/pre-commit" ]]; then
    cp .githooks/pre-commit .git/hooks/pre-commit
    chmod +x .git/hooks/pre-commit
    echo -e "${GREEN}✓ Pre-commit hook installed${NC}"
else
    echo -e "${YELLOW}⚠️  Pre-commit hook not found in .githooks/pre-commit${NC}"
fi

# Check if make check works
echo -e "${YELLOW}🔍 Testing make check...${NC}"
if make check >/dev/null 2>&1; then
    echo -e "${GREEN}✓ make check works correctly${NC}"
else
    echo -e "${YELLOW}⚠️  make check has issues - you may need to fix linting/test problems${NC}"
    echo -e "   Run 'make check' manually to see details"
fi

echo -e ""
echo -e "${GREEN}🎉 Development environment setup complete!${NC}"
echo -e ""
echo -e "${BLUE}What happens now:${NC}"
echo -e "• Pre-commit hook will run on every commit"
echo -e "• Direct commits to master/main/RELEASE are blocked"
echo -e "• Code must pass 'make check' before committing"
echo -e "• Use feature branches for all development"
echo -e ""
echo -e "${BLUE}Example workflow:${NC}"
echo -e "  git checkout -b feat/my-feature"
echo -e "  # Make changes..."
echo -e "  git add ."
echo -e "  git commit -m 'feat: add my feature'"
echo -e "  git push -u origin feat/my-feature"
echo -e ""
echo -e "${YELLOW}Happy coding! 🚀${NC}"