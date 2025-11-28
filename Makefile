# LacyLights Test Suite Makefile

GO := go
GOFLAGS := -v

# Server URLs
GO_SERVER_URL ?= http://localhost:4001/graphql
NODE_SERVER_URL ?= http://localhost:4000/graphql

# Art-Net settings
ARTNET_LISTEN_PORT ?= 6454

.PHONY: all build clean test test-contracts test-contracts-node test-contracts-go \
        test-contracts-compare test-dmx test-fade test-preview lint help deps

# =============================================================================
# DEFAULT TARGET
# =============================================================================

all: deps lint test

# =============================================================================
# DEPENDENCIES
# =============================================================================

## deps: Download and tidy dependencies
deps:
	@echo "Downloading dependencies..."
	$(GO) mod download
	$(GO) mod tidy

# =============================================================================
# BUILD
# =============================================================================

## build: Build test binaries
build:
	@echo "Building test binaries..."
	$(GO) build $(GOFLAGS) ./...

## clean: Remove build artifacts
clean:
	@echo "Cleaning..."
	$(GO) clean -testcache

# =============================================================================
# CONTRACT TESTS
# =============================================================================

## test-contracts: Run API contract tests against Go server (default)
test-contracts: test-contracts-go

## test-contracts-go: Run API contract tests against Go server
test-contracts-go:
	@echo "Running contract tests against Go server..."
	GRAPHQL_ENDPOINT=$(GO_SERVER_URL) $(GO) test $(GOFLAGS) ./contracts/api/...

## test-contracts-node: Run API contract tests against Node server
test-contracts-node:
	@echo "Running contract tests against Node server..."
	GRAPHQL_ENDPOINT=$(NODE_SERVER_URL) $(GO) test $(GOFLAGS) ./contracts/api/...

## test-contracts-compare: Run contract tests against both servers and compare
test-contracts-compare:
	@echo "Running comparison tests against both servers..."
	GO_SERVER_URL=$(GO_SERVER_URL) NODE_SERVER_URL=$(NODE_SERVER_URL) \
		$(GO) test $(GOFLAGS) ./contracts/api/... -tags=compare

# =============================================================================
# DMX BEHAVIOR TESTS
# =============================================================================

## test-dmx: Run DMX behavior tests (requires Art-Net enabled)
test-dmx:
	@echo "Running DMX behavior tests..."
	GRAPHQL_ENDPOINT=$(GO_SERVER_URL) ARTNET_LISTEN_PORT=$(ARTNET_LISTEN_PORT) \
		$(GO) test $(GOFLAGS) ./contracts/dmx/...

## test-dmx-compare: Run DMX tests comparing both servers
test-dmx-compare:
	@echo "Running DMX comparison tests..."
	GO_SERVER_URL=$(GO_SERVER_URL) NODE_SERVER_URL=$(NODE_SERVER_URL) \
		ARTNET_LISTEN_PORT=$(ARTNET_LISTEN_PORT) \
		$(GO) test $(GOFLAGS) ./contracts/dmx/... -tags=compare

# =============================================================================
# FADE TESTS
# =============================================================================

## test-fade: Run fade behavior tests
test-fade:
	@echo "Running fade behavior tests..."
	GRAPHQL_ENDPOINT=$(GO_SERVER_URL) ARTNET_LISTEN_PORT=$(ARTNET_LISTEN_PORT) \
		$(GO) test $(GOFLAGS) ./contracts/fade/...

# =============================================================================
# PREVIEW TESTS
# =============================================================================

## test-preview: Run preview mode tests
test-preview:
	@echo "Running preview mode tests..."
	GRAPHQL_ENDPOINT=$(GO_SERVER_URL) $(GO) test $(GOFLAGS) ./contracts/preview/...

# =============================================================================
# INTEGRATION & E2E TESTS
# =============================================================================

## test-integration: Run integration tests
test-integration:
	@echo "Running integration tests..."
	GRAPHQL_ENDPOINT=$(GO_SERVER_URL) $(GO) test $(GOFLAGS) ./integration/...

## test-e2e: Run end-to-end tests
test-e2e:
	@echo "Running e2e tests..."
	GRAPHQL_ENDPOINT=$(GO_SERVER_URL) $(GO) test $(GOFLAGS) ./e2e/...

# =============================================================================
# ALL TESTS
# =============================================================================

## test: Run all tests against Go server
test:
	@echo "Running all tests..."
	GRAPHQL_ENDPOINT=$(GO_SERVER_URL) ARTNET_LISTEN_PORT=$(ARTNET_LISTEN_PORT) \
		$(GO) test $(GOFLAGS) ./...

## test-all-compare: Run all comparison tests
test-all-compare: test-contracts-compare test-dmx-compare

# =============================================================================
# LINT
# =============================================================================

## lint: Run linters
lint:
	@echo "Running linters..."
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed. Running go vet..."; \
		$(GO) vet ./...; \
	fi

# =============================================================================
# HELP
# =============================================================================

## help: Show this help message
help:
	@echo "LacyLights Test Suite"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## /  /'
	@echo ""
	@echo "Environment Variables:"
	@echo "  GO_SERVER_URL      Go server endpoint (default: http://localhost:4001/graphql)"
	@echo "  NODE_SERVER_URL    Node server endpoint (default: http://localhost:4000/graphql)"
	@echo "  ARTNET_LISTEN_PORT Art-Net UDP port (default: 6454)"
