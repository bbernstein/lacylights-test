# CLAUDE.md

This file provides guidance to Claude Code when working with code in this repository.

## Project Overview

**lacylights-test** is the **central testing hub** for the entire LacyLights ecosystem. Unlike other repositories that contain unit tests for their own code, this repository validates:

- **Cross-repository integration** - How components work together
- **API contracts** - Agreements between services (GraphQL, WebSocket)
- **End-to-end flows** - Complete user journeys from UI to DMX output
- **Performance and stress** - System behavior under load (2048 channels)
- **Deployment validation** - Production readiness (RPi, macOS)

### Repository Scope

This is NOT a unit test repository. This is an **integration and system test** repository. The key difference:

- **Unit tests** → Test individual functions/components → Live in each repo
- **Integration tests** → Test how components interact → Live HERE
- **E2E tests** → Test complete user workflows → Live HERE
- **Stress tests** → Test performance under load → Live HERE

### Key Principle: Multi-Repository Context Required

**CRITICAL**: Most tests in this repository require understanding code from OTHER repositories:

- **lacylights-go** (backend) - For API contracts, GraphQL schema, DMX logic
- **lacylights-fe** (frontend) - For actual queries/mutations used by UI
- **lacylights-mcp** (MCP server) - For AI tool contracts
- **lacylights-rpi** (Raspberry Pi) - For deployment configuration
- **lacylights-mac** (macOS app) - For native app behavior

**When working on tests here, you MUST be prepared to read code from other repositories to understand what you're testing.**

## How to Work in This Repository

### 1. Always Read the Testing Plan First

**MANDATORY**: Before writing ANY new tests, read `docs/TESTING_PLAN.md` to understand:
- The overall testing strategy
- Where this test fits in the architecture
- What types of tests we need vs. what we have
- Planned vs. implemented features

### 2. Use the Explore Agent for Multi-Repo Context

When you need to understand code in other repositories (which is OFTEN):

```
Use the Task tool with subagent_type=Explore to investigate:
- lacylights-go for API schema, resolvers, business logic
- lacylights-fe for actual queries/mutations used by frontend
- lacylights-mcp for MCP tool definitions
```

**Example scenarios requiring multi-repo exploration**:
- "Write integration test for frontend scene creation" → Need to read lacylights-fe to see actual GraphQL queries
- "Test MCP tool execution" → Need to read lacylights-mcp to see tool definitions
- "Add E2E test for cue playback" → Need to read lacylights-go fade engine AND lacylights-fe UI code

### 3. Understand What You're Testing

Before writing a test:
1. **Read the actual code** being tested (in the other repo)
2. **Understand the contract** between components
3. **Identify edge cases** from the implementation
4. **Consider failure modes** (network errors, race conditions, etc.)

DON'T write tests based on assumptions. READ the code you're testing.

### 4. Repository Locations

