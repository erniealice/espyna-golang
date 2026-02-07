//go:build mock_db && mock_auth

// Package activity provides table-driven tests for the activity creation use case.
//
// The tests cover various scenarios, including success, transaction handling,
// authorization, nil requests, validation errors, data enrichment, and foreign key validation.
// Each test case is defined in a table with a specific test code, request setup,
// and assertions.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user").
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education").
//
// Usage:
//   - Run all tests: go test -tags="mock_db,mock_auth" ./...
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestCreateActivityUseCase_Execute_TableDriven
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-CREATE-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-CREATE-TRANSACTION-v1.0: WithTransaction
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-CREATE-AUTHORIZATION-v1.0: Unauthorized
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-CREATE-NIL-REQUEST-v1.0: NilRequest
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-CREATE-NIL-DATA-v1.0: NilData
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-CREATE-VALIDATION-EMPTY-NAME-v1.0: EmptyName
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-CREATE-VALIDATION-EMPTY-STAGE-ID-v1.0: EmptyStageId
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-CREATE-VALIDATION-INVALID-STAGE-ID-v1.0: InvalidStageId
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-CREATE-VALIDATION-NAME-TOO-LONG-v1.0: NameTooLong
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-CREATE-ENRICHMENT-v1.0: DataEnrichment
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-CREATE-INTEGRATION-v1.0: IntegrationTest
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-CREATE-TRANSACTION-FAILURE-v1.0: TransactionFailure
//
// Data Sources:
//   - Test cases: packages/copya/data_test/{businessType}/activity.json
//   - Mock data: packages/copya/data/{businessType}/activity.json
//   - Translations: packages/lyngua/translations/{languageCode}/{businessType}/activity.json
package activity

import (
	"context"
	"testing"

	copyatestutil "leapfor.xyz/copya/golang/testutil"
	"leapfor.xyz/espyna/internal/application/shared/testutil"
	mockDb "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock"
	"leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock/workflow"
	activitypb "leapfor.xyz/esqyma/golang/v1/domain/workflow/activity"
)

// Type alias for create activity test cases
type CreateActivityTestCase = testutil.GenericTestCase[*activitypb.CreateActivityRequest, *activitypb.CreateActivityResponse]

func createActivityTestUseCaseWithAuth(businessType string, supportsTransaction bool, shouldAuthorize bool) *CreateActivityUseCase {
	mockActivityRepo := workflow.NewMockActivityRepository(businessType)
	mockStageRepo := workflow.NewMockStageRepository(businessType)                       // For foreign key validation
	mockActivityTemplateRepo := workflow.NewMockActivityTemplateRepository(businessType) // For foreign key validation

	repositories := CreateActivityRepositories{
		Activity:         mockActivityRepo,
		Stage:            mockStageRepo,
		ActivityTemplate: mockActivityTemplateRepo,
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, shouldAuthorize)
	services := CreateActivityServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
		IDService:            standardServices.IDService,
	}

	return NewCreateActivityUseCase(repositories, services)
}

func createTestUseCaseWithFailingTransaction(businessType string) *CreateActivityUseCase {
	mockActivityRepo := workflow.NewMockActivityRepository(businessType)
	mockStageRepo := workflow.NewMockStageRepository(businessType)
	mockActivityTemplateRepo := workflow.NewMockActivityTemplateRepository(businessType)

	repositories := CreateActivityRepositories{
		Activity:         mockActivityRepo,
		Stage:            mockStageRepo,
		ActivityTemplate: mockActivityTemplateRepo,
	}

	standardServices := testutil.CreateStandardServices(false, true)
	services := CreateActivityServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   mockDb.NewFailingMockTransactionService(),
		TranslationService:   standardServices.TranslationService,
		IDService:            standardServices.IDService,
	}

	return NewCreateActivityUseCase(repositories, services)
}

