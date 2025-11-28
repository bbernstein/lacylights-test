# LacyLights Test Suite

Integration, contract, and end-to-end tests for the LacyLights lighting control system.

## Overview

This test suite validates that:
1. **API Contracts** - GraphQL APIs return identical responses from Node and Go servers
2. **DMX Behavior** - Art-Net DMX output is identical between implementations
3. **Fade Behavior** - Fade curves and timing match between servers
4. **Preview Mode** - Preview session behavior is consistent
5. **WebSocket Subscriptions** - Real-time updates are equivalent

## Architecture

```
lacylights-test/
├── pkg/
│   ├── artnet/      # Art-Net receiver for DMX packet capture
│   ├── graphql/     # GraphQL test client
│   └── websocket/   # WebSocket subscription client
├── contracts/
│   ├── api/         # Static API contract tests
│   ├── dmx/         # DMX output behavior tests
│   ├── fade/        # Fade curve and timing tests
│   └── preview/     # Preview mode tests
├── integration/     # Cross-component integration tests
└── e2e/             # Full end-to-end tests
```

## Prerequisites

- Go 1.23+
- Node server running on port 4000 (lacylights-node)
- Go server running on port 4001 (lacylights-go)
- Both servers should use the same database or be in a known state

## Running Tests

```bash
# Run all contract tests against Go server (default)
make test-contracts

# Run contract tests against Node server
make test-contracts-node

# Run tests against both servers and compare
make test-contracts-compare

# Run DMX behavior tests (requires Art-Net)
make test-dmx

# Run all tests
make test
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `GO_SERVER_URL` | `http://localhost:4001/graphql` | Go server GraphQL endpoint |
| `NODE_SERVER_URL` | `http://localhost:4000/graphql` | Node server GraphQL endpoint |
| `ARTNET_LISTEN_PORT` | `6454` | Port to listen for Art-Net packets |
| `TEST_TIMEOUT` | `30s` | Default test timeout |

## Test Categories

### API Contract Tests (`contracts/api/`)
Verify that GraphQL queries and mutations return identical responses from both servers.

### DMX Behavior Tests (`contracts/dmx/`)
Capture actual Art-Net packets and verify DMX channel values match between servers.

### Fade Tests (`contracts/fade/`)
Test fade curves, timing, and interruption handling.

### Preview Tests (`contracts/preview/`)
Test preview session creation, channel overrides, commit, and cancel.

## Writing New Tests

See existing tests in `contracts/` for examples. Key patterns:

```go
// Test against a single server
func TestSomething(t *testing.T) {
    client := graphql.NewClient(os.Getenv("GRAPHQL_ENDPOINT"))
    // ... test code
}

// Compare behavior between servers
func TestSomethingComparison(t *testing.T) {
    nodeClient := graphql.NewClient(os.Getenv("NODE_SERVER_URL"))
    goClient := graphql.NewClient(os.Getenv("GO_SERVER_URL"))
    // ... compare responses
}
```
