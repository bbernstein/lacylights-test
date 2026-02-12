# LacyLights CI and Testing Strategy

## Executive Summary

**Current State**: 129 tests across 4 repositories, but NO branch protection configured.
**Immediate Action Required**: Enable branch protection on all repos to enforce CI checks.

---

## Test Execution Methods

### Local Development

#### 1. Run All Tests (Recommended for Pre-Commit)
```bash
# From lacylights-test directory
./scripts/run-tests.sh all
```
**What it does**:
- Prompts to start/restart backend on port 4001
- Runs all contract tests including DMX/fade tests
- Requires Art-Net enabled backend

#### 2. Run CI-Safe Tests (Fast Feedback)
```bash
./scripts/run-tests.sh ci
```
**What it does**:
- Skips Art-Net dependent tests (fade/DMX)
- Runs: API, CRUD, Preview, Playback, Settings, Undo, OFL, ImportExport
- ~30 seconds execution time

#### 3. Run Specific Test Suites
```bash
./scripts/run-tests.sh contracts    # API contract tests only
./scripts/run-tests.sh preview      # Preview mode tests
./scripts/run-tests.sh settings     # Settings tests
./scripts/run-tests.sh undo         # Undo/redo tests
./scripts/run-tests.sh e2e          # Playwright E2E tests
```

#### 4. Direct Make Commands
```bash
# Contract tests
make test-contracts    # API contracts
make test-dmx          # DMX behavior (requires Art-Net)
make test-fade         # Fade behavior (requires Art-Net)
make test-effects      # Effects modulator
make test-preview      # Preview sessions
make test-settings     # System settings
make test-undo         # Undo/redo

# Integration tests
make test-integration  # All integration tests
make test-distribution # S3 distribution validation

# E2E tests
make e2e              # Run Playwright tests
make e2e-ui           # Playwright UI mode
make e2e-headed       # Headed browser mode

# All tests
make test             # All contract tests (requires Art-Net)
make test-ci          # CI-safe tests (no Art-Net)
make test-all         # Everything including integration
```

### CI/CD Pipeline

#### GitHub Actions Workflows

**lacylights-test** (.github/workflows/test.yml):
```yaml
Jobs:
  test:         # Contract tests (CI-safe)
  lint:         # Go linting
  e2e:          # Playwright E2E tests
```

**lacylights-go** (.github/workflows/ci.yml):
```yaml
Jobs:
  test:         # Unit tests + coverage checks
  lint:         # golangci-lint
  build:        # Multi-platform builds
```

**lacylights-fe** (.github/workflows/ci.yml):
```yaml
Jobs:
  lint-and-type-check:  # ESLint + TypeScript
  test:                 # Jest + coverage
  security-audit:       # npm audit
  build:                # Next.js build
```

**lacylights-mcp** (.github/workflows/ci.yml):
```yaml
Jobs:
  lint-and-build:  # ESLint + TypeScript + tests
  security-audit:  # npm audit
  mcp-validation:  # MCP server initialization
  docker-build:    # Docker image
```

---

## Current Test Coverage

### By Repository

| Repository | Test Count | Type | Coverage |
|------------|------------|------|----------|
| **lacylights-test** | 20 contract<br>2 integration<br>3 E2E | Contract + E2E | Cross-repo validation |
| **lacylights-go** | 57 unit tests | Unit | 21-100% per package |
| **lacylights-fe** | 46 unit tests | Unit | Moderate |
| **lacylights-mcp** | 1 contract test | Contract | Minimal |
| **TOTAL** | **129 tests** | All layers | Variable |

### By Test Layer

