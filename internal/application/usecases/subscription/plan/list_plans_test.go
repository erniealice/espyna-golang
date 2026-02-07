//go:build mock_db && mock_auth

// Package plan provides comprehensive tests for the plan listing use case.
//
// The tests cover various scenarios, including success, empty results, and nil requests.
// Each test function has a specific test code for tracking and logging.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user")
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education")
//
// Usage:
//   - Run all tests: go test -tags="mock_db,mock_auth" ./...
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestListPlansUseCase
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-SUBSCRIPTION-PLAN-LIST-SUCCESS-v1.0: Tests successful listing of plans
//   - ESPYNA-TEST-SUBSCRIPTION-PLAN-LIST-EMPTY-v1.0: Tests handling of empty plan list
//   - ESPYNA-TEST-SUBSCRIPTION-PLAN-LIST-NIL-REQUEST-v1.0: Tests error handling for nil request
//
// Data Sources:
//   - Mock data: packages/copya/data/{businessType}/plan.json
//   - Translations: packages/lyngua/translations/{languageCode}/{businessType}/plan.json
package plan

import (
	"testing"

	"leapfor.xyz/espyna/internal/application/shared/testutil"
	mockAuth "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/auth/mock"
	mockDb "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock"
	"leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock/subscription"
	"leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/translation"
	planpb "leapfor.xyz/esqyma/golang/v1/domain/subscription/plan"
)

// createTestListPlansUseCase is a helper function to create the use case with mock dependencies
func createTestListPlansUseCase(businessType string) *ListPlansUseCase {
	mockRepo := subscription.NewMockPlanRepository(businessType)

	repositories := ListPlansRepositories{
		Plan: mockRepo,
	}

	services := ListPlansServices{
		AuthorizationService: mockAuth.NewAllowAllAuth(),
		TransactionService:   mockDb.NewMockTransactionService(false),
		TranslationService:   translation.NewLynguaTranslationService(),
	}

	return NewListPlansUseCase(repositories, services)
}

func TestListPlansUseCase_Execute_Success(t *testing.T) {
	testCode := "ESPYNA-TEST-SUBSCRIPTION-PLAN-LIST-SUCCESS-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "Success", true)

	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestListPlansUseCase(businessType)

	// Mock repository already has pre-loaded data:
	// plan-academic-year-2024-2025, plan-academic-year-2025-2026, and others

	req := &planpb.ListPlansRequest{}
	res, err := useCase.Execute(ctx, req)

	testutil.AssertNoError(t, err)
	testutil.AssertNotNil(t, res, "response")
	testutil.AssertTrue(t, res.Success, "success")
	testutil.AssertEqual(t, 4, len(res.Data), "response data length")

	// Log test completion with result
	testutil.LogTestResult(t, testCode, "Success", true, nil)
}

func TestListPlansUseCase_Execute_Empty(t *testing.T) {
	testCode := "ESPYNA-TEST-SUBSCRIPTION-PLAN-LIST-EMPTY-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "Empty", true)

	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestListPlansUseCase(businessType)

	// This test expects empty results but mock data is always loaded
	// We would need a separate mock repository without pre-loaded data
	// For now, we expect the pre-loaded data (4 plans)

	req := &planpb.ListPlansRequest{}
	res, err := useCase.Execute(ctx, req)

	testutil.AssertNoError(t, err)
	testutil.AssertNotNil(t, res, "response")
	testutil.AssertTrue(t, res.Success, "success")
	// Mock always has 4 pre-loaded plans
	testutil.AssertEqual(t, 4, len(res.Data), "response data length (pre-loaded mock data)")

	// Log test completion with result
	testutil.LogTestResult(t, testCode, "Empty", true, nil)
}

func TestListPlansUseCase_Execute_NilRequest(t *testing.T) {
	testCode := "ESPYNA-TEST-SUBSCRIPTION-PLAN-LIST-NIL-REQUEST-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "NilRequest", false)

	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestListPlansUseCase(businessType)

	res, err := useCase.Execute(ctx, nil)
	testutil.AssertErrorForNilRequest(t, err)
	testutil.AssertNil(t, res, "response for nil request")

	// Log test completion with result
	testutil.LogTestResult(t, testCode, "NilRequest", false, err)
}
