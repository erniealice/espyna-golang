//go:build mock_db && mock_auth

// Package workflow provides table-driven tests for the workflow creation use case.
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
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestCreateWorkflowUseCase_Execute_TableDriven
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-WORKFLOW-WORKFLOW-CREATE-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-WORKFLOW-WORKFLOW-CREATE-TRANSACTION-v1.0: WithTransaction
//   - ESPYNA-TEST-WORKFLOW-WORKFLOW-CREATE-AUTHORIZATION-v1.0: Unauthorized
//   - ESPYNA-TEST-WORKFLOW-WORKFLOW-CREATE-NIL-REQUEST-v1.0: NilRequest
//   - ESPYNA-TEST-WORKFLOW-WORKFLOW-CREATE-NIL-DATA-v1.0: NilData
//   - ESPYNA-TEST-WORKFLOW-WORKFLOW-CREATE-VALIDATION-EMPTY-NAME-v1.0: EmptyName
//   - ESPYNA-TEST-WORKFLOW-WORKFLOW-CREATE-VALIDATION-NAME-TOO-SHORT-v1.0: NameTooShort
//   - ESPYNA-TEST-WORKFLOW-WORKFLOW-CREATE-VALIDATION-NAME-TOO-LONG-v1.0: NameTooLong
//   - ESPYNA-TEST-WORKFLOW-WORKFLOW-CREATE-VALIDATION-NAME-INVALID-CHARS-v1.0: NameInvalidChars
//   - ESPYNA-TEST-WORKFLOW-WORKFLOW-CREATE-VALIDATION-DESCRIPTION-TOO-LONG-v1.0: DescriptionTooLong
//   - ESPYNA-TEST-WORKFLOW-WORKFLOW-CREATE-VALIDATION-WORKSPACE-ID-TOO-SHORT-v1.0: WorkspaceIdTooShort
//   - ESPYNA-TEST-WORKFLOW-WORKFLOW-CREATE-ENRICHMENT-v1.0: DataEnrichment
//   - ESPYNA-TEST-WORKFLOW-WORKFLOW-CREATE-VALIDATION-MINIMAL-v1.0: MinimalValidData
//   - ESPYNA-TEST-WORKFLOW-WORKFLOW-CREATE-VALIDATION-MAXIMAL-v1.0: MaxValidData
//   - ESPYNA-TEST-WORKFLOW-WORKFLOW-CREATE-TRANSACTION-FAILURE-v1.0: TransactionFailure
//
// Data Sources:
//   - Test cases: packages/copya/data_test/{businessType}/workflow.json
//   - Mock data: packages/copya/data/{businessType}/workflow.json
//   - Translations: packages/lyngua/translations/{languageCode}/{businessType}/workflow.json
package workflow

import (
	"context"
	"strings"
	"testing"

	copyatestutil "leapfor.xyz/copya/golang/testutil"
	"leapfor.xyz/espyna/internal/application/shared/testutil"
	mockDb "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock"
	"leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock/workflow"
	workflowpb "leapfor.xyz/esqyma/golang/v1/domain/workflow/workflow"
)

// Type alias for create workflow test cases
type CreateWorkflowTestCase = testutil.GenericTestCase[*workflowpb.CreateWorkflowRequest, *workflowpb.CreateWorkflowResponse]

func createTestUseCaseWithAuth(businessType string, supportsTransaction bool, shouldAuthorize bool) *CreateWorkflowUseCase {
	mockRepo := workflow.NewMockWorkflowRepository(businessType)

	repositories := CreateWorkflowRepositories{
		Workflow: mockRepo,
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, shouldAuthorize)
	services := CreateWorkflowServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
		IDService:            standardServices.IDService,
	}

	return NewCreateWorkflowUseCase(repositories, services)
}

