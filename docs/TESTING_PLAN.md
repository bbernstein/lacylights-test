# LacyLights Test Suite - Strategic Testing Plan

**Status**: Living Document
**Last Updated**: 2025-12-19
**Purpose**: Provide strategic context and roadmap for comprehensive cross-repository testing

---

## Executive Summary

The lacylights-test repository serves as the **central testing hub** for the entire LacyLights ecosystem. While individual repositories contain unit tests for their own code, this repository validates:

- **Integration** - How components work together
- **Contracts** - API agreements between services
- **End-to-End** - Complete user journeys
- **Performance** - System behavior under load
- **Deployment** - Production readiness

This document provides the strategic roadmap for building out comprehensive test coverage.

---

## Current State (✅ Implemented)

### Contract Tests - lacylights-go Backend

**Status**: ✅ Mature and stable

The foundation of our testing is contract validation for the lacylights-go GraphQL API:

- **API Contracts** (`contracts/api/`) - GraphQL query/mutation responses
- **CRUD Operations** (`contracts/crud/`) - Create, Read, Update, Delete for all entities
- **DMX Behavior** (`contracts/dmx/`) - Art-Net packet capture and validation
- **Fade Engine** (`contracts/fade/`) - Comprehensive fade timing, curves, and FadeBehavior
- **Preview Mode** (`contracts/preview/`) - Session management and channel overrides
- **OFL Import** (`contracts/ofl/`) - Open Fixture Library integration
- **Playback** (`contracts/playback/`) - Cue list execution
- **Settings** (`contracts/settings/`) - System configuration persistence
- **Import/Export** (`contracts/importexport/`) - Data portability

### Integration Tests

**Status**: ✅ Partial implementation

- **Distribution** (`integration/distribution/`) - S3 binary downloads, checksums, version consistency
- **Fade Rate Integration** - Cross-system fade timing validation

### CI/CD Infrastructure

**Status**: ✅ Implemented (2025-12-19)

- Automatic tests on push/PR
- **Manual workflow dispatch** with branch selection for pre-merge integration testing
- Lint and test jobs in GitHub Actions
- Branch selection for: lacylights-test, lacylights-go, lacylights-fe, lacylights-mcp

---

## Planned Enhancements

### Phase 1: Frontend Integration Tests (HIGH PRIORITY)

**Goal**: Validate lacylights-fe ↔ lacylights-go integration

**Dependencies**:
- lacylights-fe repository access
- Understanding of frontend API client code
- GraphQL query/mutation contracts

**Scope**:
```
integration/frontend/
├── api_client_test.go        # Frontend API client behavior
├── subscription_test.go       # WebSocket subscription handling
├── state_management_test.go   # State sync between FE and BE
└── error_handling_test.go     # Error propagation and recovery
```

**Test Scenarios**:
1. **API Client Contract Tests**
   - Frontend makes GraphQL queries → Verify backend response format matches expectations
   - Mutations trigger proper backend state changes
   - Error responses are handled correctly

2. **Subscription Tests**
   - WebSocket subscriptions deliver real-time updates
   - Subscription reconnection on network issues
   - Multiple clients receive synchronized updates

3. **State Management**
   - Frontend state stays in sync with backend
   - Optimistic updates are rolled back on errors
   - Concurrent edits are handled properly

**Implementation Notes**:
- May require mocking or wrapping the frontend API client
- Could use GraphQL query extraction from frontend code
- Should validate against actual frontend queries, not hypothetical ones

**Why This Matters**:
Frontend integration issues are among the most common sources of bugs. Testing the actual queries and subscriptions the frontend uses against the backend prevents API contract drift.

---

### Phase 2: End-to-End Browser Tests (HIGH PRIORITY)

**Goal**: Validate complete user journeys from browser to DMX output

**Dependencies**:
- Playwright or similar E2E framework
- Both lacylights-fe and lacylights-go running
- Art-Net DMX capture capability

