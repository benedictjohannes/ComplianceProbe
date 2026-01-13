# Binary name
BINARY_NAME=compliance-probe

# Build flags for production (smaller binary)
LDFLAGS=-s -w
BUILD_FLAGS=-trimpath -ldflags="$(LDFLAGS)"

.PHONY: help schema build build-builder test clean

## help: Show this help message
help:
	@echo "ComplianceProbe Task Runner"
	@echo "Usage: make [target]"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-25s\033[0m %s\n", $$1, $$2}'

## schema: Generate the JSON schema for requirements
schema: ## Generate requirements.schema.json
	go run --tags builder . --schema > requirements.schema.json

## build: Build production binaries for all platforms
build: build-linux build-windows build-mac-intel build-mac-arm ## Build production binaries for all platforms

build-linux: ## Build production for Linux (amd64)
	GOOS=linux GOARCH=amd64 go build $(BUILD_FLAGS) -o $(BINARY_NAME)-linux .

build-windows: ## Build production for Windows (amd64)
	GOOS=windows GOARCH=amd64 go build $(BUILD_FLAGS) -o $(BINARY_NAME)-windows.exe .

build-mac-intel: ## Build production for Mac Intel (amd64)
	GOOS=darwin GOARCH=amd64 go build $(BUILD_FLAGS) -o $(BINARY_NAME)-mac-intel .

build-mac-arm: ## Build production for Mac Arm (arm64)
	GOOS=darwin GOARCH=arm64 go build $(BUILD_FLAGS) -o $(BINARY_NAME)-mac-arm .

## build-builder: Build builder binaries (with esbuild tags) for all platforms
build-builder: build-builder-linux build-builder-windows build-builder-mac-intel build-builder-mac-arm ## Build builder binaries for all platforms

build-builder-linux: ## Build builder for Linux (amd64)
	GOOS=linux GOARCH=amd64 go build --tags builder -o $(BINARY_NAME)-builder-linux .

build-builder-windows: ## Build builder for Windows (amd64)
	GOOS=windows GOARCH=amd64 go build --tags builder -o $(BINARY_NAME)-builder-windows.exe .

build-builder-mac-intel: ## Build builder for Mac Intel (amd64)
	GOOS=darwin GOARCH=amd64 go build --tags builder -o $(BINARY_NAME)-builder-mac-intel .

build-builder-mac-arm: ## Build builder for Mac Arm (arm64)
	GOOS=darwin GOARCH=arm64 go build --tags builder -o $(BINARY_NAME)-builder-mac-arm .

## testing: Run the test suite with builder tags
test: ## Run go tests
	go test --tags builder -v .

## clean: Remove all generated binaries
clean: ## Remove generated binaries
	rm -f $(BINARY_NAME)-*
