//go:build mock_db && mock_auth

// Package activity provides table-driven tests for the activity listing use case.
//
// The tests cover various scenarios including successful listing, authorization,
// pagination, and filtering of activities.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user").
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education").
//
// Usage:
//   - Run all tests: go test -tags="mock_db,mock_auth" ./...
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestListActivitiesUseCase_Execute_TableDriven
//
// Test Codes:
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-LIST-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-LIST-TRANSACTION-SUCCESS-v1.0: WithTransactionSuccess
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-LIST-AUTHORIZATION-DENIED-v1.0: AuthorizationDenied
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-LIST-NIL-REQUEST-v1.0: NilRequest
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-LIST-VERIFY-DETAILS-v1.0: VerifyActivityDetails
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-LIST-STAGE-FILTERING-v1.0: StageFiltering
//   - ESPYNA-TEST-WORKFLOW-ACTIVITY-LIST-BUSINESS-LOGIC-VALIDATION-v1.0: BusinessLogicValidation
package activity

import (
	"context"
	"testing"

	copyatestutil "leapfor.xyz/copya/golang/testutil"
	"github.com/erniealice/espyna-golang/internal/application/shared/testutil"
	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock/workflow"
	activitypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/workflow/activity"
)

// Type alias for list activities test cases
type ListActivitiesTestCase = testutil.GenericTestCase[*activitypb.ListActivitiesRequest, *activitypb.ListActivitiesResponse]

func createListTestUseCaseWithAuth(businessType string, supportsTransaction bool, shouldAuthorize bool) *ListActivitiesUseCase {
	mockActivityRepo := workflow.NewMockActivityRepository(businessType)

	repositories := ListActivitiesRepositories{
		Activity: mockActivityRepo,
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, shouldAuthorize)
	services := ListActivitiesServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
	}

	return NewListActivitiesUseCase(repositories, services)
}