**Scope**:
```
e2e/
├── setup/
│   ├── browser.go           # Browser setup utilities
│   ├── servers.go           # Start FE + BE servers
│   └── fixtures.go          # Test data and scenes
├── journeys/
│   ├── scene_creation_test.go     # Create → Save → Trigger
│   ├── fixture_management_test.go # Add fixture → Configure → Use
│   ├── cue_playback_test.go       # Create cue list → Play → Verify DMX
│   └── preview_mode_test.go       # Preview changes → Commit/Cancel
└── performance/
    ├── page_load_test.go          # Initial load time
    └── interaction_latency_test.go # UI responsiveness
```

**Test Scenarios**:

1. **Scene Creation Journey**
   - User opens browser → Logs in (if auth)
   - Navigates to scenes
   - Creates new scene with fixtures
   - Sets channel values
   - Saves scene
   - Triggers scene
   - **Validation**: DMX output matches scene configuration

2. **Fixture Management Journey**
   - Import fixture from OFL
   - Add fixture to project
   - Configure fixture address
   - Use fixture in scene
   - Verify DMX output to correct channels

3. **Cue List Playback Journey**
   - Create multiple scenes
   - Build cue list with scenes
   - Set fade times
   - Start playback
   - **Validation**: DMX transitions match cue timings

4. **Preview Mode Journey**
   - Enter preview mode
   - Modify scene (change channels)
   - Verify preview DMX output differs from live
   - Commit changes → Live updates
   - OR Cancel → Live unchanged

**Technical Approach**:
- Use Playwright Go bindings
- Start lacylights-fe dev server (or production build)
- Start lacylights-go backend
- Capture Art-Net packets during test
- Assert on both UI state and DMX output

**Why This Matters**:
E2E tests catch issues that only appear when all systems work together. They validate the entire stack from user interaction to physical hardware control.

---

### Phase 3: MCP Server Integration Tests (MEDIUM PRIORITY)

**Goal**: Validate lacylights-mcp ↔ lacylights-go integration

**Dependencies**:
- lacylights-mcp repository access
- Understanding of MCP protocol
- AI tool execution flows

**Scope**:
```
integration/mcp/
├── tool_execution_test.go    # MCP tool calls work correctly
├── auth_test.go              # Authorization and security
├── error_handling_test.go    # Error propagation
└── concurrent_test.go        # Multiple AI sessions
```

**Test Scenarios**:
1. **Tool Execution**
   - MCP server receives tool call (e.g., "createScene")
   - Server proxies to lacylights-go backend
   - Response is formatted correctly for AI
   - Tool execution modifies backend state

2. **Authorization**
   - MCP server enforces permissions
   - Unauthorized calls are rejected
   - Session management works correctly

3. **Error Handling**
   - Backend errors are translated to MCP errors
   - Error messages are AI-friendly
   - Recovery from errors is graceful

**Why This Matters**:
The MCP server enables AI control of LacyLights. Testing ensures AI interactions work reliably and securely.

---

### Phase 4: Stress and Performance Tests (MEDIUM PRIORITY)

**Goal**: Validate system behavior under extreme load

**Dependencies**:
- Performance measurement tools
- Art-Net capture for high-channel scenarios
- Resource monitoring

**Scope**:
```
stress/
├── channel_load/
│   ├── high_channel_count_test.go   # 2048 channels (4 universes)
│   └── rapid_changes_test.go        # Fast scene transitions
├── concurrent/
│   ├── multi_user_test.go           # Multiple simultaneous users
│   └── subscription_storm_test.go   # Many WebSocket connections
└── memory/
    ├── leak_detection_test.go       # Long-running stability
    └── resource_usage_test.go       # CPU/Memory profiling
```

**Test Scenarios**:

1. **High Channel Count** (Already partially implemented)
   - 2048 channels across 4 universes
   - Simultaneous fade on all channels
   - Validate timing precision maintained
   - Verify no dropped frames

2. **Rapid Scene Changes**
   - Trigger new scene every 100ms
   - Fade interruption and override
   - Verify clean state transitions
   - No memory leaks or resource exhaustion

3. **Multi-User Scenarios**
   - 10+ concurrent users
   - Each editing different scenes
   - Verify no conflicts or race conditions
   - Subscriptions deliver to all users

