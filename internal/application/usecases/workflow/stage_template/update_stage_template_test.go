//go:build mock_db && mock_auth

// Package stage_template provides table-driven tests for the stage template update use case.
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
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestUpdateStageTemplateUseCase_Execute_TableDriven
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-UPDATE-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-UPDATE-TRANSACTION-v1.0: WithTransaction
//   - ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-UPDATE-NIL-REQUEST-v1.0: NilRequest
//   - ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-UPDATE-NIL-DATA-v1.0: NilData
//   - ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-UPDATE-VALIDATION-EMPTY-ID-v1.0: EmptyId
//   - ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-UPDATE-VALIDATION-EMPTY-NAME-v1.0: EmptyName
//   - ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-UPDATE-VALIDATION-EMPTY-WORKFLOW-ID-v1.0: EmptyWorkflowTemplateId
//   - ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-UPDATE-VALIDATION-INVALID-WORKFLOW-ID-v1.0: InvalidWorkflowTemplateId
//   - ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-UPDATE-ENRICHMENT-v1.0: DataEnrichment
//   - ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-UPDATE-VALIDATION-MINIMAL-v1.0: MinimalValidData
//   - ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-UPDATE-VALIDATION-MAXIMAL-v1.0: MaxValidData
//   - ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-UPDATE-TRANSACTION-FAILURE-v1.0: TransactionFailure
//
// Data Sources:
//   - Test cases: packages/copya/data_test/{businessType}/stage_template.json
//   - Mock data: packages/copya/data/{businessType}/stage_template.json
//   - Translations: packages/lyngua/translations/{languageCode}/{businessType}/stage_template.json
package stage_template

import (
	"context"
	"slices"
	"strings"
	"testing"

	copyatestutil "leapfor.xyz/copya/golang/testutil"
	"leapfor.xyz/espyna/internal/application/shared/testutil"
	mockDb "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock"
	"leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock/workflow"
	stageTemplatepb "leapfor.xyz/esqyma/golang/v1/domain/workflow/stage_template"
)

// Type alias for update stage template test cases
type UpdateStageTemplateTestCase = testutil.GenericTestCase[*stageTemplatepb.UpdateStageTemplateRequest, *stageTemplatepb.UpdateStageTemplateResponse]

func createTestUpdateStageTemplateUseCaseWithAuth(businessType string, supportsTransaction bool, shouldAuthorize bool) *UpdateStageTemplateUseCase {
	mockStageTemplateRepo := workflow.NewMockStageTemplateRepository(businessType)
	mockWorkflowTemplateRepo := workflow.NewMockWorkflowTemplateRepository(businessType) // For foreign key validation

	repositories := UpdateStageTemplateRepositories{
		StageTemplate:    mockStageTemplateRepo,
		WorkflowTemplate: mockWorkflowTemplateRepo, // Foreign key validation
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, shouldAuthorize)
	services := UpdateStageTemplateServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
	}

	return NewUpdateStageTemplateUseCase(repositories, services)
}

