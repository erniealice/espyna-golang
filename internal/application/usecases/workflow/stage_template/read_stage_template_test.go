//go:build mock_db && mock_auth

// Package stage_template provides table-driven tests for the stage template read use case.
//
// The tests cover various scenarios, including success, transaction handling,
// nil requests, validation errors, and not-found cases.
// Each test case is defined in a table with a specific test code, request setup,
// and assertions.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user").
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education").
//
// Usage:
//   - Run all tests: go test -tags="mock_db,mock_auth" ./...
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestReadStageTemplateUseCase_Execute_TableDriven
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-READ-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-READ-TRANSACTION-v1.0: WithTransaction
//   - ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-READ-NIL-REQUEST-v1.0: NilRequest
//   - ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-READ-NIL-DATA-v1.0: NilData
//   - ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-READ-VALIDATION-EMPTY-ID-v1.0: EmptyId
//   - ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-READ-NOT-FOUND-v1.0: NonExistentId
//   - ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-READ-VALIDATION-MINIMAL-ID-v1.0: ValidMinimalId
//   - ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-READ-INTEGRATION-v1.0: RealisticDomainStageTemplate
//   - ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-READ-STRUCTURE-VALIDATION-v1.0: StageTemplateStructureValidation
//   - ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-READ-TRANSACTION-FAILURE-v1.0: TransactionFailure
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

// Type alias for read stage template test cases
type ReadStageTemplateTestCase = testutil.GenericTestCase[*stageTemplatepb.ReadStageTemplateRequest, *stageTemplatepb.ReadStageTemplateResponse]

func createTestReadStageTemplateUseCaseWithAuth(businessType string, supportsTransaction bool, shouldAuthorize bool) *ReadStageTemplateUseCase {
	mockStageTemplateRepo := workflow.NewMockStageTemplateRepository(businessType)

	repositories := ReadStageTemplateRepositories{
		StageTemplate: mockStageTemplateRepo,
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, shouldAuthorize)
	services := ReadStageTemplateServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
	}

	return NewReadStageTemplateUseCase(repositories, services)
}

