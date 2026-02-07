//go:build mock_db && mock_auth

// Package activity provides table-driven tests for the activity deletion use case.
//
// The tests cover various scenarios including successful deletion, authorization,
// not found cases, and soft delete validation.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user").
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education").
//
// Usage:
//   - Run all tests: go test -tags="mock_db,mock_auth" ./...
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestDeleteActivityUseCase_Execute_TableDriven
//
// Test Codes:
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-DELETE-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-DELETE-NOT-FOUND-v1.0: NonExistentActivity
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-DELETE-VALIDATION-EMPTY-ID-v1.0: EmptyActivityId
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-DELETE-NIL-REQUEST-v1.0: NilRequest
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-DELETE-NIL-DATA-v1.0: NilData
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-DELETE-TRANSACTION-SUCCESS-v1.0: WithTransactionSuccess
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-DELETE-AUTHORIZATION-DENIED-v1.0: AuthorizationDenied
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-DELETE-TRANSACTION-FAILURE-v1.0: TransactionFailure
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-DELETE-INTEGRATION-v1.0: MultipleValidActivities
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-DELETE-BUSINESS-LOGIC-v1.0: BusinessLogicValidation
package activity

import (
	"context"
	"fmt"
	"testing"

	copyatestutil "leapfor.xyz/copya/golang/testutil"
	"leapfor.xyz/espyna/internal/application/shared/testutil"
	mockDb "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock"
	"leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock/workflow"
	activitypb "leapfor.xyz/esqyma/golang/v1/domain/workflow/activity"
)

// Type alias for delete activity test cases
type DeleteActivityTestCase = testutil.GenericTestCase[*activitypb.DeleteActivityRequest, *activitypb.DeleteActivityResponse]

func createDeleteTestUseCaseWithAuth(businessType string, supportsTransaction bool, shouldAuthorize bool) *DeleteActivityUseCase {
	mockActivityRepo := workflow.NewMockActivityRepository(businessType)

	repositories := DeleteActivityRepositories{
		Activity: mockActivityRepo,
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, shouldAuthorize)
	services := DeleteActivityServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
	}

	return NewDeleteActivityUseCase(repositories, services)
}

func createDeleteTestUseCaseWithFailingTransaction(businessType string, shouldAuthorize bool) *DeleteActivityUseCase {
	mockActivityRepo := workflow.NewMockActivityRepository(businessType)

	repositories := DeleteActivityRepositories{
		Activity: mockActivityRepo,
	}

	standardServices := testutil.CreateStandardServices(false, shouldAuthorize)
	// Override with failing transaction service
	standardServices.TransactionService = mockDb.NewFailingMockTransactionService()
	services := DeleteActivityServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
	}

	return NewDeleteActivityUseCase(repositories, services)
}

