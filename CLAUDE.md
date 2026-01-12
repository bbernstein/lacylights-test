# CLAUDE.md

This file provides guidance to Claude Code when working with code in this repository.

## Project Overview

LacyLights Test is the **central integration testing hub** for the entire LacyLights ecosystem. Unlike other repositories that contain unit tests for their own code, this repository validates cross-repository integration, API contracts, and end-to-end behavior.

**Role in LacyLights Ecosystem:**
- Validates how components work together across repositories
- Tests API contracts between services (GraphQL, WebSocket)
- Ensures DMX output correctness
- Tests performance under load

**This is NOT a unit test repository.** Unit tests live in each component's own repo.

## Development Commands

### Testing
```bash
make test                # Run all tests
make test-contracts      # Run API contract tests
make test-dmx            # Run DMX behavior tests
make test-fade           # Run fade behavior tests
make test-preview        # Run preview mode tests
make test-settings       # Run settings contract tests
make test-integration    # Run integration tests
make test-distribution   # Run S3 distribution tests
```

### Building
```bash
make build               # Build test binaries
make clean               # Remove build artifacts
```

### Server Management
```bash
make start-go-server     # Start lacylights-go server in background
make stop-go-server      # Stop the server
make wait-for-server     # Wait for server to be ready
```

### Linting
```bash
make lint                # Run Go linters
```

## Architecture

### Directory Structure

```
lacylights-test/
├── contracts/           # API contract tests
│   ├── api/            # GraphQL API contracts
│   ├── crud/           # CRUD operation tests
│   ├── dmx/            # DMX output behavior tests
│   ├── fade/           # Fade curve and timing tests
│   ├── importexport/   # Import/export contract tests
│   ├── ofl/            # Open Fixture Library import tests
│   ├── playback/       # Cue list playback tests
│   ├── preview/        # Preview session tests
│   └── settings/       # System settings tests
├── integration/         # Cross-repo integration tests (future)
├── e2e/                # End-to-end tests (future)
├── stress/             # Performance tests (future)
├── pkg/                # Shared test utilities
│   ├── artnet/         # Art-Net packet capture
│   ├── graphql/        # GraphQL HTTP client
│   └── websocket/      # WebSocket client
└── docs/
    └── TESTING_PLAN.md # Strategic testing roadmap
```

### Key Technologies

- **Go**: Test implementation language
- **GraphQL Client**: HTTP client for API testing
- **Art-Net Receiver**: UDP packet capture for DMX validation
- **WebSocket Client**: Subscription testing

## Important Patterns

### Multi-Repository Context Required

**CRITICAL**: Most tests require understanding code from OTHER repositories:

| When testing | Read code from |
|--------------|----------------|
| GraphQL API contracts | lacylights-go (schema, resolvers) |
| Frontend integration | lacylights-fe (queries, mutations) |
| MCP tool execution | lacylights-mcp (tool definitions) |
| DMX output correctness | lacylights-go (fade engine) |

**Use the Explore agent** to read code from other repos before writing tests.

### Test Scope Decision

**Question:** "Does this test cross repository boundaries?"
- **YES** → Belongs here (integration, contract, E2E)
- **NO** → Belongs in the component's own repo (unit test)

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

### Test Isolation
- Each test creates its own data
- Tests clean up after themselves
- No dependencies between tests
- Use descriptive test names

## Testing Guidelines

### Contract Tests (`contracts/`)
- Validate API behavior matches expectations
- Test GraphQL schema contracts
- Verify DMX output correctness
- Black-box testing (external behavior, not internals)

### Integration Tests (`integration/`)
- Test cross-repo data flow
- Validate component interactions
- Test error propagation
- System-level behavior

### Writing New Tests
1. **Read the actual code** being tested (in other repos)
2. **Understand the contract** between components
3. **Identify edge cases** from implementation
4. **Consider failure modes** (network errors, race conditions)
5. **Allow timing tolerance** where appropriate (DMX: ±1 frame at 44Hz)

## CI/CD

| Workflow | File | Purpose |
|----------|------|---------|
| Test | `test.yml` | Run all tests on PRs and main |
| Manual | `test.yml` (workflow_dispatch) | Run with specific repo branches |

### Manual Workflow Triggering

Tests can be run against specific branches from multiple repos:
1. Go to GitHub Actions → Test workflow
2. Click "Run workflow"
3. Select branches for each repo
4. Tests validate the branch combination works together

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `GRAPHQL_ENDPOINT` | `http://localhost:4001/graphql` | Backend URL |
| `GO_SERVER_URL` | (alias for above) | Alternative name |
| `ARTNET_LISTEN_PORT` | `6454` | Art-Net UDP port |

## Related Repositories

| Repository | What to Read |
|------------|--------------|
| [lacylights-go](https://github.com/bbernstein/lacylights-go) | GraphQL schema, resolvers, fade engine |
| [lacylights-fe](https://github.com/bbernstein/lacylights-fe) | Actual queries/mutations used by UI |
| [lacylights-mcp](https://github.com/bbernstein/lacylights-mcp) | MCP tool definitions |
| [lacylights-terraform](https://github.com/bbernstein/lacylights-terraform) | Distribution infrastructure configuration |
| [lacylights-rpi](https://github.com/bbernstein/lacylights-rpi) | Raspberry Pi production platform |
| [lacylights-mac](https://github.com/bbernstein/lacylights-mac) | macOS production platform |

## Important Notes

- Tests assume lacylights-go server is running and accessible
- DMX tests require Art-Net enabled on the server
- WebSocket tests connect to the server's subscription endpoint
- **Always read `docs/TESTING_PLAN.md`** before adding new test categories

## Common Pitfalls

**Avoid:**
- Writing tests without reading the code being tested
- Testing implementation details (internal state, private functions)
- Making tests depend on each other
- Hardcoded values without explanation
- Skipping error case testing
- Flaky tests (timing-dependent without tolerance)

**Do:**
- Read actual code in other repos
- Test contracts and observable behavior
- Make tests independent and isolated
- Use descriptive names and constants
- Test error propagation and edge cases
- Allow timing tolerance where appropriate

## Test Completeness Checklist

When writing a new test:
- [ ] Test name clearly describes what's being validated
- [ ] Includes happy path AND error cases
- [ ] Creates own data, cleans up after
- [ ] Validates behavior, not implementation
- [ ] Clear assertions with helpful error messages
- [ ] Comments document the contract being validated
- [ ] Added to appropriate Make target
