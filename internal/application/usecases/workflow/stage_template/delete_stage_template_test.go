//go:build mock_db && mock_auth

// Package stage_template provides table-driven tests for the stage template deletion use case.
//
// The tests cover various scenarios, including success, transaction handling,
// authorization, nil requests, validation errors, and error handling.
// Each test case is defined in a table with a specific test code, request setup,
// and assertions.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user").
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education").
//
// Usage:
//   - Run all tests: go test -tags="mock_db,mock_auth" ./...
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestDeleteStageTemplateUseCase_Execute_TableDriven
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-DELETE-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-DELETE-NOT-FOUND-v1.0: NonExistentStageTemplate
//   - ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-DELETE-VALIDATION-EMPTY-ID-v1.0: EmptyStageTemplateId
//   - ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-DELETE-NIL-REQUEST-v1.0: NilRequest
//   - ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-DELETE-NIL-DATA-v1.0: NilData
//   - ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-DELETE-TRANSACTION-SUCCESS-v1.0: WithTransactionSuccess
//   - ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-DELETE-AUTHORIZATION-DENIED-v1.0: AuthorizationDenied
//   - ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-DELETE-TRANSACTION-FAILURE-v1.0: WithTransactionFailure
//   - ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-DELETE-INTEGRATION-v1.0: MultipleValidStageTemplates
//   - ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-DELETE-BUSINESS-LOGIC-v1.0: BusinessLogicValidation
//
// Data Sources:
//   - Test cases: packages/copya/data_test/{businessType}/stage_template.json
//   - Mock data: packages/copya/data/{businessType}/stage_template.json
//   - Translations: packages/lyngua/translations/{languageCode}/{businessType}/stage_template.json
package stage_template

import (
	"context"
	"fmt"
	"strings"
	"testing"

	copyatestutil "leapfor.xyz/copya/golang/testutil"
	"leapfor.xyz/espyna/internal/application/shared/testutil"
	mockDb "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock"
	"leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock/workflow"
	stageTemplatepb "leapfor.xyz/esqyma/golang/v1/domain/workflow/stage_template"
)

// Type alias for delete stage template test cases
type DeleteStageTemplateTestCase = testutil.GenericTestCase[*stageTemplatepb.DeleteStageTemplateRequest, *stageTemplatepb.DeleteStageTemplateResponse]

func createDeleteTestUseCaseWithAuth(businessType string, supportsTransaction bool, shouldAuthorize bool) *DeleteStageTemplateUseCase {
	mockStageTemplateRepo := workflow.NewMockStageTemplateRepository(businessType)

	repositories := DeleteStageTemplateRepositories{
		StageTemplate: mockStageTemplateRepo,
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, shouldAuthorize)
	services := DeleteStageTemplateServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
	}

	return NewDeleteStageTemplateUseCase(repositories, services)
}

func createDeleteTestUseCaseWithFailingTransaction(businessType string, shouldAuthorize bool) *DeleteStageTemplateUseCase {
	mockStageTemplateRepo := workflow.NewMockStageTemplateRepository(businessType)

	repositories := DeleteStageTemplateRepositories{
		StageTemplate: mockStageTemplateRepo,
	}

	standardServices := testutil.CreateStandardServices(false, shouldAuthorize)
	// Override with failing transaction service
	standardServices.TransactionService = mockDb.NewFailingMockTransactionService()
	services := DeleteStageTemplateServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
	}

	return NewDeleteStageTemplateUseCase(repositories, services)
}

