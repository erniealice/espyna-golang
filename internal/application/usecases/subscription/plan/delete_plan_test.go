//go:build mock_db && mock_auth

// Package plan provides comprehensive tests for the plan deletion use case.
//
// The tests cover various scenarios, including success, transaction handling,
// authorization, nil requests, validation errors, and boundary conditions.
// Each test function has a specific test code for tracking and logging.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user").
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education").
//
// Usage:
//   - Run all tests: go test -tags="mock_db,mock_auth" ./...
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestDeletePlanUseCase_Execute_Success
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-SUBSCRIPTION-PLAN-DELETE-SUCCESS-v1.0: Basic successful deletion
//   - ESPYNA-TEST-SUBSCRIPTION-PLAN-DELETE-NOT-FOUND-v1.0: Non-existent ID handling
//   - ESPYNA-TEST-SUBSCRIPTION-PLAN-DELETE-VALIDATION-NIL-REQUEST-v1.0: Nil request validation
//   - ESPYNA-TEST-SUBSCRIPTION-PLAN-DELETE-VALIDATION-NIL-DATA-v1.0: Nil data validation
//   - ESPYNA-TEST-SUBSCRIPTION-PLAN-DELETE-VALIDATION-EMPTY-ID-v1.0: Empty ID validation
//   - ESPYNA-TEST-SUBSCRIPTION-PLAN-DELETE-VALIDATION-SHORT-ID-v1.0: Short ID validation
//
// Data Sources:
//   - Test cases: packages/copya/data_test/{businessType}/plan.json
//   - Mock data: packages/copya/data/{businessType}/plan.json
//   - Translations: packages/lyngua/translations/{languageCode}/{businessType}/plan.json
package plan

import (
	"testing"

	copyatestutil "leapfor.xyz/copya/golang/testutil"
	"github.com/erniealice/espyna-golang/internal/application/shared/testutil"
	mockAuth "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/auth/mock"
	mockDb "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock"
	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock/subscription"
	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/translation"
	planpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/plan"
)

// createTestDeletePlanUseCase is a helper function to create the use case with mock dependencies
func createTestDeletePlanUseCase(businessType string, supportsTransaction bool) *DeletePlanUseCase {
	mockRepo := subscription.NewMockPlanRepository(businessType)

	repositories := DeletePlanRepositories{
		Plan: mockRepo,
	}

	services := DeletePlanServices{
		AuthorizationService: mockAuth.NewAllowAllAuth(),
		TransactionService:   mockDb.NewMockTransactionService(supportsTransaction),
		TranslationService:   translation.NewLynguaTranslationService(),
	}

	return NewDeletePlanUseCase(repositories, services)
}

func TestDeletePlanUseCase_Execute_Success(t *testing.T) {
	testCode := "ESPYNA-TEST-SUBSCRIPTION-PLAN-DELETE-SUCCESS-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "Success", true)

	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestDeletePlanUseCase(businessType, false)

	// Use existing mock data ID
	planID := "plan-academic-year-2025-2026"
	mockRepo := useCase.repositories.Plan.(*subscription.MockPlanRepository)

	req := &planpb.DeletePlanRequest{Data: &planpb.Plan{Id: &planID}}
	res, err := useCase.Execute(ctx, req)

	testutil.AssertNoError(t, err)
	testutil.AssertNotNil(t, res, "response")
	testutil.AssertTrue(t, res.Success, "success")

	// Verify the plan was deleted
	_, err = mockRepo.ReadPlan(ctx, &planpb.ReadPlanRequest{Data: &planpb.Plan{Id: &planID}})
	testutil.AssertError(t, err)

	// Log test completion with result
	testutil.LogTestResult(t, testCode, "Success", true, nil)
}

func TestDeletePlanUseCase_Execute_NotFound(t *testing.T) {
	testCode := "ESPYNA-TEST-SUBSCRIPTION-PLAN-DELETE-NOT-FOUND-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "NotFound", false)

	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestDeletePlanUseCase(businessType, false)

	deleteNotFoundResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "plan", "DeletePlan_NotFound")
	testutil.AssertTestCaseLoad(t, err, "DeletePlan_NotFound")

	nonExistentPlanId := deleteNotFoundResolver.MustGetString("nonExistentPlanId")
	req := &planpb.DeletePlanRequest{Data: &planpb.Plan{Id: &nonExistentPlanId}}
	_, err = useCase.Execute(ctx, req)

	testutil.AssertError(t, err)
	testutil.AssertTranslatedErrorWithTags(t, err, "plan.errors.not_found", map[string]any{"planId": deleteNotFoundResolver.MustGetString("nonExistentPlanId")}, useCase.services.TranslationService, ctx)

	// Log test completion with result
	testutil.LogTestResult(t, testCode, "NotFound", false, err)
}

func TestDeletePlanUseCase_Execute_ValidationErrors(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestDeletePlanUseCase(businessType, false)

	testCases := []struct {
		name     string
		testCode string
		req      *planpb.DeletePlanRequest
		wantErr  bool
	}{
		{"Nil Request", "ESPYNA-TEST-SUBSCRIPTION-PLAN-DELETE-VALIDATION-NIL-REQUEST-v1.0", nil, true},
		{"Nil Data", "ESPYNA-TEST-SUBSCRIPTION-PLAN-DELETE-VALIDATION-NIL-DATA-v1.0", &planpb.DeletePlanRequest{Data: nil}, true},
		{"Empty ID", "ESPYNA-TEST-SUBSCRIPTION-PLAN-DELETE-VALIDATION-EMPTY-ID-v1.0", &planpb.DeletePlanRequest{Data: &planpb.Plan{Id: &[]string{""}[0]}}, true},
		{"Short ID", "ESPYNA-TEST-SUBSCRIPTION-PLAN-DELETE-VALIDATION-SHORT-ID-v1.0", &planpb.DeletePlanRequest{Data: &planpb.Plan{Id: &[]string{"a"}[0]}}, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testutil.SetTestCode(t, tc.testCode)
			testutil.LogTestExecution(t, tc.testCode, tc.name, !tc.wantErr)

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
