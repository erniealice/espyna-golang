#!/bin/bash

# USAGE:
# ./packages/espyna/scripts/run-e2e-tests.sh
#
# This script runs all E2E tests for the Espyna package and saves the complete
# output to a timestamped log file. It must be run from the monorepo root.

# --- Configuration ---
PACKAGE_DIR="packages/espyna"
RESULTS_DIR="${PACKAGE_DIR}/tests/e2e/results"
TEST_COMMAND="go test ./tests/e2e -v"

# --- Script Logic ---
echo "Changing directory to package: ${PACKAGE_DIR}"
cd "${PACKAGE_DIR}" || { echo "Failed to change directory. Aborting."; exit 1; }

echo "Starting Espyna E2E Test Execution..."

# Ensure the results directory exists.
mkdir -p "tests/e2e/results"
echo "Results will be stored in: $(pwd)/tests/e2e/results"

# Generate timestamped filenames.
TIMESTAMP=$(date +%Y%m%d-%H%M)
LOG_FILE="tests/e2e/results/${TIMESTAMP}-logs.md"
FEEDBACK_FILE="tests/e2e/results/${TIMESTAMP}-feedback.md"

echo "Running command: ${TEST_COMMAND}"
echo "Saving output to: $(pwd)/${LOG_FILE}"

# Execute the test command.
eval "${TEST_COMMAND}" 2>&1 | tee "${LOG_FILE}"

# Capture the exit code of the test command.
TEST_STATUS=${PIPESTATUS[0]}



if [ $TEST_STATUS -eq 0 ]; then
    echo -e "\nAll tests passed successfully!"
else
    echo -e "\nSome tests failed. Please review the logs."
fi

echo "Script finished."
