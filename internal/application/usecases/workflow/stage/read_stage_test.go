//go:build mock_db && mock_auth

// Package stage provides table-driven tests for the stage reading use case.
//
// The tests cover various scenarios including successful reads, authorization,
// not found cases, and validation of stage data retrieval.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user").
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education").
//
// Usage:
//   - Run all tests: go test -tags="mock_db,mock_auth" ./...
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestReadStageUseCase_Execute_TableDriven
//
// Test Codes:
//   - ESPYNA-TEST-WORKFLOW-STAGE-READ-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-WORKFLOW-STAGE-READ-TRANSACTION-v1.0: WithTransaction
//   - ESPYNA-TEST-WORKFLOW-STAGE-READ-NIL-REQUEST-v1.0: NilRequest
//   - ESPYNA-TEST-WORKFLOW-STAGE-READ-NIL-DATA-v1.0: NilData
//   - ESPYNA-TEST-WORKFLOW-STAGE-READ-VALIDATION-EMPTY-ID-v1.0: EmptyId
//   - ESPYNA-TEST-WORKFLOW-STAGE-READ-NOT-FOUND-v1.0: NonExistentId
//   - ESPYNA-TEST-WORKFLOW-STAGE-READ-VALIDATION-MINIMAL-ID-v1.0: ValidMinimalId
//   - ESPYNA-TEST-WORKFLOW-STAGE-READ-INTEGRATION-v1.0: RealisticDomainStage
//   - ESPYNA-TEST-WORKFLOW-STAGE-READ-STRUCTURE-VALIDATION-v1.0: StageStructureValidation
//   - ESPYNA-TEST-WORKFLOW-STAGE-READ-TRANSACTION-FAILURE-v1.0: TransactionFailure
package stage

import (
	"context"
	"strings"
	"testing"

	copyatestutil "leapfor.xyz/copya/golang/testutil"
	"leapfor.xyz/espyna/internal/application/shared/testutil"
	mockDb "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock"
	"leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock/workflow"
	stagepb "leapfor.xyz/esqyma/golang/v1/domain/workflow/stage"
)

// Type alias for read stage test cases
type ReadStageTestCase = testutil.GenericTestCase[*stagepb.ReadStageRequest, *stagepb.ReadStageResponse]

func createTestReadStageUseCaseWithAuth(businessType string, supportsTransaction bool, shouldAuthorize bool) *ReadStageUseCase {
	mockStageRepo := workflow.NewMockStageRepository(businessType)

	repositories := ReadStageRepositories{
		Stage: mockStageRepo,
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, shouldAuthorize)
	services := ReadStageServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
	}

	return NewReadStageUseCase(repositories, services)
}

