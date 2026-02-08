//go:build mock_db && mock_auth

// Package workflow provides table-driven tests for the workflow reading use case.
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
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestReadWorkflowUseCase_Execute_TableDriven
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-WORKFLOW-WORKFLOW-READ-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-WORKFLOW-WORKFLOW-READ-TRANSACTION-v1.0: WithTransaction
//   - ESPYNA-TEST-WORKFLOW-WORKFLOW-READ-AUTHORIZATION-v1.0: Unauthorized
//   - ESPYNA-TEST-WORKFLOW-WORKFLOW-READ-NIL-REQUEST-v1.0: NilRequest
//   - ESPYNA-TEST-WORKFLOW-WORKFLOW-READ-NIL-DATA-v1.0: NilData
//   - ESPYNA-TEST-WORKFLOW-WORKFLOW-READ-VALIDATION-EMPTY-ID-v1.0: EmptyId
//   - ESPYNA-TEST-WORKFLOW-WORKFLOW-READ-VALIDATION-ID-TOO-SHORT-v1.0: IdTooShort
//   - ESPYNA-TEST-WORKFLOW-WORKFLOW-READ-VALIDATION-ID-TOO-LONG-v1.0: IdTooLong
//   - ESPYNA-TEST-WORKFLOW-WORKFLOW-READ-VALIDATION-ID-INVALID-CHARS-v1.0: IdInvalidChars
//   - ESPYNA-TEST-WORKFLOW-WORKFLOW-READ-VALIDATION-WHITESPACE-ID-v1.0: WhitespaceId
//   - ESPYNA-TEST-WORKFLOW-WORKFLOW-READ-VALIDATION-MINIMAL-ID-v1.0: ValidMinimalId
//   - ESPYNA-TEST-WORKFLOW-WORKFLOW-READ-INTEGRATION-v1.0: RealisticEducationWorkflow
//   - ESPYNA-TEST-WORKFLOW-WORKFLOW-READ-DOMAIN-SPECIFIC-v1.0: EducationDomainSpecific
//
// Test data source: packages/copya/data_test/{businessType}/workflow.json
// Mock data source: packages/copya/data/{businessType}/workflow.json
// Translation source: packages/lyngua/translations/{languageCode}/{businessType}/workflow.json

package workflow

import (
	"context"
	"testing"

	copyatestutil "leapfor.xyz/copya/golang/testutil"
	"github.com/erniealice/espyna-golang/internal/application/shared/testutil"
	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock/workflow"
	workflowpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/workflow/workflow"
)

// Type alias for read workflow test cases
type ReadWorkflowTestCase = testutil.GenericTestCase[*workflowpb.ReadWorkflowRequest, *workflowpb.ReadWorkflowResponse]

func createTestReadWorkflowUseCaseWithAuth(businessType string, supportsTransaction bool, shouldAuthorize bool) *ReadWorkflowUseCase {
	mockRepo := workflow.NewMockWorkflowRepository(businessType)

	repositories := ReadWorkflowRepositories{
		Workflow: mockRepo,
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, shouldAuthorize)
	services := ReadWorkflowServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
	}

	return NewReadWorkflowUseCase(repositories, services)
}

