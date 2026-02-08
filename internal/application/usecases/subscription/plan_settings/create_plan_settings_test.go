//go:build mock_db && mock_auth

// Package plan_settings provides table-driven tests for the plan settings creation use case.
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
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestCreatePlanSettingsUseCase_Execute_TableDriven
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-SUBSCRIPTION-PLAN-SETTINGS-CREATE-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-SUBSCRIPTION-PLAN-SETTINGS-CREATE-TRANSACTION-v1.0: WithTransaction
//   - ESPYNA-TEST-SUBSCRIPTION-PLAN-SETTINGS-CREATE-AUTHORIZATION-v1.0: Unauthorized
//   - ESPYNA-TEST-SUBSCRIPTION-PLAN-SETTINGS-CREATE-NIL-REQUEST-v1.0: NilRequest
//   - ESPYNA-TEST-SUBSCRIPTION-PLAN-SETTINGS-CREATE-NIL-DATA-v1.0: NilData
//   - ESPYNA-TEST-SUBSCRIPTION-PLAN-SETTINGS-CREATE-VALIDATION-EMPTY-NAME-v1.0: EmptyName
//   - ESPYNA-TEST-SUBSCRIPTION-PLAN-SETTINGS-CREATE-VALIDATION-EMPTY-PLAN-ID-v1.0: EmptyPlanId
//   - ESPYNA-TEST-SUBSCRIPTION-PLAN-SETTINGS-CREATE-VALIDATION-INVALID-PLAN-ID-v1.0: InvalidPlanId
//   - ESPYNA-TEST-SUBSCRIPTION-PLAN-SETTINGS-CREATE-VALIDATION-NAME-TOO-SHORT-v1.0: NameTooShort
//   - ESPYNA-TEST-SUBSCRIPTION-PLAN-SETTINGS-CREATE-VALIDATION-NAME-TOO-LONG-v1.0: NameTooLong
//   - ESPYNA-TEST-SUBSCRIPTION-PLAN-SETTINGS-CREATE-VALIDATION-DESCRIPTION-TOO-LONG-v1.0: DescriptionTooLong
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

// Type alias for create plan settings test cases
type CreatePlanSettingsTestCase = testutil.GenericTestCase[*plansettingspb.CreatePlanSettingsRequest, *plansettingspb.CreatePlanSettingsResponse]

func createTestUseCaseWithAuth(businessType string, supportsTransaction bool, shouldAuthorize bool) *CreatePlanSettingsUseCase {
	mockPlanSettingsRepo := subscription.NewMockPlanSettingsRepository(businessType)
	mockPlanRepo := subscription.NewMockPlanRepository(businessType)

	repositories := CreatePlanSettingsRepositories{
		PlanSettings: mockPlanSettingsRepo,
		Plan:         mockPlanRepo,
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, shouldAuthorize)
	services := CreatePlanSettingsServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
		IDService:            standardServices.IDService,
	}

	return NewCreatePlanSettingsUseCase(repositories, services)
}