The parent directory structure (from user's dev environment):
```
/Users/bernard/src/lacylights/
├── lacylights-test/        ← YOU ARE HERE
├── lacylights-go/          ← Backend (../lacylights-go)
├── lacylights-fe/          ← Frontend (../lacylights-fe)
├── lacylights-mcp/         ← MCP Server (../lacylights-mcp)
├── lacylights-rpi/         ← Raspberry Pi (../lacylights-rpi)
└── lacylights-mac/         ← macOS App (../lacylights-mac)
```

When you need to read code from another repo, use relative paths or Explore agent with the repo name.

### 5. Test Writing Philosophy

**Contract Tests** (`contracts/`):
- Validate API behavior matches expectations
- Test GraphQL schema contracts
- Verify DMX output correctness
- Black-box testing (external behavior, not internals)

**Integration Tests** (`integration/`):
- Test cross-repo data flow
- Validate component interactions
- Test error propagation
- System-level behavior

**E2E Tests** (`e2e/`) - Planned:
- Full user journeys in browser (Playwright)
- UI → Frontend → Backend → DMX
- Measure end-to-end latency
- Realistic user scenarios

**Stress Tests** (`stress/`) - Planned:
- Performance under load
- Resource usage monitoring
- Concurrent user scenarios
- Long-running stability

### 6. Common Patterns

**When adding a new contract test**:
1. Read the GraphQL schema in lacylights-go
2. Understand what the resolver does
3. Write test validating the contract
4. Include edge cases and error scenarios

**When adding an integration test**:
1. Identify the components involved
2. Read code in each repository
3. Understand the data flow
4. Test the integration, not individual components

**When adding an E2E test**:
1. Map out the complete user journey
2. Identify all components touched
3. Read UI code to understand interactions
4. Write browser test validating end-to-end flow

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

## Manual Workflow Triggering (GitHub Actions)

The test workflow supports **manual triggering** with branch selection. This enables pre-merge integration testing across repositories.

**How it works**:
1. Go to GitHub Actions → "Test" workflow
2. Click "Run workflow"
3. Select branches for each repo:
   - `lacylights-test` - Test suite branch
   - `lacylights-go` - Backend branch
   - `lacylights-fe` - Frontend branch (future)
   - `lacylights-mcp` - MCP server branch (future)
4. Tests run against the selected branch combination

**Use cases**:
- Testing feature branches before merging
- Validating cross-repo changes work together
- Catching integration issues early
- Testing experimental combinations

## Strategic Context for Claude

### Understanding Test Scope

**Question to ask yourself**: "Does this test cross repository boundaries?"

- **YES** → Belongs here (integration, E2E, contract between repos)
- **NO** → Belongs in the component's own repo (unit test)

**Example**:
- "Test GraphQL resolver logic" → NO (unit test in lacylights-go)
- "Test GraphQL API contract" → YES (contract test here)
- "Test React component rendering" → NO (unit test in lacylights-fe)
- "Test frontend→backend data flow" → YES (integration test here)

### Reading Other Repositories

**ALWAYS** use the Explore agent when you need to:
- Understand GraphQL schema (lacylights-go)
- See actual queries used by frontend (lacylights-fe)
- Understand MCP tool definitions (lacylights-mcp)
- Learn about deployment configuration (lacylights-rpi/mac)

**Don't guess**. Don't make assumptions. READ the actual code.

### Test Completeness Checklist

When writing a new test, ensure:
- ✅ Test name clearly describes what's being validated
- ✅ Test includes happy path AND error cases
- ✅ Test is isolated (creates own data, cleans up)
- ✅ Test validates behavior, not implementation
- ✅ Test has clear assertions with helpful error messages
- ✅ Test documents the contract being validated (comments)
- ✅ Test is added to appropriate Make target for running

### Common Pitfalls to Avoid

❌ **Don't** write tests without reading the code being tested
❌ **Don't** test implementation details (internal state, private functions)
❌ **Don't** make tests depend on each other (order matters = bad)
❌ **Don't** use hardcoded values without explaining why
❌ **Don't** skip error case testing (errors are important!)
❌ **Don't** write flaky tests (timing-dependent without tolerance)

✅ **Do** read actual code in other repos
✅ **Do** test contracts and observable behavior
✅ **Do** make tests independent and isolated
✅ **Do** use descriptive names and constants
✅ **Do** test error propagation and edge cases
✅ **Do** allow timing tolerance where appropriate

### When to Update Documentation

Update these docs when:
- Adding a new test category or type
- Changing test infrastructure
- Adding new Make targets
- Discovering important patterns or gotchas
- Implementing planned features from TESTING_PLAN.md

Files to update:
- `README.md` - User-facing documentation
- `docs/TESTING_PLAN.md` - Strategic roadmap and context
- `CLAUDE.md` - AI assistant guidance (this file)

## Quick Decision Tree

**User asks to write a test. What type?**

```
Is it testing interaction between repos?
├─ YES → Integration test (integration/)
└─ NO  → Check further
    │
    Is it testing API contract?
    ├─ YES → Contract test (contracts/)
    └─ NO  → Check further
        │
        Is it testing full user journey in browser?
        ├─ YES → E2E test (e2e/) - PLANNED
        └─ NO  → Check further
            │
            Is it testing performance/load?
            ├─ YES → Stress test (stress/) - PLANNED
            └─ NO  → Maybe belongs in component's own repo?
```

## Final Reminder

This repository is the **integration testing hub**. The goal is to validate that the LacyLights system works as a whole, that components integrate correctly, and that users can accomplish their tasks reliably.

Read `docs/TESTING_PLAN.md` for comprehensive strategic context before starting any significant work.
