//go:build mock_db && mock_auth

package testutil

import (
	"context"
	"testing"

	"leapfor.xyz/espyna/internal/application/ports"
	mockAuth "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/auth/mock"
	mockDb "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock"
	mockTranslation "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/translation/mock"
)

// StandardServices provides the complete set of services needed for testing
// This ensures ALL test files have consistent service setup including idAdapter
type StandardServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// CreateStandardServices creates a complete set of mock services for testing
// Parameters:
//   - supportsTransaction: whether the transaction service should support transactions
//   - shouldAuthorize: whether authorization should pass (true) or fail (false)
//
// Returns StandardServices with all services configured according to parameters
func CreateStandardServices(supportsTransaction, shouldAuthorize bool) StandardServices {
	var authService ports.AuthorizationService
	if shouldAuthorize {
		authService = mockAuth.NewAllowAllAuth()
	} else {
		authService = mockAuth.NewDisabledAuth()
	}

	return StandardServices{
		AuthorizationService: authService,
		TransactionService:   mockDb.NewMockTransactionService(supportsTransaction),
		TranslationService:   mockTranslation.NewMockTranslationService(),
		IDService:            ports.NewNoOpIDService(), // Use NoOp for test consistency without additional build tags
	}
}

// GenericTestCase defines a generic structure for table-driven tests that can work with any protobuf request/response types
type GenericTestCase[TRequest any, TResponse any] struct {
	Name           string
	TestCode       string // Test case code (e.g., "ESPYNA-TEST-FRAMEWORK-OBJECTIVE-SUCCESS-v1.0")
	SetupRequest   func(t *testing.T, businessType string) TRequest
	UseTransaction bool
	UseAuth        bool
	ExpectSuccess  bool
	ExpectedError  string // translation key or exact error message
	ErrorTags      map[string]any
	ExactError     bool // if true, use exact error matching instead of translation
	Assertions     func(t *testing.T, response TResponse, err error, useCase interface{}, ctx context.Context)
}

// SetTestCode sets the test case code for tracking purposes and ensures it appears in test output
func SetTestCode(t *testing.T, testCode string) {
	if testCode != "" {
		// Use t.Setenv to set the test code in environment (visible in verbose output)
		t.Setenv("CURRENT_TEST_CODE", testCode)
		// Also log the test code to ensure it appears in output regardless of test result
		t.Logf("TEST_CODE: %s", testCode)
	}
}

// LogTestExecution logs test execution details including test code for tracking
func LogTestExecution(t *testing.T, testCode, testName string, expectSuccess bool) {
	if testCode != "" {
		t.Logf("EXECUTING TEST_CODE: %s | TEST_NAME: %s | EXPECT_SUCCESS: %t", testCode, testName, expectSuccess)
	}
}

// LogTestResult logs test completion with result and test code
func LogTestResult(t *testing.T, testCode, testName string, success bool, err error) {
	status := "PASS"
	if !success {
		status = "FAIL"
	}

	if testCode != "" {
		if err != nil {
			t.Logf("COMPLETED TEST_CODE: %s | TEST_NAME: %s | STATUS: %s | ERROR: %v", testCode, testName, status, err)
		} else {
			t.Logf("COMPLETED TEST_CODE: %s | TEST_NAME: %s | STATUS: %s", testCode, testName, status)
		}
	}
}