```
┌─────────────────────────────────────────────────────────┐
│ E2E Tests (3)                                           │
│ ├─ Happy path workflow                                  │
│ ├─ Dashboard functionality                              │
│ └─ Fixture position undo/redo                          │
└─────────────────────────────────────────────────────────┘
                           │
┌─────────────────────────────────────────────────────────┐
│ Integration Tests (2)                                   │
│ ├─ Fade rate persistence                                │
│ └─ S3 distribution (7 scenarios)                        │
└─────────────────────────────────────────────────────────┘
                           │
┌─────────────────────────────────────────────────────────┐
│ Contract Tests (20)                                     │
│ ├─ API: GraphQL introspection, WiFi                     │
│ ├─ CRUD: Looks, Cues, Fixtures, Projects (7 files)     │
│ ├─ DMX: Art-Net packet validation                       │
│ ├─ Fade: Timing, curves, load (4-universe)             │
│ ├─ Effects: FX Engine modulator (16 tests)             │
│ ├─ Preview: Session management                          │
│ ├─ Playback: Cue list execution                         │
│ ├─ Settings: Configuration persistence                  │
│ ├─ Import/Export: Project operations                    │
│ ├─ OFL: Fixture library imports                         │
│ └─ Undo: Complex undo/redo scenarios                    │
└─────────────────────────────────────────────────────────┘
                           │
┌─────────────────────────────────────────────────────────┐
│ Unit Tests (104)                                        │
│ ├─ lacylights-go: 57 tests (fade, preview, DMX, etc.)  │
│ ├─ lacylights-fe: 46 tests (components, hooks)         │
│ └─ lacylights-mcp: 1 test (GraphQL client)             │
└─────────────────────────────────────────────────────────┘
```

---

## Coverage Gaps

### CRITICAL (High Priority)

1. **❌ Frontend ↔ Backend Contract Testing**
   - **Gap**: Frontend tests don't validate actual GraphQL queries
   - **Risk**: API contract drift between FE and BE
   - **Recommendation**: Add contract tests in lacylights-test that execute frontend's actual query files

2. **❌ MCP Integration Testing**
   - **Gap**: Only 1 contract test for MCP server
   - **Risk**: Tool execution failures in production
   - **Recommendation**: Add contract tests for all MCP tools against live backend

3. **❌ E2E Coverage**
   - **Gap**: Only 3 workflows tested (dashboard, happy path, fixture undo)
   - **Missing**: Fixture management, cue lists, preview mode, import/export
   - **Recommendation**: Expand to 15+ critical user journeys

### MODERATE (Medium Priority)

4. **⚠️ Cross-Repository Workflows**
   - **Gap**: No tests for FE + BE + MCP working together
   - **Risk**: Integration bugs only found in production
   - **Recommendation**: Add smoke tests that exercise full stack

5. **⚠️ Platform-Specific Code**
   - **Gap**: WiFi (39% coverage), network interfaces (64% coverage)
   - **Risk**: Raspberry Pi and macOS deployments untested
   - **Recommendation**: Add platform-specific test jobs in CI

6. **⚠️ Stress Testing**
   - **Gap**: Only 2 load tests (4-universe fade)
   - **Missing**: Multi-user, long-running, memory leak tests
   - **Recommendation**: Add performance regression suite

### LOWER PRIORITY

7. **ℹ️ Deployment Validation**
   - **Gap**: RPi/macOS installation, boot, OTA updates untested
   - **Risk**: Field deployment issues
   - **Recommendation**: Add smoke tests to lacylights-rpi/lacylights-mac repos

8. **ℹ️ Chaos Engineering**
   - **Gap**: No network failure, crash recovery, resource exhaustion tests
   - **Risk**: Ungraceful failures under stress
   - **Recommendation**: Add chaos suite (future)

---

## Branch Protection Requirements

### IMMEDIATE ACTION REQUIRED

Enable branch protection on **main** branch for all repositories:

#### Required Status Checks

**lacylights-test**:
- ✅ `test` - Contract tests (CI-safe)
- ✅ `lint` - Go linting
- ✅ `e2e` - Playwright E2E tests

**lacylights-go**:
- ✅ `test` - Unit tests + coverage validation
- ✅ `lint` - golangci-lint
- ✅ `build` - Multi-platform build verification

