# Makefile for YewSeal

# Variables
BINARY_NAME=yews.exe
MAIN_PATH=./cmd/main.go
DEV_OUTPUT_DIR=./test
BUILD_DIR=./build

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Build flags
LDFLAGS=-ldflags "-s -w"

.PHONY: all build dev clean test deps tidy run help

# Default target
all: build

# Build for production (output to build directory)
build:
	@echo "Building $(BINARY_NAME) for production..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

# Build for development (output to test directory)
dev:
	@echo "Building $(BINARY_NAME) for development..."
	@mkdir -p $(DEV_OUTPUT_DIR)
	$(GOBUILD) -o $(DEV_OUTPUT_DIR)/$(BINARY_NAME) $(MAIN_PATH)
	@echo "Dev build complete: $(DEV_OUTPUT_DIR)/$(BINARY_NAME)"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	$(GOCLEAN)
	@rm -f $(BUILD_DIR)/$(BINARY_NAME)
	@rm -f $(DEV_OUTPUT_DIR)/$(BINARY_NAME)
	@echo "Clean complete"

# Run tests
test:
	@echo "Running tests..."
	$(GOTEST) -v ./...

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GOGET) -v ./...

# Tidy dependencies
tidy:
	@echo "Tidying dependencies..."
	$(GOMOD) tidy

# Run the application (from dev build)
run: dev
	@echo "Running $(BINARY_NAME)..."
	@$(DEV_OUTPUT_DIR)/$(BINARY_NAME)

# Display help information
help:
	@echo YewSeal Makefile Commands:
	@echo   make build    - Build for production (output to build/)
	@echo   make dev      - Build for development (output to test/)
	@echo   make clean    - Clean build artifacts
	@echo   make test     - Run tests
	@echo   make deps     - Download dependencies
	@echo   make tidy     - Tidy dependencies
	@echo   make run      - Build and run the application
	@echo   make help     - Display this help message
