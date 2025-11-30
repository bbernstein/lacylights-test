# Migration Testing Guide

This document describes the migration testing strategy for validating the Go backend migration from Node.js.

## Overview

The migration test suite ensures that:

1. **Database Compatibility**: Go server can read and write the same SQLite database as Node
2. **API Parity**: GraphQL APIs return identical responses from both servers
3. **Data Integrity**: All data is preserved during migration and rollback
4. **Distribution**: Go binaries are properly distributed via S3
5. **End-to-End**: Complete migration workflows function correctly

## Test Categories

### 1. Database Migration Tests (`integration/migration_db_test.go`)

Tests that verify database-level compatibility between Node and Go servers.

#### Tests Included:

- **TestDatabaseSchemaCompatibility**: Verifies Go server can read Node's SQLite database
- **TestDatabaseTableStructure**: Validates all expected tables exist
- **TestDataPreservation**: Ensures data written by Node is preserved when read by Go
- **TestRollbackCompatibility**: Verifies data written by Go can be read by Node (rollback scenario)
- **TestComplexDataMigration**: Tests migration of complex nested data (projects, fixtures, scenes, cues)

#### Running Database Tests:

```bash
# Run all database migration tests
make test-migration-db

# Run specific test
go test -v -run TestDatabaseSchemaCompatibility ./integration/
```

#### Environment Variables:

