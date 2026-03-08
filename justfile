# Justfile for YewSeal

set shell := ["bash", "-cu"]

# Variables
binary_name := "yews"
main_path   := "./cmd/main.go"
dev_dir     := "./test"
build_dir   := "./build"
version     := `git describe --tags --always --dirty 2>/dev/null || echo "dev"`
ldflags     := "-s -w -X main.Version=" + version

# Install to $GOPATH/bin (global)
install:
    @echo "Installing {{binary_name}} to GOPATH/bin..."
    go install -ldflags "{{ldflags}}" {{main_path}}
    @echo "Installed! Run 'yews --version' to verify."

# Default: build for production
default: build

# Build for production (Windows)
build:
    @echo "Building {{binary_name}}.exe for production..."
    @mkdir -p {{build_dir}}
    go build -ldflags "{{ldflags}}" -o {{build_dir}}/{{binary_name}}.exe {{main_path}}
    @echo "Build complete: {{build_dir}}/{{binary_name}}.exe"

# Build for development (output to test/)
dev:
    @echo "Building {{binary_name}}.exe for development..."
    @mkdir -p {{dev_dir}}
    go build -ldflags "{{ldflags}}" -o {{dev_dir}}/{{binary_name}}.exe {{main_path}}
    @echo "Dev build complete: {{dev_dir}}/{{binary_name}}.exe"

# Build and run the application
run: dev
    @echo "Running {{binary_name}}..."
    @{{dev_dir}}/{{binary_name}}.exe

# Run tests
test:
    @echo "Running tests..."
    go test -v ./...

# Clean build artifacts
clean:
    @echo "Cleaning build artifacts..."
    go clean
    rm -rf {{build_dir}}
    rm -f {{dev_dir}}/{{binary_name}}.exe
    @echo "Clean complete"

# Download dependencies
deps:
    @echo "Downloading dependencies..."
    go get -v ./...

# Tidy dependencies
tidy:
    @echo "Tidying dependencies..."
    go mod tidy

# ── Cross-compilation ──────────────────────────────────────────

# (private) Build for a specific platform
[private]
_build os arch ext="":
    @echo "Building for {{os}}/{{arch}}..."
    @mkdir -p {{build_dir}}
    GOOS={{os}} GOARCH={{arch}} go build -ldflags "{{ldflags}}" \
        -o {{build_dir}}/{{binary_name}}-{{os}}-{{arch}}{{ext}} {{main_path}}

# Build for all platforms
build-all: build-linux build-windows build-darwin
    @echo "All platforms built successfully!"

# Build for all Linux platforms
build-linux: build-linux-amd64 build-linux-arm64
    @echo "Linux builds complete"

# Build for all Windows platforms
build-windows: build-windows-amd64 build-windows-arm64
    @echo "Windows builds complete"

# Build for all macOS platforms
build-darwin: build-darwin-amd64 build-darwin-arm64
    @echo "Darwin builds complete"

build-linux-amd64:   (_build "linux"   "amd64")
build-linux-arm64:   (_build "linux"   "arm64")
build-windows-amd64: (_build "windows" "amd64" ".exe")
build-windows-arm64: (_build "windows" "arm64" ".exe")
build-darwin-amd64:  (_build "darwin"  "amd64")
build-darwin-arm64:  (_build "darwin"  "arm64")

# ── Release ────────────────────────────────────────────────────

# Build all platforms and create release archives
release: build-all
    @echo "Creating release archives..."
    @cd {{build_dir}} && \
        for f in {{binary_name}}-linux-* {{binary_name}}-darwin-*; do \
            [ -f "$f" ] && tar -czf "$f.tar.gz" "$f" && rm "$f"; \
        done; \
        for f in {{binary_name}}-windows-*.exe; do \
            [ -f "$f" ] && zip -q "${f%.exe}.zip" "$f" && rm "$f"; \
        done
    @echo "Release archives created in {{build_dir}}/"
