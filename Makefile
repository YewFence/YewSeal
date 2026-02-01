# Makefile for YewSeal

# Variables
BINARY_NAME=yews
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
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS=-ldflags "-s -w -X main.Version=$(VERSION)"

# Platforms for cross-compilation

PLATFORMS := linux/amd64 linux/arm64 windows/amd64 windows/arm64 darwin/amd64 darwin/arm64

.PHONY: all build dev clean test deps tidy run help \
	build-all build-linux build-windows build-darwin \
	build-linux-amd64 build-linux-arm64 \
	build-windows-amd64 build-windows-arm64 \
	build-darwin-amd64 build-darwin-arm64 \
	release

# Default target
all: build

# Build for production (Windows, output to build directory)
build:
	@echo "Building $(BINARY_NAME).exe for production..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME).exe $(MAIN_PATH)
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME).exe"

# Build for development (output to test directory)
dev:
	@echo "Building $(BINARY_NAME).exe for development..."
	@mkdir -p $(DEV_OUTPUT_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(DEV_OUTPUT_DIR)/$(BINARY_NAME).exe $(MAIN_PATH)
	@echo "Dev build complete: $(DEV_OUTPUT_DIR)/$(BINARY_NAME).exe"


# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	$(GOCLEAN)
	@rm -rf $(BUILD_DIR)
	@rm -f $(DEV_OUTPUT_DIR)/$(BINARY_NAME).exe
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
	@$(DEV_OUTPUT_DIR)/$(BINARY_NAME).exe

# ============================================================
# Cross-compilation targets
# ============================================================

# Build all platforms
build-all: build-linux build-windows build-darwin
	@echo "All platforms built successfully!"

# Build all Linux platforms
build-linux: build-linux-amd64 build-linux-arm64
	@echo "Linux builds complete"

# Build all Windows platforms
build-windows: build-windows-amd64 build-windows-arm64
	@echo "Windows builds complete"

# Build all macOS platforms
build-darwin: build-darwin-amd64 build-darwin-arm64
	@echo "Darwin builds complete"

# Individual platform targets
build-linux-amd64:
	@echo "Building for Linux amd64..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(MAIN_PATH)

build-linux-arm64:
	@echo "Building for Linux arm64..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 $(MAIN_PATH)

build-windows-amd64:
	@echo "Building for Windows amd64..."
	@mkdir -p $(BUILD_DIR)
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(MAIN_PATH)

build-windows-arm64:
	@echo "Building for Windows arm64..."
	@mkdir -p $(BUILD_DIR)
	GOOS=windows GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-arm64.exe $(MAIN_PATH)

build-darwin-amd64:
	@echo "Building for macOS amd64..."
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(MAIN_PATH)

build-darwin-arm64:
	@echo "Building for macOS arm64..."
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(MAIN_PATH)

# ============================================================
# Release target (build all and create archives)
# ============================================================

release: build-all
	@echo "Creating release archives..."
	@cd $(BUILD_DIR) && \
		for f in $(BINARY_NAME)-linux-* $(BINARY_NAME)-darwin-*; do \
			[ -f "$$f" ] && tar -czf "$$f.tar.gz" "$$f" && rm "$$f"; \
		done; \
		for f in $(BINARY_NAME)-windows-*.exe; do \
			[ -f "$$f" ] && zip -q "$${f%.exe}.zip" "$$f" && rm "$$f"; \
		done
	@echo "Release archives created in $(BUILD_DIR)/"

# Display help information
help:
	@echo "YewSeal Makefile Commands:"
	@echo ""
	@echo "  Development:"
	@echo "    make build      - Build for production (Windows)"
	@echo "    make dev        - Build for development (output to test/)"
	@echo "    make run        - Build and run the application"
	@echo "    make test       - Run tests"
	@echo "    make clean      - Clean build artifacts"
	@echo "    make deps       - Download dependencies"
	@echo "    make tidy       - Tidy dependencies"
	@echo ""
	@echo "  Cross-compilation:"
	@echo "    make build-all       - Build for all platforms"
	@echo "    make build-linux     - Build for all Linux platforms"
	@echo "    make build-windows   - Build for all Windows platforms"
	@echo "    make build-darwin    - Build for all macOS platforms"
	@echo "    make build-{os}-{arch} - Build for specific platform"
	@echo "                           (e.g., make build-linux-amd64)"
	@echo ""
	@echo "  Release:"
	@echo "    make release    - Build all platforms and create archives"
	@echo ""
	@echo "  Supported platforms:"
	@echo "    linux/amd64, linux/arm64"
	@echo "    windows/amd64, windows/arm64"
	@echo "    darwin/amd64, darwin/arm64"
