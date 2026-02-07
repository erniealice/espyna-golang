//go:build mock_db && mock_auth

// Package plan_settings provides table-driven tests for the plan settings listing use case.
//
// The tests cover various scenarios, including success, transaction handling,
// authorization, nil requests, validation errors, and boundary conditions.
// Each test case is defined in a table with a specific test code, request setup,
// and assertions.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user").
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education").
//
// Usage:
//   - Run all tests: go test -tags="mock_db,mock_auth" ./...
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestListPlanSettingsUseCase_Execute_TableDriven
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-SUBSCRIPTION-PLANSETTINGS-LIST-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-SUBSCRIPTION-PLANSETTINGS-LIST-TRANSACTION-v1.0: WithTransaction
//   - ESPYNA-TEST-SUBSCRIPTION-PLANSETTINGS-LIST-AUTHORIZATION-v1.0: Unauthorized
//   - ESPYNA-TEST-SUBSCRIPTION-PLANSETTINGS-LIST-NIL-REQUEST-v1.0: NilRequest
//
// Data Sources:
//   - Test cases: packages/copya/data_test/{businessType}/plan_settings.json
//   - Mock data: packages/copya/data/{businessType}/plan_settings.json
//   - Translations: packages/lyngua/translations/{languageCode}/{businessType}/plan_settings.json
package plan_settings

import (
	"context"
	"testing"

	copyatestutil "leapfor.xyz/copya/golang/testutil"
	"leapfor.xyz/espyna/internal/application/shared/testutil"
	"leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock/subscription"
	plansettingspb "leapfor.xyz/esqyma/golang/v1/domain/subscription/plan_settings"
)

// Type alias for list plan settings test cases
type ListPlanSettingsTestCase = testutil.GenericTestCase[*plansettingspb.ListPlanSettingsRequest, *plansettingspb.ListPlanSettingsResponse]

func createTestListUseCaseWithAuth(businessType string, supportsTransaction bool, shouldAuthorize bool) *ListPlanSettingsUseCase {
	mockRepo := subscription.NewMockPlanSettingsRepository(businessType)

	repositories := ListPlanSettingsRepositories{
		PlanSettings: mockRepo,
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, shouldAuthorize)
	services := ListPlanSettingsServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
	}

	return NewListPlanSettingsUseCase(repositories, services)
}

func TestListPlanSettingsUseCase_Execute_TableDriven(t *testing.T) {
	businessType := testutil.GetTestBusinessType()

	// Load test data resolvers before defining test cases
	listSuccessResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "plan_settings", "ListPlanSettings_Success")
	testutil.AssertTestCaseLoad(t, err, "ListPlanSettings_Success")

	authorizationUnauthorizedResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "plan_settings", "Authorization_Unauthorized")
	testutil.AssertTestCaseLoad(t, err, "Authorization_Unauthorized")
	_ = authorizationUnauthorizedResolver // Used in Unauthorized test case

	testCases := []ListPlanSettingsTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-PLANSETTINGS-LIST-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *plansettingspb.ListPlanSettingsRequest {
				return &plansettingspb.ListPlanSettingsRequest{}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *plansettingspb.ListPlanSettingsResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				expectedCount := listSuccessResolver.MustGetInt("expectedPlanSettingsCount")
				testutil.AssertEqual(t, expectedCount, len(response.Data), "plan settings count")

				// Verify we have plan settings with proper structure
				if len(response.Data) > 0 {
					firstPlanSettings := response.Data[0]
					testutil.AssertNonEmptyString(t, firstPlanSettings.Id, "first plan settings ID")
					testutil.AssertNonEmptyString(t, firstPlanSettings.Name, "first plan settings name")
					testutil.AssertTrue(t, firstPlanSettings.Active, "first plan settings active status")
					testutil.AssertFieldSet(t, firstPlanSettings.DateCreated, "DateCreated")
				}
			},
		},
		{
			Name:     "WithTransaction",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-PLANSETTINGS-LIST-TRANSACTION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *plansettingspb.ListPlanSettingsRequest {
				return &plansettingspb.ListPlanSettingsRequest{}
			},
			UseTransaction: true,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *plansettingspb.ListPlanSettingsResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				expectedCount := listSuccessResolver.MustGetInt("expectedPlanSettingsCount")
				testutil.AssertEqual(t, expectedCount, len(response.Data), "plan settings count")
			},
		},
		{
			Name:     "Unauthorized",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-PLANSETTINGS-LIST-AUTHORIZATION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *plansettingspb.ListPlanSettingsRequest {
				return &plansettingspb.ListPlanSettingsRequest{}
			},
			UseTransaction: false,
			UseAuth:        false,
			ExpectSuccess:  false,
			ExpectedError:  "plan_settings.errors.authorization_failed",
			Assertions: func(t *testing.T, response *plansettingspb.ListPlanSettingsResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertAuthorizationError(t, err)
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-PLANSETTINGS-LIST-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *plansettingspb.ListPlanSettingsRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "plan_settings.validation.request_required",
			Assertions: func(t *testing.T, response *plansettingspb.ListPlanSettingsResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilRequest(t, err)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			// Set test code and log execution start
			testutil.SetTestCode(t, tc.TestCode)
			testutil.LogTestExecution(t, tc.TestCode, tc.Name, tc.ExpectSuccess)

			ctx := testutil.CreateTestContext()
			businessType := testutil.GetTestBusinessType()
			useCase := createTestListUseCaseWithAuth(businessType, tc.UseTransaction, tc.UseAuth)

			req := tc.SetupRequest(t, businessType)
			response, err := useCase.Execute(ctx, req)

			// Determine actual success/failure
			actualSuccess := err == nil && tc.ExpectSuccess

			if tc.ExpectSuccess {
				testutil.AssertNoError(t, err)
				testutil.AssertNotNil(t, response, "response")
			} else {
				testutil.AssertError(t, err)
				if tc.ExpectedError != "" {
					if tc.ErrorTags != nil {
						testutil.AssertTranslatedErrorWithTags(t, err, tc.ExpectedError, tc.ErrorTags, useCase.services.TranslationService, ctx)
					} else {
						testutil.AssertTranslatedError(t, err, tc.ExpectedError, useCase.services.TranslationService, ctx)
					}
				}
			}

			if tc.Assertions != nil {
				tc.Assertions(t, response, err, useCase, ctx)
			}

			// Log test completion with result
			testutil.LogTestResult(t, tc.TestCode, tc.Name, actualSuccess, err)
		})
	}
}
