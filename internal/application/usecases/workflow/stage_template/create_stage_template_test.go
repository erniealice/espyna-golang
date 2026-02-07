//go:build mock_db && mock_auth

// Package stage_template provides table-driven tests for the stage template creation use case.
//
// The tests cover various scenarios, including success, transaction handling,
// authorization, nil requests, validation errors, data enrichment, and boundary conditions.
// Each test case is defined in a table with a specific test code, request setup,
// and assertions.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user").
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education").
//
// Usage:
//   - Run all tests: go test -tags="mock_db,mock_auth" ./...
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestCreateStageTemplateUseCase_Execute_TableDriven
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-CREATE-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-CREATE-TRANSACTION-v1.0: WithTransaction
//   - ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-CREATE-AUTHORIZATION-v1.0: Unauthorized
//   - ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-CREATE-NIL-REQUEST-v1.0: NilRequest
//   - ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-CREATE-NIL-DATA-v1.0: NilData
//   - ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-CREATE-VALIDATION-EMPTY-NAME-v1.0: EmptyName
//   - ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-CREATE-VALIDATION-EMPTY-WORKFLOW-ID-v1.0: EmptyWorkflowTemplateId
//   - ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-CREATE-VALIDATION-INVALID-WORKFLOW-ID-v1.0: InvalidWorkflowTemplateId
//   - ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-CREATE-VALIDATION-NAME-TOO-SHORT-v1.0: NameTooShort
//   - ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-CREATE-VALIDATION-NAME-TOO-LONG-v1.0: NameTooLong
//   - ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-CREATE-VALIDATION-DESCRIPTION-TOO-LONG-v1.0: DescriptionTooLong
//   - ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-CREATE-ENRICHMENT-v1.0: DataEnrichment
//   - ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-CREATE-VALIDATION-MINIMAL-v1.0: MinimalValidData
//   - ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-CREATE-VALIDATION-MAXIMAL-v1.0: MaxValidData
//   - ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-CREATE-TRANSACTION-FAILURE-v1.0: TransactionFailure
//
// Data Sources:
//   - Test cases: packages/copya/data_test/{businessType}/stage_template.json
//   - Mock data: packages/copya/data/{businessType}/stage_template.json
//   - Translations: packages/lyngua/translations/{languageCode}/{businessType}/stage_template.json
package stage_template

import (
	"context"
	"strings"
	"testing"

	copyatestutil "leapfor.xyz/copya/golang/testutil"
	"leapfor.xyz/espyna/internal/application/shared/testutil"
	mockDb "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock"
	"leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock/workflow"
	stageTemplatepb "leapfor.xyz/esqyma/golang/v1/domain/workflow/stage_template"
)

// Type alias for create stage template test cases
type CreateStageTemplateTestCase = testutil.GenericTestCase[*stageTemplatepb.CreateStageTemplateRequest, *stageTemplatepb.CreateStageTemplateResponse]

func createTestUseCaseWithAuth(businessType string, supportsTransaction bool, shouldAuthorize bool) *CreateStageTemplateUseCase {
	mockStageTemplateRepo := workflow.NewMockStageTemplateRepository(businessType)
	mockWorkflowTemplateRepo := workflow.NewMockWorkflowTemplateRepository(businessType) // For foreign key validation

	repositories := CreateStageTemplateRepositories{
		StageTemplate:    mockStageTemplateRepo,
		WorkflowTemplate: mockWorkflowTemplateRepo, // Foreign key validation
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, shouldAuthorize)
	services := CreateStageTemplateServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
		IDService:            standardServices.IDService,
	}

	return NewCreateStageTemplateUseCase(repositories, services)
}

