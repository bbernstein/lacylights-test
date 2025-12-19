# LacyLights Test Suite

**Comprehensive cross-repository testing for the LacyLights lighting control system.**

## Vision

This repository serves as the **central testing hub** for the entire LacyLights ecosystem. It goes beyond single-repository unit tests to validate the system as a whole through:

- **Contract Testing** - API contracts between components
- **Integration Testing** - Cross-repo interactions and data flow
- **End-to-End Testing** - Full user journeys from browser to DMX output
- **Stress Testing** - Performance under load (4 universes, 2048 DMX channels)
- **Deployment Testing** - Raspberry Pi and macOS native app validation
- **Distribution Testing** - Binary downloads, checksums, version consistency

## Scope

### What This Repo Tests

This test suite validates **cross-cutting concerns** that span multiple repositories:

1. **Backend Contracts** (`lacylights-go`)
   - GraphQL API responses
   - DMX output via Art-Net
   - Fade engine timing and behavior
   - WebSocket subscriptions
   - System settings persistence

2. **Frontend Integration** (`lacylights-fe`) - *Planned*
   - API integration correctness
   - Real-time subscription handling
   - State management across components
   - UI interaction flows

3. **MCP Server Integration** (`lacylights-mcp`) - *Planned*
   - AI tool execution correctness
   - API proxying and authorization
   - Error handling and recovery

4. **End-to-End Flows** - *Planned*
   - Browser → Frontend → Backend → DMX output
   - Playwright tests covering complete user journeys
   - Multi-device interaction scenarios

5. **Deployment Validation** - *Planned*
   - Raspberry Pi turnkey product behavior
   - macOS native app functionality
   - Update mechanisms and distribution

6. **Performance & Stress**
   - High channel count (2048 channels / 4 universes)
   - Rapid scene changes and fade interruptions
   - Concurrent user sessions
   - Memory and resource usage under load

### What This Repo Does NOT Test

- **Unit tests** - These belong in each individual repo
- **Component-level tests** - Test individual components in their own repos
- **Library-specific tests** - Test framework-specific code in its repo

## Architecture

### Current Structure

```
lacylights-test/
├── pkg/                    # Reusable test utilities
│   ├── artnet/            # Art-Net packet receiver for DMX capture
│   ├── graphql/           # GraphQL HTTP client
│   └── websocket/         # WebSocket subscription client
├── contracts/             # Contract tests (API behavior validation)
│   ├── api/              # GraphQL API contract tests
│   ├── crud/             # CRUD operation tests
│   ├── dmx/              # DMX output behavior tests
│   ├── fade/             # Fade curve, timing, FadeBehavior tests
│   ├── ofl/              # OFL import tests
│   ├── playback/         # Cue list playback tests
│   ├── preview/          # Preview mode tests
│   ├── settings/         # System settings tests
│   └── importexport/     # Import/export tests
├── integration/           # Cross-repo integration tests
│   └── distribution/     # S3 binary distribution tests
└── docs/                  # Testing plans and documentation
```

### Planned Structure

```
lacylights-test/
├── e2e/                   # End-to-end Playwright tests (PLANNED)
│   ├── fixtures/         # Test fixtures and scene setups
│   ├── journeys/         # Complete user journey tests
│   └── performance/      # Performance measurement tests
├── stress/                # Stress and load tests (PLANNED)
│   ├── channel-load/     # High channel count scenarios
│   ├── concurrent/       # Multi-user scenarios
│   └── memory/           # Memory leak detection
└── deployment/            # Deployment validation (PLANNED)
    ├── rpi/              # Raspberry Pi tests
    └── macos/            # macOS native app tests
```

## Manual Workflow Triggering

The test workflow can be manually triggered from GitHub Actions to test any combination of branches across repos:

