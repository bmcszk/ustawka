.PHONY: build run test test-unit test-e2e clean

# Binary name
BINARY_NAME=ustawka

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
	@go test -v -short ./...

# Run end-to-end tests only (excluding short tests)
test-e2e:
	@echo "Running end-to-end tests..."
	@go test -v -run "TestRealAPI" ./...

# Clean build files
clean:
	@echo "Cleaning..."
	@go clean
	@rm -f $(BINARY_NAME)

# Install dependencies
deps:
	@echo "Installing dependencies..."
	@go mod download

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
