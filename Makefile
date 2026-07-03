# Makefile for FLEX-GO
# Builds all tools with dynamic version information

VERSION ?= 1.0.0
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_ARCH := $(shell go env GOOS)/$(shell go env GOARCH)
GO_VERSION := $(shell go version | cut -d' ' -f3)

# Build flags for version information
LDFLAGS := -X 'github.com/sqpp/flex-golang.Version=$(VERSION)' \
           -X 'github.com/sqpp/flex-golang.BuildTime=$(BUILD_TIME)' \
           -X 'github.com/sqpp/flex-golang.GitCommit=$(GIT_COMMIT)' \
           -X 'github.com/sqpp/flex-golang.Author=marcell' \
           -X 'github.com/sqpp/flex-golang.ProjectURL=https://pagercast.com' \
           -X 'github.com/sqpp/flex-golang.BuildArch=$(BUILD_ARCH)' \
           -X 'github.com/sqpp/flex-golang.BuildGoVer=$(GO_VERSION)'

BINARY_NAME=flex-decode
ifeq ($(OS),Windows_NT)
    BINARY_NAME=flex-decode.exe
endif

# Default target
.PHONY: all
all: build

# Build all tools
.PHONY: build
build:
	@echo "Building FLEX-GO v$(VERSION)..."
	@mkdir -p bin
	go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY_NAME) ./cmd/flex-decode
	@echo "Build complete!"

# Install tools
.PHONY: install
install:
	go install -ldflags "$(LDFLAGS)" ./cmd/flex-decode

# Test
.PHONY: test
test:
	go test -v ./...

# Clean build artifacts
.PHONY: clean
clean:
	rm -rf bin/

# Show version information
.PHONY: version
version:
	@echo "Version: $(VERSION)"
	@echo "Build Time: $(BUILD_TIME)"
	@echo "Git Commit: $(GIT_COMMIT)"

# Cross-compile for multiple platforms
.PHONY: cross-compile
cross-compile:
	@echo "Cross-compiling for multiple platforms..."
	@mkdir -p bin
	GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o bin/flex-decode-linux-amd64 ./cmd/flex-decode
	GOOS=linux GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o bin/flex-decode-linux-arm64 ./cmd/flex-decode
	GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o bin/flex-decode-windows-amd64.exe ./cmd/flex-decode
	GOOS=darwin GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o bin/flex-decode-darwin-amd64 ./cmd/flex-decode
	GOOS=darwin GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o bin/flex-decode-darwin-arm64 ./cmd/flex-decode
	@echo "Cross-compilation complete!"

# Help
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build        - Build all tools"
	@echo "  install      - Install tools to GOPATH/bin"
	@echo "  test         - Run tests"
	@echo "  clean        - Remove build artifacts"
	@echo "  version      - Show version information"
	@echo "  cross-compile - Build for multiple platforms"
	@echo "  help         - Show this help"
