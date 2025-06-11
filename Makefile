.PHONY: check build run test test-unit test-e2e lint clean install-lint install-gotestsum

# Binary name
BINARY_NAME=ustawka
GOTEST=gotestsum --junitfile unit-tests.xml --
GOLANGCI_LINT_CMD := golangci-lint

# Check: lint, and unit tests (no Docker)
check: lint test-unit
	@echo "Linters, and unit tests completed."

# Build the application
build:
	@echo "Building..."
	@go build -o $(BINARY_NAME) main.go

# Run the application
run:
	@echo "Running..."
	@go run main.go

# Run all tests
test: test-unit test-e2e

# Run unit tests only (marked with testing.Short())
test-unit:
	@echo "Running unit tests..."
	@$(GOTEST) -v -short ./...

# Run end-to-end tests only (excluding short tests)
test-e2e:
	@echo "Running end-to-end tests..."
	@$(GOTEST) -v -run "TestRealAPI" ./...

lint:
	@echo "Running linters..."
	@$(GOLANGCI_LINT_CMD) run --build-tags=e2e ./...

# Clean build files
clean:
	@echo "Cleaning..."
	@go clean
	@rm -f $(BINARY_NAME)

# Install dependencies
deps:
	@echo "Installing dependencies..."
	@go mod download

# Dependencies
install-lint: ## Install golangci-lint
	@echo "Installing golangci-lint..."
	@go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.1.6

install-gotestsum: ## Install gotestsum
	@echo "Installing gotestsum..."
	@go install gotest.tools/gotestsum@latest

# Help
help:
	@echo "Available targets:"
	@echo "  build      - Build the application"
	@echo "  run        - Run the application"
	@echo "  test       - Run all tests"
	@echo "  test-unit  - Run unit tests only (marked with testing.Short())"
	@echo "  test-e2e   - Run end-to-end tests only"
	@echo "  clean      - Clean build files"
	@echo "  deps       - Install dependencies"
	@echo "  help       - Show this help message" 