func TestListActivitiesUseCase_Execute_TableDriven(t *testing.T) {
	businessType := testutil.GetTestBusinessType()

	// Load test data resolvers before defining test cases
	listSuccessResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "activity", "ListActivities_Success")
	testutil.AssertTestCaseLoad(t, err, "ListActivities_Success")

	testCases := []ListActivitiesTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-WORKFLOW-ACTIVITY-LIST-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *activitypb.ListActivitiesRequest {
				return &activitypb.ListActivitiesRequest{}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *activitypb.ListActivitiesResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertNotNil(t, response.Data, "response data")
				expectedCount := listSuccessResolver.MustGetInt("expectedActivityCount")
				testutil.AssertEqual(t, expectedCount, len(response.Data), "activity count")
				expectedActivityIds := listSuccessResolver.MustGetStringArray("expectedActivityIds")
				activityIds := make(map[string]bool)
				for _, activity := range response.Data {
					activityIds[activity.Id] = true
				}
				for _, expectedId := range expectedActivityIds {
					testutil.AssertTrue(t, activityIds[expectedId], "expected activity '"+expectedId+"' found")
				}
			},
		},
		{
			Name:     "WithTransactionSuccess",
			TestCode: "ESPYNA-TEST-WORKFLOW-ACTIVITY-LIST-TRANSACTION-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *activitypb.ListActivitiesRequest {
				return &activitypb.ListActivitiesRequest{}
			},
			UseTransaction: true,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *activitypb.ListActivitiesResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				expectedCount := listSuccessResolver.MustGetInt("expectedActivityCount")
				testutil.AssertEqual(t, expectedCount, len(response.Data), "activity count with transaction")
			},
		},
		{
			Name:     "AuthorizationDenied",
			TestCode: "ESPYNA-TEST-WORKFLOW-ACTIVITY-LIST-AUTHORIZATION-DENIED-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *activitypb.ListActivitiesRequest {
				return &activitypb.ListActivitiesRequest{}
			},
			UseTransaction: false,
			UseAuth:        false,
			ExpectSuccess:  false,
			ExpectedError:  "activity.errors.authorization_failed",
			Assertions: func(t *testing.T, response *activitypb.ListActivitiesResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertAuthorizationError(t, err)
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-WORKFLOW-ACTIVITY-LIST-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *activitypb.ListActivitiesRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "activity.validation.request_required",
			Assertions: func(t *testing.T, response *activitypb.ListActivitiesResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilRequest(t, err)
			},
		},
		{
			Name:     "VerifyActivityDetails",
			TestCode: "ESPYNA-TEST-WORKFLOW-ACTIVITY-LIST-VERIFY-DETAILS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *activitypb.ListActivitiesRequest {
				return &activitypb.ListActivitiesRequest{}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *activitypb.ListActivitiesResponse, err error, useCase interface{}, ctx context.Context) {
				resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, testutil.GetTestBusinessType(), "activity", "ListActivities_VerifyDetails")
				testutil.AssertTestCaseLoad(t, err, "ListActivities_VerifyDetails")
				verificationTargetsRaw := resolver.GetTestCase().DataReferences["verificationTargets"]
				verificationTargets := testutil.AssertArray(t, verificationTargetsRaw, "verificationTargets")

				for _, targetInterface := range verificationTargets {
					target := targetInterface.(map[string]interface{})
					targetId := target["id"].(string)
					expectedName := target["expectedName"].(string)
					expectedStageId := target["expectedStageId"].(string)
					expectedActive := target["expectedActive"].(bool)

					// Find the activity in the response
					var foundActivity *activitypb.Activity
					for _, activity := range response.Data {
						if activity.Id == targetId {
							foundActivity = activity
							break
						}
					}

					testutil.AssertNotNil(t, foundActivity, targetId+" activity")
					if foundActivity != nil {
						testutil.AssertStringEqual(t, expectedName, foundActivity.Name, targetId+" activity name")
						testutil.AssertStringEqual(t, expectedStageId, foundActivity.StageId, targetId+" stage ID")
						testutil.AssertTrue(t, foundActivity.Active == expectedActive, targetId+" activity active")
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

func TestListActivitiesUseCase_Execute_VerifyActivityDetails(t *testing.T) {
	testCode := "ESPYNA-TEST-WORKFLOW-ACTIVITY-LIST-VERIFY-DETAILS-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "VerifyActivityDetails", true)

	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createListTestUseCaseWithAuth(businessType, false, true)

	// Load test case using centralized TestCaseResolver
	resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "activity", "ListActivities_VerifyDetails")
	testutil.AssertTestCaseLoad(t, err, "ListActivities_VerifyDetails")

	req := &activitypb.ListActivitiesRequest{}

	response, err := useCase.Execute(ctx, req)

	testutil.AssertNoError(t, err)

	testutil.AssertNotNil(t, response, "response")
	testutil.AssertGreaterThan(t, len(response.Data), 0, "activities count")

	// Find and verify specific activities using test data
	verificationTargetsRaw := resolver.GetTestCase().DataReferences["verificationTargets"]
	verificationTargets := testutil.AssertArray(t, verificationTargetsRaw, "verificationTargets")

	for _, targetInterface := range verificationTargets {
		target := targetInterface.(map[string]interface{})
		targetId := target["id"].(string)
		expectedName := target["expectedName"].(string)
		expectedStageId := target["expectedStageId"].(string)
		expectedActive := target["expectedActive"].(bool)

		// Find the activity in the response
		var foundActivity *activitypb.Activity
		for _, activity := range response.Data {
			if activity.Id == targetId {
				foundActivity = activity
				break
			}
		}

		testutil.AssertNotNil(t, foundActivity, targetId+" activity")
		if foundActivity != nil {
			testutil.AssertStringEqual(t, expectedName, foundActivity.Name, targetId+" activity name")
			testutil.AssertStringEqual(t, expectedStageId, foundActivity.StageId, targetId+" stage ID")
			testutil.AssertTrue(t, foundActivity.Active == expectedActive, targetId+" activity active")
		}
	}

	// Log test completion with result
	testutil.LogTestResult(t, testCode, "VerifyActivityDetails", true, nil)
}

func TestListActivitiesUseCase_Execute_StageFiltering(t *testing.T) {
	testCode := "ESPYNA-TEST-WORKFLOW-ACTIVITY-LIST-STAGE-FILTERING-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "StageFiltering", true)

	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createListTestUseCaseWithAuth(businessType, false, true)

	// Load test case using centralized TestCaseResolver
	resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "activity", "ListActivities_StageFiltering")
	testutil.AssertTestCaseLoad(t, err, "ListActivities_StageFiltering")

	req := &activitypb.ListActivitiesRequest{}

	response, err := useCase.Execute(ctx, req)

	testutil.AssertNoError(t, err)

	testutil.AssertNotNil(t, response, "response")
	testutil.AssertGreaterThan(t, len(response.Data), 0, "activities count")

	// Verify activities are associated with different stages
	stageCounts := make(map[string]int)
	for _, activity := range response.Data {
		stageCounts[activity.StageId]++
	}

	// Should have activities from multiple stages based on mock data
	minExpectedStages := resolver.MustGetInt("minExpectedStages")
	testutil.AssertGreaterThanOrEqual(t, len(stageCounts), minExpectedStages, "stage diversity")

	// Get stage associations from test data
	expectedStageAssociationsRaw := resolver.GetTestCase().DataReferences["expectedStageAssociations"]
	expectedStageAssociationsInterface := testutil.AssertMap(t, expectedStageAssociationsRaw, "expectedStageAssociations")
	expectedStageAssociations := make(map[string]string)
	for key, value := range expectedStageAssociationsInterface {
		expectedStageAssociations[key] = value.(string)
	}

	for _, activity := range response.Data {
		expectedStage, exists := expectedStageAssociations[activity.Id]
		if exists {
			testutil.AssertStringEqual(t, expectedStage, activity.StageId, "stage association for "+activity.Id)
		}
	}

	// Log test completion with result
	testutil.LogTestResult(t, testCode, "StageFiltering", true, nil)
}

