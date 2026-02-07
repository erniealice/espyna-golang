//go:build mock_db && mock_auth

// Package activity provides table-driven tests for the activity reading use case.
//
// The tests cover various scenarios including successful reads, authorization,
// not found cases, and validation of activity data retrieval.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user").
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education").
//
// Usage:
//   - Run all tests: go test -tags="mock_db,mock_auth" ./...
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestReadActivityUseCase_Execute_TableDriven
//
// Test Codes:
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-READ-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-READ-TRANSACTION-v1.0: WithTransaction
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-READ-NIL-REQUEST-v1.0: NilRequest
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-READ-NIL-DATA-v1.0: NilData
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-READ-VALIDATION-EMPTY-ID-v1.0: EmptyId
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-READ-NOT-FOUND-v1.0: NonExistentId
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-READ-VALIDATION-MINIMAL-ID-v1.0: ValidMinimalId
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-READ-INTEGRATION-v1.0: RealisticDomainActivity
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-READ-STRUCTURE-VALIDATION-v1.0: ActivityStructureValidation
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-READ-TRANSACTION-FAILURE-v1.0: TransactionFailure
package activity

import (
	"context"
	"testing"

	copyatestutil "leapfor.xyz/copya/golang/testutil"
	"leapfor.xyz/espyna/internal/application/shared/testutil"
	mockDb "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock"
	"leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock/workflow"
	activitypb "leapfor.xyz/esqyma/golang/v1/domain/workflow/activity"
)

// Type alias for read activity test cases
type ReadActivityTestCase = testutil.GenericTestCase[*activitypb.ReadActivityRequest, *activitypb.ReadActivityResponse]

func createTestReadActivityUseCaseWithAuth(businessType string, supportsTransaction bool, shouldAuthorize bool) *ReadActivityUseCase {
	mockActivityRepo := workflow.NewMockActivityRepository(businessType)

	repositories := ReadActivityRepositories{
		Activity: mockActivityRepo,
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, shouldAuthorize)
	services := ReadActivityServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
	}

	return NewReadActivityUseCase(repositories, services)
}

