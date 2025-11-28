# CLAUDE.md

This file provides guidance to Claude Code when working with code in this repository.

## Project Overview

LacyLights Test Suite is a Go-based testing framework for validating the LacyLights lighting control system. It tests contract compatibility between Node and Go server implementations, DMX behavior via Art-Net capture, and WebSocket subscriptions.

## Development Commands

### Testing
- `make test` - Run all tests
- `make test-contracts` - Run API contract tests against Go server
- `make test-contracts-node` - Run API contract tests against Node server
- `make test-contracts-compare` - Run tests against both and compare
- `make test-dmx` - Run DMX behavior tests
- `make lint` - Run linters

### Building
- `make build` - Build test binaries
- `make clean` - Remove build artifacts

## Architecture

### Package Structure

- `pkg/artnet/` - Art-Net packet receiver for capturing DMX output
- `pkg/graphql/` - GraphQL HTTP client for API testing
- `pkg/websocket/` - WebSocket client for subscription testing

### Test Structure

- `contracts/api/` - GraphQL API contract tests
- `contracts/dmx/` - DMX output behavior tests
- `contracts/fade/` - Fade curve and timing tests
- `contracts/preview/` - Preview session tests
- `integration/` - Cross-component integration tests
- `e2e/` - End-to-end tests

## Testing Philosophy

1. **Black Box Testing** - Tests observe external behavior (API responses, Art-Net packets), not internal state
2. **Server Agnostic** - Same tests run against Node or Go server via environment variables
3. **Comparison Mode** - Tests can compare behavior between two servers
4. **Timing Tolerance** - DMX tests allow for small timing differences (±1 frame at 44Hz)

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

### Server Comparison
```go
nodeResp, goResp := runOnBothServers(query)
assert.Equal(t, nodeResp, goResp)
```

## Environment Variables

- `GO_SERVER_URL` - Go server endpoint (default: http://localhost:4001/graphql)
- `NODE_SERVER_URL` - Node server endpoint (default: http://localhost:4000/graphql)
- `GRAPHQL_ENDPOINT` - Single server endpoint for non-comparison tests
- `ARTNET_LISTEN_PORT` - Art-Net UDP port (default: 6454)

## Important Notes

- Tests assume servers are running and accessible
- DMX tests require Art-Net to be enabled on the server being tested
- Comparison tests require both servers to share database state or be in identical known states
- WebSocket tests connect to the server's subscription endpoint
