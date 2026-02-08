//go:build mock_db && mock_auth

// Package workflow provides table-driven tests for the workflow update use case.
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
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestUpdateWorkflowUseCase_Execute_TableDriven
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-WORKFLOW-WORKFLOW-UPDATE-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-WORKFLOW-WORKFLOW-UPDATE-TRANSACTION-v1.0: WithTransaction
//   - ESPYNA-TEST-WORKFLOW-WORKFLOW-UPDATE-AUTHORIZATION-v1.0: Unauthorized
//   - ESPYNA-TEST-WORKFLOW-WORKFLOW-UPDATE-NIL-REQUEST-v1.0: NilRequest
//   - ESPYNA-TEST-WORKFLOW-WORKFLOW-UPDATE-NIL-DATA-v1.0: NilData
//   - ESPYNA-TEST-WORKFLOW-WORKFLOW-UPDATE-VALIDATION-EMPTY-ID-v1.0: EmptyId
//   - ESPYNA-TEST-WORKFLOW-WORKFLOW-UPDATE-VALIDATION-EMPTY-NAME-v1.0: EmptyName
//   - ESPYNA-TEST-WORKFLOW-WORKFLOW-UPDATE-VALIDATION-NAME-TOO-SHORT-v1.0: NameTooShort
//   - ESPYNA-TEST-WORKFLOW-WORKFLOW-UPDATE-VALIDATION-NAME-TOO-LONG-v1.0: NameTooLong
//   - ESPYNA-TEST-WORKFLOW-WORKFLOW-UPDATE-VALIDATION-NAME-INVALID-CHARS-v1.0: NameInvalidChars
//   - ESPYNA-TEST-WORKFLOW-WORKFLOW-UPDATE-VALIDATION-DESCRIPTION-TOO-LONG-v1.0: DescriptionTooLong
//   - ESPYNA-TEST-WORKFLOW-WORKFLOW-UPDATE-VALIDATION-WORKSPACE-ID-TOO-SHORT-v1.0: WorkspaceIdTooShort
//   - ESPYNA-TEST-WORKFLOW-WORKFLOW-UPDATE-ENRICHMENT-v1.0: DataEnrichment
//   - ESPYNA-TEST-WORKFLOW-WORKFLOW-UPDATE-VALIDATION-MINIMAL-v1.0: MinimalValidData
//   - ESPYNA-TEST-WORKFLOW-WORKFLOW-UPDATE-VALIDATION-MAXIMAL-v1.0: MaxValidData
//   - ESPYNA-TEST-WORKFLOW-WORKFLOW-UPDATE-NOT-FOUND-v1.0: WorkflowNotFound
//   - ESPYNA-TEST-WORKFLOW-WORKFLOW-UPDATE-TRANSACTION-FAILURE-v1.0: TransactionFailure
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
	"github.com/erniealice/espyna-golang/internal/application/shared/testutil"
	mockDb "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock"
	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock/workflow"
	workflowpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/workflow/workflow"
)

// Type alias for update workflow test cases
type UpdateWorkflowTestCase = testutil.GenericTestCase[*workflowpb.UpdateWorkflowRequest, *workflowpb.UpdateWorkflowResponse]

func createTestUpdateUseCaseWithAuth(businessType string, supportsTransaction bool, shouldAuthorize bool) *UpdateWorkflowUseCase {
	mockRepo := workflow.NewMockWorkflowRepository(businessType)

	repositories := UpdateWorkflowRepositories{
		Workflow: mockRepo,
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, shouldAuthorize)
	services := UpdateWorkflowServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
	}

	return NewUpdateWorkflowUseCase(repositories, services)
}