4. **Long-Running Stability**
   - Run system for hours with activity
   - Monitor memory usage over time
   - Detect memory leaks or resource leaks
   - Verify stable performance

**Performance Targets**:
- **Fade Update Rate**: 60Hz (configurable) maintained even at 2048 channels
- **Scene Trigger Latency**: < 50ms from API call to first DMX packet
- **WebSocket Latency**: < 100ms for subscription updates
- **Memory**: Stable over 24+ hours of operation

**Why This Matters**:
Real-world usage can stress systems in ways unit tests don't cover. Professional lighting control requires reliable high-performance operation.

---

### Phase 5: Deployment Validation Tests (LOWER PRIORITY)

**Goal**: Validate Raspberry Pi and macOS deployments

**Dependencies**:
- lacylights-rpi repository access
- lacylights-mac repository access
- Deployment artifacts

**Scope**:
```
deployment/
├── rpi/
│   ├── boot_test.go              # System starts correctly
│   ├── network_config_test.go    # WiFi and networking
│   ├── update_mechanism_test.go  # OTA updates work
│   └── hardware_test.go          # DMX output hardware
└── macos/
    ├── install_test.go           # App installation
    ├── permissions_test.go       # macOS permissions
    └── update_test.go            # Auto-update mechanism
```

**Test Scenarios**:

1. **Raspberry Pi Boot Test**
   - System boots from SD card
   - Services start automatically
   - Web UI is accessible
   - DMX output is functional

2. **RPi Network Configuration**
   - WiFi AP mode for setup
   - Client mode for normal operation
   - mDNS/Bonjour for discovery
   - Static IP configuration

3. **OTA Updates (RPi)**
   - Check for updates
   - Download and install update
   - System reboots successfully
   - User data is preserved

4. **macOS Installation**
   - DMG mounts correctly
   - App copies to Applications
   - First run experience
   - Permissions requests (network, etc.)

**Why This Matters**:
Deployment-specific issues are difficult to catch without testing the actual deployment process and runtime environment.

---

## Test Data Strategy

### Fixture Library

**Current**: Tests use simple fixtures defined in test code

**Future**:
- Shared fixture library in `e2e/fixtures/`
- Realistic fixture definitions (moving heads, LED bars, etc.)
- OFL-imported fixtures for testing
- Reusable across all test types

### Scene Templates

Create reusable scene templates for common test scenarios:
- **Simple Scene**: Single fixture, static colors
- **Complex Scene**: Multiple fixtures, various channel types
- **Fade Scene**: Smooth transitions between states
- **Snap Scene**: Instant changes (gobos, macros)

### Project Templates

Standard test projects:
- **Minimal**: 1 fixture, 1 scene
- **Standard**: 10 fixtures, 5 scenes, 2 cue lists
- **Large**: 50 fixtures, 20 scenes, 10 cue lists
- **Stress**: 100+ fixtures, 4 universes

---

## Testing Principles

### 1. Fail Fast, Fail Clear

Tests should fail immediately when something is wrong, with clear error messages indicating:
- What was expected
- What actually happened
- Which component likely caused the issue

### 2. Isolation

Tests should be independent:
- Each test creates its own data
- Tests clean up after themselves
- No shared mutable state between tests
- Parallel execution should work

### 3. Realistic Scenarios

Tests should mirror real-world usage:
- Use realistic fixture definitions
- Test actual user workflows
- Include edge cases (interruptions, errors, etc.)
- Measure real performance characteristics

### 4. Cross-Repository Awareness

When writing tests, consider:
- Which repositories are involved?
- What are the contracts between them?
- How do errors propagate?
- What happens when one component is down?

### 5. Documentation Through Tests

Tests serve as executable documentation:
- Test names describe behavior
- Comments explain why, not what
- Examples show how APIs should be used
- Integration tests document system architecture

---

## Implementation Priorities

### Immediate (Q1 2025)
1. ✅ Manual workflow triggering with branch selection
2. Frontend integration tests (API client, subscriptions)
3. Basic E2E tests (scene creation, playback)

### Short Term (Q2 2025)
4. Expanded E2E coverage (all major workflows)
5. MCP server integration tests
6. Enhanced stress testing (concurrent users, long-running)