- `NODE_SERVER_URL`: Node GraphQL endpoint (default: http://localhost:4000/graphql)
- `GO_SERVER_URL`: Go GraphQL endpoint (default: http://localhost:4001/graphql)
- `DATABASE_PATH`: Path to shared SQLite database (optional)

### 2. API Comparison Tests (`integration/migration_api_test.go`)

Tests that verify GraphQL API responses are identical between Node and Go servers.

#### Tests Included:

- **TestGraphQLAPIComparison**: Compares responses for common queries
- **TestMutationAPIComparison**: Verifies mutations work identically
- **TestErrorHandlingComparison**: Ensures error responses are consistent
- **TestConcurrentRequestsComparison**: Validates concurrent request handling
- **TestSubscriptionAPIComparison**: Checks WebSocket subscription endpoints
- **TestSchemaIntrospectionComparison**: Verifies GraphQL schemas are identical

#### Running API Tests:

```bash
# Run all API comparison tests
make test-migration-api

# Run specific test
go test -v -run TestGraphQLAPIComparison ./integration/
```

#### What Gets Compared:

- Query responses (structure and data)
- Mutation behavior
- Error messages and codes
- Schema introspection results
- Subscription field availability

### 3. Distribution Tests (`integration/migration_distribution_test.go`)

Tests that verify Go binary distribution via S3.

#### Tests Included:

- **TestLatestJSONEndpoint**: Verifies latest.json is accessible and valid
- **TestBinaryDownload**: Tests binary downloads for current platform
- **TestChecksumValidation**: Validates checksums for all platforms
- **TestBinaryExecutable**: Verifies downloaded binary is executable
- **TestVersionConsistency**: Checks version format and timestamp
- **TestAllPlatformsAvailable**: Ensures all expected platform binaries exist
- **TestDistributionCDN**: Verifies CDN/S3 configuration (CORS, caching)

#### Running Distribution Tests:

```bash
# Run all distribution tests
make test-migration-distribution

# Run specific test
go test -v -run TestLatestJSONEndpoint ./integration/

# Skip slow download tests
go test -v -short ./integration/
```

#### Environment Variables:

- `S3_BASE_URL`: S3 bucket URL (default: https://lacylights-binaries.s3.amazonaws.com)

#### Supported Platforms:

- linux-amd64
- linux-arm64
- darwin-amd64
- darwin-arm64
- windows-amd64

### 4. End-to-End Migration Tests (`e2e/migration_e2e_test.go`)

Tests that simulate complete migration workflows.

#### Tests Included:

- **TestFullMigrationWorkflow**: Simulates complete Node → Go migration
  - Create data with Node
  - Verify Go can read it
  - Modify data with Go
  - Verify Node can read modifications

- **TestRollbackScenario**: Simulates Go → Node rollback
  - Create data with Go
  - Verify Node can read it
  - Continue operations with Node

- **TestDataIntegrityDuringMigration**: Verifies data integrity throughout migration
  - Compares complete project state across servers
  - Validates fixture counts, scene counts, etc.

- **TestMigrationPerformance**: Compares performance between Node and Go
  - Benchmarks query response times
  - Ensures Go has reasonable performance

#### Running E2E Tests:

```bash
# Run all e2e migration tests (may take several minutes)
make test-migration-e2e

# Run specific test
go test -v -timeout 10m -run TestFullMigrationWorkflow ./e2e/

# Skip long-running tests
go test -v -short ./e2e/
```

## Running All Migration Tests

### Quick Test (Excludes Slow Tests)

```bash
make test-migration-quick
```

This runs all migration tests with `-short` flag, skipping:
- Binary downloads
- Performance benchmarks
- Long-running e2e scenarios

### Full Migration Test Suite

```bash
make test-migration
```

This runs all migration tests including:
- Database compatibility tests
- API comparison tests
- Distribution tests
- End-to-end migration tests

**Expected Duration**: 5-15 minutes depending on network speed

## Prerequisites

### Running Servers

Both Node and Go servers must be running and accessible:

```bash
# Terminal 1: Start Node server
cd lacylights-node
npm start
# Should be running on http://localhost:4000

# Terminal 2: Start Go server
cd lacylights-go
go run ./cmd/server
# Should be running on http://localhost:4001
```

### Shared Database

For database compatibility tests, both servers should use the same database:

```bash
# Option 1: Both servers use same database file
export DATABASE_URL="file:./shared.db"

# Option 2: Node uses its database, Go reads it
# (Tests will verify Go can read Node's database)
```

### Dependencies

```bash
cd lacylights-test
go mod download
```

## CI/CD Integration

### GitHub Actions Example

```yaml
name: Migration Tests

on: [push, pull_request]

jobs:
  migration-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Setup Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.23'

      - name: Setup Node.js
        uses: actions/setup-node@v3
        with:
          node-version: '20'

      - name: Start Node Server
        run: |
          cd lacylights-node
          npm install
          npm start &
          sleep 5

      - name: Start Go Server
        run: |
          cd lacylights-go
          go run ./cmd/server &
          sleep 5

      - name: Run Migration Tests
        run: |
          cd lacylights-test
          make test-migration-quick
```

## Test Fixtures and Data

### Test Project Structure

Migration tests create temporary projects with:

- **3 Fixtures**: Front PAR 1, Front PAR 2, Back PAR 1
- **3 Scenes**: Red Wash, Blue Wash, White Wash
- **1 Cue List**: With 3 cues referencing the scenes

All test data is cleaned up after tests complete.

### Database Fixtures

For database-specific tests, fixtures include:

- SQLite schema creation
- Sample project data
- Fixture definitions
- Scene configurations

## Troubleshooting

### Tests Fail: "Connection Refused"

**Problem**: Server not running or wrong port

**Solution**:
```bash
# Check server status
curl http://localhost:4000/graphql
curl http://localhost:4001/graphql

# Check environment variables
echo $NODE_SERVER_URL
echo $GO_SERVER_URL
```

### Tests Fail: "Database Locked"

**Problem**: Both servers trying to write to same database

**Solution**:
```bash
# Use WAL mode in SQLite
sqlite3 dev.db "PRAGMA journal_mode=WAL;"

# Or use separate databases
export NODE_DB="node.db"
export GO_DB="go.db"
```

### Tests Fail: "Checksum Mismatch"

**Problem**: Downloaded binary doesn't match expected checksum

**Solution**:
```bash
# Verify latest.json is up to date
curl https://lacylights-binaries.s3.amazonaws.com/latest.json

# Check if binaries have been updated recently
# Re-run tests to ensure consistency
```

### Distribution Tests Skip

**Problem**: "Platform not found in artifacts"

**Solution**: This is expected if testing on a platform that doesn't have binaries yet. Tests will skip gracefully.

## Test Coverage

### Database Coverage

- ✅ Schema compatibility
- ✅ Data preservation (Node → Go)
- ✅ Rollback compatibility (Go → Node)
- ✅ Complex nested data
- ✅ All 17 table structures

### API Coverage

- ✅ Query responses
- ✅ Mutation behavior
- ✅ Error handling
- ✅ Concurrent requests
- ✅ WebSocket subscriptions
- ✅ Schema introspection

### Distribution Coverage

- ✅ Binary downloads
- ✅ Checksum validation
- ✅ Platform availability
- ✅ Version consistency
- ✅ CDN configuration

### E2E Coverage

- ✅ Full migration workflow
- ✅ Rollback scenario
- ✅ Data integrity
- ✅ Performance comparison

## Best Practices

### 1. Run Quick Tests During Development

```bash
make test-migration-quick
```

### 2. Run Full Suite Before Deployment

```bash
make test-migration
```

### 3. Monitor Performance

```bash
# Run with verbose output to see timing
go test -v -run TestMigrationPerformance ./e2e/
```

### 4. Clean Up Test Data

Tests automatically clean up, but if interrupted:

```bash
# Check for leftover test projects
# (Look for projects with "Test" in the name)
```

### 5. Use Shared Database for Integration

When testing migration, use the same database:

```bash
export DATABASE_URL="file:./migration-test.db"
# Start both servers with this database
```

## Future Enhancements

Potential additions to the migration test suite:

1. **Load Testing**: Simulate heavy concurrent usage during migration
2. **Network Interruption**: Test resilience to network failures
3. **Partial Migration**: Test incremental migration scenarios
4. **Version Skew**: Test compatibility across different versions
5. **Real-Time Updates**: Test WebSocket subscription migration
6. **Database Migration Scripts**: Test automated schema migration tools

## Related Documentation

- [Contract Testing Plan](CONTRACT_TESTING_PLAN.md)
- [Go Rewrite Plan](../LACYLIGHTS_GO_REWRITE_PLAN.md)
- [Go Rewrite Progress](../LACYLIGHTS_GO_REWRITE_PROGRESS.md)
- [Main README](README.md)
