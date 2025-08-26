# Makefile for ai-code-template-go
# This makefile provides targets that mirror the CI pipeline and help with development

.PHONY: help test lint security vulnerability-check build clean setup deps verify mod-tidy-check all ci-local clean-template

# Default Go version for local development
GO_VERSION := 1.23
BINARY_NAME := ai-code-template-go
BUILD_DIR := ./bin

# Colors for output
GREEN := \033[32m
YELLOW := \033[33m
RED := \033[31m
NC := \033[0m # No Color

## help: Display this help message
help:
	@echo "Available targets:"
	@echo "  $(GREEN)Development targets:$(NC)"
	@echo "    setup              - Install required tools and dependencies"
	@echo "    deps               - Download and verify Go dependencies"
	@echo "    clean              - Remove build artifacts"
	@echo "    clean-template     - Clean up template code to prepare for new project"
	@echo ""
	@echo "  $(GREEN)Testing targets (mirror CI):$(NC)"
	@echo "    test               - Run all tests with race detection and coverage"
	@echo "    lint               - Run golangci-lint"
	@echo "    security           - Run Gosec security scanner"
	@echo "    vulnerability-check- Run govulncheck for vulnerability scanning"
	@echo "    build              - Build binaries for multiple platforms"
	@echo "    mod-tidy-check     - Check if go mod tidy is needed"
	@echo ""
	@echo "  $(GREEN)Docker targets:$(NC)"
	@echo "    docker-build       - Build Docker image"
	@echo "    docker-run         - Run Docker container"
	@echo "    docker-compose-up  - Start services with docker-compose"
	@echo "    docker-compose-down- Stop services with docker-compose"
	@echo ""
	@echo "  $(GREEN)Code generation targets:$(NC)"
	@echo "    generate           - Generate code (if using go generate)"
	@echo "    benchmark          - Run benchmarks"
	@echo "    profile            - Run tests with profiling"
	@echo ""
	@echo "  $(GREEN)Convenience targets:$(NC)"
	@echo "    all                - Run all quality checks (test, lint, security, vuln-check)"
	@echo "    ci-local           - Run the same checks as CI pipeline"

## setup: Install required development tools
setup:
	@echo "$(YELLOW)Installing development tools...$(NC)"
	@command -v golangci-lint >/dev/null 2>&1 || { \
		echo "Installing golangci-lint..."; \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin latest; \
	}
	@command -v govulncheck >/dev/null 2>&1 || { \
		echo "Installing govulncheck..."; \
		go install golang.org/x/vuln/cmd/govulncheck@latest; \
	}
	@command -v gosec >/dev/null 2>&1 || { \
		echo "Installing gosec..."; \
		go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest; \
	}
	@echo "$(GREEN)Development tools installed successfully!$(NC)"

## deps: Download and verify dependencies
deps:
	@echo "$(YELLOW)Downloading dependencies...$(NC)"
	go mod download
	@echo "$(YELLOW)Verifying dependencies...$(NC)"
	go mod verify
	@echo "$(GREEN)Dependencies ready!$(NC)"

## test: Run tests with race detection and coverage
test:
	@echo "$(YELLOW)Running tests...$(NC)"
	go test -v -race -coverprofile=coverage.out ./...
	@echo "$(GREEN)Tests completed!$(NC)"
	@echo "$(YELLOW)Coverage report:$(NC)"
	go tool cover -func=coverage.out

## lint: Run golangci-lint
lint:
	@echo "$(YELLOW)Running linter...$(NC)"
	golangci-lint run --timeout=10m
	@echo "$(GREEN)Linting completed!$(NC)"

## security: Run Gosec security scanner
security:
	@echo "$(YELLOW)Running security scan...$(NC)"
	gosec -no-fail -fmt text ./...
	@echo "$(GREEN)Security scan completed!$(NC)"

## vulnerability-check: Run govulncheck
vulnerability-check:
	@echo "$(YELLOW)Checking for vulnerabilities...$(NC)"
	govulncheck ./...
	@echo "$(GREEN)Vulnerability check completed!$(NC)"

## build: Build binaries for multiple platforms
build:
	@echo "$(YELLOW)Building binaries...$(NC)"
	mkdir -p $(BUILD_DIR)

	@echo "Building for Linux AMD64..."
	GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./

	@echo "Building for Linux ARM64..."
	GOOS=linux GOARCH=arm64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 ./

	@echo "Building for macOS AMD64..."
	GOOS=darwin GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 ./

	@echo "Building for macOS ARM64..."
	GOOS=darwin GOARCH=arm64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 ./

	@echo "Building for Windows AMD64..."
	GOOS=windows GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe ./

	@echo "$(GREEN)All builds completed!$(NC)"
	@echo "$(YELLOW)Built binaries:$(NC)"
	@ls -la $(BUILD_DIR)/

## mod-tidy-check: Check if go mod tidy is needed
mod-tidy-check:
	@echo "$(YELLOW)Checking if go mod tidy is needed...$(NC)"
	@go mod tidy
	@git diff --exit-code go.mod go.sum || { \
		echo "$(RED)Error: go.mod or go.sum is not tidy. Please run 'go mod tidy' and commit the changes.$(NC)"; \
		exit 1; \
	}
	@echo "$(GREEN)go.mod and go.sum are tidy!$(NC)"

## verify: Verify the module and dependencies
verify:
	@echo "$(YELLOW)Verifying module...$(NC)"
	go mod verify
	@echo "$(GREEN)Module verification completed!$(NC)"

## clean: Remove build artifacts and coverage files
clean:
	@echo "$(YELLOW)Cleaning build artifacts...$(NC)"
	rm -rf $(BUILD_DIR)
	rm -f coverage.out
	rm -f results.sarif
	@echo "$(GREEN)Clean completed!$(NC)"

## clean-template: Clean up template code and prepare for new project development
clean-template:
	@echo "$(YELLOW)Cleaning up template code...$(NC)"
	@echo "$(RED)WARNING: This will modify your repository to remove template-specific code.$(NC)"
	@echo "$(YELLOW)This action will:$(NC)"
	@echo "  - Update README.md to remove template-specific content"
	@echo "  - Replace main.go with a minimal starter"
	@echo "  - Update go.mod module path"
	@echo "  - Remove AGENTS.md"
	@echo "  - Remove this target from Makefile"
	@echo ""
	@read -p "Enter your new module path (e.g., github.com/username/project-name): " module_path && \
	read -p "Enter your project name: " project_name && \
	echo "$(YELLOW)Updating module path to $$module_path...$(NC)" && \
	go mod edit -module $$module_path && \
	echo "$(YELLOW)Creating minimal main.go...$(NC)" && \
	cat > main.go << 'EOF' && \
package main\
\
import (\
	"fmt"\
	"log"\
)\
\
func main() {\
	fmt.Println("Hello from $$project_name!")\
	log.Println("Application started successfully")\
}\
EOF\
	echo "$(YELLOW)Creating minimal main_test.go...$(NC)" && \
	cat > main_test.go << 'EOF' && \
package main\
\
import "testing"\
\
func TestMain(t *testing.T) {\
	// Add your tests here\
	t.Log("Test suite ready")\
}\
EOF\
	echo "$(YELLOW)Updating README.md...$(NC)" && \
	cat > README.md << 'EOF' && \
# $$project_name\
\
A Go application built with AI assistance.\
\
## Getting Started\
\
```bash\
# Install dependencies\
go mod tidy\
\
# Run tests\
make test\
\
# Build the application\
make build\
\
# Run the application\
go run main.go\
```\
\
## Development\
\
This project includes a comprehensive development setup:\
\
- CI/CD with GitHub Actions\
- Code quality checks with golangci-lint\
- Security scanning with gosec and govulncheck\
- Cross-platform builds with GoReleaser\
- Automated dependency management\
\
Use `make help` to see all available commands.\
\
## Contributing\
\
Please read [CONTRIBUTING.md](CONTRIBUTING.md) for details on our code of conduct and the process for submitting pull requests.\
EOF\
	echo "$(YELLOW)Removing template-specific files...$(NC)" && \
	rm -f AGENTS.md && \
	echo "$(YELLOW)Updating Makefile...$(NC)" && \
	sed -i '/## clean-template:/,/^$$/d' Makefile && \
	sed -i 's/ai-code-template-go/'"$$project_name"'/g' Makefile && \
	echo "$(YELLOW)Running go mod tidy...$(NC)" && \
	go mod tidy && \
	echo "$(GREEN)Template cleanup completed!$(NC)" && \
	echo "$(YELLOW)Next steps:$(NC)" && \
	echo "  1. Review and commit the changes" && \
	echo "  2. Update .goreleaser.yml with your project details" && \
	echo "  3. Update CONTRIBUTING.md and other documentation" && \
	echo "  4. Start building your application!"

## all: Run all quality checks
all: deps test lint security vulnerability-check mod-tidy-check
	@echo "$(GREEN)All quality checks passed!$(NC)"

## docker-build: Build Docker image
docker-build:
	@echo "$(YELLOW)Building Docker image...$(NC)"
	docker build -t $(BINARY_NAME):latest .
	@echo "$(GREEN)Docker image built successfully!$(NC)"

## docker-run: Run Docker container
docker-run:
	@echo "$(YELLOW)Running Docker container...$(NC)"
	docker run -p 8080:8080 $(BINARY_NAME):latest

## docker-compose-up: Start services with docker-compose
docker-compose-up:
	@echo "$(YELLOW)Starting services with docker-compose...$(NC)"
	docker-compose up -d
	@echo "$(GREEN)Services started!$(NC)"

## docker-compose-down: Stop services with docker-compose
docker-compose-down:
	@echo "$(YELLOW)Stopping services with docker-compose...$(NC)"
	docker-compose down
	@echo "$(GREEN)Services stopped!$(NC)"

## generate: Generate code (if using go generate)
generate:
	@echo "$(YELLOW)Generating code...$(NC)"
	go generate ./...
	@echo "$(GREEN)Code generation completed!$(NC)"

## benchmark: Run benchmarks
benchmark:
	@echo "$(YELLOW)Running benchmarks...$(NC)"
	go test -bench=. -benchmem ./...
	@echo "$(GREEN)Benchmarks completed!$(NC)"

## profile: Run tests with profiling
profile:
	@echo "$(YELLOW)Running tests with profiling...$(NC)"
	go test -cpuprofile=cpu.prof -memprofile=mem.prof ./...
	@echo "$(GREEN)Profiling completed!$(NC)"

## ci-local: Run the same checks as CI pipeline
ci-local: all build
	@echo "$(GREEN)Local CI pipeline completed successfully!$(NC)"

# Default target
.DEFAULT_GOAL := help