**lacylights-fe**:
- ✅ `lint-and-type-check` - ESLint + TypeScript strict mode
- ✅ `test` - Jest tests + coverage
- ✅ `security-audit` - npm audit

**lacylights-mcp**:
- ✅ `lint-and-build` - ESLint + TypeScript + tests
- ✅ `security-audit` - npm audit
- ✅ `mcp-validation` - MCP server initialization

#### Additional Protection Settings

For all repos, require:
- ✅ Require a pull request before merging
- ✅ Require approvals: 1 (for solo developer, can be self-approval)
- ✅ Dismiss stale pull request approvals when new commits are pushed
- ✅ Require status checks to pass before merging
- ✅ Require branches to be up to date before merging
- ✅ Include administrators (enforce rules for everyone)

### How to Configure (via GitHub UI)

For each repo:
1. Go to Settings → Branches
2. Add branch protection rule for `main`
3. Enable "Require status checks to pass before merging"
4. Select the required checks listed above
5. Enable "Require branches to be up to date before merging"
6. Save changes

### How to Configure (Automated Script - Recommended)

Use the provided script to enable branch protection on all repositories:

```bash
./scripts/enable-branch-protection.sh
```

This script automatically configures all four repositories with the required status checks.

### How to Configure (Manual via GitHub CLI)

Alternatively, you can manually configure each repository:

```bash
# Manual commands to enable branch protection on all repos
gh api -X PUT "repos/bbernstein/lacylights-test/branches/main/protection" \
  --input - << 'EOF'
{
  "required_status_checks": {
    "strict": true,
    "contexts": ["test", "lint", "e2e"]
  },
  "enforce_admins": true,
  "required_pull_request_reviews": {
    "dismiss_stale_reviews": true,
    "required_approving_review_count": 1
  },
  "restrictions": null,
  "allow_force_pushes": false,
  "allow_deletions": false
}
EOF

# Repeat for lacylights-go (contexts: test, lint, build)
# Repeat for lacylights-fe (contexts: lint-and-type-check, test, security-audit)
# Repeat for lacylights-mcp (contexts: lint-and-build, security-audit, mcp-validation)
```

---

## Test Execution Matrix

### When to Run What

| Scenario | Command | Duration | Coverage |
|----------|---------|----------|----------|
| **Quick feedback** | `./scripts/run-tests.sh ci` | ~30s | API, CRUD, Preview, Settings |
| **Pre-commit** | `./scripts/run-tests.sh all` | ~2min | All contract tests + DMX/fade |
| **Pre-PR** | Run in each repo:<br>`make test` (go)<br>`npm test` (fe/mcp)<br>`./scripts/run-tests.sh all` (test) | ~5min | Full local validation |
| **PR validation** | Automatic via GitHub Actions | ~8min | All CI checks across repos |
| **Pre-release** | Manual workflow dispatch with feature branches | ~15min | Cross-repo integration |
| **Performance** | `make test-load` | ~3min | 4-universe load (2048 channels) |
| **Full validation** | `make test-all && make e2e` | ~10min | Everything including E2E |

---

## Recommended Developer Workflow

### 1. Before Starting Work
```bash
# Ensure all repos are on main and up to date
cd /path/to/your/lacylights  # Replace with your actual path
for repo in lacylights-go lacylights-fe lacylights-mcp lacylights-test; do
  (cd "$repo" && git checkout main && git pull)
done
```

### 2. During Development
```bash
# Quick feedback after changes
cd lacylights-test
./scripts/run-tests.sh ci
```

### 3. Before Committing
```bash
# Run full test suite in each modified repo
cd lacylights-go && make test && make lint
cd ../lacylights-fe && npm test && npm run lint
cd ../lacylights-mcp && npm test && npm run lint
cd ../lacylights-test && ./scripts/run-tests.sh all
```

### 4. Before Creating PR
```bash
# Verify all checks will pass in CI
cd lacylights-test
./scripts/run-tests.sh e2e  # E2E tests
```

