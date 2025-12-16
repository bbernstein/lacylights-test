# CLAUDE.md

This file provides guidance to Claude Code when working with code in this repository.

## Project Overview

LacyLights Test Suite is a Go-based testing framework for validating the LacyLights lighting control system (lacylights-go). It tests GraphQL API contracts, DMX behavior via Art-Net capture, and WebSocket subscriptions.

## Development Commands

### Testing
- `make test` - Run all tests
- `make test-contracts` - Run API contract tests against Go server
- `make test-dmx` - Run DMX behavior tests
- `make test-fade` - Run fade behavior tests
- `make test-preview` - Run preview mode tests
- `make test-settings` - Run settings contract tests
- `make test-integration` - Run integration tests (includes fade rate tests)
- `make test-distribution` - Run S3 distribution tests
- `make lint` - Run linters

### Building
- `make build` - Build test binaries
- `make clean` - Remove build artifacts

### Server Management
- `make start-go-server` - Start lacylights-go server in background
- `make stop-go-server` - Stop the server
- `make wait-for-server` - Wait for server to be ready

## Architecture

### Package Structure

- `pkg/artnet/` - Art-Net packet receiver for capturing DMX output
- `pkg/graphql/` - GraphQL HTTP client for API testing
- `pkg/websocket/` - WebSocket client for subscription testing

### Test Structure

- `contracts/api/` - GraphQL API contract tests
- `contracts/crud/` - CRUD operation tests for all entities
- `contracts/dmx/` - DMX output behavior tests
- `contracts/fade/` - Fade curve, timing, and FadeBehavior tests
- `contracts/ofl/` - OFL (Open Fixture Library) import tests
- `contracts/playback/` - Cue list playback tests
- `contracts/preview/` - Preview session tests
- `contracts/settings/` - System settings tests (fade update rate, etc.)
- `contracts/importexport/` - Import/export functionality tests
- `integration/` - S3 distribution tests and fade rate integration tests

## Testing Philosophy

1. **Black Box Testing** - Tests observe external behavior (API responses, Art-Net packets), not internal state
2. **Contract Testing** - Tests validate GraphQL API contracts against lacylights-go
3. **Timing Tolerance** - DMX tests allow for small timing differences (±1 frame at 44Hz)

## Key Patterns

### GraphQL Client Usage
```go
client := graphql.NewClient("http://localhost:4001/graphql")
var resp struct {
    Project struct {
        ID   string
        Name string
    }
}
err := client.Query(ctx, `query { project(id: "...") { id name } }`, nil, &resp)
```

### Art-Net Capture
```go
receiver := artnet.NewReceiver(":6454")
frames := receiver.CaptureFrames(ctx, 5*time.Second)
// frames contains all DMX packets received
```

## Environment Variables

- `GRAPHQL_ENDPOINT` - Server endpoint (default: http://localhost:4001/graphql)
- `GO_SERVER_URL` - Alias for GRAPHQL_ENDPOINT
- `ARTNET_LISTEN_PORT` - Art-Net UDP port (default: 6454)

## Important Notes

- Tests assume lacylights-go server is running and accessible
- DMX tests require Art-Net to be enabled on the server being tested
- WebSocket tests connect to the server's subscription endpoint
- lacylights-node is deprecated; all tests target lacylights-go only
