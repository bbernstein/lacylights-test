# LacyLights Test Suite Makefile

GO := go
GOFLAGS := -v

# Server URLs
GO_SERVER_URL ?= http://localhost:4001/graphql

# Art-Net settings
ARTNET_LISTEN_PORT ?= 6455

.PHONY: all build clean test test-contracts test-contracts-go \
        test-dmx test-fade test-preview lint help deps \
        start-go-server stop-go-server wait-for-server test-load run-load-tests

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

## test-contracts: Run API contract tests against Go server
test-contracts: test-contracts-go

## test-contracts-go: Run API contract tests against Go server
test-contracts-go:
	@echo "Running contract tests against Go server..."
	GRAPHQL_ENDPOINT=$(GO_SERVER_URL) $(GO) test $(GOFLAGS) ./contracts/api/...

# =============================================================================
# DMX BEHAVIOR TESTS
# =============================================================================

## test-dmx: Run DMX behavior tests (requires Art-Net enabled)
test-dmx:
	@echo "Running DMX behavior tests..."
	GRAPHQL_ENDPOINT=$(GO_SERVER_URL) ARTNET_LISTEN_PORT=$(ARTNET_LISTEN_PORT) \
		$(GO) test $(GOFLAGS) ./contracts/dmx/...

# =============================================================================
# FADE TESTS
# =============================================================================

## test-fade: Run fade behavior tests
test-fade:
	@echo "Running fade behavior tests..."
	GRAPHQL_ENDPOINT=$(GO_SERVER_URL) ARTNET_LISTEN_PORT=$(ARTNET_LISTEN_PORT) ARTNET_BROADCAST=127.0.0.1 \
		$(GO) test $(GOFLAGS) ./contracts/fade/...

# =============================================================================
# PREVIEW TESTS
# =============================================================================

## test-preview: Run preview mode tests
test-preview:
	@echo "Running preview mode tests..."
	GRAPHQL_ENDPOINT=$(GO_SERVER_URL) $(GO) test $(GOFLAGS) ./contracts/preview/...

# =============================================================================
# INTEGRATION TESTS
# =============================================================================

## test-integration: Run integration tests
test-integration:
	@echo "Running integration tests..."
	GRAPHQL_ENDPOINT=$(GO_SERVER_URL) $(GO) test $(GOFLAGS) ./integration/...

## test-distribution: Run S3 distribution tests
test-distribution:
	@echo "Running S3 distribution tests..."
	$(GO) test $(GOFLAGS) -run "TestLatestJSON|TestBinaryDownload|TestChecksum|TestBinaryExecutable|TestVersionConsistency|TestAllPlatformsAvailable|TestDistributionCDN" ./integration/...

# =============================================================================
# ALL TESTS
# =============================================================================

## test: Run all tests against Go server
test:
	@echo "Running all tests..."
	GRAPHQL_ENDPOINT=$(GO_SERVER_URL) ARTNET_LISTEN_PORT=$(ARTNET_LISTEN_PORT) \
		$(GO) test $(GOFLAGS) ./...

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
	@echo "  ARTNET_LISTEN_PORT Art-Net UDP port (default: 6454)"
	@echo "  GO_SERVER_DIR      Path to lacylights-go repo (default: ../lacylights-go)"

# =============================================================================
# GO SERVER MANAGEMENT
# =============================================================================

GO_SERVER_DIR ?= ../lacylights-go
GO_SERVER_PORT ?= 4001
GO_SERVER_DB ?= file:./dev.db

## start-go-server: Start the Go server in background
start-go-server:
	@echo "Starting Go server on port $(GO_SERVER_PORT)..."
	@lsof -ti:$(GO_SERVER_PORT) | xargs kill -9 2>/dev/null || true
	@cd $(GO_SERVER_DIR) && \
		DATABASE_URL="$(GO_SERVER_DB)" PORT=$(GO_SERVER_PORT) ARTNET_BROADCAST=127.0.0.1 ARTNET_PORT=6455 go run ./cmd/server > /tmp/lacylights-go-server.log 2>&1 &
	@sleep 1
	@$(MAKE) wait-for-server
	@echo "Go server started. Logs at /tmp/lacylights-go-server.log"

## stop-go-server: Stop the Go server
stop-go-server:
	@echo "Stopping Go server on port $(GO_SERVER_PORT)..."
	@lsof -ti:$(GO_SERVER_PORT) | xargs kill -9 2>/dev/null || true
	@echo "Server stopped."

## wait-for-server: Wait for Go server to be ready (max 30 seconds)
wait-for-server:
	@echo "Waiting for server to be ready..."
	@for i in $$(seq 1 30); do \
		if curl -sf http://localhost:$(GO_SERVER_PORT)/graphql -X POST \
			-H "Content-Type: application/json" \
			-d '{"query":"{ __typename }"}' > /dev/null 2>&1; then \
			echo "Server ready!"; \
			exit 0; \
		fi; \
		sleep 1; \
	done; \
	echo "Server not ready after 30 seconds"; \
	exit 1

# =============================================================================
# LOAD TESTS
# =============================================================================

## test-load: Run 4-universe load tests (2048 channels)
test-load:
	@echo "Running 4-universe load tests..."
	GRAPHQL_ENDPOINT=$(GO_SERVER_URL) $(GO) test $(GOFLAGS) -timeout 180s \
		-run "TestFadeAllChannels4Universes|TestFadeUpAllChannels4Universes" ./contracts/fade/...

## run-load-tests: Start server, run load tests, then stop server
run-load-tests: start-go-server
	@echo ""
	@echo "Running load tests..."
	@GRAPHQL_ENDPOINT=$(GO_SERVER_URL) $(GO) test $(GOFLAGS) -timeout 180s \
		-run "TestFadeAllChannels4Universes|TestFadeUpAllChannels4Universes" ./contracts/fade/... || \
		($(MAKE) stop-go-server && exit 1)
	@$(MAKE) stop-go-server
	@echo ""
	@echo "Load tests completed successfully!"
