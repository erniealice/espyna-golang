//go:build mock_db && mock_auth

// Package plan provides table-driven tests for the plan creation use case.
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
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestCreatePlanUseCase_Execute_TableDriven
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-SUBSCRIPTION-PLAN-CREATE-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-SUBSCRIPTION-PLAN-CREATE-TRANSACTION-v1.0: WithTransaction
//   - ESPYNA-TEST-SUBSCRIPTION-PLAN-CREATE-AUTHORIZATION-v1.0: Unauthorized
//   - ESPYNA-TEST-SUBSCRIPTION-PLAN-CREATE-NIL-REQUEST-v1.0: NilRequest
//   - ESPYNA-TEST-SUBSCRIPTION-PLAN-CREATE-NIL-DATA-v1.0: NilData
//   - ESPYNA-TEST-SUBSCRIPTION-PLAN-CREATE-VALIDATION-EMPTY-NAME-v1.0: EmptyName
//   - ESPYNA-TEST-SUBSCRIPTION-PLAN-CREATE-VALIDATION-NAME-TOO-SHORT-v1.0: NameTooShort
//   - ESPYNA-TEST-SUBSCRIPTION-PLAN-CREATE-VALIDATION-NAME-TOO-LONG-v1.0: NameTooLong
//   - ESPYNA-TEST-SUBSCRIPTION-PLAN-CREATE-VALIDATION-DESCRIPTION-TOO-LONG-v1.0: DescriptionTooLong
//   - ESPYNA-TEST-SUBSCRIPTION-PLAN-CREATE-ENRICHMENT-v1.0: DataEnrichment
//
// Data Sources:
//   - Test cases: packages/copya/data_test/{businessType}/plan.json
//   - Mock data: packages/copya/data/{businessType}/plan.json
//   - Translations: packages/lyngua/translations/{languageCode}/{businessType}/plan.json
package plan

import (
	"context"
	"testing"
	"time"

	copyatestutil "leapfor.xyz/copya/golang/testutil"
	"leapfor.xyz/espyna/internal/application/shared/testutil"
	"leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock/subscription"
	planpb "leapfor.xyz/esqyma/golang/v1/domain/subscription/plan"
)

// Type alias for create plan test cases
type CreatePlanTestCase = testutil.GenericTestCase[*planpb.CreatePlanRequest, *planpb.CreatePlanResponse]

// strPtr is a helper to convert string to *string for optional fields
func strPtr(s string) *string { return &s }

func createTestCreatePlanUseCase(businessType string, supportsTransaction bool, shouldAuthorize bool) *CreatePlanUseCase {
	mockRepo := subscription.NewMockPlanRepository(businessType)

	repositories := CreatePlanRepositories{
		Plan: mockRepo,
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, shouldAuthorize)
	services := CreatePlanServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
		IDService:            standardServices.IDService,
	}

	return NewCreatePlanUseCase(repositories, services)
}