func TestDeleteActivityUseCase_Execute_TableDriven(t *testing.T) {
	businessType := testutil.GetTestBusinessType()

	// Load test data resolvers before defining test cases
	commonDataResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "activity", "Activity_CommonData")
	testutil.AssertTestCaseLoad(t, err, "Activity_CommonData")

	createSuccessResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "activity", "CreateActivity_Success")
	testutil.AssertTestCaseLoad(t, err, "CreateActivity_Success")

	testCases := []DeleteActivityTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-WORKFLOW-ACTIVITY-DELETE-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *activitypb.DeleteActivityRequest {
				return &activitypb.DeleteActivityRequest{
					Data: &activitypb.Activity{
						Id: createSuccessResolver.MustGetString("primaryActivityId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *activitypb.DeleteActivityResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "successful deletion")
			},
		},
		{
			Name:     "NonExistentActivity",
			TestCode: "ESPYNA-TEST-WORKFLOW-ACTIVITY-DELETE-NOT-FOUND-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *activitypb.DeleteActivityRequest {
				return &activitypb.DeleteActivityRequest{
					Data: &activitypb.Activity{
						Id: commonDataResolver.MustGetString("nonExistentId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "Activity deletion failed: activity with ID 'activity-non-existent-123' not found",
			ExactError:     true,
			Assertions: func(t *testing.T, response *activitypb.DeleteActivityResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertError(t, err)
				testutil.AssertNil(t, response, "response for non-existent activity")
			},
		},
		{
			Name:     "EmptyActivityId",
			TestCode: "ESPYNA-TEST-WORKFLOW-ACTIVITY-DELETE-VALIDATION-EMPTY-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *activitypb.DeleteActivityRequest {
				return &activitypb.DeleteActivityRequest{
					Data: &activitypb.Activity{
						Id: "",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "activity.validation.id_required",
			Assertions: func(t *testing.T, response *activitypb.DeleteActivityResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty activity ID")
				testutil.AssertNil(t, response, "response for invalid input")
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-WORKFLOW-ACTIVITY-DELETE-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *activitypb.DeleteActivityRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "activity.validation.request_required",
			Assertions: func(t *testing.T, response *activitypb.DeleteActivityResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilRequest(t, err)
				testutil.AssertNil(t, response, "response for nil request")
			},
		},
		{
			Name:     "NilData",
			TestCode: "ESPYNA-TEST-WORKFLOW-ACTIVITY-DELETE-NIL-DATA-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *activitypb.DeleteActivityRequest {
				return &activitypb.DeleteActivityRequest{Data: nil}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "activity.validation.request_required",
			Assertions: func(t *testing.T, response *activitypb.DeleteActivityResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilData(t, err)
				testutil.AssertNil(t, response, "response for nil data")
			},
		},
		{
			Name:     "WithTransactionSuccess",
			TestCode: "ESPYNA-TEST-WORKFLOW-ACTIVITY-DELETE-TRANSACTION-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *activitypb.DeleteActivityRequest {
				return &activitypb.DeleteActivityRequest{
					Data: &activitypb.Activity{
						Id: commonDataResolver.MustGetString("secondaryActivityId"),
					},
				}
			},
			UseTransaction: true,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *activitypb.DeleteActivityResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "successful deletion with transaction")
			},
		},
		{
			Name:     "AuthorizationDenied",
			TestCode: "ESPYNA-TEST-WORKFLOW-ACTIVITY-DELETE-AUTHORIZATION-DENIED-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *activitypb.DeleteActivityRequest {
				return &activitypb.DeleteActivityRequest{
					Data: &activitypb.Activity{
						Id: commonDataResolver.MustGetString("businessRulesActivityId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        false,
			ExpectSuccess:  false,
			ExpectedError:  "activity.errors.authorization_failed",
			Assertions: func(t *testing.T, response *activitypb.DeleteActivityResponse, err error, useCase interface{}, ctx context.Context) {
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
					if tc.ExactError {
						testutil.AssertStringEqual(t, tc.ExpectedError, err.Error(), "error message")
					} else if tc.ErrorTags != nil {
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

func TestDeleteActivityUseCase_Execute_WithTransaction_Failure(t *testing.T) {
	testCode := "ESPYNA-TEST-WORKFLOW-ACTIVITY-DELETE-TRANSACTION-FAILURE-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "WithTransactionFailure", false)

	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createDeleteTestUseCaseWithFailingTransaction(businessType, true)

	// Load test case using centralized TestCaseResolver
	resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "activity", "Activity_CommonData")
	testutil.AssertTestCaseLoad(t, err, "Activity_CommonData")

	req := &activitypb.DeleteActivityRequest{
		Data: &activitypb.Activity{
			Id: resolver.MustGetString("thirdActivityId"),
		},
	}

	_, err = useCase.Execute(ctx, req)

	testutil.AssertTransactionError(t, err)

	expectedError := "Transaction execution failed [DEFAULT]: transaction error [TRANSACTION_GENERAL] during run_in_transaction: mock run in transaction failed"
	testutil.AssertStringEqual(t, expectedError, err.Error(), "error message")

	// Log test completion with result
	testutil.LogTestResult(t, testCode, "WithTransactionFailure", false, err)
}

func TestDeleteActivityUseCase_Execute_MultipleValidActivities(t *testing.T) {
	testCode := "ESPYNA-TEST-WORKFLOW-ACTIVITY-DELETE-INTEGRATION-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "MultipleValidActivities", true)

	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()

	// Load test case using centralized TestCaseResolver
	resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "activity", "ListActivities_Success")
	testutil.AssertTestCaseLoad(t, err, "ListActivities_Success")

	activityIds := resolver.MustGetStringArray("expectedActivityIds")

	// Create test cases dynamically based on available activity IDs
	testCases := make([]struct {
		name        string
		activityId  string
		expectError bool
	}, len(activityIds))

	for i, activityId := range activityIds {
		testCases[i] = struct {
			name        string
			activityId  string
			expectError bool
		}{
			name:        fmt.Sprintf("Delete activity %d (%s)", i+1, activityId),
			activityId:  activityId,
			expectError: false,
		}
	}

	useCase := createDeleteTestUseCaseWithAuth(businessType, false, true)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			req := &activitypb.DeleteActivityRequest{
				Data: &activitypb.Activity{
					Id: tc.activityId,
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
	testutil.LogTestResult(t, testCode, "MultipleValidActivities", true, nil)
}

func TestDeleteActivityUseCase_Execute_BusinessLogicValidation(t *testing.T) {
	testCode := "ESPYNA-TEST-WORKFLOW-ACTIVITY-DELETE-BUSINESS-LOGIC-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "BusinessLogicValidation", true)

	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createDeleteTestUseCaseWithAuth(businessType, false, true)

	// Load test case using centralized TestCaseResolver
	commonDataResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "activity", "Activity_CommonData")
	testutil.AssertTestCaseLoad(t, err, "Activity_CommonData")

	validationResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "activity", "ValidationError_NameTooLongGenerated")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_NameTooLongGenerated")

	tests := []struct {
		name          string
		activityData  *activitypb.Activity
		expectError   bool
		expectedError string
	}{
		{
			name: "Valid activity ID",
			activityData: &activitypb.Activity{
				Id: commonDataResolver.MustGetString("primaryActivityId"),
			},
			expectError: false,
		},
		{
			name: "Invalid activity ID format",
			activityData: &activitypb.Activity{
				Id: "invalid-id-format",
			},
			expectError: true,
		},
		{
			name: "Extremely long activity ID",
			activityData: &activitypb.Activity{
				Id: validationResolver.MustGetString("tooLongNameGenerated"),
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			req := &activitypb.DeleteActivityRequest{
				Data: tt.activityData,
			}

			response, err := useCase.Execute(ctx, req)

			if tt.expectError {
				testutil.AssertError(t, err)
				if tt.expectedError != "" {
					testutil.AssertStringEqual(t, tt.expectedError, err.Error(), "error message")
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
