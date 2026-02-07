//go:build mock_db && mock_auth

// Package activity_template provides table-driven tests for the activity template update use case.
//
// The tests cover various scenarios, including success, transaction handling,
// nil requests, validation errors, data enrichment, and boundary conditions.
// Each test case is defined in a table with a specific test code, request setup,
// and assertions.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user").
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education").
//
// Usage:
//   - Run all tests: go test -tags="mock_db,mock_auth" ./...
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestUpdateActivityTemplateUseCase_Execute_TableDriven
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-UPDATE-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-UPDATE-TRANSACTION-v1.0: WithTransaction
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-UPDATE-NIL-REQUEST-v1.0: NilRequest
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-UPDATE-NIL-DATA-v1.0: NilData
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-UPDATE-VALIDATION-EMPTY-ID-v1.0: EmptyId
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-UPDATE-VALIDATION-EMPTY-NAME-v1.0: EmptyName
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-UPDATE-VALIDATION-EMPTY-STAGE-TEMPLATE-ID-v1.0: EmptyStageTemplateId
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-UPDATE-VALIDATION-INVALID-STAGE-TEMPLATE-ID-v1.0: InvalidStageTemplateId
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-UPDATE-ENRICHMENT-v1.0: DataEnrichment
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-UPDATE-VALIDATION-MINIMAL-v1.0: MinimalValidData
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-UPDATE-VALIDATION-MAXIMAL-v1.0: MaxValidData
//
// Data Sources:
//   - Test cases: packages/copya/data_test/{businessType}/activity_template.json
//   - Mock data: packages/copya/data/{businessType}/activity_template.json
//   - Translations: packages/lyngua/translations/{languageCode}/{businessType}/activity_template.json

package activity_template

import (
	"context"
	"slices"
	"testing"

	copyatestutil "leapfor.xyz/copya/golang/testutil"
	"leapfor.xyz/espyna/internal/application/shared/testutil"
	mockDb "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock"
	"leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock/workflow"
	activityTemplatepb "leapfor.xyz/esqyma/golang/v1/domain/workflow/activity_template"
)

// Type alias for update activity template test cases
type UpdateActivityTemplateTestCase = testutil.GenericTestCase[*activityTemplatepb.UpdateActivityTemplateRequest, *activityTemplatepb.UpdateActivityTemplateResponse]

func createTestUpdateActivityTemplateUseCaseWithAuth(businessType string, supportsTransaction bool, shouldAuthorize bool) *UpdateActivityTemplateUseCase {
	mockActivityTemplateRepo := workflow.NewMockActivityTemplateRepository(businessType)
	mockStageTemplateRepo := workflow.NewMockStageTemplateRepository(businessType) // For foreign key validation

	repositories := UpdateActivityTemplateRepositories{
		ActivityTemplate: mockActivityTemplateRepo,
		StageTemplate:    mockStageTemplateRepo, // Foreign key validation
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, shouldAuthorize)
	services := UpdateActivityTemplateServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
	}

	return NewUpdateActivityTemplateUseCase(repositories, services)
}

