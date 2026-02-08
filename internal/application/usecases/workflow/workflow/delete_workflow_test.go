//go:build mock_db && mock_auth

// Package workflow provides table-driven tests for the workflow deletion use case.
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
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestDeleteWorkflowUseCase_Execute_TableDriven
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-WORKFLOW-WORKFLOW-DELETE-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-WORKFLOW-WORKFLOW-DELETE-TRANSACTION-v1.0: WithTransaction
//   - ESPYNA-TEST-WORKFLOW-WORKFLOW-DELETE-AUTHORIZATION-v1.0: Unauthorized
//   - ESPYNA-TEST-WORKFLOW-WORKFLOW-DELETE-NIL-REQUEST-v1.0: NilRequest
//   - ESPYNA-TEST-WORKFLOW-WORKFLOW-DELETE-NIL-DATA-v1.0: NilData
//   - ESPYNA-TEST-WORKFLOW-WORKFLOW-DELETE-VALIDATION-EMPTY-ID-v1.0: EmptyId
//   - ESPYNA-TEST-WORKFLOW-WORKFLOW-DELETE-VALIDATION-ID-TOO-SHORT-v1.0: IdTooShort
//   - ESPYNA-TEST-WORKFLOW-WORKFLOW-DELETE-VALIDATION-ID-TOO-LONG-v1.0: IdTooLong
//   - ESPYNA-TEST-WORKFLOW-WORKFLOW-DELETE-VALIDATION-ID-INVALID-CHARS-v1.0: IdInvalidChars
//   - ESPYNA-TEST-WORKFLOW-WORKFLOW-DELETE-VALIDATION-WHITESPACE-ID-v1.0: WhitespaceId
//   - ESPYNA-TEST-WORKFLOW-WORKFLOW-DELETE-VALIDATION-MINIMAL-ID-v1.0: ValidMinimalId
//   - ESPYNA-TEST-WORKFLOW-WORKFLOW-DELETE-NOT-FOUND-v1.0: WorkflowNotFound
//   - ESPYNA-TEST-WORKFLOW-WORKFLOW-DELETE-TRANSACTION-FAILURE-v1.0: TransactionFailure
//
// Data Sources:
//   - Test cases: packages/copya/data_test/{businessType}/workflow.json
//   - Mock data: packages/copya/data/{businessType}/workflow.json
//   - Translations: packages/lyngua/translations/{languageCode}/{businessType}/workflow.json
package workflow

import (
	"context"
	"testing"

	copyatestutil "leapfor.xyz/copya/golang/testutil"
	"github.com/erniealice/espyna-golang/internal/application/shared/testutil"
	mockDb "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock"
	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock/workflow"
	workflowpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/workflow/workflow"
)

// Type alias for delete workflow test cases
type DeleteWorkflowTestCase = testutil.GenericTestCase[*workflowpb.DeleteWorkflowRequest, *workflowpb.DeleteWorkflowResponse]

func createTestDeleteUseCaseWithAuth(businessType string, supportsTransaction bool, shouldAuthorize bool) *DeleteWorkflowUseCase {
	mockRepo := workflow.NewMockWorkflowRepository(businessType)

	repositories := DeleteWorkflowRepositories{
		Workflow: mockRepo,
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, shouldAuthorize)
	services := DeleteWorkflowServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
	}

	return NewDeleteWorkflowUseCase(repositories, services)
}

