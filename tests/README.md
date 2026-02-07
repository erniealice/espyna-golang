# Espyna Test Suite

This directory contains comprehensive tests for the hexagonal architecture implementation in the Espyna package.

## Test Structure

```
tests/
├── integration/           # Multi-provider integration tests
│   ├── provider_switch_test.go    # Test switching between providers
│   ├── table_mapping_test.go      # Test configurable table/collection names
│   └── health_check_test.go       # Test provider health monitoring
├── e2e/                   # End-to-end server tests
│   └── server_test.go             # Test different server types with providers
├── testutil/              # Shared test utilities
│   ├── provider_helpers.go        # Provider setup and cleanup helpers
│   └── test_data.go               # Standardized test data
└── README.md              # This file
```

## Running Tests

### All Tests
```bash
cd packages/espyna
go test ./tests/...
```

### Integration Tests Only
```bash
cd packages/espyna
go test ./tests/integration/...
```

### End-to-End Tests Only
```bash
cd packages/espyna
go test ./tests/e2e/...
```

### Specific Test
```bash
cd packages/espyna
go test ./tests/integration/ -run TestProviderSwitching
```

### Verbose Output
```bash
cd packages/espyna
go test -v ./tests/...
```

### Run Tests in Parallel
```bash
cd packages/espyna
go test -parallel 4 ./tests/...
```

## Test Environment Setup

### Mock Provider
- Always available
- No external dependencies
- Uses in-memory storage

### PostgreSQL Provider
- Requires PostgreSQL instance
- Set environment variables:
  ```bash
  export TEST_POSTGRES_HOST=localhost
  export TEST_POSTGRES_PORT=5432
  export TEST_POSTGRES_DB=espyna_test
  export TEST_POSTGRES_USER=postgres
  export TEST_POSTGRES_PASSWORD=your_password
  ```

### Firestore Provider
- Requires Firestore emulator or test project
- Set environment variables:
  ```bash
  export TEST_FIRESTORE_PROJECT=espyna-test
  export FIRESTORE_EMULATOR_HOST=localhost:8080  # For emulator
  ```

## Test Coverage

### Provider Switching Tests
- Validates that the same business operations work across all providers
- Tests CRUD operations for Client and Admin entities
- Ensures data consistency regardless of provider

### Table/Collection Mapping Tests
- Tests configurable table/collection names
- Validates that custom names work correctly
- Tests default name fallback behavior

### Health Check Tests
- Tests provider health monitoring
- Validates failover scenarios
- Tests timeout handling

### End-to-End Tests
- Tests complete HTTP server functionality
- Validates different server types (Vanilla, Gin, Fiber)
- Tests concurrent request handling

## Test Data

All tests use standardized test data from `testutil/test_data.go`:
- Deterministic IDs and timestamps
- Consistent entity relationships
- Isolated test data per test run

## Architecture Validation

These tests validate key aspects of the hexagonal architecture:

1. **Provider Abstraction**: Business logic works identically across providers
2. **Configuration Flexibility**: Table/collection names are configurable
3. **Health Monitoring**: System can detect and respond to provider failures
4. **Server Independence**: All server types work with all providers
5. **Concurrent Safety**: System handles concurrent operations correctly

## Test Isolation

- Each test creates its own container instance
- Test data is isolated per test run
- Cleanup ensures no test pollution
- Tests can run in parallel safely

## Performance Considerations

- Mock provider tests are fastest (in-memory)
- PostgreSQL tests require database connectivity
- Firestore tests require emulator or network connectivity
- Use `-parallel` flag to speed up execution

## Troubleshooting

### Tests Skip Due to Provider Unavailability
- Check that required providers are properly configured
- Verify environment variables are set correctly
- Ensure external services (PostgreSQL, Firestore) are running

### Tests Fail Due to Timeouts
- Increase timeout values in test configuration
- Check network connectivity to external services
- Verify that providers are responding correctly

### Tests Fail Due to Data Conflicts
- Ensure proper test cleanup
- Use unique test data identifiers
- Check that test isolation is working correctly