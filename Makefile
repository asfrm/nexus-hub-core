# Nexus Hub Core - Makefile
# Comprehensive build, test, and deployment automation

# Binary name
BINARY_NAME := nexus-hub

# Version information
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GIT_COMMIT := $(shell git rev-parse HEAD)
BUILD_DATE := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

# Go parameters
GOCMD := go
GOBUILD := $(GOCMD) build
GOTEST := $(GOCMD) test
GOVET := $(GOCMD) vet
GOFMT := gofmt
GOMOD := $(GOCMD) mod
GO_VERSION := 1.26.2

# Build flags
LDFLAGS := -ldflags="-s -w -X main.Version=$(VERSION) -X main.GitCommit=$(GIT_COMMIT) -X main.BuildDate=$(BUILD_DATE)"
CGO_ENABLED := 0

# Build targets
TARGETS := \
	linux-amd64 \
	linux-arm64 \
	linux-armv7 \
	darwin-amd64 \
	darwin-arm64 \
	windows-amd64

# Build directory
BUILD_DIR := build

# Docker parameters
DOCKER_IMAGE := ghcr.io/$(shell git remote get-url origin 2>/dev/null | sed 's/.*\///' | sed 's/\.git//')
DOCKER_TAG ?= $(VERSION)

# Colors for output
GREEN := \033[0;32m
YELLOW := \033[0;33m
RED := \033[0;31m
NC := \033[0m

.PHONY: all build test clean release docker help

# Default target
all: clean deps lint test build

## Help: Show this help message
help: Makefile
	@echo "Nexus Hub Core - Build System"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Build Targets:"
	@echo "  build              Build for current OS/arch"
	@echo "  build-all          Build all platforms"
	@echo "  build-<platform>   Build for specific platform (linux-amd64, darwin-arm64, etc.)"
	@echo ""
	@echo "Test & Quality:"
	@echo "  test               Run tests"
	@echo "  test-coverage      Run tests with coverage"
	@echo "  lint               Run linters"
	@echo "  vet                Run go vet"
	@echo "  fmt                Format code"
	@echo "  fmt-check          Check formatting"
	@echo "  mod-tidy           Tidy go modules"
	@echo ""
	@echo "Docker:"
	@echo "  docker             Build Docker image"
	@echo "  docker-run         Run Docker container"
	@echo "  docker-push        Push to registry"
	@echo ""
	@echo "Release:"
	@echo "  release            Create release archive"
	@echo "  release-all        Create all release archives"
	@echo ""
	@echo "Utility:"
	@echo "  clean              Clean build artifacts"
	@echo "  deps               Download dependencies"
	@echo "  help               Show this help"
	@echo "  version            Show version info"

## Build: Build for current platform
build:
	@echo "$(GREEN)Building $(BINARY_NAME) $(VERSION)$(NC)"
	CGO_ENABLED=$(CGO_ENABLED) $(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) -v .
	@echo "$(GREEN)Build complete: $(BINARY_NAME)$(NC)"

## Build All: Build for all supported platforms
build-all: $(addprefix build-,$(TARGETS))

## Build Platform-Specific: Build for specific platform
build-linux-amd64:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=$(CGO_ENABLED) $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 -v .

build-linux-arm64:
	GOOS=linux GOARCH=arm64 CGO_ENABLED=$(CGO_ENABLED) $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 -v .

build-linux-armv7:
	GOOS=linux GOARCH=arm GOARM=7 CGO_ENABLED=$(CGO_ENABLED) $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-armv7 -v .

build-darwin-amd64:
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=$(CGO_ENABLED) $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 -v .

build-darwin-arm64:
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=$(CGO_ENABLED) $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 -v .

build-windows-amd64:
	GOOS=windows GOARCH=amd64 CGO_ENABLED=$(CGO_ENABLED) $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe -v .

## Test: Run all tests
test:
	@echo "$(GREEN)Running tests...$(NC)"
	$(GOTEST) -v -race -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -func=coverage.out
	@echo "$(GREEN)Tests complete$(NC)"

## Test Coverage: Run tests with HTML coverage report
test-coverage: test
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)Coverage report generated: coverage.html$(NC)"

## Lint: Run golangci-lint
lint:
	@echo "$(GREEN)Running linters...$(NC)"
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run --timeout=5m; \
	elif [ -f $(shell go env GOPATH)/bin/golangci-lint ]; then \
		$(shell go env GOPATH)/bin/golangci-lint run --timeout=5m; \
	else \
		echo "$(YELLOW)golangci-lint not installed, installing...$(NC)"; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
		$(shell go env GOPATH)/bin/golangci-lint run --timeout=5m; \
	fi
	@echo "$(GREEN)Linting complete$(NC)"

## Vet: Run go vet
vet:
	@echo "$(GREEN)Running go vet...$(NC)"
	$(GOVET) ./...
	@echo "$(GREEN)Vet complete$(NC)"

## Format: Format all Go code
fmt:
	@echo "$(GREEN)Formatting code...$(NC)"
	$(GOFMT) -s -w .
	@echo "$(GREEN)Formatting complete$(NC)"

## Format Check: Check if code is properly formatted
fmt-check:
	@echo "$(GREEN)Checking code formatting...$(NC)"
	@unformatted=$$($(GOFMT) -l .); \
	if [ -n "$$unformatted" ]; then \
		echo "$(RED)Unformatted files found:$(NC)"; \
		echo "$$unformatted"; \
		exit 1; \
	fi
	@echo "$(GREEN)All files properly formatted$(NC)"

## Mod Tidy: Clean up go.mod and go.sum
mod-tidy:
	@echo "$(GREEN)Tidying modules...$(NC)"
	$(GOMOD) tidy
	@echo "$(GREEN)Modules tidy$(NC)"

## Dependencies: Download all dependencies
deps:
	@echo "$(GREEN)Downloading dependencies...$(NC)"
	$(GOMOD) download
	@echo "$(GREEN)Dependencies downloaded$(NC)"

## Clean: Remove build artifacts
clean:
	@echo "$(YELLOW)Cleaning build artifacts...$(NC)"
	rm -rf $(BUILD_DIR)
	rm -f $(BINARY_NAME)
	rm -f *.out
	rm -f *.html
	rm -rf dist/
	rm -rf uploads/
	@echo "$(YELLOW)Clean complete$(NC)"

## Docker: Build Docker image
docker:
	@echo "$(GREEN)Building Docker image $(DOCKER_IMAGE):$(DOCKER_TAG)$(NC)"
	docker build \
		--build-arg VERSION=$(VERSION) \
		--build-arg GIT_COMMIT=$(GIT_COMMIT) \
		--build-arg BUILD_DATE=$(BUILD_DATE) \
		-t $(DOCKER_IMAGE):$(DOCKER_TAG) \
		-t $(DOCKER_IMAGE):latest \
		.
	@echo "$(GREEN)Docker build complete$(NC)"

## Docker Run: Run Docker container
docker-run: docker
	@echo "$(GREEN)Running Docker container...$(NC)"
	docker run --rm -p 8080:8080 $(DOCKER_IMAGE):$(DOCKER_TAG)

## Docker Push: Push Docker image to registry
docker-push:
	@echo "$(GREEN)Pushing Docker image...$(NC)"
	docker push $(DOCKER_IMAGE):$(DOCKER_TAG)
	docker push $(DOCKER_IMAGE):latest
	@echo "$(GREEN)Docker push complete$(NC)"

## Release: Create release archive for current platform
release: build
	@echo "$(GREEN)Creating release archive...$(NC)"
	@mkdir -p $(BUILD_DIR)
	@if [ "$$(uname -s)" = "Darwin" ]; then \
		tar -czf $(BUILD_DIR)/$(BINARY_NAME)-$(VERSION)-$$(uname -s | tr 'A-Z' 'a-z')-$$(uname -m).tar.gz $(BINARY_NAME); \
	else \
		tar -czf $(BUILD_DIR)/$(BINARY_NAME)-$(VERSION)-$$(uname -s | tr 'A-Z' 'a-z')-$$(uname -m).tar.gz $(BINARY_NAME); \
	fi
	@echo "$(GREEN)Release archive created: $(BUILD_DIR)/$(BINARY_NAME)-$(VERSION)-$$(uname -s | tr 'A-Z' 'a-z')-$$(uname -m).tar.gz$(NC)"

## Release All: Create release archives for all platforms
release-all: clean deps build-all
	@echo "$(GREEN)Creating release archives...$(NC)"
	@mkdir -p $(BUILD_DIR)
	@for target in $(TARGETS); do \
		case $$target in \
			windows-*) \
				ext=".exe" ;; \
			*) \
				ext="" ;; \
		esac; \
		binary=$(BINARY_NAME)-$$target$$ext; \
		if echo $$target | grep -q windows; then \
			cd $(BUILD_DIR) && zip $$binary.zip $$binary && cd ..; \
		else \
			tar -czf $(BUILD_DIR)/$$binary.tar.gz -C $(BUILD_DIR) $$binary; \
		fi; \
	done
	@echo "$(GREEN)All release archives created$(NC)"
	@ls -lh $(BUILD_DIR)/

## Version: Show version information
version:
	@echo "Nexus Hub Core"
	@echo "Version:    $(VERSION)"
	@echo "Git Commit: $(GIT_COMMIT)"
	@echo "Build Date: $(BUILD_DATE)"
	@echo "Go Version: $(GO_VERSION)"
	@echo "CGO Enabled: $(CGO_ENABLED)"

## Install: Install binary to GOPATH
install:
	@echo "$(GREEN)Installing $(BINARY_NAME)...$(NC)"
	$(GOCMD) install $(LDFLAGS) .
	@echo "$(GREEN)Installation complete$(NC)"

## Run: Build and run
run: build
	@echo "$(GREEN)Running $(BINARY_NAME)...$(NC)"
	./$(BINARY_NAME) -debug

## Security: Run security scanners
security:
	@echo "$(GREEN)Running security scans...$(NC)"
	@if command -v gosec >/dev/null 2>&1; then \
		gosec -exclude-generated ./...; \
	else \
		echo "$(YELLOW)Installing gosec...$(NC)"; \
		go install github.com/securego/gosec/v2/cmd/gosec@latest; \
		gosec -exclude-generated ./...; \
	fi
	@echo "$(GREEN)Security scan complete$(NC)"

## CI: Run all CI checks locally
ci: clean deps fmt-check vet lint test build
	@echo "$(GREEN)✅ All CI checks passed$(NC)"