func TestDeleteWorkflowUseCase_Execute_TableDriven(t *testing.T) {
	businessType := testutil.GetTestBusinessType()

	// Load test data resolvers before defining test cases
	deleteSuccessResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "workflow", "DeleteWorkflow_Success")
	testutil.AssertTestCaseLoad(t, err, "DeleteWorkflow_Success")

	authorizationUnauthorizedResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "workflow", "Authorization_Unauthorized")
	testutil.AssertTestCaseLoad(t, err, "Authorization_Unauthorized")

	testCases := []DeleteWorkflowTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-WORKFLOW-WORKFLOW-DELETE-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *workflowpb.DeleteWorkflowRequest {
				return &workflowpb.DeleteWorkflowRequest{
					Data: &workflowpb.Workflow{
						Id: deleteSuccessResolver.MustGetString("deletableWorkflowId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *workflowpb.DeleteWorkflowResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				// Delete operations don't return data, only success status
			},
		},
		{
			Name:     "WithTransaction",
			TestCode: "ESPYNA-TEST-WORKFLOW-WORKFLOW-DELETE-TRANSACTION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *workflowpb.DeleteWorkflowRequest {
				return &workflowpb.DeleteWorkflowRequest{
					Data: &workflowpb.Workflow{
						Id: deleteSuccessResolver.MustGetString("anotherDeletableWorkflowId"),
					},
				}
			},
			UseTransaction: true,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *workflowpb.DeleteWorkflowResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				// Delete operations don't return data, only success status
			},
		},
		{
			Name:     "Unauthorized",
			TestCode: "ESPYNA-TEST-WORKFLOW-WORKFLOW-DELETE-AUTHORIZATION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *workflowpb.DeleteWorkflowRequest {
				return &workflowpb.DeleteWorkflowRequest{
					Data: &workflowpb.Workflow{
						Id: authorizationUnauthorizedResolver.MustGetString("existingWorkflowId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        false,
			ExpectSuccess:  false,
			ExpectedError:  "Authorization failed for workflows",
			Assertions: func(t *testing.T, response *workflowpb.DeleteWorkflowResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertAuthorizationError(t, err)
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-WORKFLOW-WORKFLOW-DELETE-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *workflowpb.DeleteWorkflowRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "Request is required for workflows",
			Assertions: func(t *testing.T, response *workflowpb.DeleteWorkflowResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilRequest(t, err)
			},
		},
		{
			Name:     "NilData",
			TestCode: "ESPYNA-TEST-WORKFLOW-WORKFLOW-DELETE-NIL-DATA-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *workflowpb.DeleteWorkflowRequest {
				return &workflowpb.DeleteWorkflowRequest{Data: nil}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "Workflow data is required",
			Assertions: func(t *testing.T, response *workflowpb.DeleteWorkflowResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilData(t, err)
			},
		},
		{
			Name:     "EmptyId",
			TestCode: "ESPYNA-TEST-WORKFLOW-WORKFLOW-DELETE-VALIDATION-EMPTY-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *workflowpb.DeleteWorkflowRequest {
				return &workflowpb.DeleteWorkflowRequest{
					Data: &workflowpb.Workflow{
						Id: "",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "Workflow ID is required",
			Assertions: func(t *testing.T, response *workflowpb.DeleteWorkflowResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty ID")
			},
		},
		{
			Name:     "IdTooShort",
			TestCode: "ESPYNA-TEST-WORKFLOW-WORKFLOW-DELETE-VALIDATION-ID-TOO-SHORT-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *workflowpb.DeleteWorkflowRequest {
				return &workflowpb.DeleteWorkflowRequest{
					Data: &workflowpb.Workflow{
						Id: "ab", // Too short - keeping hardcoded for validation test consistency
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "Workflow ID format is invalid",
			Assertions: func(t *testing.T, response *workflowpb.DeleteWorkflowResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "ID too short")
			},
		},
		{
			Name:     "IdTooLong",
			TestCode: "ESPYNA-TEST-WORKFLOW-WORKFLOW-DELETE-VALIDATION-ID-TOO-LONG-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *workflowpb.DeleteWorkflowRequest {
				resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "workflow", "ValidationError_IdTooLongGenerated")
				testutil.AssertTestCaseLoad(t, err, "ValidationError_IdTooLongGenerated")
				return &workflowpb.DeleteWorkflowRequest{
					Data: &workflowpb.Workflow{
						Id: resolver.MustGetString("tooLongIdGenerated"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "Workflow ID format is invalid",
			Assertions: func(t *testing.T, response *workflowpb.DeleteWorkflowResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "ID too long")
			},
		},
		{
			Name:     "IdInvalidChars",
			TestCode: "ESPYNA-TEST-WORKFLOW-WORKFLOW-DELETE-VALIDATION-ID-INVALID-CHARS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *workflowpb.DeleteWorkflowRequest {
				return &workflowpb.DeleteWorkflowRequest{
					Data: &workflowpb.Workflow{
						Id: "workflow@123#invalid", // Invalid characters
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "Workflow ID format is invalid",
			Assertions: func(t *testing.T, response *workflowpb.DeleteWorkflowResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "invalid characters")
			},
		},
		{
			Name:     "WhitespaceId",
			TestCode: "ESPYNA-TEST-WORKFLOW-WORKFLOW-DELETE-VALIDATION-WHITESPACE-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *workflowpb.DeleteWorkflowRequest {
				return &workflowpb.DeleteWorkflowRequest{
					Data: &workflowpb.Workflow{
						Id: "   ", // Whitespace only
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "Workflow ID format is invalid",
			Assertions: func(t *testing.T, response *workflowpb.DeleteWorkflowResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "whitespace only")
			},
		},
		{
			Name:     "ValidMinimalId",
			TestCode: "ESPYNA-TEST-WORKFLOW-WORKFLOW-DELETE-VALIDATION-MINIMAL-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *workflowpb.DeleteWorkflowRequest {
				return &workflowpb.DeleteWorkflowRequest{
					Data: &workflowpb.Workflow{
						Id: "abc", // Minimal valid ID (3 characters) - but workflow doesn't exist
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "Workflow not found",
			Assertions: func(t *testing.T, response *workflowpb.DeleteWorkflowResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertError(t, err)
			},
		},
		{
			Name:     "WorkflowNotFound",
			TestCode: "ESPYNA-TEST-WORKFLOW-WORKFLOW-DELETE-NOT-FOUND-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *workflowpb.DeleteWorkflowRequest {
				return &workflowpb.DeleteWorkflowRequest{
					Data: &workflowpb.Workflow{
						Id: "non-existent-workflow-id",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "Workflow not found",
			Assertions: func(t *testing.T, response *workflowpb.DeleteWorkflowResponse, err error, useCase interface{}, ctx context.Context) {
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

func TestDeleteWorkflowUseCase_Execute_TransactionFailure(t *testing.T) {
	testCode := "ESPYNA-TEST-WORKFLOW-WORKFLOW-DELETE-TRANSACTION-FAILURE-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "TransactionFailure", false)

	// Create a use case with a failing transaction service
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	mockRepo := workflow.NewMockWorkflowRepository(businessType)

	repositories := DeleteWorkflowRepositories{
		Workflow: mockRepo,
	}

	standardServices := testutil.CreateStandardServices(false, true)
	services := DeleteWorkflowServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   mockDb.NewFailingMockTransactionService(),
		TranslationService:   standardServices.TranslationService,
	}

	useCase := NewDeleteWorkflowUseCase(repositories, services)

	// Load test case using centralized TestCaseResolver
	resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "workflow", "DeleteWorkflow_Success")
	testutil.AssertTestCaseLoad(t, err, "DeleteWorkflow_Success")

	req := &workflowpb.DeleteWorkflowRequest{
		Data: &workflowpb.Workflow{
			Id: resolver.MustGetString("deletableWorkflowId"),
		},
	}

	_, err = useCase.Execute(ctx, req)

	// Verify that a transaction error occurred
	testutil.AssertTransactionError(t, err)

	// Log completion of transaction failure test
	testutil.LogTestResult(t, testCode, "TransactionFailure", false, err)
}