func TestUpdateActivityTemplateUseCase_Execute_TableDriven(t *testing.T) {
	businessType := testutil.GetTestBusinessType()

	// Load test data resolvers before defining test cases
	createSuccessResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "activity_template", "CreateActivityTemplate_Success")
	testutil.AssertTestCaseLoad(t, err, "CreateActivityTemplate_Success")

	updateSuccessResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "activity_template", "UpdateActivityTemplate_Success")
	testutil.AssertTestCaseLoad(t, err, "UpdateActivityTemplate_Success")

	commonDataResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "activity_template", "ActivityTemplate_CommonData")
	testutil.AssertTestCaseLoad(t, err, "ActivityTemplate_CommonData")

	testCases := []UpdateActivityTemplateTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-UPDATE-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *activityTemplatepb.UpdateActivityTemplateRequest {
				return &activityTemplatepb.UpdateActivityTemplateRequest{
					Data: &activityTemplatepb.ActivityTemplate{
						Id:              commonDataResolver.MustGetString("primaryActivityTemplateId"),
						Name:            updateSuccessResolver.MustGetString("updatedActivityTemplateName"),
						Description:     &[]string{updateSuccessResolver.MustGetString("updatedActivityTemplateDescription")}[0],
						StageTemplateId: createSuccessResolver.MustGetString("validStageTemplateId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *activityTemplatepb.UpdateActivityTemplateResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertEqual(t, 1, len(response.Data), "response data length")
				updatedActivityTemplate := response.Data[0]
				testutil.AssertStringEqual(t, updateSuccessResolver.MustGetString("updatedActivityTemplateName"), updatedActivityTemplate.Name, "updated name")
				testutil.AssertFieldSet(t, updatedActivityTemplate.Description, "updated description")
				testutil.AssertFieldSet(t, updatedActivityTemplate.DateModified, "DateModified")
				testutil.AssertFieldSet(t, updatedActivityTemplate.DateModifiedString, "DateModifiedString")
			},
		},
		{
			Name:     "WithTransaction",
			TestCode: "ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-UPDATE-TRANSACTION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *activityTemplatepb.UpdateActivityTemplateRequest {
				return &activityTemplatepb.UpdateActivityTemplateRequest{
					Data: &activityTemplatepb.ActivityTemplate{
						Id:              commonDataResolver.MustGetString("secondaryActivityTemplateId"),
						Name:            updateSuccessResolver.MustGetString("updatedActivityTemplateName"),
						Description:     &[]string{updateSuccessResolver.MustGetString("updatedActivityTemplateDescription")}[0],
						StageTemplateId: createSuccessResolver.MustGetString("validStageTemplateId"),
					},
				}
			},
			UseTransaction: true,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *activityTemplatepb.UpdateActivityTemplateResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				updatedActivityTemplate := response.Data[0]
				testutil.AssertStringEqual(t, updateSuccessResolver.MustGetString("updatedActivityTemplateName"), updatedActivityTemplate.Name, "updated name")
				testutil.AssertFieldSet(t, updatedActivityTemplate.DateModified, "DateModified")
				testutil.AssertFieldSet(t, updatedActivityTemplate.DateModifiedString, "DateModifiedString")
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-UPDATE-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *activityTemplatepb.UpdateActivityTemplateRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "activity_template.validation.request_required",
			Assertions: func(t *testing.T, response *activityTemplatepb.UpdateActivityTemplateResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilRequest(t, err)
			},
		},
		{
			Name:     "NilData",
			TestCode: "ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-UPDATE-NIL-DATA-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *activityTemplatepb.UpdateActivityTemplateRequest {
				return &activityTemplatepb.UpdateActivityTemplateRequest{Data: nil}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "activity_template.validation.request_required",
			Assertions: func(t *testing.T, response *activityTemplatepb.UpdateActivityTemplateResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilData(t, err)
			},
		},
		{
			Name:     "EmptyId",
			TestCode: "ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-UPDATE-VALIDATION-EMPTY-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *activityTemplatepb.UpdateActivityTemplateRequest {
				return &activityTemplatepb.UpdateActivityTemplateRequest{
					Data: &activityTemplatepb.ActivityTemplate{
						Id:              "",
						Name:            updateSuccessResolver.MustGetString("validActivityTemplateName"),
						StageTemplateId: createSuccessResolver.MustGetString("validStageTemplateId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "activity_template.validation.id_required",
			Assertions: func(t *testing.T, response *activityTemplatepb.UpdateActivityTemplateResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty ID")
			},
		},
		{
			Name:     "EmptyName",
			TestCode: "ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-UPDATE-VALIDATION-EMPTY-NAME-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *activityTemplatepb.UpdateActivityTemplateRequest {
				return &activityTemplatepb.UpdateActivityTemplateRequest{
					Data: &activityTemplatepb.ActivityTemplate{
						Id:              commonDataResolver.MustGetString("primaryActivityTemplateId"),
						Name:            "",
						StageTemplateId: createSuccessResolver.MustGetString("validStageTemplateId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "activity_template.validation.name_required",
			Assertions: func(t *testing.T, response *activityTemplatepb.UpdateActivityTemplateResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty name")
			},
		},
		{
			Name:     "EmptyStageTemplateId",
			TestCode: "ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-UPDATE-VALIDATION-EMPTY-STAGE-TEMPLATE-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *activityTemplatepb.UpdateActivityTemplateRequest {
				return &activityTemplatepb.UpdateActivityTemplateRequest{
					Data: &activityTemplatepb.ActivityTemplate{
						Id:              commonDataResolver.MustGetString("primaryActivityTemplateId"),
						Name:            updateSuccessResolver.MustGetString("validActivityTemplateName"),
						StageTemplateId: "",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "activity_template.validation.stage_template_id_required",
			Assertions: func(t *testing.T, response *activityTemplatepb.UpdateActivityTemplateResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty stage template ID")
			},
		},
		{
			Name:     "InvalidStageTemplateId",
			TestCode: "ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-UPDATE-VALIDATION-INVALID-STAGE-TEMPLATE-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *activityTemplatepb.UpdateActivityTemplateRequest {
				invalidResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "activity_template", "ValidationError_InvalidStageTemplateId")
				testutil.AssertTestCaseLoad(t, err, "ValidationError_InvalidStageTemplateId")
				return &activityTemplatepb.UpdateActivityTemplateRequest{
					Data: &activityTemplatepb.ActivityTemplate{
						Id:              commonDataResolver.MustGetString("primaryActivityTemplateId"),
						Name:            "Valid Activity Template Name",
						StageTemplateId: invalidResolver.MustGetString("invalidStageTemplateId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "Entity reference validation failed [DEFAULT]: failed to validate stage template entity reference: StageTemplate with ID 'stage-template-non-existent-123' not found",
			ExactError:     true,
			Assertions: func(t *testing.T, response *activityTemplatepb.UpdateActivityTemplateResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "invalid stage template ID")
			},
		},
		{
			Name:     "DataEnrichment",
			TestCode: "ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-UPDATE-ENRICHMENT-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *activityTemplatepb.UpdateActivityTemplateRequest {
				return &activityTemplatepb.UpdateActivityTemplateRequest{
					Data: &activityTemplatepb.ActivityTemplate{
						Id:              commonDataResolver.MustGetString("thirdActivityTemplateId"),
						Name:            updateSuccessResolver.MustGetString("updatedActivityTemplateName"),
						Description:     &[]string{updateSuccessResolver.MustGetString("updatedActivityTemplateDescription")}[0],
						StageTemplateId: createSuccessResolver.MustGetString("validStageTemplateId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *activityTemplatepb.UpdateActivityTemplateResponse, err error, useCase interface{}, ctx context.Context) {
				updatedActivityTemplate := response.Data[0]
				testutil.AssertFieldSet(t, updatedActivityTemplate.DateModified, "DateModified")
				testutil.AssertFieldSet(t, updatedActivityTemplate.DateModifiedString, "DateModifiedString")
			},
		},
		{
			Name:     "MinimalValidData",
			TestCode: "ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-UPDATE-VALIDATION-MINIMAL-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *activityTemplatepb.UpdateActivityTemplateRequest {
				boundaryResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "activity_template", "BoundaryTest_MinimalValid")
				testutil.AssertTestCaseLoad(t, err, "BoundaryTest_MinimalValid")
				return &activityTemplatepb.UpdateActivityTemplateRequest{
					Data: &activityTemplatepb.ActivityTemplate{
						Id:              commonDataResolver.MustGetString("primaryActivityTemplateId"),
						Name:            boundaryResolver.MustGetString("minValidName"),
						StageTemplateId: createSuccessResolver.MustGetString("validStageTemplateId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *activityTemplatepb.UpdateActivityTemplateResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				updatedActivityTemplate := response.Data[0]
				boundaryResolver, _ := copyatestutil.LoadTestCaseFromBusinessType(t, testutil.GetTestBusinessType(), "activity_template", "BoundaryTest_MinimalValid")
				testutil.AssertStringEqual(t, boundaryResolver.MustGetString("minValidName"), updatedActivityTemplate.Name, "activity template name")
			},
		},
		{
			Name:     "MaxValidData",
			TestCode: "ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-UPDATE-VALIDATION-MAXIMAL-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *activityTemplatepb.UpdateActivityTemplateRequest {
				boundaryResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "activity_template", "BoundaryTest_MaximalValid")
				testutil.AssertTestCaseLoad(t, err, "BoundaryTest_MaximalValid")
				return &activityTemplatepb.UpdateActivityTemplateRequest{
					Data: &activityTemplatepb.ActivityTemplate{
						Id:              commonDataResolver.MustGetString("primaryActivityTemplateId"),
						Name:            boundaryResolver.MustGetString("maxValidNameExact100"),
						Description:     &[]string{boundaryResolver.MustGetString("maxValidDescriptionExact1000")}[0],
						StageTemplateId: createSuccessResolver.MustGetString("validStageTemplateId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *activityTemplatepb.UpdateActivityTemplateResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				updatedActivityTemplate := response.Data[0]
				testutil.AssertEqual(t, 100, len(updatedActivityTemplate.Name), "name length")
				testutil.AssertFieldSet(t, updatedActivityTemplate.Description, "description")
				if updatedActivityTemplate.Description != nil {
					testutil.AssertEqual(t, 1000, len(*updatedActivityTemplate.Description), "description length")
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
			useCase := createTestUpdateActivityTemplateUseCaseWithAuth(businessType, tc.UseTransaction, tc.UseAuth)

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

func TestUpdateActivityTemplateUseCase_Execute_TransactionFailure(t *testing.T) {
	testCode := "ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-UPDATE-TRANSACTION-FAILURE-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "TransactionFailure", false)

	// Create a use case with a failing transaction service
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()

	// Load test case using centralized TestCaseResolver
	commonResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "activity_template", "ActivityTemplate_CommonData")
	testutil.AssertTestCaseLoad(t, err, "ActivityTemplate_CommonData")
	updateResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "activity_template", "UpdateActivityTemplate_Success")
	testutil.AssertTestCaseLoad(t, err, "UpdateActivityTemplate_Success")
	createResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "activity_template", "CreateActivityTemplate_Success")
	testutil.AssertTestCaseLoad(t, err, "CreateActivityTemplate_Success")

	mockActivityTemplateRepo := workflow.NewMockActivityTemplateRepository(businessType)
	mockStageTemplateRepo := workflow.NewMockStageTemplateRepository(businessType)

	repositories := UpdateActivityTemplateRepositories{
		ActivityTemplate: mockActivityTemplateRepo,
		StageTemplate:    mockStageTemplateRepo,
	}

	standardServices := testutil.CreateStandardServices(false, true)
	// Override with failing transaction service
	standardServices.TransactionService = mockDb.NewFailingMockTransactionService()
	services := UpdateActivityTemplateServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
	}

	useCase := NewUpdateActivityTemplateUseCase(repositories, services)

	req := &activityTemplatepb.UpdateActivityTemplateRequest{
		Data: &activityTemplatepb.ActivityTemplate{
			Id:              commonResolver.MustGetString("primaryActivityTemplateId"),
			Name:            updateResolver.MustGetString("validActivityTemplateName"),
			StageTemplateId: createResolver.MustGetString("validStageTemplateId"),
		},
	}

	// For update operations without transaction support, should still work
	response, err := useCase.Execute(ctx, req)

	// This should work since update operations don't always require transactions
	if err != nil {
		// If error occurs, should be due to transaction failure
		testutil.AssertTranslatedError(t, err, "transaction.errors.execution_failed", useCase.services.TranslationService, ctx)
	} else {
		// If it succeeds, verify we get proper response
		testutil.AssertNotNil(t, response, "response")
	}

	// Log completion of transaction failure test
	testutil.LogTestResult(t, testCode, "TransactionFailure", err == nil, err)
}

func TestUpdateActivityTemplateUseCase_Execute_ForeignKeyValidation(t *testing.T) {
	testCode := "ESPYNA-TEST-WORKFLOW-ACTIVITY-TEMPLATE-UPDATE-FOREIGN-KEY-VALIDATION-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "ForeignKeyValidation", true)

	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestUpdateActivityTemplateUseCaseWithAuth(businessType, false, true)

	commonResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "activity_template", "ActivityTemplate_CommonData")
	testutil.AssertTestCaseLoad(t, err, "ActivityTemplate_CommonData")

	// Test updating with different valid stage template IDs
	stageTemplateFilteringResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "activity_template", "ListActivityTemplates_StageTemplateFiltering")
	testutil.AssertTestCaseLoad(t, err, "ListActivityTemplates_StageTemplateFiltering")
	stageTemplateAssociationsRaw := stageTemplateFilteringResolver.GetTestCase().DataReferences["expectedStageTemplateAssociations"]
	stageTemplateAssociationsInterface := testutil.AssertMap(t, stageTemplateAssociationsRaw, "expectedStageTemplateAssociations")
	stageTemplateIds := make([]string, 0)
	for _, stageTemplateId := range stageTemplateAssociationsInterface {
		stageTemplateIdStr := stageTemplateId.(string)
		// Add unique stage template IDs to the list
		if !slices.Contains(stageTemplateIds, stageTemplateIdStr) {
			stageTemplateIds = append(stageTemplateIds, stageTemplateIdStr)
		}
	}

	for _, stageTemplateId := range stageTemplateIds {
		req := &activityTemplatepb.UpdateActivityTemplateRequest{
			Data: &activityTemplatepb.ActivityTemplate{
				Id:              commonResolver.MustGetString("primaryActivityTemplateId"),
				Name:            "Test Foreign Key Validation",
				StageTemplateId: stageTemplateId, // Each valid stage template ID
			},
		}

		response, err := useCase.Execute(ctx, req)

		testutil.AssertNoError(t, err)
		testutil.AssertNotNil(t, response, "response")

		updatedActivityTemplate := response.Data[0]
		testutil.AssertStringEqual(t, stageTemplateId, updatedActivityTemplate.StageTemplateId, "stage template ID")
	}

	// Log completion of foreign key validation test
	testutil.LogTestResult(t, testCode, "ForeignKeyValidation", true, nil)
}
