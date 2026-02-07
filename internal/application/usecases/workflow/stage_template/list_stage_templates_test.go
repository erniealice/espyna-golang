//go:build mock_db && mock_auth

// Package stage_template provides table-driven tests for the stage template listing use case.
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
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestListStageTemplatesUseCase_Execute_TableDriven
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-LIST-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-LIST-TRANSACTION-SUCCESS-v1.0: WithTransactionSuccess
//   - ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-LIST-AUTHORIZATION-DENIED-v1.0: AuthorizationDenied
//   - ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-LIST-NIL-REQUEST-v1.0: NilRequest
//   - ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-LIST-VERIFY-DETAILS-v1.0: VerifyStageTemplateDetails
//   - ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-LIST-WORKFLOW-FILTERING-v1.0: WorkflowFiltering
//   - ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-LIST-BUSINESS-LOGIC-VALIDATION-v1.0: BusinessLogicValidation
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
	"leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock/workflow"
	stageTemplatepb "leapfor.xyz/esqyma/golang/v1/domain/workflow/stage_template"
)

// Type alias for list stage templates test cases
type ListStageTemplatesTestCase = testutil.GenericTestCase[*stageTemplatepb.ListStageTemplatesRequest, *stageTemplatepb.ListStageTemplatesResponse]

func createListTestUseCaseWithAuth(businessType string, supportsTransaction bool, shouldAuthorize bool) *ListStageTemplatesUseCase {
	mockStageTemplateRepo := workflow.NewMockStageTemplateRepository(businessType)

	repositories := ListStageTemplatesRepositories{
		StageTemplate: mockStageTemplateRepo,
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, shouldAuthorize)
	services := ListStageTemplatesServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
	}

	return NewListStageTemplatesUseCase(repositories, services)
}

