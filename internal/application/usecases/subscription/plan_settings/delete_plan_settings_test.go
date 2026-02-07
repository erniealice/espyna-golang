//go:build mock_db && mock_auth

// Package plan_settings provides table-driven tests for the plan settings deletion use case.
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
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestDeletePlanSettingsUseCase_Execute_TableDriven
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-SUBSCRIPTION-PLANSETTINGS-DELETE-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-SUBSCRIPTION-PLANSETTINGS-DELETE-TRANSACTION-v1.0: WithTransaction
//   - ESPYNA-TEST-SUBSCRIPTION-PLANSETTINGS-DELETE-AUTHORIZATION-v1.0: Unauthorized
//   - ESPYNA-TEST-SUBSCRIPTION-PLANSETTINGS-DELETE-NIL-REQUEST-v1.0: NilRequest
//   - ESPYNA-TEST-SUBSCRIPTION-PLANSETTINGS-DELETE-NIL-DATA-v1.0: NilData
//   - ESPYNA-TEST-SUBSCRIPTION-PLANSETTINGS-DELETE-NOT-FOUND-v1.0: NotFound
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

// Type alias for delete plan settings test cases
type DeletePlanSettingsTestCase = testutil.GenericTestCase[*plansettingspb.DeletePlanSettingsRequest, *plansettingspb.DeletePlanSettingsResponse]

func createTestDeleteUseCaseWithAuth(businessType string, supportsTransaction bool, shouldAuthorize bool) *DeletePlanSettingsUseCase {
	mockRepo := subscription.NewMockPlanSettingsRepository(businessType)

	repositories := DeletePlanSettingsRepositories{
		PlanSettings: mockRepo,
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, shouldAuthorize)
	services := DeletePlanSettingsServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
	}

	return NewDeletePlanSettingsUseCase(repositories, services)
}

func TestDeletePlanSettingsUseCase_Execute_TableDriven(t *testing.T) {
	businessType := testutil.GetTestBusinessType()

	// Load test data resolvers before defining test cases
	deleteSuccessResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "plan_settings", "DeletePlanSettings_Success")
	testutil.AssertTestCaseLoad(t, err, "DeletePlanSettings_Success")

	authorizationUnauthorizedResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "plan_settings", "Authorization_Unauthorized")
	testutil.AssertTestCaseLoad(t, err, "Authorization_Unauthorized")

	deleteNotFoundResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "plan_settings", "DeletePlanSettings_NotFound")
	testutil.AssertTestCaseLoad(t, err, "DeletePlanSettings_NotFound")

	testCases := []DeletePlanSettingsTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-PLANSETTINGS-DELETE-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *plansettingspb.DeletePlanSettingsRequest {
				return &plansettingspb.DeletePlanSettingsRequest{
					Data: &plansettingspb.PlanSettings{
						Id: deleteSuccessResolver.MustGetString("existingPlanSettingsId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *plansettingspb.DeletePlanSettingsResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertTrue(t, deleteSuccessResolver.MustGetBool("expectedSuccess"), "expected success from resolver")
			},
		},
		{
			Name:     "WithTransaction",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-PLANSETTINGS-DELETE-TRANSACTION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *plansettingspb.DeletePlanSettingsRequest {
				return &plansettingspb.DeletePlanSettingsRequest{
					Data: &plansettingspb.PlanSettings{
						Id: deleteSuccessResolver.MustGetString("existingPlanSettingsId"),
					},
				}
			},
			UseTransaction: true,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *plansettingspb.DeletePlanSettingsResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
			},
		},
		{
			Name:     "Unauthorized",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-PLANSETTINGS-DELETE-AUTHORIZATION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *plansettingspb.DeletePlanSettingsRequest {
				return &plansettingspb.DeletePlanSettingsRequest{
					Data: &plansettingspb.PlanSettings{
						Id: authorizationUnauthorizedResolver.MustGetString("targetPlanSettingsId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        false,
			ExpectSuccess:  false,
			ExpectedError:  "plan_settings.errors.authorization_failed",
			Assertions: func(t *testing.T, response *plansettingspb.DeletePlanSettingsResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertAuthorizationError(t, err)
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-PLANSETTINGS-DELETE-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *plansettingspb.DeletePlanSettingsRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "plan_settings.validation.request_required",
			Assertions: func(t *testing.T, response *plansettingspb.DeletePlanSettingsResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilRequest(t, err)
			},
		},
		{
			Name:     "NilData",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-PLANSETTINGS-DELETE-NIL-DATA-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *plansettingspb.DeletePlanSettingsRequest {
				return &plansettingspb.DeletePlanSettingsRequest{Data: nil}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "plan_settings.validation.data_required",
			Assertions: func(t *testing.T, response *plansettingspb.DeletePlanSettingsResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilData(t, err)
			},
		},
		{
			Name:     "NotFound",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-PLANSETTINGS-DELETE-NOT-FOUND-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *plansettingspb.DeletePlanSettingsRequest {
				return &plansettingspb.DeletePlanSettingsRequest{
					Data: &plansettingspb.PlanSettings{
						Id: deleteNotFoundResolver.MustGetString("nonExistentPlanSettingsId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "plan_settings.errors.not_found",
			Assertions: func(t *testing.T, response *plansettingspb.DeletePlanSettingsResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "not found")
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
			useCase := createTestDeleteUseCaseWithAuth(businessType, tc.UseTransaction, tc.UseAuth)

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