### Medium Term (Q3-Q4 2025)
7. Deployment validation (RPi, macOS)
8. Performance benchmarking and regression testing
9. Chaos engineering (network failures, service crashes)

---

## Success Metrics

### Coverage Metrics
- **Contract Tests**: All GraphQL operations covered ✅
- **Integration Tests**: All inter-repo interactions covered (Target: 80%)
- **E2E Tests**: All critical user journeys covered (Target: 90%)
- **Stress Tests**: All performance targets validated (Target: 100%)

### Quality Metrics
- **CI Pass Rate**: > 95% (tests should be reliable)
- **Issue Detection**: Catch integration issues before merging
- **Performance**: No degradation in latency or throughput
- **Stability**: No flaky tests (< 1% failure rate on reruns)

### Process Metrics
- **PR Testing**: All PRs tested against target branch before merge
- **Cross-Repo Testing**: Feature branches tested together before merge
- **Release Confidence**: All tests pass before production release

---

## Tools and Technologies

### Current Stack
- **Language**: Go 1.23+
- **Testing**: Go `testing` package, `testify` assertions
- **API Testing**: Custom GraphQL client
- **DMX Capture**: Custom Art-Net receiver
- **CI/CD**: GitHub Actions

### Planned Additions
- **Browser Testing**: Playwright (Go bindings)
- **Performance**: Go profiling tools, custom metrics
- **Monitoring**: Prometheus/Grafana for long-running tests
- **Reporting**: HTML test reports, coverage dashboards

---

## Context for AI Assistants (Claude)

When working on tests in this repository, Claude should:

### 1. Understand Multi-Repo Context
- Tests often require knowledge of multiple repos (fe, go, mcp)
- Use Task/Explore agents to read code from other repos
- Understand contracts and interactions between components

### 2. Access Required Repositories
When writing tests that involve:
- **lacylights-go**: Read backend code for API contracts
- **lacylights-fe**: Read frontend code for actual queries used
- **lacylights-mcp**: Read MCP server for tool definitions
- **Other repos**: Access as needed

### 3. Test Structure Conventions
- Contract tests in `contracts/`
- Integration tests in `integration/`
- E2E tests in `e2e/` (planned)
- Stress tests in `stress/` (planned)

### 4. Focus on Realistic Scenarios
- Don't just test happy paths
- Include error cases, edge cases, interruptions
- Test what users actually do, not theoretical scenarios

### 5. Performance Awareness
- DMX tests need timing validation
- Stress tests need resource monitoring
- E2E tests should measure latency

### 6. Documentation
- Update this plan when adding new test categories
- Update README when adding new test types
- Comment complex test scenarios

---

## Open Questions

### 1. Test Data Persistence
- Should we have a test database that persists across runs?
- Or always start fresh with each test?
- **Current**: Fresh DB per test run (CI) or shared local DB (dev)

### 2. Test Execution Time
- Some stress tests may run for minutes or hours
- How do we balance coverage vs CI speed?
- **Current**: Fast tests in CI, optional long-running tests

### 3. Deployment Test Infrastructure
- How do we test RPi without physical hardware?
- Emulation? QEMU? Real devices?
- **Open**: Needs investigation

### 4. Frontend E2E Test Strategy
- Test against dev server or production build?
- Mock backend or real backend?
- **Proposed**: Real backend, dev server for speed

---

## Conclusion

This testing repository is critical infrastructure for the LacyLights project. By validating integration, contracts, end-to-end flows, and performance, we ensure that the system works reliably in production.

The roadmap above provides a clear path forward. The immediate priorities are frontend integration and E2E tests, which will catch the most common sources of bugs.

**Remember**: The goal is not 100% coverage of every line of code. The goal is **confidence** that the system works as a whole, that components integrate correctly, and that users can accomplish their tasks reliably.

---

**Next Actions**:
1. Implement frontend integration tests (Phase 1)
2. Set up Playwright for E2E testing (Phase 2)
3. Expand stress test scenarios (Phase 4)
4. Update this document as we learn and evolve