func TestCreatePlanUseCase_Execute_TableDriven(t *testing.T) {
	businessType := testutil.GetTestBusinessType()

	// Load test data resolvers before defining test cases
	createSuccessResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "plan", "CreatePlan_Success")
	testutil.AssertTestCaseLoad(t, err, "CreatePlan_Success")

	validationErrorEmptyNameResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "plan", "ValidationError_EmptyName")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_EmptyName")

	validationErrorNameTooShortResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "plan", "ValidationError_NameTooShort")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_NameTooShort")

	validationErrorNameTooLongResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "plan", "ValidationError_NameTooLong")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_NameTooLong")

	validationErrorDescriptionTooLongResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "plan", "ValidationError_DescriptionTooLong")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_DescriptionTooLong")

	testCases := []CreatePlanTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-PLAN-CREATE-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *planpb.CreatePlanRequest {
				return &planpb.CreatePlanRequest{
					Data: &planpb.Plan{
						Name:        createSuccessResolver.MustGetString("newPlanName"),
						Description: strPtr(createSuccessResolver.MustGetString("newPlanDescription")),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *planpb.CreatePlanResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertEqual(t, 1, len(response.Data), "response data length")
				createdPlan := response.Data[0]
				testutil.AssertStringEqual(t, createSuccessResolver.MustGetString("newPlanName"), createdPlan.Name, "plan name")
				testutil.AssertStringEqual(t, createSuccessResolver.MustGetString("newPlanDescription"), *createdPlan.Description, "plan description")
				testutil.AssertNonEmptyString(t, *createdPlan.Id, "plan ID")
				testutil.AssertTrue(t, createdPlan.Active, "plan active status")
				testutil.AssertFieldSet(t, createdPlan.DateCreated, "DateCreated")
				testutil.AssertFieldSet(t, createdPlan.DateCreatedString, "DateCreatedString")
			},
		},
		{
			Name:     "WithTransaction",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-PLAN-CREATE-TRANSACTION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *planpb.CreatePlanRequest {
				return &planpb.CreatePlanRequest{
					Data: &planpb.Plan{
						Name:        createSuccessResolver.MustGetString("newPlanName"),
						Description: strPtr(createSuccessResolver.MustGetString("newPlanDescription")),
					},
				}
			},
			UseTransaction: true,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *planpb.CreatePlanResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertEqual(t, 1, len(response.Data), "response data length")
				createdPlan := response.Data[0]
				testutil.AssertStringEqual(t, createSuccessResolver.MustGetString("newPlanName"), createdPlan.Name, "plan name")
			},
		},
		{
			Name:     "Unauthorized",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-PLAN-CREATE-AUTHORIZATION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *planpb.CreatePlanRequest {
				return &planpb.CreatePlanRequest{
					Data: &planpb.Plan{
						Name:        createSuccessResolver.MustGetString("newPlanName"),
						Description: strPtr(createSuccessResolver.MustGetString("newPlanDescription")),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        false,
			ExpectSuccess:  false,
			ExpectedError:  "plan.errors.authorization_failed",
			Assertions: func(t *testing.T, response *planpb.CreatePlanResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertAuthorizationError(t, err)
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-PLAN-CREATE-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *planpb.CreatePlanRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "plan.validation.request_required",
			Assertions: func(t *testing.T, response *planpb.CreatePlanResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilRequest(t, err)
			},
		},
		{
			Name:     "NilData",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-PLAN-CREATE-NIL-DATA-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *planpb.CreatePlanRequest {
				return &planpb.CreatePlanRequest{Data: nil}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "plan.validation.data_required",
			Assertions: func(t *testing.T, response *planpb.CreatePlanResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "nil data")
			},
		},
		{
			Name:     "EmptyName",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-PLAN-CREATE-VALIDATION-EMPTY-NAME-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *planpb.CreatePlanRequest {
				return &planpb.CreatePlanRequest{
					Data: &planpb.Plan{
						Name:        validationErrorEmptyNameResolver.MustGetString("emptyName"),
						Description: strPtr(createSuccessResolver.MustGetString("newPlanDescription")),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "plan.validation.name_required",
			Assertions: func(t *testing.T, response *planpb.CreatePlanResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty name")
			},
		},
		{
			Name:     "NameTooShort",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-PLAN-CREATE-VALIDATION-NAME-TOO-SHORT-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *planpb.CreatePlanRequest {
				return &planpb.CreatePlanRequest{
					Data: &planpb.Plan{
						Name:        validationErrorNameTooShortResolver.MustGetString("tooShortName"),
						Description: strPtr(createSuccessResolver.MustGetString("newPlanDescription")),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "plan.validation.name_too_short",
			Assertions: func(t *testing.T, response *planpb.CreatePlanResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "name too short")
			},
		},
		{
			Name:     "NameTooLong",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-PLAN-CREATE-VALIDATION-NAME-TOO-LONG-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *planpb.CreatePlanRequest {
				return &planpb.CreatePlanRequest{
					Data: &planpb.Plan{
						Name:        validationErrorNameTooLongResolver.MustGetString("tooLongName"),
						Description: strPtr(createSuccessResolver.MustGetString("newPlanDescription")),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "plan.validation.name_too_long",
			Assertions: func(t *testing.T, response *planpb.CreatePlanResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "name too long")
			},
		},
		{
			Name:     "DescriptionTooLong",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-PLAN-CREATE-VALIDATION-DESCRIPTION-TOO-LONG-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *planpb.CreatePlanRequest {
				return &planpb.CreatePlanRequest{
					Data: &planpb.Plan{
						Name:        createSuccessResolver.MustGetString("newPlanName"),
						Description: strPtr(validationErrorDescriptionTooLongResolver.MustGetString("tooLongDescription")),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "plan.validation.description_too_long",
			Assertions: func(t *testing.T, response *planpb.CreatePlanResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "description too long")
			},
		},
		{
			Name:     "DataEnrichment",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-PLAN-CREATE-ENRICHMENT-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *planpb.CreatePlanRequest {
				return &planpb.CreatePlanRequest{
					Data: &planpb.Plan{
						Name:        createSuccessResolver.MustGetString("newPlanName"),
						Description: strPtr(createSuccessResolver.MustGetString("newPlanDescription")),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *planpb.CreatePlanResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertEqual(t, 1, len(response.Data), "response data length")
				createdPlan := response.Data[0]
				testutil.AssertNonEmptyString(t, *createdPlan.Id, "plan ID")
				testutil.AssertTrue(t, createdPlan.Active, "plan active status")
				testutil.AssertFieldSet(t, createdPlan.DateCreated, "DateCreated")
				testutil.AssertFieldSet(t, createdPlan.DateCreatedString, "DateCreatedString")
				testutil.AssertFieldSet(t, createdPlan.DateModified, "DateModified")
				testutil.AssertFieldSet(t, createdPlan.DateModifiedString, "DateModifiedString")

				// Verify DateCreated is recent (within 5 seconds)
				if createdPlan.DateCreated != nil {
					now := time.Now().UnixMilli()
					testutil.AssertTrue(t, *createdPlan.DateCreated >= now-5000 && *createdPlan.DateCreated <= now+5000, "DateCreated is recent")
				} else {
					t.Errorf("DateCreated is nil")
				}
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
			useCase := createTestCreatePlanUseCase(businessType, tc.UseTransaction, tc.UseAuth)

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
