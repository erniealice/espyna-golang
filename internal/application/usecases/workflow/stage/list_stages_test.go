//go:build mock_db && mock_auth

// Package stage provides table-driven tests for the stage listing use case.
//
// The tests cover various scenarios, including success, transaction handling,
// authorization, nil requests, and filtering.
// Each test case is defined in a table with a specific test code, request setup,
// and assertions.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user").
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education").
//
// Usage:
//   - Run all tests: go test -tags="mock_db,mock_auth" ./...
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestListStagesUseCase_Execute_TableDriven
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-WORKFLOW-STAGE-LIST-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-WORKFLOW-STAGE-LIST-TRANSACTION-SUCCESS-v1.0: WithTransactionSuccess
//   - ESPYNA-TEST-WORKFLOW-STAGE-LIST-AUTHORIZATION-DENIED-v1.0: AuthorizationDenied
//   - ESPYNA-TEST-WORKFLOW-STAGE-LIST-NIL-REQUEST-v1.0: NilRequest
//   - ESPYNA-TEST-WORKFLOW-STAGE-LIST-VERIFY-DETAILS-v1.0: VerifyStageDetails
//   - ESPYNA-TEST-WORKFLOW-STAGE-LIST-WORKFLOW-FILTERING-v1.0: WorkflowFiltering
//   - ESPYNA-TEST-WORKFLOW-STAGE-LIST-BUSINESS-LOGIC-VALIDATION-v1.0: BusinessLogicValidation
//
// Data Sources:
//   - Test cases: packages/copya/data_test/{businessType}/stage.json
//   - Mock data: packages/copya/data/{businessType}/stage.json
//   - Translations: packages/lyngua/translations/{languageCode}/{businessType}/stage.json
package stage

import (
	"context"
	"testing"

	copyatestutil "leapfor.xyz/copya/golang/testutil"
	"leapfor.xyz/espyna/internal/application/shared/testutil"
	"leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock/workflow"
	stagepb "leapfor.xyz/esqyma/golang/v1/domain/workflow/stage"
)

// Type alias for list stages test cases
type ListStagesTestCase = testutil.GenericTestCase[*stagepb.ListStagesRequest, *stagepb.ListStagesResponse]

func createListTestUseCaseWithAuth(businessType string, supportsTransaction bool, shouldAuthorize bool) *ListStagesUseCase {
	mockStageRepo := workflow.NewMockStageRepository(businessType)

	repositories := ListStagesRepositories{
		Stage: mockStageRepo,
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, shouldAuthorize)
	services := ListStagesServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
	}

	return NewListStagesUseCase(repositories, services)
}