func TestCreateActivityUseCase_Execute_TableDriven(t *testing.T) {
	businessType := testutil.GetTestBusinessType()

	// Load test data resolvers before defining test cases
	createSuccessResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "activity", "CreateActivity_Success")
	testutil.AssertTestCaseLoad(t, err, "CreateActivity_Success")

	authorizationUnauthorizedResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "activity", "Authorization_Unauthorized")
	testutil.AssertTestCaseLoad(t, err, "Authorization_Unauthorized")

	validationErrorEmptyNameResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "activity", "ValidationError_EmptyName")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_EmptyName")

	validationErrorInvalidStageIdResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "activity", "ValidationError_InvalidStageId")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_InvalidStageId")

	testCases := []CreateActivityTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-WORKFLOW-ACTIVITY-CREATE-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *activitypb.CreateActivityRequest {
				return &activitypb.CreateActivityRequest{
					Data: &activitypb.Activity{
						Name:               createSuccessResolver.MustGetString("newActivityName"),
						Description:        &[]string{createSuccessResolver.MustGetString("newActivityDescription")}[0],
						StageId:            createSuccessResolver.MustGetString("validStageId"),
						ActivityTemplateId: createSuccessResolver.MustGetString("validActivityTemplateId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *activitypb.CreateActivityResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertEqual(t, 1, len(response.Data), "response data length")
				createdActivity := response.Data[0]
				testutil.AssertStringEqual(t, createSuccessResolver.MustGetString("newActivityName"), createdActivity.Name, "activity name")
				testutil.AssertNonEmptyString(t, createdActivity.Id, "activity ID")
				testutil.AssertTrue(t, createdActivity.Active, "activity active status")
				testutil.AssertFieldSet(t, createdActivity.DateCreated, "DateCreated")
				testutil.AssertFieldSet(t, createdActivity.DateCreatedString, "DateCreatedString")
			},
		},
		{
			Name:     "WithTransaction",
			TestCode: "ESPYNA-TEST-WORKFLOW-ACTIVITY-CREATE-TRANSACTION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *activitypb.CreateActivityRequest {
				return &activitypb.CreateActivityRequest{
					Data: &activitypb.Activity{
						Name:               createSuccessResolver.MustGetString("newActivityName"),
						Description:        &[]string{createSuccessResolver.MustGetString("newActivityDescription")}[0],
						StageId:            createSuccessResolver.MustGetString("validStageId"),
						ActivityTemplateId: createSuccessResolver.MustGetString("validActivityTemplateId"),
					},
				}
			},
			UseTransaction: true,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *activitypb.CreateActivityResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				createdActivity := response.Data[0]
				testutil.AssertStringEqual(t, createSuccessResolver.MustGetString("newActivityName"), createdActivity.Name, "activity name")
			},
		},
		{
			Name:     "Unauthorized",
			TestCode: "ESPYNA-TEST-WORKFLOW-ACTIVITY-CREATE-AUTHORIZATION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *activitypb.CreateActivityRequest {
				return &activitypb.CreateActivityRequest{
					Data: &activitypb.Activity{
						Name:               authorizationUnauthorizedResolver.MustGetString("unauthorizedActivityName"),
						Description:        &[]string{authorizationUnauthorizedResolver.MustGetString("unauthorizedActivityDescription")}[0],
						StageId:            authorizationUnauthorizedResolver.MustGetString("unauthorizedStageId"),
						ActivityTemplateId: authorizationUnauthorizedResolver.MustGetString("unauthorizedActivityTemplateId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        false,
			ExpectSuccess:  false,
			ExpectedError:  "activity.errors.authorization_failed",
			Assertions: func(t *testing.T, response *activitypb.CreateActivityResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertAuthorizationError(t, err)
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-WORKFLOW-ACTIVITY-CREATE-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *activitypb.CreateActivityRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "activity.validation.request_required",
			Assertions: func(t *testing.T, response *activitypb.CreateActivityResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilRequest(t, err)
			},
		},
		{
			Name:     "NilData",
			TestCode: "ESPYNA-TEST-WORKFLOW-ACTIVITY-CREATE-NIL-DATA-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *activitypb.CreateActivityRequest {
				return &activitypb.CreateActivityRequest{Data: nil}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "activity.validation.data_required",
			Assertions: func(t *testing.T, response *activitypb.CreateActivityResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilData(t, err)
			},
		},
		{
			Name:     "EmptyName",
			TestCode: "ESPYNA-TEST-WORKFLOW-ACTIVITY-CREATE-VALIDATION-EMPTY-NAME-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *activitypb.CreateActivityRequest {
				return &activitypb.CreateActivityRequest{
					Data: &activitypb.Activity{
						Name:               validationErrorEmptyNameResolver.MustGetString("emptyName"),
						StageId:            validationErrorEmptyNameResolver.MustGetString("validStageId"),
						ActivityTemplateId: validationErrorEmptyNameResolver.MustGetString("validActivityTemplateId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "activity.validation.name_required",
			Assertions: func(t *testing.T, response *activitypb.CreateActivityResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty name")
			},
		},
		{
			Name:     "EmptyStageId",
			TestCode: "ESPYNA-TEST-WORKFLOW-ACTIVITY-CREATE-VALIDATION-EMPTY-STAGE-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *activitypb.CreateActivityRequest {
				return &activitypb.CreateActivityRequest{
					Data: &activitypb.Activity{
						Name:               createSuccessResolver.MustGetString("newActivityName"),
						StageId:            "",
						ActivityTemplateId: createSuccessResolver.MustGetString("validActivityTemplateId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "activity.validation.stage_id_required",
			Assertions: func(t *testing.T, response *activitypb.CreateActivityResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty stage ID")
			},
		},
		{
			Name:     "InvalidStageId",
			TestCode: "ESPYNA-TEST-WORKFLOW-ACTIVITY-CREATE-VALIDATION-INVALID-STAGE-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *activitypb.CreateActivityRequest {
				return &activitypb.CreateActivityRequest{
					Data: &activitypb.Activity{
						Name:               validationErrorInvalidStageIdResolver.MustGetString("validName"),
						StageId:            validationErrorInvalidStageIdResolver.MustGetString("invalidStageId"),
						ActivityTemplateId: validationErrorInvalidStageIdResolver.MustGetString("validActivityTemplateId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "Entity reference validation failed [DEFAULT]: Failed to validate stage entity reference [DEFAULT]: stage with ID 'stage-non-existent-123' not found",
			ExactError:     true,
			Assertions: func(t *testing.T, response *activitypb.CreateActivityResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "invalid stage ID")
			},
		},
		{
			Name:     "NameTooLong",
			TestCode: "ESPYNA-TEST-WORKFLOW-ACTIVITY-CREATE-VALIDATION-NAME-TOO-LONG-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *activitypb.CreateActivityRequest {
				resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "activity", "ValidationError_NameTooLongGenerated")
				testutil.AssertTestCaseLoad(t, err, "ValidationError_NameTooLongGenerated")
				return &activitypb.CreateActivityRequest{
					Data: &activitypb.Activity{
						Name:               resolver.MustGetString("tooLongNameGenerated"),
						StageId:            createSuccessResolver.MustGetString("validStageId"),
						ActivityTemplateId: createSuccessResolver.MustGetString("validActivityTemplateId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "activity.validation.name_too_long",
			Assertions: func(t *testing.T, response *activitypb.CreateActivityResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "name too long")
			},
		},
		{
			Name:     "DataEnrichment",
			TestCode: "ESPYNA-TEST-WORKFLOW-ACTIVITY-CREATE-ENRICHMENT-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *activitypb.CreateActivityRequest {
				return &activitypb.CreateActivityRequest{
					Data: &activitypb.Activity{
						Name:               createSuccessResolver.MustGetString("newActivityName"),
						StageId:            createSuccessResolver.MustGetString("validStageId"),
						ActivityTemplateId: createSuccessResolver.MustGetString("validActivityTemplateId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *activitypb.CreateActivityResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				createdActivity := response.Data[0]
				testutil.AssertNonEmptyString(t, createdActivity.Id, "activity ID")
				testutil.AssertTrue(t, createdActivity.Active, "activity active status")
				testutil.AssertFieldSet(t, createdActivity.DateCreated, "DateCreated")
				testutil.AssertFieldSet(t, createdActivity.DateCreatedString, "DateCreatedString")
			},
		},
		{
			Name:     "WithForeignKeyValidation",
			TestCode: "ESPYNA-TEST-WORKFLOW-ACTIVITY-CREATE-INTEGRATION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *activitypb.CreateActivityRequest {
				return &activitypb.CreateActivityRequest{
					Data: &activitypb.Activity{
						Name:               createSuccessResolver.MustGetString("newActivityName"),
						Description:        &[]string{createSuccessResolver.MustGetString("newActivityDescription")}[0],
						StageId:            createSuccessResolver.MustGetString("validStageId"),
						ActivityTemplateId: createSuccessResolver.MustGetString("validActivityTemplateId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *activitypb.CreateActivityResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				createdActivity := response.Data[0]
				testutil.AssertStringEqual(t, createSuccessResolver.MustGetString("validStageId"), createdActivity.StageId, "stage ID")
				testutil.AssertStringEqual(t, createSuccessResolver.MustGetString("validActivityTemplateId"), createdActivity.ActivityTemplateId, "activity template ID")
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
			useCase := createActivityTestUseCaseWithAuth(businessType, tc.UseTransaction, tc.UseAuth)

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
					if tc.ExactError {
						testutil.AssertStringEqual(t, tc.ExpectedError, err.Error(), "error message")
					} else if tc.ErrorTags != nil {
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

// Individual test functions for edge cases that need special setup
func TestCreateActivityUseCase_Execute_TransactionFailure(t *testing.T) {
	testCode := "ESPYNA-TEST-WORKFLOW-ACTIVITY-CREATE-TRANSACTION-FAILURE-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "TransactionFailure", false)

	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestUseCaseWithFailingTransaction(businessType)

	// Load test case using centralized TestCaseResolver
	resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "activity", "CreateActivity_Success")
	testutil.AssertTestCaseLoad(t, err, "CreateActivity_Success")

	req := &activitypb.CreateActivityRequest{
		Data: &activitypb.Activity{
			Name:               resolver.MustGetString("newActivityName"),
			Description:        &[]string{resolver.MustGetString("newActivityDescription")}[0],
			StageId:            resolver.MustGetString("validStageId"),
			ActivityTemplateId: resolver.MustGetString("validActivityTemplateId"),
		},
	}

	_, err = useCase.Execute(ctx, req)

	testutil.AssertError(t, err)

	// Log test completion with result
	testutil.LogTestResult(t, testCode, "TransactionFailure", false, err)
}