func TestListStageTemplatesUseCase_Execute_TableDriven(t *testing.T) {
	businessType := testutil.GetTestBusinessType()

	// Load test data resolvers before defining test cases
	listSuccessResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "stage_template", "ListStageTemplates_Success")
	testutil.AssertTestCaseLoad(t, err, "ListStageTemplates_Success")

	testCases := []ListStageTemplatesTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-LIST-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *stageTemplatepb.ListStageTemplatesRequest {
				return &stageTemplatepb.ListStageTemplatesRequest{}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *stageTemplatepb.ListStageTemplatesResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertNotNil(t, response.Data, "response data")
				expectedCount := listSuccessResolver.MustGetInt("expectedStageTemplateCount")
				testutil.AssertEqual(t, expectedCount, len(response.Data), "stage template count")
				expectedStageTemplateIds := listSuccessResolver.MustGetStringArray("expectedStageTemplateIds")
				stageTemplateIds := make(map[string]bool)
				for _, st := range response.Data {
					stageTemplateIds[st.Id] = true
				}
				for _, expectedId := range expectedStageTemplateIds {
					testutil.AssertTrue(t, stageTemplateIds[expectedId], "expected stage template '"+expectedId+"' found")
				}
			},
		},
		{
			Name:     "WithTransactionSuccess",
			TestCode: "ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-LIST-TRANSACTION-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *stageTemplatepb.ListStageTemplatesRequest {
				return &stageTemplatepb.ListStageTemplatesRequest{}
			},
			UseTransaction: true,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *stageTemplatepb.ListStageTemplatesResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				expectedCount := listSuccessResolver.MustGetInt("expectedStageTemplateCount")
				testutil.AssertEqual(t, expectedCount, len(response.Data), "stage template count with transaction")
			},
		},
		{
			Name:     "AuthorizationDenied",
			TestCode: "ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-LIST-AUTHORIZATION-DENIED-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *stageTemplatepb.ListStageTemplatesRequest {
				return &stageTemplatepb.ListStageTemplatesRequest{}
			},
			UseTransaction: false,
			UseAuth:        false,
			ExpectSuccess:  false,
			ExpectedError:  "Authorization failed for stage templates",
			Assertions: func(t *testing.T, response *stageTemplatepb.ListStageTemplatesResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertAuthorizationError(t, err)
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-LIST-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *stageTemplatepb.ListStageTemplatesRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "Request is required for stage templates",
			Assertions: func(t *testing.T, response *stageTemplatepb.ListStageTemplatesResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilRequest(t, err)
			},
		},
		{
			Name:     "VerifyStageTemplateDetails",
			TestCode: "ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-LIST-VERIFY-DETAILS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *stageTemplatepb.ListStageTemplatesRequest {
				return &stageTemplatepb.ListStageTemplatesRequest{}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *stageTemplatepb.ListStageTemplatesResponse, err error, useCase interface{}, ctx context.Context) {
				resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, testutil.GetTestBusinessType(), "stage_template", "ListStageTemplates_VerifyDetails")
				testutil.AssertTestCaseLoad(t, err, "ListStageTemplates_VerifyDetails")
				verificationTargetsRaw := resolver.GetTestCase().DataReferences["verificationTargets"]
				verificationTargets := testutil.AssertArray(t, verificationTargetsRaw, "verificationTargets")

				for _, targetInterface := range verificationTargets {
					target := targetInterface.(map[string]interface{})
					targetId := target["id"].(string)
					expectedName := target["expectedName"].(string)
					expectedWorkflowTemplateId := target["expectedWorkflowTemplateId"].(string)
					expectedActive := target["expectedActive"].(bool)

					// Find the stage template in the response
					var foundSt *stageTemplatepb.StageTemplate
					for _, st := range response.Data {
						if st.Id == targetId {
							foundSt = st
							break
						}
					}

					testutil.AssertNotNil(t, foundSt, targetId+" stage template")
					if foundSt != nil {
						testutil.AssertStringEqual(t, expectedName, foundSt.Name, targetId+" stage template name")
						testutil.AssertStringEqual(t, expectedWorkflowTemplateId, foundSt.WorkflowTemplateId, targetId+" workflow ID")
						testutil.AssertTrue(t, foundSt.Active == expectedActive, targetId+" stage template active")
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

func TestListStageTemplatesUseCase_Execute_VerifyStageTemplateDetails(t *testing.T) {
	testCode := "ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-LIST-VERIFY-DETAILS-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "VerifyStageTemplateDetails", true)

	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createListTestUseCaseWithAuth(businessType, false, true)

	// Load test case using centralized TestCaseResolver
	resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "stage_template", "ListStageTemplates_VerifyDetails")
	testutil.AssertTestCaseLoad(t, err, "ListStageTemplates_VerifyDetails")

	req := &stageTemplatepb.ListStageTemplatesRequest{}

	response, err := useCase.Execute(ctx, req)

	testutil.AssertNoError(t, err)

	testutil.AssertNotNil(t, response, "response")
	testutil.AssertGreaterThan(t, len(response.Data), 0, "stage templates count")

	// Find and verify specific stage templates using test data
	verificationTargetsRaw := resolver.GetTestCase().DataReferences["verificationTargets"]
	verificationTargets := testutil.AssertArray(t, verificationTargetsRaw, "verificationTargets")

	for _, targetInterface := range verificationTargets {
		target := targetInterface.(map[string]interface{})
		targetId := target["id"].(string)
		expectedName := target["expectedName"].(string)
		expectedWorkflowTemplateId := target["expectedWorkflowTemplateId"].(string)
		expectedActive := target["expectedActive"].(bool)

		// Find the stage template in the response
		var foundSt *stageTemplatepb.StageTemplate
		for _, st := range response.Data {
			if st.Id == targetId {
				foundSt = st
				break
			}
		}

		testutil.AssertNotNil(t, foundSt, targetId+" stage template")
		if foundSt != nil {
			testutil.AssertStringEqual(t, expectedName, foundSt.Name, targetId+" stage template name")
			testutil.AssertStringEqual(t, expectedWorkflowTemplateId, foundSt.WorkflowTemplateId, targetId+" workflow ID")
			testutil.AssertTrue(t, foundSt.Active == expectedActive, targetId+" stage template active")
		}
	}

	// Log test completion with result
	testutil.LogTestResult(t, testCode, "VerifyStageTemplateDetails", true, nil)
}

func TestListStageTemplatesUseCase_Execute_WorkflowFiltering(t *testing.T) {
	testCode := "ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-LIST-WORKFLOW-FILTERING-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "WorkflowFiltering", true)

	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createListTestUseCaseWithAuth(businessType, false, true)

	// Load test case using centralized TestCaseResolver
	resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "stage_template", "ListStageTemplates_WorkflowFiltering")
	testutil.AssertTestCaseLoad(t, err, "ListStageTemplates_WorkflowFiltering")

	req := &stageTemplatepb.ListStageTemplatesRequest{}

	response, err := useCase.Execute(ctx, req)

	testutil.AssertNoError(t, err)

	testutil.AssertNotNil(t, response, "response")
	testutil.AssertGreaterThan(t, len(response.Data), 0, "stage templates count")

	// Verify stage templates are associated with different workflows
	workflowCounts := make(map[string]int)
	for _, st := range response.Data {
		workflowCounts[st.WorkflowTemplateId]++
	}

	// Should have stage templates from multiple workflows based on mock data
	minExpectedWorkflows := resolver.MustGetInt("minExpectedWorkflows")
	testutil.AssertGreaterThanOrEqual(t, len(workflowCounts), minExpectedWorkflows, "workflow diversity")

	// Get workflow associations from test data
	expectedWorkflowAssociationsRaw := resolver.GetTestCase().DataReferences["expectedWorkflowAssociations"]
	expectedWorkflowAssociationsInterface := testutil.AssertMap(t, expectedWorkflowAssociationsRaw, "expectedWorkflowAssociations")
	expectedWorkflowAssociations := make(map[string]string)
	for key, value := range expectedWorkflowAssociationsInterface {
		expectedWorkflowAssociations[key] = value.(string)
	}

	for _, st := range response.Data {
		expectedWorkflow, exists := expectedWorkflowAssociations[st.Id]
		if exists {
			testutil.AssertStringEqual(t, expectedWorkflow, st.WorkflowTemplateId, "workflow association for "+st.Id)
		}
	}

	// Log test completion with result
	testutil.LogTestResult(t, testCode, "WorkflowFiltering", true, nil)
}

