//go:build mock_db && mock_auth

// Package workflow provides table-driven tests for the workflow listing use case.
//
// The tests cover basic scenarios including success, transaction handling,
// authorization, and simple pagination for the ListWorkflows operation.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user").
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education").
//
// Usage:
//   - Run all tests: go test -tags="mock_db,mock_auth" ./...
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestListWorkflowsUseCase_Execute_TableDriven
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-WORKFLOW-WORKFLOW-LIST-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-WORKFLOW-WORKFLOW-LIST-TRANSACTION-v1.0: WithTransaction
//   - ESPYNA-TEST-WORKFLOW-WORKFLOW-LIST-AUTHORIZATION-v1.0: Unauthorized
//   - ESPYNA-TEST-WORKFLOW-WORKFLOW-LIST-NIL-REQUEST-v1.0: NilRequest
//   - ESPYNA-TEST-WORKFLOW-WORKFLOW-LIST-PAGINATION-v1.0: Pagination
//
// Data Sources:
//   - Test cases: packages/copya/data_test/{businessType}/workflow.json
//   - Mock data: packages/copya/data/{businessType}/workflow.json
//   - Translations: packages/lyngua/translations/{languageCode}/{businessType}/workflow.json

package workflow

import (
	"context"
	"fmt"
	"testing"

	copyatestutil "leapfor.xyz/copya/golang/testutil"
	"github.com/erniealice/espyna-golang/internal/application/shared/testutil"
	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock/workflow"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	workflowpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/workflow/workflow"
)

// Type alias for list workflows test cases
type ListWorkflowsTestCase = testutil.GenericTestCase[*workflowpb.ListWorkflowsRequest, *workflowpb.ListWorkflowsResponse]

func createTestListUseCaseWithAuth(businessType string, supportsTransaction bool, shouldAuthorize bool) *ListWorkflowsUseCase {
	mockRepo := workflow.NewMockWorkflowRepository(businessType)

	repositories := ListWorkflowsRepositories{
		Workflow: mockRepo,
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, shouldAuthorize)
	services := ListWorkflowsServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
	}

	return NewListWorkflowsUseCase(repositories, services)
}

func TestListWorkflowsUseCase_Execute_TableDriven(t *testing.T) {
	businessType := testutil.GetTestBusinessType()

	// Load test data resolvers before defining test cases
	listSuccessResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "workflow", "ListWorkflows_Success")
	testutil.AssertTestCaseLoad(t, err, "ListWorkflows_Success")

	testCases := []ListWorkflowsTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-WORKFLOW-WORKFLOW-LIST-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *workflowpb.ListWorkflowsRequest {
				return &workflowpb.ListWorkflowsRequest{
					Pagination: &commonpb.PaginationRequest{
						Limit: 10,
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *workflowpb.ListWorkflowsResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertGreaterThanOrEqual(t, len(response.Data), 0, "response data length")
				expectedWorkflowCount := listSuccessResolver.GetIntWithDefault("expectedWorkflowCount", 5)
				testutil.AssertEqual(t, expectedWorkflowCount, len(response.Data), "workflow count")

				// Verify workflow structure
				for i, workflow := range response.Data {
					testutil.AssertNonEmptyString(t, workflow.Id, fmt.Sprintf("workflow %d ID", i))
					testutil.AssertNonEmptyString(t, workflow.Name, fmt.Sprintf("workflow %d name", i))
				}
			},
		},
		{
			Name:     "WithTransaction",
			TestCode: "ESPYNA-TEST-WORKFLOW-WORKFLOW-LIST-TRANSACTION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *workflowpb.ListWorkflowsRequest {
				return &workflowpb.ListWorkflowsRequest{
					Pagination: &commonpb.PaginationRequest{
						Limit: 5,
					},
				}
			},
			UseTransaction: true,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *workflowpb.ListWorkflowsResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertGreaterThan(t, len(response.Data), 0, "workflows with transaction")
			},
		},
		{
			Name:     "Unauthorized",
			TestCode: "ESPYNA-TEST-WORKFLOW-WORKFLOW-LIST-AUTHORIZATION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *workflowpb.ListWorkflowsRequest {
				return &workflowpb.ListWorkflowsRequest{
					Pagination: &commonpb.PaginationRequest{
						Limit: 10,
					},
				}
			},
			UseTransaction: false,
			UseAuth:        false,
			ExpectSuccess:  false,
			ExpectedError:  "workflow.errors.authorization_failed",
			Assertions: func(t *testing.T, response *workflowpb.ListWorkflowsResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertAuthorizationError(t, err)
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-WORKFLOW-WORKFLOW-LIST-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *workflowpb.ListWorkflowsRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *workflowpb.ListWorkflowsResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertGreaterThanOrEqual(t, len(response.Data), 0, "workflows with nil request")
			},
		},
		{
			Name:     "Pagination",
			TestCode: "ESPYNA-TEST-WORKFLOW-WORKFLOW-LIST-PAGINATION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *workflowpb.ListWorkflowsRequest {
				return &workflowpb.ListWorkflowsRequest{
					Pagination: &commonpb.PaginationRequest{
						Limit: 3, // Only get 3 items per page
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *workflowpb.ListWorkflowsResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertTrue(t, len(response.Data) <= 3, "should not exceed limit")
				// Basic list doesn't return pagination details
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
			useCase := createTestListUseCaseWithAuth(businessType, tc.UseTransaction, tc.UseAuth)

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