func TestReadWorkflowUseCase_Execute_TableDriven(t *testing.T) {
	businessType := testutil.GetTestBusinessType()

	// Load test data resolvers before defining test cases
	createSuccessResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "workflow", "CreateWorkflow_Success")
	testutil.AssertTestCaseLoad(t, err, "CreateWorkflow_Success")

	testCases := []ReadWorkflowTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-WORKFLOW-WORKFLOW-READ-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *workflowpb.ReadWorkflowRequest {
				return &workflowpb.ReadWorkflowRequest{
					Data: &workflowpb.Workflow{
						Id: createSuccessResolver.MustGetString("primaryWorkflowId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *workflowpb.ReadWorkflowResponse, err error, useCase interface{}, ctx context.Context) {
				t.Log("Workflow found successfully - mock data is properly configured")
			},
		},
		{
			Name:     "WithTransaction",
			TestCode: "ESPYNA-TEST-WORKFLOW-WORKFLOW-READ-TRANSACTION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *workflowpb.ReadWorkflowRequest {
				return &workflowpb.ReadWorkflowRequest{
					Data: &workflowpb.Workflow{
						Id: createSuccessResolver.MustGetString("secondaryWorkflowId"),
					},
				}
			},
			UseTransaction: true,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *workflowpb.ReadWorkflowResponse, err error, useCase interface{}, ctx context.Context) {
				t.Log("Workflow found successfully with transaction")
			},
		},
		{
			Name:     "Unauthorized",
			TestCode: "ESPYNA-TEST-WORKFLOW-WORKFLOW-READ-AUTHORIZATION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *workflowpb.ReadWorkflowRequest {
				return &workflowpb.ReadWorkflowRequest{
					Data: &workflowpb.Workflow{
						Id: createSuccessResolver.MustGetString("primaryWorkflowId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        false,
			ExpectSuccess:  false,
			ExpectedError:  "Authorization failed for workflows",
			Assertions: func(t *testing.T, response *workflowpb.ReadWorkflowResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertAuthorizationError(t, err)
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-WORKFLOW-WORKFLOW-READ-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *workflowpb.ReadWorkflowRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "Request is required for workflows",
			Assertions: func(t *testing.T, response *workflowpb.ReadWorkflowResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilRequest(t, err)
			},
		},
		{
			Name:     "NilData",
			TestCode: "ESPYNA-TEST-WORKFLOW-WORKFLOW-READ-NIL-DATA-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *workflowpb.ReadWorkflowRequest {
				return &workflowpb.ReadWorkflowRequest{Data: nil}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "Workflow data is required",
			Assertions: func(t *testing.T, response *workflowpb.ReadWorkflowResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilData(t, err)
			},
		},
		{
			Name:     "EmptyId",
			TestCode: "ESPYNA-TEST-WORKFLOW-WORKFLOW-READ-VALIDATION-EMPTY-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *workflowpb.ReadWorkflowRequest {
				return &workflowpb.ReadWorkflowRequest{
					Data: &workflowpb.Workflow{
						Id: "",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "Workflow ID is required for read operations",
			Assertions: func(t *testing.T, response *workflowpb.ReadWorkflowResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty ID")
			},
		},
		{
			Name:     "IdTooShort",
			TestCode: "ESPYNA-TEST-WORKFLOW-WORKFLOW-READ-VALIDATION-ID-TOO-SHORT-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *workflowpb.ReadWorkflowRequest {
				return &workflowpb.ReadWorkflowRequest{
					Data: &workflowpb.Workflow{
						Id: "ab", // Too short - keeping hardcoded for validation test consistency
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "Workflow ID format is invalid",
			Assertions: func(t *testing.T, response *workflowpb.ReadWorkflowResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "ID too short")
			},
		},
		{
			Name:     "IdTooLong",
			TestCode: "ESPYNA-TEST-WORKFLOW-WORKFLOW-READ-VALIDATION-ID-TOO-LONG-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *workflowpb.ReadWorkflowRequest {
				resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "workflow", "ValidationError_IdTooLongGenerated")
				testutil.AssertTestCaseLoad(t, err, "ValidationError_IdTooLongGenerated")
				return &workflowpb.ReadWorkflowRequest{
					Data: &workflowpb.Workflow{
						Id: resolver.MustGetString("tooLongIdGenerated"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "Workflow ID format is invalid",
			Assertions: func(t *testing.T, response *workflowpb.ReadWorkflowResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "ID too long")
			},
		},
		{
			Name:     "IdInvalidChars",
			TestCode: "ESPYNA-TEST-WORKFLOW-WORKFLOW-READ-VALIDATION-ID-INVALID-CHARS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *workflowpb.ReadWorkflowRequest {
				return &workflowpb.ReadWorkflowRequest{
					Data: &workflowpb.Workflow{
						Id: "workflow@123#invalid", // Invalid characters
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "Workflow ID format is invalid",
			Assertions: func(t *testing.T, response *workflowpb.ReadWorkflowResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "invalid characters")
			},
		},
		{
			Name:     "WhitespaceId",
			TestCode: "ESPYNA-TEST-WORKFLOW-WORKFLOW-READ-VALIDATION-WHITESPACE-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *workflowpb.ReadWorkflowRequest {
				return &workflowpb.ReadWorkflowRequest{
					Data: &workflowpb.Workflow{
						Id: "   ", // Whitespace only
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "Workflow ID format is invalid",
			Assertions: func(t *testing.T, response *workflowpb.ReadWorkflowResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "whitespace only")
			},
		},
		{
			Name:     "ValidMinimalId",
			TestCode: "ESPYNA-TEST-WORKFLOW-WORKFLOW-READ-VALIDATION-MINIMAL-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *workflowpb.ReadWorkflowRequest {
				return &workflowpb.ReadWorkflowRequest{
					Data: &workflowpb.Workflow{
						Id: "abc", // Minimal valid ID (3 characters) - but workflow doesn't exist
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "Workflow not found",
			Assertions: func(t *testing.T, response *workflowpb.ReadWorkflowResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertError(t, err)
			},
		},
		{
			Name:     "RealisticWorkflow",
			TestCode: "ESPYNA-TEST-WORKFLOW-WORKFLOW-READ-INTEGRATION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *workflowpb.ReadWorkflowRequest {
				return &workflowpb.ReadWorkflowRequest{
					Data: &workflowpb.Workflow{
						Id: createSuccessResolver.MustGetString("thirdWorkflowId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *workflowpb.ReadWorkflowResponse, err error, useCase interface{}, ctx context.Context) {
				t.Log("Workflow found successfully - realistic workflow test")
				if response != nil && len(response.Data) > 0 {
					workflow := response.Data[0]
					testutil.AssertNonEmptyString(t, workflow.Id, "workflow ID")
					testutil.AssertNonEmptyString(t, workflow.Name, "workflow name")
					testutil.AssertFieldSet(t, workflow.DateCreated, "DateCreated")
					testutil.AssertFieldSet(t, workflow.DateCreatedString, "DateCreatedString")
				}
			},
		},
		{
			Name:     "EducationDomainSpecific",
			TestCode: "ESPYNA-TEST-WORKFLOW-WORKFLOW-READ-DOMAIN-SPECIFIC-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *workflowpb.ReadWorkflowRequest {
				return &workflowpb.ReadWorkflowRequest{
					Data: &workflowpb.Workflow{
						Id: createSuccessResolver.MustGetString("educationWorkflowId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *workflowpb.ReadWorkflowResponse, err error, useCase interface{}, ctx context.Context) {
				t.Log("Education domain specific workflow found successfully")
				if response != nil && len(response.Data) > 0 {
					workflow := response.Data[0]
					testutil.AssertStringEqual(t, "Education Workflow", workflow.Name, "education workflow name")

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
			useCase := createTestReadWorkflowUseCaseWithAuth(businessType, tc.UseTransaction, tc.UseAuth)

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