func TestListActivitiesUseCase_Execute_BusinessLogicValidation(t *testing.T) {
	testCode := "ESPYNA-TEST-WORKFLOW-ACTIVITY-LIST-BUSINESS-LOGIC-VALIDATION-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "BusinessLogicValidation", true)

	// Load test case using centralized TestCaseResolver
	resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, testutil.GetTestBusinessType(), "activity", "ListActivities_BusinessLogic")
	testutil.AssertTestCaseLoad(t, err, "ListActivities_BusinessLogic")

	testCases := resolver.GetTestCase().DataReferences["testCases"].([]interface{})

	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createListTestUseCaseWithAuth(businessType, false, true)

	for _, testCaseInterface := range testCases {
		testCase := testCaseInterface.(map[string]interface{})
		t.Run(testCase["name"].(string), func(t *testing.T) {
			var request *activitypb.ListActivitiesRequest
			if testCase["request"] == nil {
				request = nil
			} else {
				request = &activitypb.ListActivitiesRequest{}
			}

			response, err := useCase.Execute(ctx, request)

			expectError := testCase["expectError"].(bool)
			minActivities := int(testCase["minActivities"].(float64))

			if expectError {
				testutil.AssertError(t, err)
				testutil.AssertNil(t, response, "response for "+testCase["name"].(string))
			} else {
				testutil.AssertNoError(t, err)
				testutil.AssertNotNil(t, response, "response for "+testCase["name"].(string))
				if response != nil {
					testutil.AssertGreaterThanOrEqual(t, len(response.Data), minActivities, "activity count for "+testCase["name"].(string))
				}
			}
		})
	}

	// Log test completion with result
	testutil.LogTestResult(t, testCode, "BusinessLogicValidation", true, nil)
}