func TestCreateWorkflowUseCase_Execute_TableDriven(t *testing.T) {
	businessType := testutil.GetTestBusinessType()

	// Load test data resolvers before defining test cases
	createSuccessResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "workflow", "CreateWorkflow_Success")
	testutil.AssertTestCaseLoad(t, err, "CreateWorkflow_Success")

	authorizationUnauthorizedResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "workflow", "Authorization_Unauthorized")
	testutil.AssertTestCaseLoad(t, err, "Authorization_Unauthorized")

	validationErrorEmptyNameResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "workflow", "ValidationError_EmptyName")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_EmptyName")

	validationErrorNameTooShortResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "workflow", "ValidationError_NameTooShort")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_NameTooShort")

	testCases := []CreateWorkflowTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-WORKFLOW-WORKFLOW-CREATE-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *workflowpb.CreateWorkflowRequest {
				return &workflowpb.CreateWorkflowRequest{
					Data: &workflowpb.Workflow{
						Name:        createSuccessResolver.MustGetString("newWorkflowName"),
						Description: &[]string{createSuccessResolver.MustGetString("newWorkflowDescription")}[0],
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *workflowpb.CreateWorkflowResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertEqual(t, 1, len(response.Data), "response data length")
				createdWorkflow := response.Data[0]
				testutil.AssertStringEqual(t, createSuccessResolver.MustGetString("newWorkflowName"), createdWorkflow.Name, "workflow name")
				testutil.AssertNonEmptyString(t, createdWorkflow.Id, "workflow ID")
				testutil.AssertTrue(t, createdWorkflow.Active, "workflow active status")
				testutil.AssertFieldSet(t, createdWorkflow.DateCreated, "DateCreated")
				testutil.AssertFieldSet(t, createdWorkflow.DateCreatedString, "DateCreatedString")
			},
		},
		{
			Name:     "WithTransaction",
			TestCode: "ESPYNA-TEST-WORKFLOW-WORKFLOW-CREATE-TRANSACTION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *workflowpb.CreateWorkflowRequest {
				return &workflowpb.CreateWorkflowRequest{
					Data: &workflowpb.Workflow{
						Name:        createSuccessResolver.MustGetString("newWorkflowName"),
						Description: &[]string{createSuccessResolver.MustGetString("newWorkflowDescription")}[0],
					},
				}
			},
			UseTransaction: true,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *workflowpb.CreateWorkflowResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				createdWorkflow := response.Data[0]
				testutil.AssertStringEqual(t, createSuccessResolver.MustGetString("newWorkflowName"), createdWorkflow.Name, "workflow name")
			},
		},
		{
			Name:     "Unauthorized",
			TestCode: "ESPYNA-TEST-WORKFLOW-WORKFLOW-CREATE-AUTHORIZATION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *workflowpb.CreateWorkflowRequest {
				return &workflowpb.CreateWorkflowRequest{
					Data: &workflowpb.Workflow{
						Name:        authorizationUnauthorizedResolver.MustGetString("unauthorizedWorkflowName"),
						Description: &[]string{authorizationUnauthorizedResolver.MustGetString("unauthorizedWorkflowDescription")}[0],
					},
				}
			},
			UseTransaction: false,
			UseAuth:        false,
			ExpectSuccess:  false,
			ExpectedError:  "Authorization failed for workflows",
			Assertions: func(t *testing.T, response *workflowpb.CreateWorkflowResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertAuthorizationError(t, err)
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-WORKFLOW-WORKFLOW-CREATE-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *workflowpb.CreateWorkflowRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "Request is required for workflows",
			Assertions: func(t *testing.T, response *workflowpb.CreateWorkflowResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilRequest(t, err)
			},
		},
		{
			Name:     "NilData",
			TestCode: "ESPYNA-TEST-WORKFLOW-WORKFLOW-CREATE-NIL-DATA-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *workflowpb.CreateWorkflowRequest {
				return &workflowpb.CreateWorkflowRequest{Data: nil}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "Workflow data is required",
			Assertions: func(t *testing.T, response *workflowpb.CreateWorkflowResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilData(t, err)
			},
		},
		{
			Name:     "EmptyName",
			TestCode: "ESPYNA-TEST-WORKFLOW-WORKFLOW-CREATE-VALIDATION-EMPTY-NAME-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *workflowpb.CreateWorkflowRequest {
				return &workflowpb.CreateWorkflowRequest{
					Data: &workflowpb.Workflow{
						Name: validationErrorEmptyNameResolver.MustGetString("emptyName"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "Workflow name is required",
			Assertions: func(t *testing.T, response *workflowpb.CreateWorkflowResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty name")
			},
		},
		{
			Name:     "NameTooShort",
			TestCode: "ESPYNA-TEST-WORKFLOW-WORKFLOW-CREATE-VALIDATION-NAME-TOO-SHORT-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *workflowpb.CreateWorkflowRequest {
				return &workflowpb.CreateWorkflowRequest{
					Data: &workflowpb.Workflow{
						Name: validationErrorNameTooShortResolver.MustGetString("tooShortName"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "Workflow name must be at least 2 characters long",
			Assertions: func(t *testing.T, response *workflowpb.CreateWorkflowResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "name too short")
			},
		},
		{
			Name:     "NameTooLong",
			TestCode: "ESPYNA-TEST-WORKFLOW-WORKFLOW-CREATE-VALIDATION-NAME-TOO-LONG-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *workflowpb.CreateWorkflowRequest {
				resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "workflow", "ValidationError_NameTooLongGenerated")
				testutil.AssertTestCaseLoad(t, err, "ValidationError_NameTooLongGenerated")
				return &workflowpb.CreateWorkflowRequest{
					Data: &workflowpb.Workflow{
						Name: resolver.MustGetString("tooLongNameGenerated"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "Workflow name cannot exceed 100 characters",
			Assertions: func(t *testing.T, response *workflowpb.CreateWorkflowResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "name too long")
			},
		},
		{
			Name:     "NameInvalidChars",
			TestCode: "ESPYNA-TEST-WORKFLOW-WORKFLOW-CREATE-VALIDATION-NAME-INVALID-CHARS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *workflowpb.CreateWorkflowRequest {
				return &workflowpb.CreateWorkflowRequest{
					Data: &workflowpb.Workflow{
						Name: "Invalid@Name#With$Special%Characters",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "Workflow name contains invalid characters",
			Assertions: func(t *testing.T, response *workflowpb.CreateWorkflowResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "invalid characters")
			},
		},
		{
			Name:     "DescriptionTooLong",
			TestCode: "ESPYNA-TEST-WORKFLOW-WORKFLOW-CREATE-VALIDATION-DESCRIPTION-TOO-LONG-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *workflowpb.CreateWorkflowRequest {
				resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "workflow", "ValidationError_DescriptionTooLongGenerated")
				testutil.AssertTestCaseLoad(t, err, "ValidationError_DescriptionTooLongGenerated")
				return &workflowpb.CreateWorkflowRequest{
					Data: &workflowpb.Workflow{
						Name:        resolver.MustGetString("validName"),
						Description: &[]string{resolver.MustGetString("tooLongDescriptionGenerated")}[0],
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "Workflow description cannot exceed 1000 characters",
			Assertions: func(t *testing.T, response *workflowpb.CreateWorkflowResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "description too long")
			},
		},
		{
			Name:     "WorkspaceIdTooShort",
			TestCode: "ESPYNA-TEST-WORKFLOW-WORKFLOW-CREATE-VALIDATION-WORKSPACE-ID-TOO-SHORT-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *workflowpb.CreateWorkflowRequest {
				return &workflowpb.CreateWorkflowRequest{
					Data: &workflowpb.Workflow{
						Name:        "Valid Workflow Name",
						WorkspaceId: &[]string{"ab"}[0], // Too short
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "Workspace ID is invalid",
			Assertions: func(t *testing.T, response *workflowpb.CreateWorkflowResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "workspace ID")
			},
		},
		{
			Name:     "DataEnrichment",
			TestCode: "ESPYNA-TEST-WORKFLOW-WORKFLOW-CREATE-ENRICHMENT-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *workflowpb.CreateWorkflowRequest {
				return &workflowpb.CreateWorkflowRequest{
					Data: &workflowpb.Workflow{
						Name: "Data Enrichment Test",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *workflowpb.CreateWorkflowResponse, err error, useCase interface{}, ctx context.Context) {
				createdWorkflow := response.Data[0]
				testutil.AssertNonEmptyString(t, createdWorkflow.Id, "generated ID")
				testutil.AssertFieldSet(t, createdWorkflow.DateCreated, "DateCreated")
				testutil.AssertFieldSet(t, createdWorkflow.DateCreatedString, "DateCreatedString")
				testutil.AssertFieldSet(t, createdWorkflow.DateModified, "DateModified")
				testutil.AssertFieldSet(t, createdWorkflow.DateModifiedString, "DateModifiedString")
				testutil.AssertTrue(t, createdWorkflow.Active, "Active")
				testutil.AssertNotNil(t, createdWorkflow.Version, "Version")
				testutil.AssertEqual(t, int32(1), *createdWorkflow.Version, "default version")
			},
		},
		{
			Name:     "MinimalValidData",
			TestCode: "ESPYNA-TEST-WORKFLOW-WORKFLOW-CREATE-VALIDATION-MINIMAL-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *workflowpb.CreateWorkflowRequest {
				resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "workflow", "BoundaryTest_MinimalValid")
				testutil.AssertTestCaseLoad(t, err, "BoundaryTest_MinimalValid")
				return &workflowpb.CreateWorkflowRequest{
					Data: &workflowpb.Workflow{
						Name: resolver.MustGetString("minValidName"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *workflowpb.CreateWorkflowResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				createdWorkflow := response.Data[0]
				resolver, _ := copyatestutil.LoadTestCaseFromBusinessType(t, testutil.GetTestBusinessType(), "workflow", "BoundaryTest_MinimalValid")
				testutil.AssertStringEqual(t, resolver.MustGetString("minValidName"), createdWorkflow.Name, "workflow name")
			},
		},
		{
			Name:     "MaxValidData",
			TestCode: "ESPYNA-TEST-WORKFLOW-WORKFLOW-CREATE-VALIDATION-MAXIMAL-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *workflowpb.CreateWorkflowRequest {
				resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "workflow", "BoundaryTest_MaximalValid")
				testutil.AssertTestCaseLoad(t, err, "BoundaryTest_MaximalValid")
				return &workflowpb.CreateWorkflowRequest{
					Data: &workflowpb.Workflow{
						Name:        resolver.MustGetString("maxValidNameExact100"),
						Description: &[]string{resolver.MustGetString("maxValidDescriptionExact1000")}[0],
						WorkspaceId: &[]string{resolver.MustGetString("validWorkspaceId")}[0],
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *workflowpb.CreateWorkflowResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				createdWorkflow := response.Data[0]
				testutil.AssertEqual(t, 100, len(createdWorkflow.Name), "name length")
				if createdWorkflow.Description != nil {
					testutil.AssertEqual(t, 1000, len(*createdWorkflow.Description), "description length")
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

func TestCreateWorkflowUseCase_Execute_TransactionFailure(t *testing.T) {
	testCode := "ESPYNA-TEST-WORKFLOW-WORKFLOW-CREATE-TRANSACTION-FAILURE-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "TransactionFailure", false)

	// Create a use case with a failing transaction service
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	mockRepo := workflow.NewMockWorkflowRepository(businessType)

	repositories := CreateWorkflowRepositories{
		Workflow: mockRepo,
	}

	standardServices := testutil.CreateStandardServices(false, true)
	services := CreateWorkflowServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   mockDb.NewFailingMockTransactionService(),
		TranslationService:   standardServices.TranslationService,
		IDService:            standardServices.IDService,
	}

	useCase := NewCreateWorkflowUseCase(repositories, services)

	// Load test case using centralized TestCaseResolver
	resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "workflow", "CreateWorkflow_Success")
	testutil.AssertTestCaseLoad(t, err, "CreateWorkflow_Success")

	req := &workflowpb.CreateWorkflowRequest{
		Data: &workflowpb.Workflow{
			Name: resolver.MustGetString("newWorkflowName"),
		},
	}

	_, err = useCase.Execute(ctx, req)

	// Verify that a transaction error occurred
	testutil.AssertTransactionError(t, err)

	// Log completion of transaction failure test
	testutil.LogTestResult(t, testCode, "TransactionFailure", false, err)
}