func TestReadActivityUseCase_Execute_TableDriven(t *testing.T) {
	businessType := testutil.GetTestBusinessType()

	// Load test data resolvers before defining test cases
	commonDataResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "activity", "Activity_CommonData")
	testutil.AssertTestCaseLoad(t, err, "Activity_CommonData")

	readSuccessResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "activity", "ReadActivity_Success")
	testutil.AssertTestCaseLoad(t, err, "ReadActivity_Success")

	testCases := []ReadActivityTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-WORKFLOW-ACTIVITY-READ-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *activitypb.ReadActivityRequest {
				return &activitypb.ReadActivityRequest{
					Data: &activitypb.Activity{
						Id: readSuccessResolver.MustGetString("targetActivityId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *activitypb.ReadActivityResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertEqual(t, 1, len(response.Data), "response data length")
				activity := response.Data[0]
				testutil.AssertNonEmptyString(t, activity.Name, "activity name")
				testutil.AssertNonEmptyString(t, activity.StageId, "stage ID")
				testutil.AssertFieldSet(t, activity.DateCreated, "DateCreated")
				testutil.AssertFieldSet(t, activity.DateCreatedString, "DateCreatedString")
			},
		},
		{
			Name:     "WithTransaction",
			TestCode: "ESPYNA-TEST-WORKFLOW-ACTIVITY-READ-TRANSACTION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *activitypb.ReadActivityRequest {
				return &activitypb.ReadActivityRequest{
					Data: &activitypb.Activity{
						Id: commonDataResolver.MustGetString("secondaryActivityId"),
					},
				}
			},
			UseTransaction: true,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *activitypb.ReadActivityResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				activity := response.Data[0]
				testutil.AssertStringEqual(t, commonDataResolver.MustGetString("secondaryActivityId"), activity.Id, "activity ID")
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-WORKFLOW-ACTIVITY-READ-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *activitypb.ReadActivityRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "activity.validation.request_required",
			Assertions: func(t *testing.T, response *activitypb.ReadActivityResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilRequest(t, err)
			},
		},
		{
			Name:     "NilData",
			TestCode: "ESPYNA-TEST-WORKFLOW-ACTIVITY-READ-NIL-DATA-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *activitypb.ReadActivityRequest {
				return &activitypb.ReadActivityRequest{Data: nil}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "activity.validation.request_required",
			Assertions: func(t *testing.T, response *activitypb.ReadActivityResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilData(t, err)
			},
		},
		{
			Name:     "EmptyId",
			TestCode: "ESPYNA-TEST-WORKFLOW-ACTIVITY-READ-VALIDATION-EMPTY-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *activitypb.ReadActivityRequest {
				return &activitypb.ReadActivityRequest{
					Data: &activitypb.Activity{Id: ""},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "activity.validation.id_required",
			Assertions: func(t *testing.T, response *activitypb.ReadActivityResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty ID")
			},
		},
		{
			Name:     "NonExistentId",
			TestCode: "ESPYNA-TEST-WORKFLOW-ACTIVITY-READ-NOT-FOUND-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *activitypb.ReadActivityRequest {
				return &activitypb.ReadActivityRequest{
					Data: &activitypb.Activity{Id: commonDataResolver.MustGetString("nonExistentId")},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "activity.errors.not_found",
			ErrorTags:      map[string]any{"activityId": commonDataResolver.MustGetString("nonExistentId")},
			Assertions: func(t *testing.T, response *activitypb.ReadActivityResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertError(t, err)
			},
		},
		{
			Name:     "ValidMinimalId",
			TestCode: "ESPYNA-TEST-WORKFLOW-ACTIVITY-READ-VALIDATION-MINIMAL-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *activitypb.ReadActivityRequest {
				return &activitypb.ReadActivityRequest{
					Data: &activitypb.Activity{Id: commonDataResolver.MustGetString("minimalValidId")},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "activity.errors.not_found",
			ErrorTags:      map[string]any{"activityId": commonDataResolver.MustGetString("minimalValidId")},
			Assertions: func(t *testing.T, response *activitypb.ReadActivityResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertError(t, err)
			},
		},
		{
			Name:     "RealisticDomainActivity",
			TestCode: "ESPYNA-TEST-WORKFLOW-ACTIVITY-READ-INTEGRATION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *activitypb.ReadActivityRequest {
				return &activitypb.ReadActivityRequest{
					Data: &activitypb.Activity{
						Id: commonDataResolver.MustGetString("thirdActivityId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *activitypb.ReadActivityResponse, err error, useCase interface{}, ctx context.Context) {
				activity := response.Data[0]
				testutil.AssertNonEmptyString(t, activity.Name, "activity name")
				testutil.AssertFieldSet(t, activity.Description, "description")
				testutil.AssertNonEmptyString(t, activity.StageId, "stage ID")
				testutil.AssertNonEmptyString(t, activity.StageId, "stage linkage")
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
			useCase := createTestReadActivityUseCaseWithAuth(businessType, tc.UseTransaction, tc.UseAuth)

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

func TestReadActivityUseCase_Execute_ActivityStructureValidation(t *testing.T) {
	testCode := "ESPYNA-TEST-WORKFLOW-ACTIVITY-READ-STRUCTURE-VALIDATION-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "ActivityStructureValidation", true)

	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestReadActivityUseCaseWithAuth(businessType, false, true)

	// Load test case using centralized TestCaseResolver
	resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "activity", "ListActivities_Success")
	testutil.AssertTestCaseLoad(t, err, "ListActivities_Success")

	// Test with multiple real activity IDs from mock data
	activityIds := resolver.MustGetStringArray("expectedActivityIds")

	for _, activityId := range activityIds {
		req := &activitypb.ReadActivityRequest{
			Data: &activitypb.Activity{
				Id: activityId,
			},
		}

		response, err := useCase.Execute(ctx, req)

		testutil.AssertNoError(t, err)

		activity := response.Data[0]

		// Validate activity structure
		testutil.AssertStringEqual(t, activityId, activity.Id, "activity ID")

		testutil.AssertNonEmptyString(t, activity.Name, "activity name")

		testutil.AssertTrue(t, activity.Active, "activity active status")

		testutil.AssertNonEmptyString(t, activity.StageId, "stage ID")

		// Audit fields
		testutil.AssertFieldSet(t, activity.DateCreated, "DateCreated")

		testutil.AssertFieldSet(t, activity.DateCreatedString, "DateCreatedString")
	}

	// Log completion of structure validation test
	testutil.LogTestResult(t, testCode, "ActivityStructureValidation", true, nil)
}

func TestReadActivityUseCase_Execute_TransactionFailure(t *testing.T) {
	testCode := "ESPYNA-TEST-WORKFLOW-ACTIVITY-READ-TRANSACTION-FAILURE-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "TransactionFailure", false)

	// Create a use case with a failing transaction service
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	mockActivityRepo := workflow.NewMockActivityRepository(businessType)

	repositories := ReadActivityRepositories{
		Activity: mockActivityRepo,
	}

	standardServices := testutil.CreateStandardServices(false, true)
	// Override with failing transaction service
	standardServices.TransactionService = mockDb.NewFailingMockTransactionService()
	services := ReadActivityServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
	}

	useCase := NewReadActivityUseCase(repositories, services)

	// Load test case using centralized TestCaseResolver
	resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "activity", "Activity_CommonData")
	testutil.AssertTestCaseLoad(t, err, "Activity_CommonData")

	req := &activitypb.ReadActivityRequest{
		Data: &activitypb.Activity{
			Id: resolver.MustGetString("nonExistentId"),
		},
	}

	// For read operations, transaction failure should not affect the operation
	// since read operations typically don't use transactions
	response, err := useCase.Execute(ctx, req)

	// This should either work (no transaction used) or fail gracefully
	if err != nil {
		// If it fails, verify it's due to the activity not existing, not transaction failure
		testutil.AssertTranslatedErrorWithTags(t, err, "activity.errors.not_found",
			map[string]any{"activityId": resolver.MustGetString("nonExistentId")}, useCase.services.TranslationService, ctx)
	} else {
		// If it succeeds, verify we get a proper response
		testutil.AssertNotNil(t, response, "response")
	}

	// Log completion of transaction failure test
	testutil.LogTestResult(t, testCode, "TransactionFailure", err == nil, err)
}