func TestReadStageUseCase_Execute_TableDriven(t *testing.T) {
	businessType := testutil.GetTestBusinessType()

	// Load test data resolvers before defining test cases
	commonDataResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "stage", "Stage_CommonData")
	testutil.AssertTestCaseLoad(t, err, "Stage_CommonData")

	readSuccessResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "stage", "ReadStage_Success")
	testutil.AssertTestCaseLoad(t, err, "ReadStage_Success")

	testCases := []ReadStageTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-WORKFLOW-STAGE-READ-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *stagepb.ReadStageRequest {
				return &stagepb.ReadStageRequest{
					Data: &stagepb.Stage{
						Id: readSuccessResolver.MustGetString("targetStageId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *stagepb.ReadStageResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertEqual(t, 1, len(response.Data), "response data length")
				stage := response.Data[0]
				testutil.AssertNonEmptyString(t, stage.Name, "stage name")
				testutil.AssertNonEmptyString(t, stage.WorkflowInstanceId, "workflow instance ID")
				testutil.AssertFieldSet(t, stage.DateCreated, "DateCreated")
				testutil.AssertFieldSet(t, stage.DateCreatedString, "DateCreatedString")
			},
		},
		{
			Name:     "WithTransaction",
			TestCode: "ESPYNA-TEST-WORKFLOW-STAGE-READ-TRANSACTION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *stagepb.ReadStageRequest {
				return &stagepb.ReadStageRequest{
					Data: &stagepb.Stage{
						Id: commonDataResolver.MustGetString("secondaryStageId"),
					},
				}
			},
			UseTransaction: true,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *stagepb.ReadStageResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				stage := response.Data[0]
				testutil.AssertStringEqual(t, commonDataResolver.MustGetString("secondaryStageId"), stage.Id, "stage ID")
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-WORKFLOW-STAGE-READ-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *stagepb.ReadStageRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "Request is required for stages",
			Assertions: func(t *testing.T, response *stagepb.ReadStageResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilRequest(t, err)
			},
		},
		{
			Name:     "NilData",
			TestCode: "ESPYNA-TEST-WORKFLOW-STAGE-READ-NIL-DATA-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *stagepb.ReadStageRequest {
				return &stagepb.ReadStageRequest{Data: nil}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "Request is required for stages",
			Assertions: func(t *testing.T, response *stagepb.ReadStageResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilData(t, err)
			},
		},
		{
			Name:     "EmptyId",
			TestCode: "ESPYNA-TEST-WORKFLOW-STAGE-READ-VALIDATION-EMPTY-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *stagepb.ReadStageRequest {
				return &stagepb.ReadStageRequest{
					Data: &stagepb.Stage{Id: ""},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "Stage ID is required",
			Assertions: func(t *testing.T, response *stagepb.ReadStageResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty ID")
			},
		},
		{
			Name:     "NonExistentId",
			TestCode: "ESPYNA-TEST-WORKFLOW-STAGE-READ-NOT-FOUND-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *stagepb.ReadStageRequest {
				return &stagepb.ReadStageRequest{
					Data: &stagepb.Stage{Id: commonDataResolver.MustGetString("nonExistentId")},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "Stage with ID 'stage-non-existent-123' not found",
			ErrorTags:      map[string]any{"stageId": commonDataResolver.MustGetString("nonExistentId")},
			Assertions: func(t *testing.T, response *stagepb.ReadStageResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertError(t, err)
			},
		},
		{
			Name:     "ValidMinimalId",
			TestCode: "ESPYNA-TEST-WORKFLOW-STAGE-READ-VALIDATION-MINIMAL-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *stagepb.ReadStageRequest {
				return &stagepb.ReadStageRequest{
					Data: &stagepb.Stage{Id: commonDataResolver.MustGetString("minimalValidId")},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "Stage with ID 'abc' not found",
			Assertions: func(t *testing.T, response *stagepb.ReadStageResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertError(t, err)
			},
		},
		{
			Name:     "RealisticDomainStage",
			TestCode: "ESPYNA-TEST-WORKFLOW-STAGE-READ-INTEGRATION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *stagepb.ReadStageRequest {
				return &stagepb.ReadStageRequest{
					Data: &stagepb.Stage{
						Id: commonDataResolver.MustGetString("thirdStageId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *stagepb.ReadStageResponse, err error, useCase interface{}, ctx context.Context) {
				stage := response.Data[0]
				testutil.AssertNonEmptyString(t, stage.Name, "stage name")
				testutil.AssertFieldSet(t, stage.Description, "description")
				testutil.AssertNonEmptyString(t, stage.WorkflowInstanceId, "workflow instance ID")
				testutil.AssertNonEmptyString(t, stage.WorkflowInstanceId, "workflow instance linkage")
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
			useCase := createTestReadStageUseCaseWithAuth(businessType, tc.UseTransaction, tc.UseAuth)

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

func TestReadStageUseCase_Execute_StageStructureValidation(t *testing.T) {
	testCode := "ESPYNA-TEST-WORKFLOW-STAGE-READ-STRUCTURE-VALIDATION-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "StageStructureValidation", true)

	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestReadStageUseCaseWithAuth(businessType, false, true)

	// Load test case using centralized TestCaseResolver
	resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "stage", "ListStages_Success")
	testutil.AssertTestCaseLoad(t, err, "ListStages_Success")

	// Test with multiple real stage IDs from mock data
	stageIds := resolver.MustGetStringArray("expectedStageIds")

	for _, stageId := range stageIds {
		req := &stagepb.ReadStageRequest{
			Data: &stagepb.Stage{
				Id: stageId,
			},
		}

		response, err := useCase.Execute(ctx, req)

		testutil.AssertNoError(t, err)

		stage := response.Data[0]

		// Validate stage structure
		testutil.AssertStringEqual(t, stageId, stage.Id, "stage ID")

		testutil.AssertNonEmptyString(t, stage.Name, "stage name")

		testutil.AssertTrue(t, stage.Active, "stage active status")

		testutil.AssertNonEmptyString(t, stage.WorkflowInstanceId, "workflow instance ID")

		// Audit fields
		testutil.AssertFieldSet(t, stage.DateCreated, "DateCreated")

		testutil.AssertFieldSet(t, stage.DateCreatedString, "DateCreatedString")
	}

	// Log completion of structure validation test
	testutil.LogTestResult(t, testCode, "StageStructureValidation", true, nil)
}

func TestReadStageUseCase_Execute_TransactionFailure(t *testing.T) {
	testCode := "ESPYNA-TEST-WORKFLOW-STAGE-READ-TRANSACTION-FAILURE-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "TransactionFailure", false)

	// Create a use case with a failing transaction service
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	mockStageRepo := workflow.NewMockStageRepository(businessType)

	repositories := ReadStageRepositories{
		Stage: mockStageRepo,
	}

	standardServices := testutil.CreateStandardServices(false, true)
	// Override with failing transaction service
	standardServices.TransactionService = mockDb.NewFailingMockTransactionService()
	services := ReadStageServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
	}

	useCase := NewReadStageUseCase(repositories, services)

	// Load test case using centralized TestCaseResolver
	resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "stage", "Stage_CommonData")
	testutil.AssertTestCaseLoad(t, err, "Stage_CommonData")

	req := &stagepb.ReadStageRequest{
		Data: &stagepb.Stage{
			Id: resolver.MustGetString("nonExistentId"),
		},
	}

	// For read operations, transaction failure should not affect the operation
	// since read operations typically don't use transactions
	response, err := useCase.Execute(ctx, req)

	// This should either work (no transaction used) or fail gracefully
	if err != nil {
		// If it fails, verify it's due to the stage not existing, not transaction failure
		expectedError := "Stage with ID 'stage-non-existent-123' not found"
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