1. Go to [Actions tab](https://github.com/bbernstein/lacylights-test/actions)
2. Select "Test" workflow
3. Click "Run workflow"
4. Select branches for:
   - `lacylights-test` - The test suite branch
   - `lacylights-go` - Backend server branch
   - `lacylights-fe` - Frontend branch (future use)
   - `lacylights-mcp` - MCP server branch (future use)
5. Click "Run workflow"

This enables **pre-merge integration testing** across repositories to catch integration issues before merging feature branches.

## Prerequisites

- Go 1.23+
- lacylights-go server running on port 4001 (for local testing)
- (Optional) lacylights-fe server on port 3000 (for future E2E tests)
- (Optional) Playwright for browser testing (future)

## Running Tests

```bash
# Run all tests
make test

# Run specific test categories
make test-contracts   # API contract tests
make test-dmx         # DMX behavior tests (requires Art-Net)
make test-fade        # Fade behavior tests (includes Art-Net capture)
make test-preview     # Preview mode tests
make test-settings    # Settings contract tests
make test-integration # Integration tests (includes fade rate tests)
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

### 8. Settings Tests (`contracts/settings/`)
Test system settings configuration:
- Fade update rate configuration (default 60Hz)
- Settings persistence and validation

### 9. Integration Tests (`integration/`)
- S3 distribution tests for binary downloads
- Checksum validation
- Platform availability verification

## Writing New Tests

### Test Types and When to Use Them

**Contract Tests** (`contracts/`)
- Test API behavior and data structures
- Validate GraphQL schema contracts
- Use when verifying API responses, mutations, subscriptions
- Example: Does `createScene` return the expected fields?

**Integration Tests** (`integration/`)
- Test cross-repo interactions
- Validate data flow between components
- Use when testing how multiple systems work together
- Example: Frontend → Backend → DMX output flow

**End-to-End Tests** (`e2e/`) - *Planned*
- Test complete user journeys in a browser
- Use Playwright to simulate real user interactions
- Use when validating full workflows from UI to DMX
- Example: User creates scene → saves → triggers → DMX outputs

**Stress Tests** (`stress/`) - *Planned*
- Test system performance under load
- Measure resource usage and timing
- Use when validating performance requirements
- Example: 2048 channels fading simultaneously

### Code Pattern Example

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

## Testing Philosophy

### Black Box Testing
Tests observe **external behavior** (API responses, DMX packets, UI state) rather than internal implementation details. This keeps tests resilient to refactoring.

### Contract-First
Define and test the contracts between components. If contracts are honored, components can evolve independently.

### Timing Tolerance
Real-world systems have timing variability. DMX tests allow small timing differences (±1 frame) while still validating correctness.

### Cross-Repository Validation
This repo exists because **integration issues happen at boundaries**. Unit tests in individual repos can't catch problems that only appear when systems interact.

## Repository Relationships

```
┌─────────────────┐
│  lacylights-fe  │  Frontend (NextJS/React)
│  (Port 3000)    │
└────────┬────────┘
         │ GraphQL + WebSocket
         ▼
┌─────────────────┐      ┌──────────────────┐
│ lacylights-mcp  │◄─────│  lacylights-test │  ◄── YOU ARE HERE
│  (MCP Server)   │      │  (Test Suite)    │
└────────┬────────┘      └──────────────────┘
         │                        │
         │ GraphQL                │ Tests all repos
         ▼                        ▼
┌─────────────────┐      ┌──────────────────┐
│  lacylights-go  │◄─────│  Tests contracts,│
│  (Port 4000)    │      │  integration,    │
└────────┬────────┘      │  E2E, stress     │
         │               └──────────────────┘
         │ Art-Net DMX
         ▼
    [DMX Hardware]
```

## Contributing

When adding tests to this repository:

1. **Consider scope** - Does this test cross-repo boundaries? If not, it might belong in the component's own repo.
2. **Check existing tests** - Look for similar patterns in `contracts/` or `integration/`
3. **Update documentation** - Add your test category to this README
4. **Follow conventions** - Match the style and structure of existing tests
5. **Test locally first** - Use `make start-go-server && make test` before pushing
6. **Use feature branches** - Never commit directly to main
7. **Reference the plan** - See `docs/TESTING_PLAN.md` for strategic direction

## Further Documentation

- **Testing Plan**: See `docs/TESTING_PLAN.md` for the strategic roadmap
- **Claude Context**: See `CLAUDE.md` for AI assistant guidance
- **Individual Test Docs**: See comments in test files for specific scenarios