func TestListStagesUseCase_Execute_TableDriven(t *testing.T) {
	businessType := testutil.GetTestBusinessType()

	// Load test data resolvers before defining test cases
	listSuccessResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "stage", "ListStages_Success")
	testutil.AssertTestCaseLoad(t, err, "ListStages_Success")

	testCases := []ListStagesTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-WORKFLOW-STAGE-LIST-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *stagepb.ListStagesRequest {
				return &stagepb.ListStagesRequest{}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *stagepb.ListStagesResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertNotNil(t, response.Data, "response data")
				expectedCount := listSuccessResolver.MustGetInt("expectedStageCount")
				testutil.AssertEqual(t, expectedCount, len(response.Data), "stage count")
				expectedStageIds := listSuccessResolver.MustGetStringArray("expectedStageIds")
				stageIds := make(map[string]bool)
				for _, stage := range response.Data {
					stageIds[stage.Id] = true
				}
				for _, expectedId := range expectedStageIds {
					testutil.AssertTrue(t, stageIds[expectedId], "expected stage '"+expectedId+"' found")
				}
			},
		},
		{
			Name:     "WithTransactionSuccess",
			TestCode: "ESPYNA-TEST-WORKFLOW-STAGE-LIST-TRANSACTION-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *stagepb.ListStagesRequest {
				return &stagepb.ListStagesRequest{}
			},
			UseTransaction: true,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *stagepb.ListStagesResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				expectedCount := listSuccessResolver.MustGetInt("expectedStageCount")
				testutil.AssertEqual(t, expectedCount, len(response.Data), "stage count with transaction")
			},
		},
		{
			Name:     "AuthorizationDenied",
			TestCode: "ESPYNA-TEST-WORKFLOW-STAGE-LIST-AUTHORIZATION-DENIED-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *stagepb.ListStagesRequest {
				return &stagepb.ListStagesRequest{}
			},
			UseTransaction: false,
			UseAuth:        false,
			ExpectSuccess:  false,
			ExpectedError:  "Authorization failed for stages",
			Assertions: func(t *testing.T, response *stagepb.ListStagesResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertAuthorizationError(t, err)
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-WORKFLOW-STAGE-LIST-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *stagepb.ListStagesRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "Request is required for stages",
			Assertions: func(t *testing.T, response *stagepb.ListStagesResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilRequest(t, err)
			},
		},
		{
			Name:     "VerifyStageDetails",
			TestCode: "ESPYNA-TEST-WORKFLOW-STAGE-LIST-VERIFY-DETAILS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *stagepb.ListStagesRequest {
				return &stagepb.ListStagesRequest{}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *stagepb.ListStagesResponse, err error, useCase interface{}, ctx context.Context) {
				resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, testutil.GetTestBusinessType(), "stage", "ListStages_VerifyDetails")
				testutil.AssertTestCaseLoad(t, err, "ListStages_VerifyDetails")
				verificationTargetsRaw := resolver.GetTestCase().DataReferences["verificationTargets"]
				verificationTargets := testutil.AssertArray(t, verificationTargetsRaw, "verificationTargets")

				for _, targetInterface := range verificationTargets {
					target := targetInterface.(map[string]interface{})
					targetId := target["id"].(string)
					expectedName := target["expectedName"].(string)
					expectedWorkflowInstanceId := target["expectedWorkflowInstanceId"].(string)
					expectedActive := target["expectedActive"].(bool)

					// Find the stage in the response
					var foundStage *stagepb.Stage
					for _, stage := range response.Data {
						if stage.Id == targetId {
							foundStage = stage
							break
						}
					}

					testutil.AssertNotNil(t, foundStage, targetId+" stage")
					if foundStage != nil {
						testutil.AssertStringEqual(t, expectedName, foundStage.Name, targetId+" stage name")
						testutil.AssertStringEqual(t, expectedWorkflowInstanceId, foundStage.WorkflowInstanceId, targetId+" workflow instance ID")
						testutil.AssertTrue(t, foundStage.Active == expectedActive, targetId+" stage active")
					}
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
			useCase := createListTestUseCaseWithAuth(businessType, tc.UseTransaction, tc.UseAuth)

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
					if err.Error() != tc.ExpectedError {
						t.Errorf("Expected error '%s', got '%s'", tc.ExpectedError, err.Error())
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

func TestListStagesUseCase_Execute_VerifyStageDetails(t *testing.T) {
	testCode := "ESPYNA-TEST-WORKFLOW-STAGE-LIST-VERIFY-DETAILS-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "VerifyStageDetails", true)

	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createListTestUseCaseWithAuth(businessType, false, true)

	// Load test case using centralized TestCaseResolver
	resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "stage", "ListStages_VerifyDetails")
	testutil.AssertTestCaseLoad(t, err, "ListStages_VerifyDetails")

	req := &stagepb.ListStagesRequest{}

	response, err := useCase.Execute(ctx, req)

	testutil.AssertNoError(t, err)

	testutil.AssertNotNil(t, response, "response")
	testutil.AssertGreaterThan(t, len(response.Data), 0, "stages count")

	// Find and verify specific stages using test data
	verificationTargetsRaw := resolver.GetTestCase().DataReferences["verificationTargets"]
	verificationTargets := testutil.AssertArray(t, verificationTargetsRaw, "verificationTargets")

	for _, targetInterface := range verificationTargets {
		target := targetInterface.(map[string]interface{})
		targetId := target["id"].(string)
		expectedName := target["expectedName"].(string)
		expectedWorkflowInstanceId := target["expectedWorkflowInstanceId"].(string)
		expectedActive := target["expectedActive"].(bool)

		// Find the stage in the response
		var foundStage *stagepb.Stage
		for _, stage := range response.Data {
			if stage.Id == targetId {
				foundStage = stage
				break
			}
		}

		testutil.AssertNotNil(t, foundStage, targetId+" stage")
		if foundStage != nil {
			testutil.AssertStringEqual(t, expectedName, foundStage.Name, targetId+" stage name")
			testutil.AssertStringEqual(t, expectedWorkflowInstanceId, foundStage.WorkflowInstanceId, targetId+" workflow instance ID")
			testutil.AssertTrue(t, foundStage.Active == expectedActive, targetId+" stage active")
		}
	}

	// Log test completion with result
	testutil.LogTestResult(t, testCode, "VerifyStageDetails", true, nil)
}

func TestListStagesUseCase_Execute_WorkflowFiltering(t *testing.T) {
	testCode := "ESPYNA-TEST-WORKFLOW-STAGE-LIST-WORKFLOW-FILTERING-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "WorkflowFiltering", true)

	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createListTestUseCaseWithAuth(businessType, false, true)

	// Load test case using centralized TestCaseResolver
	resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "stage", "ListStages_WorkflowFiltering")
	testutil.AssertTestCaseLoad(t, err, "ListStages_WorkflowFiltering")

	req := &stagepb.ListStagesRequest{}

	response, err := useCase.Execute(ctx, req)

	testutil.AssertNoError(t, err)

	testutil.AssertNotNil(t, response, "response")
	testutil.AssertGreaterThan(t, len(response.Data), 0, "stages count")

	// Verify stages are associated with different workflows
	workflowCounts := make(map[string]int)
	for _, stage := range response.Data {
		workflowCounts[stage.WorkflowInstanceId]++
	}

	// Should have stages from multiple workflows based on mock data
	minExpectedWorkflows := resolver.MustGetInt("minExpectedWorkflows")
	testutil.AssertGreaterThanOrEqual(t, len(workflowCounts), minExpectedWorkflows, "workflow diversity")

	// Get workflow associations from test data
	expectedWorkflowAssociationsRaw := resolver.GetTestCase().DataReferences["expectedWorkflowAssociations"]
	expectedWorkflowAssociationsInterface := testutil.AssertMap(t, expectedWorkflowAssociationsRaw, "expectedWorkflowAssociations")
	expectedWorkflowAssociations := make(map[string]string)
	for key, value := range expectedWorkflowAssociationsInterface {
		expectedWorkflowAssociations[key] = value.(string)
	}

	for _, stage := range response.Data {
		expectedWorkflow, exists := expectedWorkflowAssociations[stage.Id]
		if exists {
			testutil.AssertStringEqual(t, expectedWorkflow, stage.WorkflowInstanceId, "workflow association for "+stage.Id)
		}
	}

	// Log test completion with result
	testutil.LogTestResult(t, testCode, "WorkflowFiltering", true, nil)
}

