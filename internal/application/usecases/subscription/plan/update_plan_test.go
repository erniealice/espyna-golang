//go:build mock_db && mock_auth

// Package plan provides comprehensive tests for the plan update use case.
//
// The tests cover various scenarios, including success, not found, validation errors,
// and data enrichment. Each test function has a specific test code for tracking and logging.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user").
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education").
//
// Usage:
//   - Run all tests: go test -tags="mock_db,mock_auth" ./...
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestUpdatePlanUseCase_Execute_Success
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-SUBSCRIPTION-PLAN-UPDATE-SUCCESS-v1.0: Basic successful updating
//   - ESPYNA-TEST-SUBSCRIPTION-PLAN-UPDATE-NOT-FOUND-v1.0: Non-existent ID handling
//   - ESPYNA-TEST-SUBSCRIPTION-PLAN-UPDATE-VALIDATION-v1.0: Input validation scenarios
//   - ESPYNA-TEST-SUBSCRIPTION-PLAN-UPDATE-ENRICHMENT-v1.0: Auto-generated fields verification
//
// Data Sources:
//   - Test cases: packages/copya/data_test/{businessType}/plan.json
//   - Mock data: packages/copya/data/{businessType}/plan.json
//   - Translations: packages/lyngua/translations/{languageCode}/{businessType}/plan.json
package plan

import (
	"testing"
	"time"

	"leapfor.xyz/espyna/internal/application/shared/testutil"
	mockAuth "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/auth/mock"
	mockDb "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock"
	"leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock/subscription"
	"leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/translation"
	planpb "leapfor.xyz/esqyma/golang/v1/domain/subscription/plan"
)

// strPtrUpdate is a helper to convert string to *string for optional fields
func strPtrUpdate(s string) *string { return &s }

// createTestUpdatePlanUseCase is a helper function to create the use case with mock dependencies
func createTestUpdatePlanUseCase(businessType string, supportsTransaction bool) *UpdatePlanUseCase {
	mockRepo := subscription.NewMockPlanRepository(businessType)

	repositories := UpdatePlanRepositories{
		Plan: mockRepo,
	}

	services := UpdatePlanServices{
		AuthorizationService: mockAuth.NewAllowAllAuth(),
		TransactionService:   mockDb.NewMockTransactionService(supportsTransaction),
		TranslationService:   translation.NewLynguaTranslationService(),
	}

	return NewUpdatePlanUseCase(repositories, services)
}

func TestUpdatePlanUseCase_Execute_Success(t *testing.T) {
	testCode := "ESPYNA-TEST-SUBSCRIPTION-PLAN-UPDATE-SUCCESS-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "Success", true)

	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestUpdatePlanUseCase(businessType, false)

	// Use existing mock data ID
	planID := "plan-academic-year-2024-2025"

	time.Sleep(1 * time.Second)

	req := &planpb.UpdatePlanRequest{
		Data: &planpb.Plan{
			Id:          &planID,
			Name:        "New Name",
			Description: strPtrUpdate("New Desc"),
		},
	}
	res, err := useCase.Execute(ctx, req)

	testutil.AssertNoError(t, err)
	testutil.AssertNotNil(t, res, "response")
	testutil.AssertTrue(t, res.Success, "success")
	testutil.AssertEqual(t, 1, len(res.Data), "response data length")

	updatedPlan := res.Data[0]
	testutil.AssertStringEqual(t, "New Name", updatedPlan.Name, "plan name")

	// Log test completion with result
	testutil.LogTestResult(t, testCode, "Success", true, nil)
}

func TestUpdatePlanUseCase_Execute_NotFound(t *testing.T) {
	testCode := "ESPYNA-TEST-SUBSCRIPTION-PLAN-UPDATE-NOT-FOUND-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "NotFound", false)

	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestUpdatePlanUseCase(businessType, false)

	nonexistentPlanId := "nonexistent-plan-id"
	req := &planpb.UpdatePlanRequest{Data: &planpb.Plan{Id: &nonexistentPlanId, Name: "n", Description: strPtrUpdate("d")}}
	_, err := useCase.Execute(ctx, req)

	testutil.AssertError(t, err)
	testutil.AssertTranslatedErrorWithTags(t, err, "plan.errors.not_found", map[string]any{"planId": "nonexistent-plan-id"}, useCase.services.TranslationService, ctx)

	// Log test completion with result
	testutil.LogTestResult(t, testCode, "NotFound", false, err)
}

func TestUpdatePlanUseCase_Execute_ValidationErrors(t *testing.T) {
	testCode := "ESPYNA-TEST-SUBSCRIPTION-PLAN-UPDATE-VALIDATION-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "ValidationErrors", false)

	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestUpdatePlanUseCase(businessType, false)
	longString := testutil.GenerateDefaultLongString(501)

	testCases := []struct {
		name    string
		req     *planpb.UpdatePlanRequest
		wantErr bool
	}{
		{"Nil Request", nil, true},
		{"Nil Data", &planpb.UpdatePlanRequest{Data: nil}, true},
		{"Empty ID", &planpb.UpdatePlanRequest{Data: &planpb.Plan{Id: &[]string{""}[0]}}, true},
		{"Short ID", &planpb.UpdatePlanRequest{Data: &planpb.Plan{Id: &[]string{"a"}[0]}}, true},
		{"Empty Name", &planpb.UpdatePlanRequest{Data: &planpb.Plan{Id: &[]string{"id"}[0], Name: ""}}, true},
		{"Long Name", &planpb.UpdatePlanRequest{Data: &planpb.Plan{Id: &[]string{"id"}[0], Name: longString}}, true},
		{"Long Description", &planpb.UpdatePlanRequest{Data: &planpb.Plan{Id: &[]string{"id"}[0], Name: "n", Description: strPtrUpdate(longString)}}, true},
	}

	allTestsPassed := true
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := useCase.Execute(ctx, tc.req)
			if tc.wantErr {
				testutil.AssertError(t, err)
			} else {
				testutil.AssertNoError(t, err)
			}
			if (err != nil) != tc.wantErr {
				allTestsPassed = false
			}
		})
	}

	// Log test completion with result
	testutil.LogTestResult(t, testCode, "ValidationErrors", allTestsPassed, nil)
}

func TestUpdatePlanUseCase_DataEnrichment(t *testing.T) {
	testCode := "ESPYNA-TEST-SUBSCRIPTION-PLAN-UPDATE-ENRICHMENT-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "DataEnrichment", true)

	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestUpdatePlanUseCase(businessType, false)

	// Use existing mock data ID
	planID := "plan-academic-year-2025-2026"

	req := &planpb.UpdatePlanRequest{Data: &planpb.Plan{Id: &planID, Name: "Academic Year 2025-2026 Updated", Description: strPtrUpdate("Updated plan description")}}
	res, err := useCase.Execute(ctx, req)

	testutil.AssertNoError(t, err)
	testutil.AssertNotNil(t, res, "response")

	updatedPlan := res.Data[0]
	testutil.AssertFieldSet(t, updatedPlan.DateModified, "DateModified")

	// Log test completion with result
	testutil.LogTestResult(t, testCode, "DataEnrichment", true, nil)
}
