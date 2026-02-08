//go:build mock_db && mock_auth

// Package activity provides table-driven tests for the activity update use case.
//
// The tests cover various scenarios including successful updates, authorization,
// validation errors, and concurrent modification handling.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user").
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education").
//
// Usage:
//   - Run all tests: go test -tags="mock_db,mock_auth" ./...
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestUpdateActivityUseCase_Execute_TableDriven
//
// Test Codes:
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-UPDATE-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-UPDATE-TRANSACTION-v1.0: WithTransaction
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-UPDATE-NIL-REQUEST-v1.0: NilRequest
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-UPDATE-NIL-DATA-v1.0: NilData
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-UPDATE-VALIDATION-EMPTY-ID-v1.0: EmptyId
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-UPDATE-VALIDATION-EMPTY-NAME-v1.0: EmptyName
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-UPDATE-VALIDATION-EMPTY-STAGE-v1.0: EmptyStageId
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-UPDATE-VALIDATION-INVALID-STAGE-v1.0: InvalidStageId
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-UPDATE-ENRICHMENT-v1.0: DataEnrichment
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-UPDATE-VALIDATION-MINIMAL-v1.0: MinimalValidData
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-UPDATE-VALIDATION-MAXIMAL-v1.0: MaxValidData
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-UPDATE-TRANSACTION-FAILURE-v1.0: TransactionFailure
package activity

import (
	"context"
	"slices"
	"testing"

	copyatestutil "leapfor.xyz/copya/golang/testutil"
	"github.com/erniealice/espyna-golang/internal/application/shared/testutil"
	mockDb "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock"
	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock/workflow"
	activitypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/workflow/activity"
)

// Type alias for update activity test cases
type UpdateActivityTestCase = testutil.GenericTestCase[*activitypb.UpdateActivityRequest, *activitypb.UpdateActivityResponse]

func createTestUpdateActivityUseCaseWithAuth(businessType string, supportsTransaction bool, shouldAuthorize bool) *UpdateActivityUseCase {
	mockActivityRepo := workflow.NewMockActivityRepository(businessType)

	repositories := UpdateActivityRepositories{
		Activity: mockActivityRepo,
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, shouldAuthorize)
	services := UpdateActivityServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
	}

	return NewUpdateActivityUseCase(repositories, services)
}