func TestListStagesUseCase_Execute_BusinessLogicValidation(t *testing.T) {
	testCode := "ESPYNA-TEST-WORKFLOW-STAGE-LIST-BUSINESS-LOGIC-VALIDATION-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "BusinessLogicValidation", true)

	// Load test case using centralized TestCaseResolver
	resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, testutil.GetTestBusinessType(), "stage", "ListStages_BusinessLogic")
	testutil.AssertTestCaseLoad(t, err, "ListStages_BusinessLogic")

	testCases := resolver.GetTestCase().DataReferences["testCases"].([]interface{})

	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createListTestUseCaseWithAuth(businessType, false, true)

	for _, testCaseInterface := range testCases {
		testCase := testCaseInterface.(map[string]interface{})
		t.Run(testCase["name"].(string), func(t *testing.T) {
			var request *stagepb.ListStagesRequest
			if testCase["request"] == nil {
				request = nil
			} else {
				request = &stagepb.ListStagesRequest{}
			}

			response, err := useCase.Execute(ctx, request)

			expectError := testCase["expectError"].(bool)
			minStages := int(testCase["minStages"].(float64))

			if expectError {
				testutil.AssertError(t, err)
				testutil.AssertNil(t, response, "response for "+testCase["name"].(string))
			} else {
				testutil.AssertNoError(t, err)
				testutil.AssertNotNil(t, response, "response for "+testCase["name"].(string))
				if response != nil {
					testutil.AssertGreaterThanOrEqual(t, len(response.Data), minStages, "stage count for "+testCase["name"].(string))
				}
			}
		})
	}

	// Log test completion with result
	testutil.LogTestResult(t, testCode, "BusinessLogicValidation", true, nil)
}