func TestDeleteStageTemplateUseCase_Execute_TableDriven(t *testing.T) {
	businessType := testutil.GetTestBusinessType()

	// Load test data resolvers before defining test cases
	commonDataResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "stage_template", "StageTemplate_CommonData")
	testutil.AssertTestCaseLoad(t, err, "StageTemplate_CommonData")

	createSuccessResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "stage_template", "CreateStageTemplate_Success")
	testutil.AssertTestCaseLoad(t, err, "CreateStageTemplate_Success")

	testCases := []DeleteStageTemplateTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-DELETE-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *stageTemplatepb.DeleteStageTemplateRequest {
				return &stageTemplatepb.DeleteStageTemplateRequest{
					Data: &stageTemplatepb.StageTemplate{
						Id: createSuccessResolver.MustGetString("primaryStageTemplateId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *stageTemplatepb.DeleteStageTemplateResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "successful deletion")
			},
		},
		{
			Name:     "NonExistentStageTemplate",
			TestCode: "ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-DELETE-NOT-FOUND-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *stageTemplatepb.DeleteStageTemplateRequest {
				return &stageTemplatepb.DeleteStageTemplateRequest{
					Data: &stageTemplatepb.StageTemplate{
						Id: commonDataResolver.MustGetString("nonExistentId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "Stage template deletion failed: Stage template with ID 'stage-template-non-existent-123' not found",
			Assertions: func(t *testing.T, response *stageTemplatepb.DeleteStageTemplateResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertError(t, err)
				testutil.AssertNil(t, response, "response for non-existent stage template")
			},
		},
		{
			Name:     "EmptyStageTemplateId",
			TestCode: "ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-DELETE-VALIDATION-EMPTY-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *stageTemplatepb.DeleteStageTemplateRequest {
				return &stageTemplatepb.DeleteStageTemplateRequest{
					Data: &stageTemplatepb.StageTemplate{
						Id: "",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "Stage template ID is required",
			Assertions: func(t *testing.T, response *stageTemplatepb.DeleteStageTemplateResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty stage template ID")
				testutil.AssertNil(t, response, "response for invalid input")
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-DELETE-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *stageTemplatepb.DeleteStageTemplateRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "Request is required for stage templates",
			Assertions: func(t *testing.T, response *stageTemplatepb.DeleteStageTemplateResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilRequest(t, err)
				testutil.AssertNil(t, response, "response for nil request")
			},
		},
		{
			Name:     "NilData",
			TestCode: "ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-DELETE-NIL-DATA-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *stageTemplatepb.DeleteStageTemplateRequest {
				return &stageTemplatepb.DeleteStageTemplateRequest{Data: nil}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "Request is required for stage templates",
			Assertions: func(t *testing.T, response *stageTemplatepb.DeleteStageTemplateResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilData(t, err)
				testutil.AssertNil(t, response, "response for nil data")
			},
		},
		{
			Name:     "WithTransactionSuccess",
			TestCode: "ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-DELETE-TRANSACTION-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *stageTemplatepb.DeleteStageTemplateRequest {
				return &stageTemplatepb.DeleteStageTemplateRequest{
					Data: &stageTemplatepb.StageTemplate{
						Id: commonDataResolver.MustGetString("secondaryStageTemplateId"),
					},
				}
			},
			UseTransaction: true,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *stageTemplatepb.DeleteStageTemplateResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "successful deletion with transaction")
			},
		},
		{
			Name:     "AuthorizationDenied",
			TestCode: "ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-DELETE-AUTHORIZATION-DENIED-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *stageTemplatepb.DeleteStageTemplateRequest {
				return &stageTemplatepb.DeleteStageTemplateRequest{
					Data: &stageTemplatepb.StageTemplate{
						Id: commonDataResolver.MustGetString("businessRulesStageTemplateId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        false,
			ExpectSuccess:  false,
			ExpectedError:  "Authorization failed for stage templates",
			Assertions: func(t *testing.T, response *stageTemplatepb.DeleteStageTemplateResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertAuthorizationError(t, err)
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
			useCase := createDeleteTestUseCaseWithAuth(businessType, tc.UseTransaction, tc.UseAuth)

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

func TestDeleteStageTemplateUseCase_Execute_WithTransaction_Failure(t *testing.T) {
	testCode := "ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-DELETE-TRANSACTION-FAILURE-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "WithTransactionFailure", false)

	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createDeleteTestUseCaseWithFailingTransaction(businessType, true)

	// Load test case using centralized TestCaseResolver
	resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "stage_template", "StageTemplate_CommonData")
	testutil.AssertTestCaseLoad(t, err, "StageTemplate_CommonData")

	req := &stageTemplatepb.DeleteStageTemplateRequest{
		Data: &stageTemplatepb.StageTemplate{
			Id: resolver.MustGetString("thirdStageTemplateId"),
		},
	}

	_, err2 := useCase.Execute(ctx, req)

	testutil.AssertTransactionError(t, err2)

	expectedError := "Transaction execution failed: transaction error [TRANSACTION_GENERAL] during run_in_transaction: mock run in transaction failed"
	testutil.AssertStringEqual(t, expectedError, err2.Error(), "error message")

	// Log test completion with result
	testutil.LogTestResult(t, testCode, "WithTransactionFailure", false, err2)
}

func TestDeleteStageTemplateUseCase_Execute_MultipleValidStageTemplates(t *testing.T) {
	testCode := "ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-DELETE-INTEGRATION-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "MultipleValidStageTemplates", true)

	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()

	// Load test case using centralized TestCaseResolver
	resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "stage_template", "ListStageTemplates_Success")
	testutil.AssertTestCaseLoad(t, err, "ListStageTemplates_Success")

	stageTemplateIds := resolver.MustGetStringArray("expectedStageTemplateIds")

	// Create test cases dynamically based on available stage template IDs
	testCases := make([]struct {
		name            string
		stageTemplateId string
		expectError     bool
	}, len(stageTemplateIds))

	for i, stageTemplateId := range stageTemplateIds {
		testCases[i] = struct {
			name            string
			stageTemplateId string
			expectError     bool
		}{
			name:            fmt.Sprintf("Delete stage template %d (%s)", i+1, stageTemplateId),
			stageTemplateId: stageTemplateId,
			expectError:     false,
		}
	}

	useCase := createDeleteTestUseCaseWithAuth(businessType, false, true)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			req := &stageTemplatepb.DeleteStageTemplateRequest{
				Data: &stageTemplatepb.StageTemplate{
					Id: tc.stageTemplateId,
				},
			}

			response, err := useCase.Execute(ctx, req)

			if tc.expectError {
				testutil.AssertError(t, err)
			} else {
				testutil.AssertNoError(t, err)

				testutil.AssertNotNil(t, response, "response")

				testutil.AssertTrue(t, response.Success, "successful deletion")
			}
		})
	}

	// Log test completion with result
	testutil.LogTestResult(t, testCode, "MultipleValidStageTemplates", true, nil)
}

func TestDeleteStageTemplateUseCase_Execute_BusinessLogicValidation(t *testing.T) {
	testCode := "ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-DELETE-BUSINESS-LOGIC-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "BusinessLogicValidation", true)

	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createDeleteTestUseCaseWithAuth(businessType, false, true)

	// Load test case using centralized TestCaseResolver
	resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "stage_template", "StageTemplate_CommonData")
	testutil.AssertTestCaseLoad(t, err, "StageTemplate_CommonData")

	tests := []struct {
		name              string
		stageTemplateData *stageTemplatepb.StageTemplate
		expectError       bool
		expectedError     string
	}{
		{
			name: "Valid stage template ID",
			stageTemplateData: &stageTemplatepb.StageTemplate{
				Id: resolver.MustGetString("primaryStageTemplateId"),
			},
			expectError: false,
		},
		{
			name: "Invalid stage template ID format",
			stageTemplateData: &stageTemplatepb.StageTemplate{
				Id: "invalid-id-format",
			},
			expectError: true,
		},
		{
			name: "Extremely long stage template ID",
			stageTemplateData: &stageTemplatepb.StageTemplate{
				Id: fmt.Sprintf("stage-template-%s", strings.Repeat("A", 300)), // Create long string for validation test
			},
			expectError: true,
		},
		{
			name: "Active stage template (cannot delete)",
			stageTemplateData: &stageTemplatepb.StageTemplate{
				Id: resolver.MustGetString("activeStageTemplateId"),
			},
			expectError:   true,
			expectedError: "Cannot delete active stage template",
		},
		{
			name: "Inactive stage template (can delete)",
			stageTemplateData: &stageTemplatepb.StageTemplate{
				Id: resolver.MustGetString("inactiveStageTemplateId"),
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			req := &stageTemplatepb.DeleteStageTemplateRequest{
				Data: tt.stageTemplateData,
			}

			response, err := useCase.Execute(ctx, req)

			if tt.expectError {
				testutil.AssertError(t, err)
				if tt.expectedError != "" {
					if !strings.Contains(err.Error(), tt.expectedError) {
						t.Errorf("Expected error containing '%s', got '%s'", tt.expectedError, err.Error())
					}
				}
				testutil.AssertNil(t, response, "response")
			} else {
				testutil.AssertNoError(t, err)
				testutil.AssertNotNil(t, response, "response")
			}
		})
	}

	// Log test completion with result
	testutil.LogTestResult(t, testCode, "BusinessLogicValidation", true, nil)
}