func TestReadStageTemplateUseCase_Execute_TableDriven(t *testing.T) {
	businessType := testutil.GetTestBusinessType()

	// Load test data resolvers before defining test cases
	commonDataResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "stage_template", "StageTemplate_CommonData")
	testutil.AssertTestCaseLoad(t, err, "StageTemplate_CommonData")

	readSuccessResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "stage_template", "ReadStageTemplate_Success")
	testutil.AssertTestCaseLoad(t, err, "ReadStageTemplate_Success")

	testCases := []ReadStageTemplateTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-READ-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *stageTemplatepb.ReadStageTemplateRequest {
				return &stageTemplatepb.ReadStageTemplateRequest{
					Data: &stageTemplatepb.StageTemplate{
						Id: readSuccessResolver.MustGetString("targetStageTemplateId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *stageTemplatepb.ReadStageTemplateResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertEqual(t, 1, len(response.Data), "response data length")
				stageTemplate := response.Data[0]
				testutil.AssertNonEmptyString(t, stageTemplate.Name, "stage template name")
				testutil.AssertNonEmptyString(t, stageTemplate.WorkflowTemplateId, "workflow ID")
				testutil.AssertFieldSet(t, stageTemplate.DateCreated, "DateCreated")
				testutil.AssertFieldSet(t, stageTemplate.DateCreatedString, "DateCreatedString")
			},
		},
		{
			Name:     "WithTransaction",
			TestCode: "ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-READ-TRANSACTION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *stageTemplatepb.ReadStageTemplateRequest {
				return &stageTemplatepb.ReadStageTemplateRequest{
					Data: &stageTemplatepb.StageTemplate{
						Id: commonDataResolver.MustGetString("secondaryStageTemplateId"),
					},
				}
			},
			UseTransaction: true,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *stageTemplatepb.ReadStageTemplateResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				stageTemplate := response.Data[0]
				testutil.AssertStringEqual(t, commonDataResolver.MustGetString("secondaryStageTemplateId"), stageTemplate.Id, "stage template ID")
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-READ-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *stageTemplatepb.ReadStageTemplateRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "Request is required for stage templates",
			Assertions: func(t *testing.T, response *stageTemplatepb.ReadStageTemplateResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilRequest(t, err)
			},
		},
		{
			Name:     "NilData",
			TestCode: "ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-READ-NIL-DATA-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *stageTemplatepb.ReadStageTemplateRequest {
				return &stageTemplatepb.ReadStageTemplateRequest{Data: nil}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "Request is required for stage templates",
			Assertions: func(t *testing.T, response *stageTemplatepb.ReadStageTemplateResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilData(t, err)
			},
		},
		{
			Name:     "EmptyId",
			TestCode: "ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-READ-VALIDATION-EMPTY-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *stageTemplatepb.ReadStageTemplateRequest {
				return &stageTemplatepb.ReadStageTemplateRequest{
					Data: &stageTemplatepb.StageTemplate{Id: ""},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "Stage template ID is required",
			Assertions: func(t *testing.T, response *stageTemplatepb.ReadStageTemplateResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty ID")
			},
		},
		{
			Name:     "NonExistentId",
			TestCode: "ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-READ-NOT-FOUND-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *stageTemplatepb.ReadStageTemplateRequest {
				return &stageTemplatepb.ReadStageTemplateRequest{
					Data: &stageTemplatepb.StageTemplate{Id: commonDataResolver.MustGetString("nonExistentId")},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "Stage template with ID 'stage-template-non-existent-123' not found",
			ErrorTags:      map[string]any{"stageTemplateId": commonDataResolver.MustGetString("nonExistentId")},
			Assertions: func(t *testing.T, response *stageTemplatepb.ReadStageTemplateResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertError(t, err)
			},
		},
		{
			Name:     "ValidMinimalId",
			TestCode: "ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-READ-VALIDATION-MINIMAL-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *stageTemplatepb.ReadStageTemplateRequest {
				return &stageTemplatepb.ReadStageTemplateRequest{
					Data: &stageTemplatepb.StageTemplate{Id: commonDataResolver.MustGetString("minimalValidId")},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "Stage template with ID 'abc' not found",
			Assertions: func(t *testing.T, response *stageTemplatepb.ReadStageTemplateResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertError(t, err)
			},
		},
		{
			Name:     "RealisticDomainStageTemplate",
			TestCode: "ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-READ-INTEGRATION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *stageTemplatepb.ReadStageTemplateRequest {
				return &stageTemplatepb.ReadStageTemplateRequest{
					Data: &stageTemplatepb.StageTemplate{
						Id: commonDataResolver.MustGetString("thirdStageTemplateId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *stageTemplatepb.ReadStageTemplateResponse, err error, useCase interface{}, ctx context.Context) {
				stageTemplate := response.Data[0]
				testutil.AssertNonEmptyString(t, stageTemplate.Name, "stage template name")
				testutil.AssertFieldSet(t, stageTemplate.Description, "description")
				testutil.AssertNonEmptyString(t, stageTemplate.WorkflowTemplateId, "workflow ID")
				testutil.AssertNonEmptyString(t, stageTemplate.WorkflowTemplateId, "workflow linkage")
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
			useCase := createTestReadStageTemplateUseCaseWithAuth(businessType, tc.UseTransaction, tc.UseAuth)

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

func TestReadStageTemplateUseCase_Execute_StageTemplateStructureValidation(t *testing.T) {
	testCode := "ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-READ-STRUCTURE-VALIDATION-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "StageTemplateStructureValidation", true)

	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestReadStageTemplateUseCaseWithAuth(businessType, false, true)

	// Load test case using centralized TestCaseResolver
	resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "stage_template", "ListStageTemplates_Success")
	testutil.AssertTestCaseLoad(t, err, "ListStageTemplates_Success")

	// Test with multiple real stage template IDs from mock data
	stageTemplateIds := resolver.MustGetStringArray("expectedStageTemplateIds")

	for _, stageTemplateId := range stageTemplateIds {
		req := &stageTemplatepb.ReadStageTemplateRequest{
			Data: &stageTemplatepb.StageTemplate{
				Id: stageTemplateId,
			},
		}

		response, err := useCase.Execute(ctx, req)

		testutil.AssertNoError(t, err)

		stageTemplate := response.Data[0]

		// Validate stage template structure
		testutil.AssertStringEqual(t, stageTemplateId, stageTemplate.Id, "stage template ID")

		testutil.AssertNonEmptyString(t, stageTemplate.Name, "stage template name")

		testutil.AssertTrue(t, stageTemplate.Active, "stage template active status")

		testutil.AssertNonEmptyString(t, stageTemplate.WorkflowTemplateId, "workflow ID")

		// Audit fields
		testutil.AssertFieldSet(t, stageTemplate.DateCreated, "DateCreated")

		testutil.AssertFieldSet(t, stageTemplate.DateCreatedString, "DateCreatedString")
	}

	// Log completion of structure validation test
	testutil.LogTestResult(t, testCode, "StageTemplateStructureValidation", true, nil)
}

func TestReadStageTemplateUseCase_Execute_TransactionFailure(t *testing.T) {
	testCode := "ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-READ-TRANSACTION-FAILURE-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "TransactionFailure", false)

	// Create a use case with a failing transaction service
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	mockStageTemplateRepo := workflow.NewMockStageTemplateRepository(businessType)

	repositories := ReadStageTemplateRepositories{
		StageTemplate: mockStageTemplateRepo,
	}

	standardServices := testutil.CreateStandardServices(false, true)
	// Override with failing transaction service
	standardServices.TransactionService = mockDb.NewFailingMockTransactionService()
	services := ReadStageTemplateServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
	}

	useCase := NewReadStageTemplateUseCase(repositories, services)

	// Load test case using centralized TestCaseResolver
	resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "stage_template", "StageTemplate_CommonData")
	testutil.AssertTestCaseLoad(t, err, "StageTemplate_CommonData")

	req := &stageTemplatepb.ReadStageTemplateRequest{
		Data: &stageTemplatepb.StageTemplate{
			Id: resolver.MustGetString("nonExistentId"),
		},
	}

	// For read operations, transaction failure should not affect the operation
	// since read operations typically don't use transactions
	response, err := useCase.Execute(ctx, req)

	// This should either work (no transaction used) or fail gracefully
	if err != nil {
		// If it fails, verify it's due to the stage template not existing, not transaction failure
		expectedError := "Stage template with ID 'stage-template-non-existent-123' not found"
		if !strings.Contains(err.Error(), expectedError) {
			t.Errorf("Expected error containing '%s', got '%s'", expectedError, err.Error())
		}
	} else {
		// If it succeeds, verify we get a proper response
		testutil.AssertNotNil(t, response, "response")
	}

	// Log completion of transaction failure test
	testutil.LogTestResult(t, testCode, "TransactionFailure", err == nil, err)
}
