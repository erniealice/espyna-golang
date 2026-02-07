# Espyna Test Revamp Plan Summary

This document outlines the plan to overhaul the testing strategy for the `espyna` Go package.

**Objective**: Replace the outdated, hardcoded E2E tests with a robust, data-driven unit and integration test suite.

**Key Changes**:
1.  **Centralized Data**: All mock data will be loaded from the `@leapfor/copya` package, eliminating hardcoded data in test files.
2.  **Data-Driven Approach**: Test cases will be defined in JSON files within a new `packages/espyna/tests/unit/data_test` directory.
3.  **New Test Suite**: A new test suite will be created in `packages/espyna/tests/unit`, organized by domain.
4.  **Full Coverage**: The new suite will test all CRUDL endpoints defined in `internal/composition/routes.go`.
5.  **Standard Tooling**: Tests will be executed using standard `go test` commands.
6.  **Logging**: Test runs will generate timestamped log files in a `results` directory.

This initiative will improve test reliability, maintainability, and scalability. For the full detailed plan, see `packages/espyna/docs/plans/20250906-tests/index.md`.
