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

# Run fade behavior tests (includes Art-Net capture)
make test-fade

# Run migration tests (new!)
make test-migration          # All migration tests
make test-migration-quick    # Quick migration tests (excludes slow tests)
make test-migration-db       # Database compatibility tests
make test-migration-api      # API comparison tests
make test-migration-e2e      # End-to-end migration tests

# Run all tests
make test
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `GO_SERVER_URL` | `http://localhost:4001/graphql` | Go server GraphQL endpoint |
| `NODE_SERVER_URL` | `http://localhost:4000/graphql` | Node server GraphQL endpoint |
| `ARTNET_LISTEN_PORT` | `6455` | Port to listen for Art-Net packets (default 6455 for testing) |
| `ARTNET_BROADCAST` | `127.0.0.1` | Broadcast address for Art-Net (use localhost for testing) |
| `TEST_TIMEOUT` | `30s` | Default test timeout |

> **Note:** Tests use Art-Net port **6455** and localhost broadcast (`127.0.0.1`) by default to avoid conflicts with other Art-Net software running on the standard port 6454.

## Test Categories & Coverage

### 1. API Contract Tests (`contracts/api/`)
Verify that GraphQL queries and mutations return identical responses from both servers.
- **Coverage**: Basic CRUD for Scenes, Fixtures, CueLists.
- **Missing**: Complex nested queries, edge case validation.

### 2. DMX Behavior Tests (`contracts/dmx/`)
Capture actual Art-Net packets and verify DMX channel values match between servers.
- **Coverage**: Basic channel output, universe mapping.

### 3. Fade Tests (`contracts/fade/`)
Comprehensive testing of the fade engine.
- **Coverage**:
  - Linear and Sine easing curves
  - Fade interruption (new scene, blackout)
  - Cross-fading between scenes
  - Cue list timing and transitions
  - Preview mode isolation (ensure preview doesn't affect live output)
  - Art-Net frame capture verification
- **Status**: All tests passing. `TestCueFadeTimeOverride` requires server support for `fadeInTime` override (implemented).

### 4. Preview Tests (`contracts/preview/`)
Test preview session creation, channel overrides, commit, and cancel.
- **Coverage**: Session lifecycle, channel updates.
- **Missing**: Multi-user preview sessions.

## Migration Testing

Comprehensive migration tests have been added to validate the Go backend migration. See [MIGRATION_TESTING.md](MIGRATION_TESTING.md) for detailed documentation.

**New Test Categories**:

1. **Database Migration Tests** - Verify Go can read/write Node's SQLite database
2. **API Comparison Tests** - Ensure GraphQL APIs return identical responses
3. **Distribution Tests** - Validate S3 binary downloads and checksums
4. **End-to-End Tests** - Simulate complete migration workflows

**Quick Start**:
```bash
# Run quick migration tests (2-3 minutes)
make test-migration-quick

# Run full migration suite (5-15 minutes)
make test-migration
```

## Areas Needing Coverage

The following areas still need significant test coverage:

1. **Complex Scenarios**:
   - "Live Busking" (rapid changes)
   - Long-running stability tests
   - Network interruption recovery

2. **WebSocket Subscriptions**:
   - Real-time updates for faders/buttons
   - Connection stability
   - Event ordering

3. **System Integration**:
   - ✅ Database migration verification (added!)
   - Import/Export project fidelity

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
