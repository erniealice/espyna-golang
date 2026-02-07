//go:build mock_db && mock_auth

// Package plan provides comprehensive tests for the plan reading use case.
//
// The tests cover various scenarios, including success, not found, and validation errors.
// Each test function has a specific test code for tracking and logging.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user").
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education").
//
// Usage:
//   - Run all tests: go test -tags="mock_db,mock_auth" ./...
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestReadPlanUseCase_Execute_Success
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-SUBSCRIPTION-PLAN-READ-SUCCESS-v1.0: Basic successful reading
//   - ESPYNA-TEST-SUBSCRIPTION-PLAN-READ-NOT-FOUND-v1.0: Non-existent ID handling
//   - ESPYNA-TEST-SUBSCRIPTION-PLAN-READ-VALIDATION-NIL-REQUEST-v1.0: Nil request validation
//   - ESPYNA-TEST-SUBSCRIPTION-PLAN-READ-VALIDATION-NIL-DATA-v1.0: Nil data validation
//   - ESPYNA-TEST-SUBSCRIPTION-PLAN-READ-VALIDATION-EMPTY-ID-v1.0: Empty ID validation
//   - ESPYNA-TEST-SUBSCRIPTION-PLAN-READ-VALIDATION-SHORT-ID-v1.0: Short ID validation
//
// Data Sources:
//   - Test cases: packages/copya/data_test/{businessType}/plan.json
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

// createTestReadPlanUseCase is a helper function to create the use case with mock dependencies
func createTestReadPlanUseCase(businessType string) *ReadPlanUseCase {
	mockRepo := subscription.NewMockPlanRepository(businessType)

	repositories := ReadPlanRepositories{
		Plan: mockRepo,
	}

	services := ReadPlanServices{
		AuthorizationService: mockAuth.NewAllowAllAuth(),
		TransactionService:   mockDb.NewMockTransactionService(false),
		TranslationService:   translation.NewLynguaTranslationService(),
	}

	return NewReadPlanUseCase(repositories, services)
}

func TestReadPlanUseCase_Execute_Success(t *testing.T) {
	testCode := "ESPYNA-TEST-SUBSCRIPTION-PLAN-READ-SUCCESS-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "Success", true)

	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestReadPlanUseCase(businessType)

	// Use existing mock data ID
	planID := "plan-academic-year-2024-2025"

	req := &planpb.ReadPlanRequest{Data: &planpb.Plan{Id: &planID}}
	res, err := useCase.Execute(ctx, req)

	testutil.AssertNoError(t, err)
	testutil.AssertNotNil(t, res, "response")
	testutil.AssertTrue(t, res.Success, "success")
	testutil.AssertDataLength(t, 1, len(res.Data), "response data")

	readPlan := res.Data[0]
	testutil.AssertStringEqual(t, planID, *readPlan.Id, "plan ID")

	// Log test completion with result
	testutil.LogTestResult(t, testCode, "Success", true, nil)
}

func TestReadPlanUseCase_Execute_NotFound(t *testing.T) {
	testCode := "ESPYNA-TEST-SUBSCRIPTION-PLAN-READ-NOT-FOUND-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "NotFound", false)

	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestReadPlanUseCase(businessType)

	nonexistentPlanId := "nonexistent-plan-id"
	req := &planpb.ReadPlanRequest{Data: &planpb.Plan{Id: &nonexistentPlanId}}
	_, err := useCase.Execute(ctx, req)

	testutil.AssertError(t, err)
	testutil.AssertTranslatedError(t, err, "plan.errors.not_found", useCase.services.TranslationService, ctx)

	// Log test completion with result
	testutil.LogTestResult(t, testCode, "NotFound", false, err)
}

func TestReadPlanUseCase_Execute_ValidationErrors(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestReadPlanUseCase(businessType)

	testCases := []struct {
		name     string
		testCode string
		req      *planpb.ReadPlanRequest
		wantErr  bool
	}{
		{"Nil Request", "ESPYNA-TEST-SUBSCRIPTION-PLAN-READ-VALIDATION-NIL-REQUEST-v1.0", nil, true},
		{"Nil Data", "ESPYNA-TEST-SUBSCRIPTION-PLAN-READ-VALIDATION-NIL-DATA-v1.0", &planpb.ReadPlanRequest{Data: nil}, true},
		{"Empty ID", "ESPYNA-TEST-SUBSCRIPTION-PLAN-READ-VALIDATION-EMPTY-ID-v1.0", &planpb.ReadPlanRequest{Data: &planpb.Plan{Id: &[]string{""}[0]}}, true},
		{"Short ID", "ESPYNA-TEST-SUBSCRIPTION-PLAN-READ-VALIDATION-SHORT-ID-v1.0", &planpb.ReadPlanRequest{Data: &planpb.Plan{Id: &[]string{"a"}[0]}}, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testutil.SetTestCode(t, tc.testCode)
			testutil.LogTestExecution(t, tc.testCode, tc.name, false)

			_, err := useCase.Execute(ctx, tc.req)
			if tc.wantErr {
				testutil.AssertError(t, err)
			} else {
				testutil.AssertNoError(t, err)
			}

			// Log test completion with result
			testutil.LogTestResult(t, tc.testCode, tc.name, !tc.wantErr, err)
		})
	}
}
