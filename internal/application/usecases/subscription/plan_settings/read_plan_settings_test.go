//go:build mock_db && mock_auth

// Package plan_settings provides table-driven tests for the plan settings reading use case.
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
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestReadPlanSettingsUseCase_Execute_TableDriven
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-SUBSCRIPTION-PLANSETTINGS-READ-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-SUBSCRIPTION-PLANSETTINGS-READ-TRANSACTION-v1.0: WithTransaction
//   - ESPYNA-TEST-SUBSCRIPTION-PLANSETTINGS-READ-AUTHORIZATION-v1.0: Unauthorized
//   - ESPYNA-TEST-SUBSCRIPTION-PLANSETTINGS-READ-NIL-REQUEST-v1.0: NilRequest
//   - ESPYNA-TEST-SUBSCRIPTION-PLANSETTINGS-READ-NIL-DATA-v1.0: NilData
//   - ESPYNA-TEST-SUBSCRIPTION-PLANSETTINGS-READ-NOT-FOUND-v1.0: NotFound
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
	"github.com/erniealice/espyna-golang/internal/application/shared/testutil"
	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock/subscription"
	plansettingspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/plan_settings"
)

// Type alias for read plan settings test cases
type ReadPlanSettingsTestCase = testutil.GenericTestCase[*plansettingspb.ReadPlanSettingsRequest, *plansettingspb.ReadPlanSettingsResponse]

func createTestReadUseCaseWithAuth(businessType string, supportsTransaction bool, shouldAuthorize bool) *ReadPlanSettingsUseCase {
	mockRepo := subscription.NewMockPlanSettingsRepository(businessType)

	repositories := ReadPlanSettingsRepositories{
		PlanSettings: mockRepo,
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, shouldAuthorize)
	services := ReadPlanSettingsServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
	}

	return NewReadPlanSettingsUseCase(repositories, services)
}

func TestReadPlanSettingsUseCase_Execute_TableDriven(t *testing.T) {
	businessType := testutil.GetTestBusinessType()

	// Load test data resolvers before defining test cases
	readSuccessResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "plan_settings", "ReadPlanSettings_Success")
	testutil.AssertTestCaseLoad(t, err, "ReadPlanSettings_Success")

	authorizationUnauthorizedResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "plan_settings", "Authorization_Unauthorized")
	testutil.AssertTestCaseLoad(t, err, "Authorization_Unauthorized")

	readNotFoundResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "plan_settings", "ReadPlanSettings_NotFound")
	testutil.AssertTestCaseLoad(t, err, "ReadPlanSettings_NotFound")

	testCases := []ReadPlanSettingsTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-PLANSETTINGS-READ-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *plansettingspb.ReadPlanSettingsRequest {
				return &plansettingspb.ReadPlanSettingsRequest{
					Data: &plansettingspb.PlanSettings{
						Id: readSuccessResolver.MustGetString("existingPlanSettingsId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *plansettingspb.ReadPlanSettingsResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertEqual(t, 1, len(response.Data), "response data length")
				readPlanSettings := response.Data[0]
				testutil.AssertStringEqual(t, readSuccessResolver.MustGetString("expectedPlanSettingsName"), readPlanSettings.Name, "plan settings name")
				testutil.AssertStringEqual(t, readSuccessResolver.MustGetString("existingPlanSettingsId"), readPlanSettings.Id, "plan settings ID")
				testutil.AssertStringEqual(t, readSuccessResolver.MustGetString("expectedPlanId"), readPlanSettings.PlanId, "plan ID")
				testutil.AssertTrue(t, readPlanSettings.Active, "plan settings active status")
				testutil.AssertFieldSet(t, readPlanSettings.DateCreated, "DateCreated")
				testutil.AssertFieldSet(t, readPlanSettings.DateCreatedString, "DateCreatedString")
			},
		},
		{
			Name:     "WithTransaction",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-PLANSETTINGS-READ-TRANSACTION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *plansettingspb.ReadPlanSettingsRequest {
				return &plansettingspb.ReadPlanSettingsRequest{
					Data: &plansettingspb.PlanSettings{
						Id: readSuccessResolver.MustGetString("existingPlanSettingsId"),
					},
				}
			},
			UseTransaction: true,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *plansettingspb.ReadPlanSettingsResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				readPlanSettings := response.Data[0]
				testutil.AssertStringEqual(t, readSuccessResolver.MustGetString("expectedPlanSettingsName"), readPlanSettings.Name, "plan settings name")
			},
		},
		{
			Name:     "Unauthorized",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-PLANSETTINGS-READ-AUTHORIZATION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *plansettingspb.ReadPlanSettingsRequest {
				return &plansettingspb.ReadPlanSettingsRequest{
					Data: &plansettingspb.PlanSettings{
						Id: authorizationUnauthorizedResolver.MustGetString("targetPlanSettingsId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        false,
			ExpectSuccess:  false,
			ExpectedError:  "plan_settings.errors.authorization_failed",
			Assertions: func(t *testing.T, response *plansettingspb.ReadPlanSettingsResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertAuthorizationError(t, err)
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-PLANSETTINGS-READ-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *plansettingspb.ReadPlanSettingsRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "plan_settings.validation.request_required",
			Assertions: func(t *testing.T, response *plansettingspb.ReadPlanSettingsResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilRequest(t, err)
			},
		},
		{
			Name:     "NilData",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-PLANSETTINGS-READ-NIL-DATA-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *plansettingspb.ReadPlanSettingsRequest {
				return &plansettingspb.ReadPlanSettingsRequest{Data: nil}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "plan_settings.validation.data_required",
			Assertions: func(t *testing.T, response *plansettingspb.ReadPlanSettingsResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilData(t, err)
			},
		},
		{
			Name:     "NotFound",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-PLANSETTINGS-READ-NOT-FOUND-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *plansettingspb.ReadPlanSettingsRequest {
				return &plansettingspb.ReadPlanSettingsRequest{
					Data: &plansettingspb.PlanSettings{
						Id: readNotFoundResolver.MustGetString("nonExistentPlanSettingsId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "plan_settings.errors.not_found",
			ErrorTags:      map[string]any{"planSettingsId": readNotFoundResolver.MustGetString("nonExistentPlanSettingsId")},
			Assertions: func(t *testing.T, response *plansettingspb.ReadPlanSettingsResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertError(t, err)
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
			useCase := createTestReadUseCaseWithAuth(businessType, tc.UseTransaction, tc.UseAuth)

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