func TestUpdateWorkflowUseCase_Execute_TableDriven(t *testing.T) {
	businessType := testutil.GetTestBusinessType()

	// Load test data resolvers before defining test cases
	updateSuccessResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "workflow", "UpdateWorkflow_Success")
	testutil.AssertTestCaseLoad(t, err, "UpdateWorkflow_Success")

	authorizationUnauthorizedResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "workflow", "Authorization_Unauthorized")
	testutil.AssertTestCaseLoad(t, err, "Authorization_Unauthorized")

	validationErrorEmptyNameResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "workflow", "ValidationError_EmptyName")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_EmptyName")

	validationErrorNameTooShortResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "workflow", "ValidationError_NameTooShort")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_NameTooShort")

	testCases := []UpdateWorkflowTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-WORKFLOW-WORKFLOW-UPDATE-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *workflowpb.UpdateWorkflowRequest {
				return &workflowpb.UpdateWorkflowRequest{
					Data: &workflowpb.Workflow{
						Id:          updateSuccessResolver.MustGetString("existingWorkflowId"),
						Name:        updateSuccessResolver.MustGetString("updatedWorkflowName"),
						Description: &[]string{updateSuccessResolver.MustGetString("updatedWorkflowDescription")}[0],
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *workflowpb.UpdateWorkflowResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertEqual(t, 1, len(response.Data), "response data length")
				updatedWorkflow := response.Data[0]
				testutil.AssertStringEqual(t, updateSuccessResolver.MustGetString("updatedWorkflowName"), updatedWorkflow.Name, "workflow name")
				testutil.AssertEqual(t, updateSuccessResolver.MustGetString("existingWorkflowId"), updatedWorkflow.Id, "workflow ID")
				testutil.AssertFieldSet(t, updatedWorkflow.DateModified, "DateModified")
				testutil.AssertFieldSet(t, updatedWorkflow.DateModifiedString, "DateModifiedString")
			},
		},
		{
			Name:     "WithTransaction",
			TestCode: "ESPYNA-TEST-WORKFLOW-WORKFLOW-UPDATE-TRANSACTION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *workflowpb.UpdateWorkflowRequest {
				return &workflowpb.UpdateWorkflowRequest{
					Data: &workflowpb.Workflow{
						Id:          updateSuccessResolver.MustGetString("existingWorkflowId"),
						Name:        updateSuccessResolver.MustGetString("updatedWorkflowName"),
						Description: &[]string{updateSuccessResolver.MustGetString("updatedWorkflowDescription")}[0],
					},
				}
			},
			UseTransaction: true,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *workflowpb.UpdateWorkflowResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				updatedWorkflow := response.Data[0]
				testutil.AssertStringEqual(t, updateSuccessResolver.MustGetString("updatedWorkflowName"), updatedWorkflow.Name, "workflow name")
			},
		},
		{
			Name:     "Unauthorized",
			TestCode: "ESPYNA-TEST-WORKFLOW-WORKFLOW-UPDATE-AUTHORIZATION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *workflowpb.UpdateWorkflowRequest {
				return &workflowpb.UpdateWorkflowRequest{
					Data: &workflowpb.Workflow{
						Id:          authorizationUnauthorizedResolver.MustGetString("existingWorkflowId"),
						Name:        authorizationUnauthorizedResolver.MustGetString("unauthorizedWorkflowName"),
						Description: &[]string{authorizationUnauthorizedResolver.MustGetString("unauthorizedWorkflowDescription")}[0],
					},
				}
			},
			UseTransaction: false,
			UseAuth:        false,
			ExpectSuccess:  false,
			ExpectedError:  "Authorization failed for workflows",
			Assertions: func(t *testing.T, response *workflowpb.UpdateWorkflowResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertAuthorizationError(t, err)
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-WORKFLOW-WORKFLOW-UPDATE-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *workflowpb.UpdateWorkflowRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "Request is required for workflows",
			Assertions: func(t *testing.T, response *workflowpb.UpdateWorkflowResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilRequest(t, err)
			},
		},
		{
			Name:     "NilData",
			TestCode: "ESPYNA-TEST-WORKFLOW-WORKFLOW-UPDATE-NIL-DATA-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *workflowpb.UpdateWorkflowRequest {
				return &workflowpb.UpdateWorkflowRequest{Data: nil}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "Workflow data is required",
			Assertions: func(t *testing.T, response *workflowpb.UpdateWorkflowResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilData(t, err)
			},
		},
		{
			Name:     "EmptyId",
			TestCode: "ESPYNA-TEST-WORKFLOW-WORKFLOW-UPDATE-VALIDATION-EMPTY-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *workflowpb.UpdateWorkflowRequest {
				return &workflowpb.UpdateWorkflowRequest{
					Data: &workflowpb.Workflow{
						Id:   "",
						Name: "Valid Name",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "Workflow ID is required",
			Assertions: func(t *testing.T, response *workflowpb.UpdateWorkflowResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty ID")
			},
		},
		{
			Name:     "EmptyName",
			TestCode: "ESPYNA-TEST-WORKFLOW-WORKFLOW-UPDATE-VALIDATION-EMPTY-NAME-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *workflowpb.UpdateWorkflowRequest {
				return &workflowpb.UpdateWorkflowRequest{
					Data: &workflowpb.Workflow{
						Id:   updateSuccessResolver.MustGetString("existingWorkflowId"),
						Name: validationErrorEmptyNameResolver.MustGetString("emptyName"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "Workflow name is required",
			Assertions: func(t *testing.T, response *workflowpb.UpdateWorkflowResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty name")
			},
		},
		{
			Name:     "NameTooShort",
			TestCode: "ESPYNA-TEST-WORKFLOW-WORKFLOW-UPDATE-VALIDATION-NAME-TOO-SHORT-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *workflowpb.UpdateWorkflowRequest {
				return &workflowpb.UpdateWorkflowRequest{
					Data: &workflowpb.Workflow{
						Id:   updateSuccessResolver.MustGetString("existingWorkflowId"),
						Name: validationErrorNameTooShortResolver.MustGetString("tooShortName"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "Workflow name must be at least 2 characters long",
			Assertions: func(t *testing.T, response *workflowpb.UpdateWorkflowResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "name too short")
			},
		},
		{
			Name:     "NameTooLong",
			TestCode: "ESPYNA-TEST-WORKFLOW-WORKFLOW-UPDATE-VALIDATION-NAME-TOO-LONG-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *workflowpb.UpdateWorkflowRequest {
				resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "workflow", "ValidationError_NameTooLongGenerated")
				testutil.AssertTestCaseLoad(t, err, "ValidationError_NameTooLongGenerated")
				return &workflowpb.UpdateWorkflowRequest{
					Data: &workflowpb.Workflow{
						Id:   updateSuccessResolver.MustGetString("existingWorkflowId"),
						Name: resolver.MustGetString("tooLongNameGenerated"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "Workflow name cannot exceed 100 characters",
			Assertions: func(t *testing.T, response *workflowpb.UpdateWorkflowResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "name too long")
			},
		},
		{
			Name:     "NameInvalidChars",
			TestCode: "ESPYNA-TEST-WORKFLOW-WORKFLOW-UPDATE-VALIDATION-NAME-INVALID-CHARS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *workflowpb.UpdateWorkflowRequest {
				return &workflowpb.UpdateWorkflowRequest{
					Data: &workflowpb.Workflow{
						Id:   updateSuccessResolver.MustGetString("existingWorkflowId"),
						Name: "Invalid@Name#With$Special%Characters",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "Workflow name contains invalid characters",
			Assertions: func(t *testing.T, response *workflowpb.UpdateWorkflowResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "invalid characters")
			},
		},
		{
			Name:     "DescriptionTooLong",
			TestCode: "ESPYNA-TEST-WORKFLOW-WORKFLOW-UPDATE-VALIDATION-DESCRIPTION-TOO-LONG-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *workflowpb.UpdateWorkflowRequest {
				resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "workflow", "ValidationError_DescriptionTooLongGenerated")
				testutil.AssertTestCaseLoad(t, err, "ValidationError_DescriptionTooLongGenerated")
				return &workflowpb.UpdateWorkflowRequest{
					Data: &workflowpb.Workflow{
						Id:          updateSuccessResolver.MustGetString("existingWorkflowId"),
						Name:        resolver.MustGetString("validName"),
						Description: &[]string{resolver.MustGetString("tooLongDescriptionGenerated")}[0],
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "Workflow description cannot exceed 1000 characters",
			Assertions: func(t *testing.T, response *workflowpb.UpdateWorkflowResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "description too long")
			},
		},
		{
			Name:     "WorkspaceIdTooShort",
			TestCode: "ESPYNA-TEST-WORKFLOW-WORKFLOW-UPDATE-VALIDATION-WORKSPACE-ID-TOO-SHORT-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *workflowpb.UpdateWorkflowRequest {
				return &workflowpb.UpdateWorkflowRequest{
					Data: &workflowpb.Workflow{
						Id:          updateSuccessResolver.MustGetString("existingWorkflowId"),
						Name:        "Valid Workflow Name",
						WorkspaceId: &[]string{"ab"}[0], // Too short
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "Workspace ID is invalid",
			Assertions: func(t *testing.T, response *workflowpb.UpdateWorkflowResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "workspace ID")
			},
		},
		{
			Name:     "DataEnrichment",
			TestCode: "ESPYNA-TEST-WORKFLOW-WORKFLOW-UPDATE-ENRICHMENT-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *workflowpb.UpdateWorkflowRequest {
				return &workflowpb.UpdateWorkflowRequest{
					Data: &workflowpb.Workflow{
						Id:   updateSuccessResolver.MustGetString("existingWorkflowId"),
						Name: "Data Enrichment Test Update",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *workflowpb.UpdateWorkflowResponse, err error, useCase interface{}, ctx context.Context) {
				updatedWorkflow := response.Data[0]
				testutil.AssertFieldSet(t, updatedWorkflow.DateModified, "DateModified")
				testutil.AssertFieldSet(t, updatedWorkflow.DateModifiedString, "DateModifiedString")
			},
		},
		{
			Name:     "MinimalValidData",
			TestCode: "ESPYNA-TEST-WORKFLOW-WORKFLOW-UPDATE-VALIDATION-MINIMAL-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *workflowpb.UpdateWorkflowRequest {
				resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "workflow", "BoundaryTest_MinimalValid")
				testutil.AssertTestCaseLoad(t, err, "BoundaryTest_MinimalValid")
				return &workflowpb.UpdateWorkflowRequest{
					Data: &workflowpb.Workflow{
						Id:   updateSuccessResolver.MustGetString("existingWorkflowId"),
						Name: resolver.MustGetString("minValidName"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *workflowpb.UpdateWorkflowResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				updatedWorkflow := response.Data[0]
				resolver, _ := copyatestutil.LoadTestCaseFromBusinessType(t, testutil.GetTestBusinessType(), "workflow", "BoundaryTest_MinimalValid")
				testutil.AssertStringEqual(t, resolver.MustGetString("minValidName"), updatedWorkflow.Name, "workflow name")
			},
		},
		{
			Name:     "MaxValidData",
			TestCode: "ESPYNA-TEST-WORKFLOW-WORKFLOW-UPDATE-VALIDATION-MAXIMAL-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *workflowpb.UpdateWorkflowRequest {
				resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "workflow", "BoundaryTest_MaximalValid")
				testutil.AssertTestCaseLoad(t, err, "BoundaryTest_MaximalValid")
				return &workflowpb.UpdateWorkflowRequest{
					Data: &workflowpb.Workflow{
						Id:          updateSuccessResolver.MustGetString("existingWorkflowId"),
						Name:        resolver.MustGetString("maxValidNameExact100"),
						Description: &[]string{resolver.MustGetString("maxValidDescriptionExact1000")}[0],
						WorkspaceId: &[]string{resolver.MustGetString("validWorkspaceId")}[0],
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *workflowpb.UpdateWorkflowResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				updatedWorkflow := response.Data[0]
				testutil.AssertEqual(t, 100, len(updatedWorkflow.Name), "name length")
				if updatedWorkflow.Description != nil {
					testutil.AssertEqual(t, 1000, len(*updatedWorkflow.Description), "description length")
				}
			},
		},
		{
			Name:     "WorkflowNotFound",
			TestCode: "ESPYNA-TEST-WORKFLOW-WORKFLOW-UPDATE-NOT-FOUND-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *workflowpb.UpdateWorkflowRequest {
				return &workflowpb.UpdateWorkflowRequest{
					Data: &workflowpb.Workflow{
						Id:   "non-existent-workflow-id",
						Name: "Updated Name",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "Workflow not found",
			Assertions: func(t *testing.T, response *workflowpb.UpdateWorkflowResponse, err error, useCase interface{}, ctx context.Context) {
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
			useCase := createTestUpdateUseCaseWithAuth(businessType, tc.UseTransaction, tc.UseAuth)

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

func TestUpdateWorkflowUseCase_Execute_TransactionFailure(t *testing.T) {
	testCode := "ESPYNA-TEST-WORKFLOW-WORKFLOW-UPDATE-TRANSACTION-FAILURE-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "TransactionFailure", false)

	// Create a use case with a failing transaction service
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	mockRepo := workflow.NewMockWorkflowRepository(businessType)

	repositories := UpdateWorkflowRepositories{
		Workflow: mockRepo,
	}

	standardServices := testutil.CreateStandardServices(false, true)
	services := UpdateWorkflowServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   mockDb.NewFailingMockTransactionService(),
		TranslationService:   standardServices.TranslationService,
	}

	useCase := NewUpdateWorkflowUseCase(repositories, services)

	// Load test case using centralized TestCaseResolver
	resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "workflow", "UpdateWorkflow_Success")
	testutil.AssertTestCaseLoad(t, err, "UpdateWorkflow_Success")

	req := &workflowpb.UpdateWorkflowRequest{
		Data: &workflowpb.Workflow{
			Id:   resolver.MustGetString("existingWorkflowId"),
			Name: "Updated Workflow Name",
		},
	}

	_, err = useCase.Execute(ctx, req)

	// Verify that a transaction error occurred
	testutil.AssertTransactionError(t, err)

	// Log completion of transaction failure test
	testutil.LogTestResult(t, testCode, "TransactionFailure", false, err)
}