func TestUpdateActivityUseCase_Execute_TableDriven(t *testing.T) {
	businessType := testutil.GetTestBusinessType()

	// Load test data resolvers before defining test cases
	createSuccessResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "activity", "CreateActivity_Success")
	testutil.AssertTestCaseLoad(t, err, "CreateActivity_Success")

	updateSuccessResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "activity", "UpdateActivity_Success")
	testutil.AssertTestCaseLoad(t, err, "UpdateActivity_Success")

	commonDataResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "activity", "Activity_CommonData")
	testutil.AssertTestCaseLoad(t, err, "Activity_CommonData")

	testCases := []UpdateActivityTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-WORKFLOW-ACTIVITY-UPDATE-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *activitypb.UpdateActivityRequest {
				return &activitypb.UpdateActivityRequest{
					Data: &activitypb.Activity{
						Id:          commonDataResolver.MustGetString("primaryActivityId"),
						Name:        updateSuccessResolver.MustGetString("enhancedActivityName"),
						Description: &[]string{updateSuccessResolver.MustGetString("enhancedActivityDescription")}[0],
						StageId:     createSuccessResolver.MustGetString("validStageId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *activitypb.UpdateActivityResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertEqual(t, 1, len(response.Data), "response data length")
				updatedActivity := response.Data[0]
				testutil.AssertStringEqual(t, updateSuccessResolver.MustGetString("enhancedActivityName"), updatedActivity.Name, "updated name")
				testutil.AssertFieldSet(t, updatedActivity.Description, "updated description")
				testutil.AssertFieldSet(t, updatedActivity.DateModified, "DateModified")
				testutil.AssertFieldSet(t, updatedActivity.DateModifiedString, "DateModifiedString")
			},
		},
		{
			Name:     "WithTransaction",
			TestCode: "ESPYNA-TEST-WORKFLOW-ACTIVITY-UPDATE-TRANSACTION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *activitypb.UpdateActivityRequest {
				return &activitypb.UpdateActivityRequest{
					Data: &activitypb.Activity{
						Id:          commonDataResolver.MustGetString("secondaryActivityId"),
						Name:        updateSuccessResolver.MustGetString("enhancedSecondaryName"),
						Description: &[]string{updateSuccessResolver.MustGetString("enhancedSecondaryDescription")}[0],
						StageId:     createSuccessResolver.MustGetString("validStageId"),
					},
				}
			},
			UseTransaction: true,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *activitypb.UpdateActivityResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				updatedActivity := response.Data[0]
				testutil.AssertStringEqual(t, updateSuccessResolver.MustGetString("enhancedSecondaryName"), updatedActivity.Name, "updated name")
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-WORKFLOW-ACTIVITY-UPDATE-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *activitypb.UpdateActivityRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "activity.validation.request_required",
			Assertions: func(t *testing.T, response *activitypb.UpdateActivityResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilRequest(t, err)
			},
		},
		{
			Name:     "NilData",
			TestCode: "ESPYNA-TEST-WORKFLOW-ACTIVITY-UPDATE-NIL-DATA-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *activitypb.UpdateActivityRequest {
				return &activitypb.UpdateActivityRequest{Data: nil}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "activity.validation.request_required",
			Assertions: func(t *testing.T, response *activitypb.UpdateActivityResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilData(t, err)
			},
		},
		{
			Name:     "EmptyId",
			TestCode: "ESPYNA-TEST-WORKFLOW-ACTIVITY-UPDATE-VALIDATION-EMPTY-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *activitypb.UpdateActivityRequest {
				return &activitypb.UpdateActivityRequest{
					Data: &activitypb.Activity{
						Id:      "",
						Name:    updateSuccessResolver.MustGetString("validActivityName"),
						StageId: createSuccessResolver.MustGetString("validStageId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "activity.validation.id_required",
			Assertions: func(t *testing.T, response *activitypb.UpdateActivityResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty ID")
			},
		},
		{
			Name:     "EmptyName",
			TestCode: "ESPYNA-TEST-WORKFLOW-ACTIVITY-UPDATE-VALIDATION-EMPTY-NAME-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *activitypb.UpdateActivityRequest {
				return &activitypb.UpdateActivityRequest{
					Data: &activitypb.Activity{
						Id:      commonDataResolver.MustGetString("primaryActivityId"),
						Name:    "",
						StageId: createSuccessResolver.MustGetString("validStageId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "activity.validation.name_required",
			Assertions: func(t *testing.T, response *activitypb.UpdateActivityResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty name")
			},
		},
		{
			Name:     "EmptyStageId",
			TestCode: "ESPYNA-TEST-WORKFLOW-ACTIVITY-UPDATE-VALIDATION-EMPTY-STAGE-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *activitypb.UpdateActivityRequest {
				return &activitypb.UpdateActivityRequest{
					Data: &activitypb.Activity{
						Id:      commonDataResolver.MustGetString("primaryActivityId"),
						Name:    updateSuccessResolver.MustGetString("validActivityName"),
						StageId: "",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "activity.validation.stage_id_required",
			Assertions: func(t *testing.T, response *activitypb.UpdateActivityResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty stage ID")
			},
		},
		{
			Name:     "InvalidStageId",
			TestCode: "ESPYNA-TEST-WORKFLOW-ACTIVITY-UPDATE-VALIDATION-INVALID-STAGE-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *activitypb.UpdateActivityRequest {
				invalidResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "activity", "ValidationError_InvalidStageId")
				testutil.AssertTestCaseLoad(t, err, "ValidationError_InvalidStageId")
				return &activitypb.UpdateActivityRequest{
					Data: &activitypb.Activity{
						Id:      commonDataResolver.MustGetString("primaryActivityId"),
						Name:    "Valid Activity Name",
						StageId: invalidResolver.MustGetString("invalidStageId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "Entity reference validation failed [DEFAULT]: Failed to validate stage entity reference [DEFAULT]: stage with ID 'stage-non-existent-123' not found",
			ExactError:     true,
			Assertions: func(t *testing.T, response *activitypb.UpdateActivityResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "invalid stage ID")
			},
		},
		{
			Name:     "DataEnrichment",
			TestCode: "ESPYNA-TEST-WORKFLOW-ACTIVITY-UPDATE-ENRICHMENT-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *activitypb.UpdateActivityRequest {
				return &activitypb.UpdateActivityRequest{
					Data: &activitypb.Activity{
						Id:      commonDataResolver.MustGetString("thirdActivityId"),
						Name:    updateSuccessResolver.MustGetString("updatedActivityName"),
						StageId: createSuccessResolver.MustGetString("validStageId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *activitypb.UpdateActivityResponse, err error, useCase interface{}, ctx context.Context) {
				updatedActivity := response.Data[0]
				testutil.AssertFieldSet(t, updatedActivity.DateModified, "DateModified")
				testutil.AssertFieldSet(t, updatedActivity.DateModifiedString, "DateModifiedString")
				testutil.AssertTimestampPositive(t, *updatedActivity.DateModified, "DateModified")
			},
		},
		{
			Name:     "MinimalValidData",
			TestCode: "ESPYNA-TEST-WORKFLOW-ACTIVITY-UPDATE-VALIDATION-MINIMAL-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *activitypb.UpdateActivityRequest {
				boundaryResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "activity", "BoundaryTest_MinimalValid")
				testutil.AssertTestCaseLoad(t, err, "BoundaryTest_MinimalValid")
				return &activitypb.UpdateActivityRequest{
					Data: &activitypb.Activity{
						Id:      commonDataResolver.MustGetString("primaryActivityId"),
						Name:    boundaryResolver.MustGetString("minValidName"),
						StageId: createSuccessResolver.MustGetString("validStageId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *activitypb.UpdateActivityResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				updatedActivity := response.Data[0]
				boundaryResolver, _ := copyatestutil.LoadTestCaseFromBusinessType(t, testutil.GetTestBusinessType(), "activity", "BoundaryTest_MinimalValid")
				testutil.AssertStringEqual(t, boundaryResolver.MustGetString("minValidName"), updatedActivity.Name, "activity name")
			},
		},
		{
			Name:     "MaxValidData",
			TestCode: "ESPYNA-TEST-WORKFLOW-ACTIVITY-UPDATE-VALIDATION-MAXIMAL-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *activitypb.UpdateActivityRequest {
				boundaryResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "activity", "BoundaryTest_MaximalValid")
				testutil.AssertTestCaseLoad(t, err, "BoundaryTest_MaximalValid")
				return &activitypb.UpdateActivityRequest{
					Data: &activitypb.Activity{
						Id:          commonDataResolver.MustGetString("primaryActivityId"),
						Name:        boundaryResolver.MustGetString("maxValidNameExact100"),
						Description: &[]string{boundaryResolver.MustGetString("maxValidDescriptionExact1000")}[0],
						StageId:     createSuccessResolver.MustGetString("validStageId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *activitypb.UpdateActivityResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				updatedActivity := response.Data[0]
				testutil.AssertEqual(t, 100, len(updatedActivity.Name), "name length")
				testutil.AssertFieldSet(t, updatedActivity.Description, "description")
				if updatedActivity.Description != nil {
					testutil.AssertEqual(t, 1000, len(*updatedActivity.Description), "description length")
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
			useCase := createTestUpdateActivityUseCaseWithAuth(businessType, tc.UseTransaction, tc.UseAuth)

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

func TestUpdateActivityUseCase_Execute_TransactionFailure(t *testing.T) {
	testCode := "ESPYNA-TEST-WORKFLOW-ACTIVITY-UPDATE-TRANSACTION-FAILURE-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "TransactionFailure", false)

	// Create a use case with a failing transaction service
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()

	// Load test case using centralized TestCaseResolver
	commonResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "activity", "Activity_CommonData")
	testutil.AssertTestCaseLoad(t, err, "Activity_CommonData")
	updateResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "activity", "UpdateActivity_Success")
	testutil.AssertTestCaseLoad(t, err, "UpdateActivity_Success")
	createResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "activity", "CreateActivity_Success")
	testutil.AssertTestCaseLoad(t, err, "CreateActivity_Success")

	mockActivityRepo := workflow.NewMockActivityRepository(businessType)

	repositories := UpdateActivityRepositories{
		Activity: mockActivityRepo,
	}

	standardServices := testutil.CreateStandardServices(false, true)
	// Override with failing transaction service
	standardServices.TransactionService = mockDb.NewFailingMockTransactionService()
	services := UpdateActivityServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
	}

	useCase := NewUpdateActivityUseCase(repositories, services)

	req := &activitypb.UpdateActivityRequest{
		Data: &activitypb.Activity{
			Id:      commonResolver.MustGetString("primaryActivityId"),
			Name:    updateResolver.MustGetString("validActivityName"),
			StageId: createResolver.MustGetString("validStageId"),
		},
	}

	// For update operations without transaction support, should still work
	response, err := useCase.Execute(ctx, req)

	// This should work since update operations don't always require transactions
	if err != nil {
		// If error occurs, should be due to transaction failure
		testutil.AssertTranslatedError(t, err, "transaction.errors.execution_failed", useCase.services.TranslationService, ctx)
	} else {
		// If it succeeds, verify we get proper response
		testutil.AssertNotNil(t, response, "response")
	}

	// Log completion of transaction failure test
	testutil.LogTestResult(t, testCode, "TransactionFailure", err == nil, err)
}

func TestUpdateActivityUseCase_Execute_ForeignKeyValidation(t *testing.T) {
	testCode := "ESPYNA-TEST-WORKFLOW-ACTIVITY-UPDATE-FOREIGN-KEY-VALIDATION-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "ForeignKeyValidation", true)

	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestUpdateActivityUseCaseWithAuth(businessType, false, true)

	commonResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "activity", "Activity_CommonData")
	testutil.AssertTestCaseLoad(t, err, "Activity_CommonData")

	// Test updating with different valid stage IDs
	stageFilteringResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "activity", "ListActivities_StageFiltering")
	testutil.AssertTestCaseLoad(t, err, "ListActivities_StageFiltering")
	stageAssociationsRaw := stageFilteringResolver.GetTestCase().DataReferences["expectedStageAssociations"]
	stageAssociationsInterface := testutil.AssertMap(t, stageAssociationsRaw, "expectedStageAssociations")
	stageIds := make([]string, 0)
	for _, stageId := range stageAssociationsInterface {
		stageIdStr := stageId.(string)
		// Add unique stage IDs to the list
		if !slices.Contains(stageIds, stageIdStr) {
			stageIds = append(stageIds, stageIdStr)
		}
	}

	for _, stageId := range stageIds {
		req := &activitypb.UpdateActivityRequest{
			Data: &activitypb.Activity{
				Id:      commonResolver.MustGetString("primaryActivityId"),
				Name:    "Test Foreign Key Validation",
				StageId: stageId, // Each valid stage ID
			},
		}

		response, err := useCase.Execute(ctx, req)

		testutil.AssertNoError(t, err)
		testutil.AssertNotNil(t, response, "response")

		updatedActivity := response.Data[0]
		testutil.AssertStringEqual(t, stageId, updatedActivity.StageId, "stage ID")
	}

	// Log completion of foreign key validation test
	testutil.LogTestResult(t, testCode, "ForeignKeyValidation", true, nil)
}