### 5. After PR Created
- ✅ Verify all CI checks pass
- ✅ Use manual workflow dispatch to test against feature branches if needed
- ✅ Request Copilot review for complex changes

---

## Coverage Enforcement

### Backend (lacylights-go)

Coverage thresholds enforced by `scripts/check-coverage.sh`:

| Package | Threshold | Rationale |
|---------|-----------|-----------|
| `pkg/artnet` | 100% | Critical DMX output |
| `internal/config` | 100% | Configuration management |
| `internal/database/models` | 100% | Data models |
| `internal/services/fade` | 97% | Fade engine core |
| `internal/services/preview` | 91% | Preview mode |
| `internal/services/dmx` | 88% | DMX management |
| `internal/graphql/resolvers` | 21% | Auto-generated + covered by contract tests |

**Enforcement**: CI fails if coverage drops below thresholds.

### Frontend (lacylights-fe)

- Coverage tracked via Codecov
- No hard thresholds (yet)
- **Recommendation**: Add minimum 60% coverage requirement

### MCP (lacylights-mcp)

- Coverage tracked via Codecov
- No hard thresholds
- **Recommendation**: Add minimum 70% coverage requirement

---

## Future Enhancements

### Short Term (Next Sprint)

1. **Enable Branch Protection** ⚠️ CRITICAL
   - Configure required status checks
   - Prevent direct pushes to main

2. **Add Frontend Contract Tests**
   - Test actual GraphQL queries used by frontend
   - Validate against live backend

3. **Expand E2E Coverage**
   - Add fixture management workflow
   - Add cue list playback workflow
   - Add preview mode workflow

### Medium Term (Next Quarter)

4. **MCP Integration Suite**
   - Test all MCP tools against live backend
   - Add authorization tests
   - Add concurrent session tests

5. **Performance Regression Suite**
   - Multi-user concurrent testing
   - Long-running stability tests
   - Memory leak detection

6. **Platform-Specific Tests**
   - Add macOS CI runner for network tests
   - Add RPi CI runner (GitHub self-hosted)

### Long Term (Future)

7. **Deployment Validation**
   - RPi boot and OTA update tests
   - macOS installation tests
   - Hardware DMX output validation

8. **Chaos Engineering**
   - Network failure simulation
   - Service crash recovery
   - Resource exhaustion testing

---

## Troubleshooting

### Tests Failing Locally

**Symptom**: All tests fail with "connection refused"
**Cause**: Backend not running on port 4001
**Solution**: Use `./scripts/run-tests.sh` which automatically manages backend

### Art-Net Tests Skipping

**Symptom**: DMX/fade tests skip with "Art-Net receiver not available"
**Cause**: Backend not configured with `ARTNET_BROADCAST=127.0.0.1`
**Solution**: Ensure backend started with Art-Net enabled (scripts handle this)

### E2E Tests Timing Out

**Symptom**: Playwright tests timeout
**Cause**: Frontend or backend not starting
**Solution**: Check logs, ensure ports 3001 and 4001 are available

### Coverage Check Failing

**Symptom**: CI fails with "Coverage below threshold"
**Cause**: New code not adequately tested
**Solution**: Add tests to bring coverage above threshold for affected package

---

## Summary

| Aspect | Status | Action Required |
|--------|--------|-----------------|
| **Test Count** | ✅ 129 tests | None - adequate |
| **CI Configuration** | ✅ Running on all PRs | None - working |
| **Branch Protection** | ❌ Not configured | **CRITICAL - Enable ASAP** |
| **Coverage** | ⚠️ Variable (21-100%) | Add FE/MCP contract tests |
| **E2E Tests** | ⚠️ 3 workflows | Expand to 15+ workflows |
| **Cross-Repo Testing** | ❌ Minimal | Add integration tests |

**Top Priority**: Enable branch protection on all repositories to enforce CI checks before merge.