func TestCreateStageTemplateUseCase_Execute_TableDriven(t *testing.T) {
	businessType := testutil.GetTestBusinessType()

	// Load test data resolvers before defining test cases
	createSuccessResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "stage_template", "CreateStageTemplate_Success")
	testutil.AssertTestCaseLoad(t, err, "CreateStageTemplate_Success")

	authorizationUnauthorizedResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "stage_template", "Authorization_Unauthorized")
	testutil.AssertTestCaseLoad(t, err, "Authorization_Unauthorized")

	validationErrorEmptyNameResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "stage_template", "ValidationError_EmptyName")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_EmptyName")

	validationErrorEmptyWorkflowTemplateIdResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "stage_template", "ValidationError_EmptyWorkflowTemplateId")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_EmptyWorkflowTemplateId")

	validationErrorInvalidWorkflowTemplateIdResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "stage_template", "ValidationError_InvalidWorkflowTemplateId")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_InvalidWorkflowTemplateId")

	testCases := []CreateStageTemplateTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-CREATE-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *stageTemplatepb.CreateStageTemplateRequest {
				return &stageTemplatepb.CreateStageTemplateRequest{
					Data: &stageTemplatepb.StageTemplate{
						Name:               createSuccessResolver.MustGetString("newStageTemplateName"),
						Description:        &[]string{createSuccessResolver.MustGetString("newStageTemplateDescription")}[0],
						WorkflowTemplateId: createSuccessResolver.MustGetString("validWorkflowTemplateId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *stageTemplatepb.CreateStageTemplateResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertEqual(t, 1, len(response.Data), "response data length")
				createdStageTemplate := response.Data[0]
				testutil.AssertStringEqual(t, createSuccessResolver.MustGetString("newStageTemplateName"), createdStageTemplate.Name, "stage template name")
				testutil.AssertNonEmptyString(t, createdStageTemplate.Id, "stage template ID")
				testutil.AssertTrue(t, createdStageTemplate.Active, "stage template active status")
				testutil.AssertFieldSet(t, createdStageTemplate.DateCreated, "DateCreated")
				testutil.AssertFieldSet(t, createdStageTemplate.DateCreatedString, "DateCreatedString")
			},
		},
		{
			Name:     "WithTransaction",
			TestCode: "ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-CREATE-TRANSACTION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *stageTemplatepb.CreateStageTemplateRequest {
				return &stageTemplatepb.CreateStageTemplateRequest{
					Data: &stageTemplatepb.StageTemplate{
						Name:               createSuccessResolver.MustGetString("newStageTemplateName"),
						Description:        &[]string{createSuccessResolver.MustGetString("newStageTemplateDescription")}[0],
						WorkflowTemplateId: createSuccessResolver.MustGetString("validWorkflowTemplateId"),
					},
				}
			},
			UseTransaction: true,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *stageTemplatepb.CreateStageTemplateResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				createdStageTemplate := response.Data[0]
				testutil.AssertStringEqual(t, createSuccessResolver.MustGetString("newStageTemplateName"), createdStageTemplate.Name, "stage template name")
			},
		},
		{
			Name:     "Unauthorized",
			TestCode: "ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-CREATE-AUTHORIZATION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *stageTemplatepb.CreateStageTemplateRequest {
				return &stageTemplatepb.CreateStageTemplateRequest{
					Data: &stageTemplatepb.StageTemplate{
						Name:               authorizationUnauthorizedResolver.MustGetString("unauthorizedStageTemplateName"),
						Description:        &[]string{authorizationUnauthorizedResolver.MustGetString("unauthorizedStageTemplateDescription")}[0],
						WorkflowTemplateId: authorizationUnauthorizedResolver.MustGetString("unauthorizedWorkflowTemplateId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        false,
			ExpectSuccess:  false,
			ExpectedError:  "Authorization failed for stage templates",
			Assertions: func(t *testing.T, response *stageTemplatepb.CreateStageTemplateResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertAuthorizationError(t, err)
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-CREATE-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *stageTemplatepb.CreateStageTemplateRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "Request is required for stage templates",
			Assertions: func(t *testing.T, response *stageTemplatepb.CreateStageTemplateResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilRequest(t, err)
			},
		},
		{
			Name:     "NilData",
			TestCode: "ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-CREATE-NIL-DATA-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *stageTemplatepb.CreateStageTemplateRequest {
				return &stageTemplatepb.CreateStageTemplateRequest{Data: nil}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "Stage template data is required",
			Assertions: func(t *testing.T, response *stageTemplatepb.CreateStageTemplateResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilData(t, err)
			},
		},
		{
			Name:     "EmptyName",
			TestCode: "ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-CREATE-VALIDATION-EMPTY-NAME-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *stageTemplatepb.CreateStageTemplateRequest {
				return &stageTemplatepb.CreateStageTemplateRequest{
					Data: &stageTemplatepb.StageTemplate{
						Name:               validationErrorEmptyNameResolver.MustGetString("emptyName"),
						WorkflowTemplateId: validationErrorEmptyNameResolver.MustGetString("validWorkflowTemplateId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "Stage template name is required",
			Assertions: func(t *testing.T, response *stageTemplatepb.CreateStageTemplateResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty name")
			},
		},
		{
			Name:     "EmptyWorkflowTemplateId",
			TestCode: "ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-CREATE-VALIDATION-EMPTY-WORKFLOW-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *stageTemplatepb.CreateStageTemplateRequest {
				return &stageTemplatepb.CreateStageTemplateRequest{
					Data: &stageTemplatepb.StageTemplate{
						Name:               validationErrorEmptyWorkflowTemplateIdResolver.MustGetString("validName"),
						WorkflowTemplateId: validationErrorEmptyWorkflowTemplateIdResolver.MustGetString("emptyWorkflowTemplateId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "Workflow ID is required",
			Assertions: func(t *testing.T, response *stageTemplatepb.CreateStageTemplateResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty workflow ID")
			},
		},
		{
			Name:     "InvalidWorkflowTemplateId",
			TestCode: "ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-CREATE-VALIDATION-INVALID-WORKFLOW-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *stageTemplatepb.CreateStageTemplateRequest {
				return &stageTemplatepb.CreateStageTemplateRequest{
					Data: &stageTemplatepb.StageTemplate{
						Name:               validationErrorInvalidWorkflowTemplateIdResolver.MustGetString("validName"),
						WorkflowTemplateId: validationErrorInvalidWorkflowTemplateIdResolver.MustGetString("invalidWorkflowTemplateId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "Referenced workflow does not exist",
			Assertions: func(t *testing.T, response *stageTemplatepb.CreateStageTemplateResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "invalid workflow ID")
			},
		},
		{
			Name:     "NameTooShort",
			TestCode: "ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-CREATE-VALIDATION-NAME-TOO-SHORT-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *stageTemplatepb.CreateStageTemplateRequest {
				return &stageTemplatepb.CreateStageTemplateRequest{
					Data: &stageTemplatepb.StageTemplate{
						Name:               "AB",
						WorkflowTemplateId: createSuccessResolver.MustGetString("validWorkflowTemplateId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "Stage template name must be at least 2 characters long",
			Assertions: func(t *testing.T, response *stageTemplatepb.CreateStageTemplateResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "name too short")
			},
		},
		{
			Name:     "NameTooLong",
			TestCode: "ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-CREATE-VALIDATION-NAME-TOO-LONG-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *stageTemplatepb.CreateStageTemplateRequest {
				resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "stage_template", "ValidationError_NameTooLongGenerated")
				testutil.AssertTestCaseLoad(t, err, "ValidationError_NameTooLongGenerated")
				return &stageTemplatepb.CreateStageTemplateRequest{
					Data: &stageTemplatepb.StageTemplate{
						Name:               resolver.MustGetString("tooLongNameGenerated"),
						WorkflowTemplateId: resolver.MustGetString("validWorkflowTemplateId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "Stage template name cannot exceed 100 characters",
			Assertions: func(t *testing.T, response *stageTemplatepb.CreateStageTemplateResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "name too long")
			},
		},
		{
			Name:     "DescriptionTooLong",
			TestCode: "ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-CREATE-VALIDATION-DESCRIPTION-TOO-LONG-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *stageTemplatepb.CreateStageTemplateRequest {
				resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "stage_template", "ValidationError_DescriptionTooLongGenerated")
				testutil.AssertTestCaseLoad(t, err, "ValidationError_DescriptionTooLongGenerated")
				return &stageTemplatepb.CreateStageTemplateRequest{
					Data: &stageTemplatepb.StageTemplate{
						Name:               resolver.MustGetString("validName"),
						Description:        &[]string{resolver.MustGetString("tooLongDescriptionGenerated")}[0],
						WorkflowTemplateId: resolver.MustGetString("validWorkflowTemplateId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "Stage template description cannot exceed 1000 characters",
			Assertions: func(t *testing.T, response *stageTemplatepb.CreateStageTemplateResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "description too long")
			},
		},
		{
			Name:     "DataEnrichment",
			TestCode: "ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-CREATE-ENRICHMENT-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *stageTemplatepb.CreateStageTemplateRequest {
				resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "stage_template", "CreateStageTemplate_Success")
				testutil.AssertTestCaseLoad(t, err, "CreateStageTemplate_Success")
				return &stageTemplatepb.CreateStageTemplateRequest{
					Data: &stageTemplatepb.StageTemplate{
						Name:               "Data Enrichment Test",
						WorkflowTemplateId: resolver.MustGetString("validWorkflowTemplateId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *stageTemplatepb.CreateStageTemplateResponse, err error, useCase interface{}, ctx context.Context) {
				createdStageTemplate := response.Data[0]
				testutil.AssertNonEmptyString(t, createdStageTemplate.Id, "generated ID")
				testutil.AssertFieldSet(t, createdStageTemplate.DateCreated, "DateCreated")
				testutil.AssertFieldSet(t, createdStageTemplate.DateCreatedString, "DateCreatedString")
				testutil.AssertTrue(t, createdStageTemplate.Active, "Active")
				testutil.AssertNotNil(t, createdStageTemplate.OrderIndex, "OrderIndex")
				testutil.AssertEqual(t, int32(1), *createdStageTemplate.OrderIndex, "default order index")
				testutil.AssertNotNil(t, createdStageTemplate.IsRequired, "IsRequired")
				testutil.AssertTrue(t, *createdStageTemplate.IsRequired, "default is required")
			},
		},
		{
			Name:     "MinimalValidData",
			TestCode: "ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-CREATE-VALIDATION-MINIMAL-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *stageTemplatepb.CreateStageTemplateRequest {
				resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "stage_template", "BoundaryTest_MinimalValid")
				testutil.AssertTestCaseLoad(t, err, "BoundaryTest_MinimalValid")
				return &stageTemplatepb.CreateStageTemplateRequest{
					Data: &stageTemplatepb.StageTemplate{
						Name:               resolver.MustGetString("minValidName"),
						WorkflowTemplateId: resolver.MustGetString("validWorkflowTemplateId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *stageTemplatepb.CreateStageTemplateResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				createdStageTemplate := response.Data[0]
				resolver, _ := copyatestutil.LoadTestCaseFromBusinessType(t, testutil.GetTestBusinessType(), "stage_template", "BoundaryTest_MinimalValid")
				testutil.AssertStringEqual(t, resolver.MustGetString("minValidName"), createdStageTemplate.Name, "stage template name")
			},
		},
		{
			Name:     "MaxValidData",
			TestCode: "ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-CREATE-VALIDATION-MAXIMAL-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *stageTemplatepb.CreateStageTemplateRequest {
				resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "stage_template", "BoundaryTest_MaximalValid")
				testutil.AssertTestCaseLoad(t, err, "BoundaryTest_MaximalValid")
				return &stageTemplatepb.CreateStageTemplateRequest{
					Data: &stageTemplatepb.StageTemplate{
						Name:               resolver.MustGetString("maxValidNameExact100"),
						Description:        &[]string{resolver.MustGetString("maxValidDescriptionExact1000")}[0],
						WorkflowTemplateId: resolver.MustGetString("validWorkflowTemplateId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *stageTemplatepb.CreateStageTemplateResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				createdStageTemplate := response.Data[0]
				testutil.AssertEqual(t, 100, len(createdStageTemplate.Name), "name length")
				if createdStageTemplate.Description != nil {
					testutil.AssertEqual(t, 1000, len(*createdStageTemplate.Description), "description length")
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

func TestCreateStageTemplateUseCase_Execute_TransactionFailure(t *testing.T) {
	testCode := "ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-CREATE-TRANSACTION-FAILURE-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "TransactionFailure", false)

	// Create a use case with a failing transaction service
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	mockStageTemplateRepo := workflow.NewMockStageTemplateRepository(businessType)
	mockWorkflowTemplateRepo := workflow.NewMockWorkflowTemplateRepository(businessType)

	repositories := CreateStageTemplateRepositories{
		StageTemplate:    mockStageTemplateRepo,
		WorkflowTemplate: mockWorkflowTemplateRepo,
	}

	standardServices := testutil.CreateStandardServices(false, true)
	services := CreateStageTemplateServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   mockDb.NewFailingMockTransactionService(),
		TranslationService:   standardServices.TranslationService,
		IDService:            standardServices.IDService,
	}

	useCase := NewCreateStageTemplateUseCase(repositories, services)

	// Load test case using centralized TestCaseResolver
	resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "stage_template", "CreateStageTemplate_Success")
	testutil.AssertTestCaseLoad(t, err, "CreateStageTemplate_Success")

	req := &stageTemplatepb.CreateStageTemplateRequest{
		Data: &stageTemplatepb.StageTemplate{
			Name:               resolver.MustGetString("newStageTemplateName"),
			WorkflowTemplateId: resolver.MustGetString("validWorkflowTemplateId"),
		},
	}

	_, err = useCase.Execute(ctx, req)

	// Verify that a transaction error occurred
	testutil.AssertTransactionError(t, err)

	// Log completion of transaction failure test
	testutil.LogTestResult(t, testCode, "TransactionFailure", false, err)
}
