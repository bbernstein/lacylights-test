# LacyLights Test Suite

Contract and integration tests for the LacyLights lighting control system (lacylights-go).

## Overview

This test suite validates that:
1. **API Contracts** - GraphQL APIs return expected responses
2. **DMX Behavior** - Art-Net DMX output is correct
3. **Fade Behavior** - Fade curves, timing, and FadeBehavior work correctly
4. **Preview Mode** - Preview session behavior is consistent
5. **OFL Import** - Open Fixture Library import works correctly
6. **WebSocket Subscriptions** - Real-time updates work

## Architecture

```
lacylights-test/
├── pkg/
│   ├── artnet/      # Art-Net receiver for DMX packet capture
│   ├── graphql/     # GraphQL test client
│   └── websocket/   # WebSocket subscription client
├── contracts/
│   ├── api/         # API contract tests
│   ├── crud/        # CRUD operation tests
│   ├── dmx/         # DMX output behavior tests
│   ├── fade/        # Fade curve, timing, and FadeBehavior tests
│   ├── ofl/         # OFL import tests
│   ├── playback/    # Cue list playback tests
│   ├── preview/     # Preview mode tests
│   └── importexport/# Import/export tests
└── integration/     # S3 distribution tests
```

## Prerequisites

- Go 1.23+
- Go server running on port 4001 (lacylights-go)

## Running Tests

```bash
# Run all tests
make test

# Run specific test categories
make test-contracts   # API contract tests
make test-dmx         # DMX behavior tests (requires Art-Net)
make test-fade        # Fade behavior tests (includes Art-Net capture)
make test-preview     # Preview mode tests
make test-integration # Integration tests
make test-distribution # S3 binary distribution tests

# Run linters
make lint

# Server management
make start-go-server  # Start lacylights-go in background
make stop-go-server   # Stop the server
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `GRAPHQL_ENDPOINT` | `http://localhost:4001/graphql` | Server GraphQL endpoint |
| `GO_SERVER_URL` | `http://localhost:4001/graphql` | Alias for GRAPHQL_ENDPOINT |
| `ARTNET_LISTEN_PORT` | `6455` | Port to listen for Art-Net packets |
| `ARTNET_BROADCAST` | `127.0.0.1` | Broadcast address for Art-Net |

> **Note:** Tests use Art-Net port **6455** and localhost broadcast (`127.0.0.1`) by default to avoid conflicts with other Art-Net software running on the standard port 6454.

## Test Categories

### 1. API Contract Tests (`contracts/api/`)
Verify that GraphQL queries and mutations return expected responses.

### 2. CRUD Tests (`contracts/crud/`)
Test Create, Read, Update, Delete operations for all entities:
- Projects, Fixtures, Scenes, Cue Lists

### 3. DMX Behavior Tests (`contracts/dmx/`)
Capture actual Art-Net packets and verify DMX channel values.

### 4. Fade Tests (`contracts/fade/`)
Comprehensive testing of the fade engine:
- Linear and Bezier easing curves
- Fade interruption (new scene, blackout)
- Cross-fading between scenes
- FadeBehavior (FADE, SNAP, SNAP_END) for channels
- Art-Net frame capture verification

### 5. OFL Tests (`contracts/ofl/`)
Test Open Fixture Library import functionality:
- Import status queries
- Check for updates
- Trigger imports
- FadeBehavior auto-detection for imported fixtures

### 6. Preview Tests (`contracts/preview/`)
Test preview session creation, channel overrides, commit, and cancel.

### 7. Playback Tests (`contracts/playback/`)
Test cue list playback, navigation, and timing.

### 8. Integration Tests (`integration/`)
- S3 distribution tests for binary downloads
- Checksum validation
- Platform availability verification

## Writing New Tests

See existing tests in `contracts/` for examples. Key pattern:

```go
func TestSomething(t *testing.T) {
    client := graphql.NewClient("") // Uses GRAPHQL_ENDPOINT env var

    var resp struct {
        Project struct {
            ID   string `json:"id"`
            Name string `json:"name"`
        } `json:"project"`
    }

    err := client.Query(ctx, `
        query GetProject($id: ID!) {
            project(id: $id) {
                id
                name
            }
        }
    `, map[string]interface{}{"id": projectID}, &resp)

    require.NoError(t, err)
    assert.Equal(t, expectedName, resp.Project.Name)
}
```
