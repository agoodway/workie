.PHONY: build install clean test deps help version

# Variables
BINARY_NAME=workie
BUILD_DIR=build
MAIN_PACKAGE=.

# Version information
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE ?= $(shell date -u +"%Y-%m-%d %H:%M:%S UTC")

# Build flags
LDFLAGS=-ldflags "-X workie/cmd.Version=$(VERSION) -X workie/cmd.Commit=$(COMMIT) -X 'workie/cmd.BuildDate=$(BUILD_DATE)'"

# Default target
help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

deps: ## Download dependencies
	go mod download
	go mod tidy

build: deps ## Build the binary
	@mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PACKAGE)
	@echo "Binary built: $(BUILD_DIR)/$(BINARY_NAME) (version: $(VERSION))"

install: deps ## Install the binary to GOPATH/bin
	go install $(LDFLAGS) $(MAIN_PACKAGE)
	@echo "Binary installed to GOPATH/bin (version: $(VERSION))"

test: ## Run tests
	go test -v ./...

clean: ## Clean build artifacts
	rm -rf $(BUILD_DIR)
	go clean

# Cross-platform builds
build-all: deps ## Build for all platforms
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(MAIN_PACKAGE)
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(MAIN_PACKAGE)
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(MAIN_PACKAGE)
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(MAIN_PACKAGE)
	@echo "Cross-platform binaries built in $(BUILD_DIR)/ (version: $(VERSION))"

version: ## Show version information that would be built
	@echo "Version: $(VERSION)"
	@echo "Commit: $(COMMIT)"
	@echo "Build Date: $(BUILD_DATE)"

dev: build ## Build and run with example arguments
	./$(BUILD_DIR)/$(BINARY_NAME) --help

dev-version: build ## Build and show version
	./$(BUILD_DIR)/$(BINARY_NAME) --version
