//go:build mock_db && mock_auth

// Package stage provides table-driven tests for the stage deletion use case.
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
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestDeleteStageUseCase_Execute_TableDriven
//
// Test Codes:
//   - ESPYNA-TEST-WORKFLOW-STAGE-DELETE-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-WORKFLOW-STAGE-DELETE-AUTHORIZATION-v1.0: Unauthorized
//   - ESPYNA-TEST-WORKFLOW-STAGE-DELETE-NOT-FOUND-v1.0: NotFound
//   - ESPYNA-TEST-WORKFLOW-STAGE-DELETE-ALREADY-DELETED-v1.0: AlreadyDeleted
//   - ESPYNA-TEST-WORKFLOW-STAGE-DELETE-HAS-DEPENDENCIES-v1.0: HasDependencies
package stage

import (
	"context"
	"testing"

	copyatestutil "leapfor.xyz/copya/golang/testutil"
	"leapfor.xyz/espyna/internal/application/shared/testutil"
	"leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock/workflow"
	stagepb "leapfor.xyz/esqyma/golang/v1/domain/workflow/stage"
)

// Type alias for delete stage test cases
type DeleteStageTestCase = testutil.GenericTestCase[*stagepb.DeleteStageRequest, *stagepb.DeleteStageResponse]

func createDeleteStageTestUseCaseWithAuth(businessType string, supportsTransaction bool, shouldAuthorize bool) *DeleteStageUseCase {
	mockStageRepo := workflow.NewMockStageRepository(businessType)

	repositories := DeleteStageRepositories{
		Stage: mockStageRepo,
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, shouldAuthorize)
	services := DeleteStageServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
	}

	return NewDeleteStageUseCase(repositories, services)
}

func TestDeleteStageUseCase_Execute_TableDriven(t *testing.T) {
	businessType := testutil.GetTestBusinessType()

	// Load test data resolvers before defining test cases
	deleteSuccessResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "stage", "DeleteStage_Success")
	testutil.AssertTestCaseLoad(t, err, "DeleteStage_Success")

	authorizationUnauthorizedResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "stage", "Authorization_Unauthorized")
	testutil.AssertTestCaseLoad(t, err, "Authorization_Unauthorized")

	notFoundResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "stage", "DeleteStage_NotFound")
	testutil.AssertTestCaseLoad(t, err, "DeleteStage_NotFound")

	alreadyDeletedResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "stage", "DeleteStage_AlreadyDeleted")
	testutil.AssertTestCaseLoad(t, err, "DeleteStage_AlreadyDeleted")

	hasDependenciesResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "stage", "DeleteStage_HasDependencies")
	testutil.AssertTestCaseLoad(t, err, "DeleteStage_HasDependencies")

	testCases := []DeleteStageTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-WORKFLOW-STAGE-DELETE-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *stagepb.DeleteStageRequest {
				return &stagepb.DeleteStageRequest{
					Data: &stagepb.Stage{
						Id: deleteSuccessResolver.MustGetString("existingStageId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *stagepb.DeleteStageResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "successful deletion")
			},
		},
		{
			Name:     "Unauthorized",
			TestCode: "ESPYNA-TEST-WORKFLOW-STAGE-DELETE-AUTHORIZATION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *stagepb.DeleteStageRequest {
				return &stagepb.DeleteStageRequest{
					Data: &stagepb.Stage{
						Id: authorizationUnauthorizedResolver.MustGetString("restrictedStageId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        false,
			ExpectSuccess:  false,
			ExpectedError:  "stage.errors.authorization_failed",
			Assertions: func(t *testing.T, response *stagepb.DeleteStageResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertAuthorizationError(t, err)
			},
		},
		{
			Name:     "NotFound",
			TestCode: "ESPYNA-TEST-WORKFLOW-STAGE-DELETE-NOT-FOUND-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *stagepb.DeleteStageRequest {
				return &stagepb.DeleteStageRequest{
					Data: &stagepb.Stage{
						Id: notFoundResolver.MustGetString("nonExistentStageId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "stage.errors.not_found",
			Assertions: func(t *testing.T, response *stagepb.DeleteStageResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertNotFoundError(t, err)
			},
		},
		{
			Name:     "AlreadyDeleted",
			TestCode: "ESPYNA-TEST-WORKFLOW-STAGE-DELETE-ALREADY-DELETED-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *stagepb.DeleteStageRequest {
				return &stagepb.DeleteStageRequest{
					Data: &stagepb.Stage{
						Id: alreadyDeletedResolver.MustGetString("alreadyDeletedStageId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "stage.errors.already_deleted",
			Assertions: func(t *testing.T, response *stagepb.DeleteStageResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertAlreadyDeletedError(t, err)
			},
		},
		{
			Name:     "HasDependencies",
			TestCode: "ESPYNA-TEST-WORKFLOW-STAGE-DELETE-HAS-DEPENDENCIES-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *stagepb.DeleteStageRequest {
				return &stagepb.DeleteStageRequest{
					Data: &stagepb.Stage{
						Id: hasDependenciesResolver.MustGetString("stageWithDependenciesId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "stage.errors.has_dependencies",
			Assertions: func(t *testing.T, response *stagepb.DeleteStageResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertHasDependenciesError(t, err)
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
			useCase := createDeleteStageTestUseCaseWithAuth(businessType, tc.UseTransaction, tc.UseAuth)

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
