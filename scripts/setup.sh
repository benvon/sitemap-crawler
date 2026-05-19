#!/bin/bash

# Setup script for sitemap-crawler
# This script helps set up the development environment

set -e

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${GREEN}Setting up sitemap-crawler development environment...${NC}"

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo -e "${RED}Error: Go is not installed. Please install Go 1.23 or later.${NC}"
    echo -e "${YELLOW}Visit: https://golang.org/doc/install${NC}"
    exit 1
fi

# Check Go version
GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
echo -e "${GREEN}Found Go version: ${GO_VERSION}${NC}"

# Install pre-commit if not present
if ! command -v pre-commit &> /dev/null; then
    echo -e "${YELLOW}Installing pre-commit...${NC}"
    if command -v pip3 &> /dev/null; then
        pip3 install pre-commit
    elif command -v pip &> /dev/null; then
        pip install pre-commit
    else
        echo -e "${YELLOW}Note: pip not found. Please install pre-commit manually.${NC}"
        echo -e "${YELLOW}Visit: https://pre-commit.com/#install${NC}"
    fi
else
    echo -e "${GREEN}pre-commit is already installed${NC}"
fi

# Download Go dependencies
echo -e "${YELLOW}Downloading Go dependencies...${NC}"
go mod download
go mod verify

echo -e "${YELLOW}Verifying module-pinned Go tools...${NC}"
go tool -modfile=tools/go.mod -n golangci-lint >/dev/null
go tool -modfile=tools/go.mod -n govulncheck >/dev/null
go tool -modfile=tools/go.mod -n gosec >/dev/null

# Install pre-commit hooks
if command -v pre-commit &> /dev/null; then
    echo -e "${YELLOW}Installing pre-commit hooks...${NC}"
    pre-commit install
else
    echo -e "${YELLOW}Skipping pre-commit hooks installation (pre-commit not found)${NC}"
fi

# Create .env file from example if it doesn't exist
if [ ! -f .env ]; then
    if [ -f .env.example ]; then
        echo -e "${YELLOW}Creating .env file from .env.example...${NC}"
        cp .env.example .env
        echo -e "${YELLOW}Please review and update .env with your configuration${NC}"
    fi
fi

# Run initial tests
echo -e "${YELLOW}Running initial tests...${NC}"
go test ./...

# Run initial linting
echo -e "${YELLOW}Running initial linting...${NC}"
go tool -modfile=tools/go.mod golangci-lint run

echo -e "${GREEN}Setup completed successfully!${NC}"
echo -e "${YELLOW}Next steps:${NC}"
echo -e "  1. Review and update .env file with your configuration"
echo -e "  2. Run 'make help' to see available commands"
echo -e "  3. Run 'make tools' to verify module-pinned Go tools"
echo -e "  4. Run 'make test' to run tests"
echo -e "  5. Run 'make build' to build the application"
echo -e "  6. Run 'make docker-build' to build Docker image"
echo -e "  7. Start developing your application!"