func TestCreatePlanSettingsUseCase_Execute_TableDriven(t *testing.T) {
	businessType := testutil.GetTestBusinessType()

	// Load test data resolvers before defining test cases
	createSuccessResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "plan_settings", "CreatePlanSettings_Success")
	testutil.AssertTestCaseLoad(t, err, "CreatePlanSettings_Success")

	authorizationUnauthorizedResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "plan_settings", "Authorization_Unauthorized")
	testutil.AssertTestCaseLoad(t, err, "Authorization_Unauthorized")

	validationErrorEmptyNameResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "plan_settings", "ValidationError_EmptyName")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_EmptyName")

	validationErrorEmptyPlanIdResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "plan_settings", "ValidationError_EmptyPlanId")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_EmptyPlanId")

	validationErrorInvalidPlanIdResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "plan_settings", "ValidationError_InvalidPlanId")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_InvalidPlanId")

	validationErrorNameTooShortResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "plan_settings", "ValidationError_NameTooShort")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_NameTooShort")

	validationErrorNameTooLongResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "plan_settings", "ValidationError_NameTooLong")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_NameTooLong")

	validationErrorDescriptionTooLongResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "plan_settings", "ValidationError_DescriptionTooLong")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_DescriptionTooLong")

	testCases := []CreatePlanSettingsTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-PLAN-SETTINGS-CREATE-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *plansettingspb.CreatePlanSettingsRequest {
				return &plansettingspb.CreatePlanSettingsRequest{
					Data: &plansettingspb.PlanSettings{
						Name:        createSuccessResolver.MustGetString("newPlanSettingsName"),
						Description: createSuccessResolver.MustGetString("newPlanSettingsDescription"),
						PlanId:      createSuccessResolver.MustGetString("validPlanId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *plansettingspb.CreatePlanSettingsResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertEqual(t, 1, len(response.Data), "response data length")
				createdPlanSettings := response.Data[0]
				testutil.AssertStringEqual(t, createSuccessResolver.MustGetString("newPlanSettingsName"), createdPlanSettings.Name, "plan settings name")
				testutil.AssertNonEmptyString(t, createdPlanSettings.Id, "plan settings ID")
				testutil.AssertTrue(t, createdPlanSettings.Active, "plan settings active status")
				testutil.AssertFieldSet(t, createdPlanSettings.DateCreated, "DateCreated")
				testutil.AssertFieldSet(t, createdPlanSettings.DateCreatedString, "DateCreatedString")
			},
		},
		{
			Name:     "WithTransaction",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-PLAN-SETTINGS-CREATE-TRANSACTION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *plansettingspb.CreatePlanSettingsRequest {
				return &plansettingspb.CreatePlanSettingsRequest{
					Data: &plansettingspb.PlanSettings{
						Name:        createSuccessResolver.MustGetString("newPlanSettingsName"),
						Description: createSuccessResolver.MustGetString("newPlanSettingsDescription"),
						PlanId:      createSuccessResolver.MustGetString("validPlanId"),
					},
				}
			},
			UseTransaction: true,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *plansettingspb.CreatePlanSettingsResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				createdPlanSettings := response.Data[0]
				testutil.AssertStringEqual(t, createSuccessResolver.MustGetString("newPlanSettingsName"), createdPlanSettings.Name, "plan settings name")
			},
		},
		{
			Name:     "Unauthorized",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-PLAN-SETTINGS-CREATE-AUTHORIZATION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *plansettingspb.CreatePlanSettingsRequest {
				return &plansettingspb.CreatePlanSettingsRequest{
					Data: &plansettingspb.PlanSettings{
						Name:        authorizationUnauthorizedResolver.MustGetString("unauthorizedPlanSettingsName"),
						Description: authorizationUnauthorizedResolver.MustGetString("unauthorizedPlanSettingsDescription"),
						PlanId:      authorizationUnauthorizedResolver.MustGetString("unauthorizedPlanId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        false,
			ExpectSuccess:  false,
			ExpectedError:  "plan_settings.errors.authorization_failed",
			Assertions: func(t *testing.T, response *plansettingspb.CreatePlanSettingsResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertAuthorizationError(t, err)
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-PLAN-SETTINGS-CREATE-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *plansettingspb.CreatePlanSettingsRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "plan_settings.validation.request_required",
			Assertions: func(t *testing.T, response *plansettingspb.CreatePlanSettingsResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilRequest(t, err)
			},
		},
		{
			Name:     "NilData",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-PLAN-SETTINGS-CREATE-NIL-DATA-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *plansettingspb.CreatePlanSettingsRequest {
				return &plansettingspb.CreatePlanSettingsRequest{Data: nil}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "plan_settings.validation.data_required",
			Assertions: func(t *testing.T, response *plansettingspb.CreatePlanSettingsResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilData(t, err)
			},
		},
		{
			Name:     "EmptyName",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-PLAN-SETTINGS-CREATE-VALIDATION-EMPTY-NAME-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *plansettingspb.CreatePlanSettingsRequest {
				return &plansettingspb.CreatePlanSettingsRequest{
					Data: &plansettingspb.PlanSettings{
						Name:   validationErrorEmptyNameResolver.MustGetString("emptyName"),
						PlanId: validationErrorEmptyNameResolver.MustGetString("validPlanId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "plan_settings.validation.name_required",
			Assertions: func(t *testing.T, response *plansettingspb.CreatePlanSettingsResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty name")
			},
		},
		{
			Name:     "EmptyPlanId",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-PLAN-SETTINGS-CREATE-VALIDATION-EMPTY-PLAN-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *plansettingspb.CreatePlanSettingsRequest {
				return &plansettingspb.CreatePlanSettingsRequest{
					Data: &plansettingspb.PlanSettings{
						Name:   validationErrorEmptyPlanIdResolver.MustGetString("validPlanSettingsName"),
						PlanId: validationErrorEmptyPlanIdResolver.MustGetString("emptyPlanId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "plan_settings.validation.plan_id_required",
			Assertions: func(t *testing.T, response *plansettingspb.CreatePlanSettingsResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty plan ID")
			},
		},
		{
			Name:     "InvalidPlanId",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-PLAN-SETTINGS-CREATE-VALIDATION-INVALID-PLAN-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *plansettingspb.CreatePlanSettingsRequest {
				return &plansettingspb.CreatePlanSettingsRequest{
					Data: &plansettingspb.PlanSettings{
						Name:   validationErrorInvalidPlanIdResolver.MustGetString("validPlanSettingsName"),
						PlanId: validationErrorInvalidPlanIdResolver.MustGetString("invalidPlanId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "plan_settings.errors.plan_not_found",
			Assertions: func(t *testing.T, response *plansettingspb.CreatePlanSettingsResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "invalid plan ID")
			},
		},
		{
			Name:     "NameTooShort",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-PLAN-SETTINGS-CREATE-VALIDATION-NAME-TOO-SHORT-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *plansettingspb.CreatePlanSettingsRequest {
				return &plansettingspb.CreatePlanSettingsRequest{
					Data: &plansettingspb.PlanSettings{
						Name:   validationErrorNameTooShortResolver.MustGetString("tooShortName"),
						PlanId: validationErrorNameTooShortResolver.MustGetString("validPlanId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "plan_settings.validation.name_too_short",
			Assertions: func(t *testing.T, response *plansettingspb.CreatePlanSettingsResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "name too short")
			},
		},
		{
			Name:     "NameTooLong",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-PLAN-SETTINGS-CREATE-VALIDATION-NAME-TOO-LONG-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *plansettingspb.CreatePlanSettingsRequest {
				return &plansettingspb.CreatePlanSettingsRequest{
					Data: &plansettingspb.PlanSettings{
						Name:   validationErrorNameTooLongResolver.MustGetString("tooLongName"),
						PlanId: validationErrorNameTooLongResolver.MustGetString("validPlanId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "plan_settings.validation.name_too_long",
			Assertions: func(t *testing.T, response *plansettingspb.CreatePlanSettingsResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "name too long")
			},
		},
		{
			Name:     "DescriptionTooLong",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-PLAN-SETTINGS-CREATE-VALIDATION-DESCRIPTION-TOO-LONG-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *plansettingspb.CreatePlanSettingsRequest {
				return &plansettingspb.CreatePlanSettingsRequest{
					Data: &plansettingspb.PlanSettings{
						Name:        validationErrorDescriptionTooLongResolver.MustGetString("validPlanSettingsName"),
						Description: validationErrorDescriptionTooLongResolver.MustGetString("tooLongDescription"),
						PlanId:      validationErrorDescriptionTooLongResolver.MustGetString("validPlanId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "plan_settings.validation.description_too_long",
			Assertions: func(t *testing.T, response *plansettingspb.CreatePlanSettingsResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "description too long")
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
			useCase := createTestUseCaseWithAuth(businessType, tc.UseTransaction, tc.UseAuth)

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