func TestListStageTemplatesUseCase_Execute_BusinessLogicValidation(t *testing.T) {
	testCode := "ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-LIST-BUSINESS-LOGIC-VALIDATION-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "BusinessLogicValidation", true)

	// Load test case using centralized TestCaseResolver
	resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, testutil.GetTestBusinessType(), "stage_template", "ListStageTemplates_BusinessLogic")
	testutil.AssertTestCaseLoad(t, err, "ListStageTemplates_BusinessLogic")

	testCases := resolver.GetTestCase().DataReferences["testCases"].([]interface{})

	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createListTestUseCaseWithAuth(businessType, false, true)

	for _, testCaseInterface := range testCases {
		testCase := testCaseInterface.(map[string]interface{})
		t.Run(testCase["name"].(string), func(t *testing.T) {
			var request *stageTemplatepb.ListStageTemplatesRequest
			if testCase["request"] == nil {
				request = nil
			} else {
				request = &stageTemplatepb.ListStageTemplatesRequest{}
			}

			response, err := useCase.Execute(ctx, request)

			expectError := testCase["expectError"].(bool)
			minStageTemplates := int(testCase["minStageTemplates"].(float64))

			if expectError {
				testutil.AssertError(t, err)
				testutil.AssertNil(t, response, "response for "+testCase["name"].(string))
			} else {
				testutil.AssertNoError(t, err)
				testutil.AssertNotNil(t, response, "response for "+testCase["name"].(string))
				if response != nil {
					testutil.AssertGreaterThanOrEqual(t, len(response.Data), minStageTemplates, "stage template count for "+testCase["name"].(string))
				}
			}
		})
	}

	// Log test completion with result
	testutil.LogTestResult(t, testCode, "BusinessLogicValidation", true, nil)
}