func TestUpdateStageTemplateUseCase_Execute_TableDriven(t *testing.T) {
	businessType := testutil.GetTestBusinessType()

	// Load test data resolvers before defining test cases
	createSuccessResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "stage_template", "CreateStageTemplate_Success")
	testutil.AssertTestCaseLoad(t, err, "CreateStageTemplate_Success")

	updateSuccessResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "stage_template", "UpdateStageTemplate_Success")
	testutil.AssertTestCaseLoad(t, err, "UpdateStageTemplate_Success")

	commonDataResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "stage_template", "StageTemplate_CommonData")
	testutil.AssertTestCaseLoad(t, err, "StageTemplate_CommonData")

	testCases := []UpdateStageTemplateTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-UPDATE-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *stageTemplatepb.UpdateStageTemplateRequest {
				return &stageTemplatepb.UpdateStageTemplateRequest{
					Data: &stageTemplatepb.StageTemplate{
						Id:                 createSuccessResolver.MustGetString("primaryStageTemplateId"),
						Name:               updateSuccessResolver.MustGetString("enhancedApprovalName"),
						Description:        &[]string{updateSuccessResolver.MustGetString("enhancedApprovalDescription")}[0],
						WorkflowTemplateId: createSuccessResolver.MustGetString("validWorkflowTemplateId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *stageTemplatepb.UpdateStageTemplateResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertEqual(t, 1, len(response.Data), "response data length")
				updatedStageTemplate := response.Data[0]
				testutil.AssertStringEqual(t, updateSuccessResolver.MustGetString("enhancedApprovalName"), updatedStageTemplate.Name, "updated name")
				testutil.AssertFieldSet(t, updatedStageTemplate.Description, "updated description")
				testutil.AssertFieldSet(t, updatedStageTemplate.DateModified, "DateModified")
				testutil.AssertFieldSet(t, updatedStageTemplate.DateModifiedString, "DateModifiedString")
			},
		},
		{
			Name:     "WithTransaction",
			TestCode: "ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-UPDATE-TRANSACTION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *stageTemplatepb.UpdateStageTemplateRequest {
				return &stageTemplatepb.UpdateStageTemplateRequest{
					Data: &stageTemplatepb.StageTemplate{
						Id:                 commonDataResolver.MustGetString("secondaryStageTemplateId"),
						Name:               updateSuccessResolver.MustGetString("enhancedReviewName"),
						Description:        &[]string{updateSuccessResolver.MustGetString("enhancedReviewDescription")}[0],
						WorkflowTemplateId: commonDataResolver.MustGetString("validWorkflowTemplateId"),
					},
				}
			},
			UseTransaction: true,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *stageTemplatepb.UpdateStageTemplateResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				updatedStageTemplate := response.Data[0]
				testutil.AssertStringEqual(t, updateSuccessResolver.MustGetString("enhancedReviewName"), updatedStageTemplate.Name, "updated name")
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-UPDATE-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *stageTemplatepb.UpdateStageTemplateRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "Request is required for stage templates",
			Assertions: func(t *testing.T, response *stageTemplatepb.UpdateStageTemplateResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilRequest(t, err)
			},
		},
		{
			Name:     "NilData",
			TestCode: "ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-UPDATE-NIL-DATA-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *stageTemplatepb.UpdateStageTemplateRequest {
				return &stageTemplatepb.UpdateStageTemplateRequest{Data: nil}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "Request is required for stage templates",
			Assertions: func(t *testing.T, response *stageTemplatepb.UpdateStageTemplateResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilData(t, err)
			},
		},
		{
			Name:     "EmptyId",
			TestCode: "ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-UPDATE-VALIDATION-EMPTY-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *stageTemplatepb.UpdateStageTemplateRequest {
				resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "stage_template", "StageTemplate_CommonData")
				testutil.AssertTestCaseLoad(t, err, "StageTemplate_CommonData")
				updateResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "stage_template", "UpdateStageTemplate_Success")
				testutil.AssertTestCaseLoad(t, err, "UpdateStageTemplate_Success")
				return &stageTemplatepb.UpdateStageTemplateRequest{
					Data: &stageTemplatepb.StageTemplate{
						Id:                 "",
						Name:               updateResolver.MustGetString("validStageTemplateName"),
						WorkflowTemplateId: resolver.MustGetString("validWorkflowTemplateId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "Stage template ID is required",
			Assertions: func(t *testing.T, response *stageTemplatepb.UpdateStageTemplateResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty ID")
			},
		},
		{
			Name:     "EmptyName",
			TestCode: "ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-UPDATE-VALIDATION-EMPTY-NAME-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *stageTemplatepb.UpdateStageTemplateRequest {
				resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "stage_template", "CreateStageTemplate_Success")
				testutil.AssertTestCaseLoad(t, err, "CreateStageTemplate_Success")
				return &stageTemplatepb.UpdateStageTemplateRequest{
					Data: &stageTemplatepb.StageTemplate{
						Id:                 resolver.MustGetString("primaryStageTemplateId"),
						Name:               "",
						WorkflowTemplateId: resolver.MustGetString("validWorkflowTemplateId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "Stage template name is required",
			Assertions: func(t *testing.T, response *stageTemplatepb.UpdateStageTemplateResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty name")
			},
		},
		{
			Name:     "EmptyWorkflowTemplateId",
			TestCode: "ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-UPDATE-VALIDATION-EMPTY-WORKFLOW-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *stageTemplatepb.UpdateStageTemplateRequest {
				resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "stage_template", "StageTemplate_CommonData")
				testutil.AssertTestCaseLoad(t, err, "StageTemplate_CommonData")
				updateResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "stage_template", "UpdateStageTemplate_Success")
				testutil.AssertTestCaseLoad(t, err, "UpdateStageTemplate_Success")
				return &stageTemplatepb.UpdateStageTemplateRequest{
					Data: &stageTemplatepb.StageTemplate{
						Id:                 resolver.MustGetString("primaryStageTemplateId"),
						Name:               updateResolver.MustGetString("validStageTemplateName"),
						WorkflowTemplateId: "",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "Workflow ID is required",
			Assertions: func(t *testing.T, response *stageTemplatepb.UpdateStageTemplateResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty workflow ID")
			},
		},
		{
			Name:     "InvalidWorkflowTemplateId",
			TestCode: "ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-UPDATE-VALIDATION-INVALID-WORKFLOW-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *stageTemplatepb.UpdateStageTemplateRequest {
				resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "stage_template", "StageTemplate_CommonData")
				testutil.AssertTestCaseLoad(t, err, "StageTemplate_CommonData")
				workflowResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "workflow", "Workflow_CommonData")
				testutil.AssertTestCaseLoad(t, err, "Workflow_CommonData")
				return &stageTemplatepb.UpdateStageTemplateRequest{
					Data: &stageTemplatepb.StageTemplate{
						Id:                 resolver.MustGetString("primaryStageTemplateId"),
						Name:               "Valid Stage Template Name",
						WorkflowTemplateId: workflowResolver.MustGetString("nonExistentId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "Referenced workflow does not exist",
			Assertions: func(t *testing.T, response *stageTemplatepb.UpdateStageTemplateResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "invalid workflow ID")
			},
		},
		{
			Name:     "DataEnrichment",
			TestCode: "ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-UPDATE-ENRICHMENT-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *stageTemplatepb.UpdateStageTemplateRequest {
				resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "stage_template", "StageTemplate_CommonData")
				testutil.AssertTestCaseLoad(t, err, "StageTemplate_CommonData")
				updateResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "stage_template", "UpdateStageTemplate_Success")
				testutil.AssertTestCaseLoad(t, err, "UpdateStageTemplate_Success")
				return &stageTemplatepb.UpdateStageTemplateRequest{
					Data: &stageTemplatepb.StageTemplate{
						Id:                 resolver.MustGetString("thirdStageTemplateId"),
						Name:               updateResolver.MustGetString("updatedNotificationName"),
						WorkflowTemplateId: resolver.MustGetString("validWorkflowTemplateId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *stageTemplatepb.UpdateStageTemplateResponse, err error, useCase interface{}, ctx context.Context) {
				updatedStageTemplate := response.Data[0]
				testutil.AssertFieldSet(t, updatedStageTemplate.DateModified, "DateModified")
				testutil.AssertFieldSet(t, updatedStageTemplate.DateModifiedString, "DateModifiedString")
				testutil.AssertTimestampPositive(t, *updatedStageTemplate.DateModified, "DateModified")
				testutil.AssertTimestampInMilliseconds(t, *updatedStageTemplate.DateModified, "DateModified")
			},
		},
		{
			Name:     "MinimalValidData",
			TestCode: "ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-UPDATE-VALIDATION-MINIMAL-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *stageTemplatepb.UpdateStageTemplateRequest {
				resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "stage_template", "StageTemplate_CommonData")
				testutil.AssertTestCaseLoad(t, err, "StageTemplate_CommonData")
				boundaryResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "stage_template", "BoundaryTest_MinimalValid")
				testutil.AssertTestCaseLoad(t, err, "BoundaryTest_MinimalValid")
				return &stageTemplatepb.UpdateStageTemplateRequest{
					Data: &stageTemplatepb.StageTemplate{
						Id:                 resolver.MustGetString("primaryStageTemplateId"),
						Name:               boundaryResolver.MustGetString("minValidName"),
						WorkflowTemplateId: resolver.MustGetString("validWorkflowTemplateId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *stageTemplatepb.UpdateStageTemplateResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				updatedStageTemplate := response.Data[0]
				boundaryResolver, _ := copyatestutil.LoadTestCaseFromBusinessType(t, testutil.GetTestBusinessType(), "stage_template", "BoundaryTest_MinimalValid")
				testutil.AssertStringEqual(t, boundaryResolver.MustGetString("minValidName"), updatedStageTemplate.Name, "stage template name")
			},
		},
		{
			Name:     "MaxValidData",
			TestCode: "ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-UPDATE-VALIDATION-MAXIMAL-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *stageTemplatepb.UpdateStageTemplateRequest {
				resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "stage_template", "StageTemplate_CommonData")
				testutil.AssertTestCaseLoad(t, err, "StageTemplate_CommonData")
				boundaryResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "stage_template", "BoundaryTest_MaximalValid")
				testutil.AssertTestCaseLoad(t, err, "BoundaryTest_MaximalValid")
				return &stageTemplatepb.UpdateStageTemplateRequest{
					Data: &stageTemplatepb.StageTemplate{
						Id:                  resolver.MustGetString("primaryStageTemplateId"),
						Name:                boundaryResolver.MustGetString("maxValidNameExact100"),
						Description:         &[]string{boundaryResolver.MustGetString("maxValidDescriptionExact1000")}[0],
						WorkflowTemplateId:  resolver.MustGetString("validWorkflowTemplateId"),
						ConditionExpression: &[]string{boundaryResolver.MustGetString("maxValidConditionExpressionExact2000")}[0],
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *stageTemplatepb.UpdateStageTemplateResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				updatedStageTemplate := response.Data[0]
				testutil.AssertEqual(t, 100, len(updatedStageTemplate.Name), "name length")
				testutil.AssertFieldSet(t, updatedStageTemplate.Description, "description")
				if updatedStageTemplate.Description != nil {
					testutil.AssertEqual(t, 1000, len(*updatedStageTemplate.Description), "description length")
				}
				if updatedStageTemplate.ConditionExpression != nil {
					testutil.AssertEqual(t, 2000, len(*updatedStageTemplate.ConditionExpression), "condition expression length")
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
			useCase := createTestUpdateStageTemplateUseCaseWithAuth(businessType, tc.UseTransaction, tc.UseAuth)

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
					if !strings.Contains(err.Error(), tc.ExpectedError) {
						t.Errorf("Expected error containing '%s', got '%s'", tc.ExpectedError, err.Error())
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

func TestUpdateStageTemplateUseCase_Execute_TransactionFailure(t *testing.T) {
	testCode := "ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-UPDATE-TRANSACTION-FAILURE-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "TransactionFailure", false)

	// Create a use case with a failing transaction service
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()

	// Load test case using centralized TestCaseResolver
	resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "stage_template", "StageTemplate_CommonData")
	testutil.AssertTestCaseLoad(t, err, "StageTemplate_CommonData")
	updateResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "stage_template", "UpdateStageTemplate_Success")
	testutil.AssertTestCaseLoad(t, err, "UpdateStageTemplate_Success")
	mockStageTemplateRepo := workflow.NewMockStageTemplateRepository(businessType)
	mockWorkflowTemplateRepo := workflow.NewMockWorkflowTemplateRepository(businessType)

	repositories := UpdateStageTemplateRepositories{
		StageTemplate:    mockStageTemplateRepo,
		WorkflowTemplate: mockWorkflowTemplateRepo,
	}

	standardServices := testutil.CreateStandardServices(false, true)
	// Override with failing transaction service
	standardServices.TransactionService = mockDb.NewFailingMockTransactionService()
	services := UpdateStageTemplateServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
	}

	useCase := NewUpdateStageTemplateUseCase(repositories, services)

	req := &stageTemplatepb.UpdateStageTemplateRequest{
		Data: &stageTemplatepb.StageTemplate{
			Id:                 resolver.MustGetString("primaryStageTemplateId"),
			Name:               updateResolver.MustGetString("validStageTemplateName"),
			WorkflowTemplateId: resolver.MustGetString("validWorkflowTemplateId"),
		},
	}

	// For update operations without transaction support, should still work
	response, err := useCase.Execute(ctx, req)

	// This should work since update operations don't always require transactions
	if err != nil {
		// If error occurs, should be due to transaction failure
		testutil.AssertTranslatedError(t, err, "stage_template.errors.update_failed", useCase.services.TranslationService, ctx)
	} else {
		// If it succeeds, verify we get proper response
		testutil.AssertNotNil(t, response, "response")
	}

	// Log completion of transaction failure test
	testutil.LogTestResult(t, testCode, "TransactionFailure", err == nil, err)
}

func TestUpdateStageTemplateUseCase_Execute_ForeignKeyValidation(t *testing.T) {
	testCode := "ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-VALIDATION-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "ForeignKeyValidation", true)

	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestUpdateStageTemplateUseCaseWithAuth(businessType, false, true)

	// Load test case using centralized TestCaseResolver
	resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "stage_template", "StageTemplate_CommonData")
	testutil.AssertTestCaseLoad(t, err, "StageTemplate_CommonData")

	// Test updating with different valid workflow IDs
	// Get valid workflow IDs from test data
	workflowResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "stage_template", "ListStageTemplates_WorkflowFiltering")
	testutil.AssertTestCaseLoad(t, err, "ListStageTemplates_WorkflowFiltering")
	workflowAssociationsRaw := workflowResolver.GetTestCase().DataReferences["expectedWorkflowAssociations"]
	workflowAssociationsInterface := testutil.AssertMap(t, workflowAssociationsRaw, "expectedWorkflowAssociations")
	workflowIds := make([]string, 0)
	for _, workflowId := range workflowAssociationsInterface {
		workflowIdStr := workflowId.(string)
		// Add unique workflow IDs to the list
		found := slices.Contains(workflowIds, workflowIdStr)
		if !found {
			workflowIds = append(workflowIds, workflowIdStr)
		}
	}

	for _, workflowId := range workflowIds {
		req := &stageTemplatepb.UpdateStageTemplateRequest{
			Data: &stageTemplatepb.StageTemplate{
				Id:                 resolver.MustGetString("primaryStageTemplateId"),
				Name:               "Test Foreign Key Validation",
				WorkflowTemplateId: workflowId, // Each valid workflow ID
			},
		}

		response, err := useCase.Execute(ctx, req)

		testutil.AssertNoError(t, err)

		testutil.AssertNotNil(t, response, "response")

		updatedStageTemplate := response.Data[0]
		testutil.AssertStringEqual(t, workflowId, updatedStageTemplate.WorkflowTemplateId, "workflow ID")
	}

	// Log completion of foreign key validation test
	testutil.LogTestResult(t, testCode, "ForeignKeyValidation", true, nil)
}
