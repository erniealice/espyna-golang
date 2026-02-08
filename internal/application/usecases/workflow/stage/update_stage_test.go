//go:build mock_db && mock_auth

// Package stage provides table-driven tests for the stage update use case.
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
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestUpdateStageUseCase_Execute_TableDriven
//
// Test Codes:
//   - ESPYNA-TEST-WORKFLOW-STAGE-UPDATE-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-WORKFLOW-STAGE-UPDATE-AUTHORIZATION-v1.0: Unauthorized
//   - ESPYNA-TEST-WORKFLOW-STAGE-UPDATE-NOT-FOUND-v1.0: NotFound
//   - ESPYNA-TEST-WORKFLOW-STAGE-UPDATE-VALIDATION-EMPTY-NAME-v1.0: EmptyName
//   - ESPYNA-TEST-WORKFLOW-STAGE-UPDATE-VALIDATION-NAME-TOO-SHORT-v1.0: NameTooShort
//   - ESPYNA-TEST-WORKFLOW-STAGE-UPDATE-CONCURRENT-MODIFICATION-v1.0: ConcurrentModification
package stage

import (
	"context"
	"testing"

	copyatestutil "leapfor.xyz/copya/golang/testutil"
	"github.com/erniealice/espyna-golang/internal/application/shared/testutil"
	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock/workflow"
	stagepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/workflow/stage"
)

// Type alias for update stage test cases
type UpdateStageTestCase = testutil.GenericTestCase[*stagepb.UpdateStageRequest, *stagepb.UpdateStageResponse]

func createUpdateStageTestUseCaseWithAuth(businessType string, supportsTransaction bool, shouldAuthorize bool) *UpdateStageUseCase {
	mockStageRepo := workflow.NewMockStageRepository(businessType)

	repositories := UpdateStageRepositories{
		Stage: mockStageRepo,
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, shouldAuthorize)
	services := UpdateStageServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
	}

	return NewUpdateStageUseCase(repositories, services)
}

func TestUpdateStageUseCase_Execute_TableDriven(t *testing.T) {
	businessType := testutil.GetTestBusinessType()

	// Load test data resolvers before defining test cases
	updateSuccessResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "stage", "UpdateStage_Success")
	testutil.AssertTestCaseLoad(t, err, "UpdateStage_Success")

	authorizationUnauthorizedResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "stage", "Authorization_Unauthorized")
	testutil.AssertTestCaseLoad(t, err, "Authorization_Unauthorized")

	notFoundResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "stage", "UpdateStage_NotFound")
	testutil.AssertTestCaseLoad(t, err, "UpdateStage_NotFound")

	validationErrorEmptyNameResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "stage", "ValidationError_EmptyName")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_EmptyName")

	concurrentModificationResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "stage", "ConcurrentModification")
	testutil.AssertTestCaseLoad(t, err, "ConcurrentModification")

	testCases := []UpdateStageTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-WORKFLOW-STAGE-UPDATE-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *stagepb.UpdateStageRequest {
				return &stagepb.UpdateStageRequest{
					Data: &stagepb.Stage{
						Id:          updateSuccessResolver.MustGetString("existingStageId"),
						Name:        updateSuccessResolver.MustGetString("updatedStageName"),
						Description: &[]string{updateSuccessResolver.MustGetString("updatedStageDescription")}[0],
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *stagepb.UpdateStageResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertEqual(t, 1, len(response.Data), "response data length")
				updatedStage := response.Data[0]
				testutil.AssertStringEqual(t, updateSuccessResolver.MustGetString("existingStageId"), updatedStage.Id, "stage ID")
				testutil.AssertStringEqual(t, updateSuccessResolver.MustGetString("updatedStageName"), updatedStage.Name, "updated stage name")
				testutil.AssertFieldSet(t, updatedStage.DateModified, "DateModified")
			},
		},
		{
			Name:     "Unauthorized",
			TestCode: "ESPYNA-TEST-WORKFLOW-STAGE-UPDATE-AUTHORIZATION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *stagepb.UpdateStageRequest {
				return &stagepb.UpdateStageRequest{
					Data: &stagepb.Stage{
						Id:   authorizationUnauthorizedResolver.MustGetString("restrictedStageId"),
						Name: authorizationUnauthorizedResolver.MustGetString("updatedName"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        false,
			ExpectSuccess:  false,
			ExpectedError:  "stage.errors.authorization_failed",
			Assertions: func(t *testing.T, response *stagepb.UpdateStageResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertAuthorizationError(t, err)
			},
		},
		{
			Name:     "NotFound",
			TestCode: "ESPYNA-TEST-WORKFLOW-STAGE-UPDATE-NOT-FOUND-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *stagepb.UpdateStageRequest {
				return &stagepb.UpdateStageRequest{
					Data: &stagepb.Stage{
						Id:   notFoundResolver.MustGetString("nonExistentStageId"),
						Name: notFoundResolver.MustGetString("updatedName"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "stage.errors.not_found",
			Assertions: func(t *testing.T, response *stagepb.UpdateStageResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertNotFoundError(t, err)
			},
		},
		{
			Name:     "EmptyName",
			TestCode: "ESPYNA-TEST-WORKFLOW-STAGE-UPDATE-VALIDATION-EMPTY-NAME-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *stagepb.UpdateStageRequest {
				return &stagepb.UpdateStageRequest{
					Data: &stagepb.Stage{
						Id:   validationErrorEmptyNameResolver.MustGetString("existingStageId"),
						Name: validationErrorEmptyNameResolver.MustGetString("emptyName"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "stage.validation.name_required",
			Assertions: func(t *testing.T, response *stagepb.UpdateStageResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty name")
			},
		},
		{
			Name:     "NameTooShort",
			TestCode: "ESPYNA-TEST-WORKFLOW-STAGE-UPDATE-VALIDATION-NAME-TOO-SHORT-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *stagepb.UpdateStageRequest {
				return &stagepb.UpdateStageRequest{
					Data: &stagepb.Stage{
						Id:   validationErrorEmptyNameResolver.MustGetString("existingStageId"),
						Name: "AB",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "stage.validation.name_too_short",
			Assertions: func(t *testing.T, response *stagepb.UpdateStageResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "name too short")
			},
		},
		{
			Name:     "ConcurrentModification",
			TestCode: "ESPYNA-TEST-WORKFLOW-STAGE-UPDATE-CONCURRENT-MODIFICATION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *stagepb.UpdateStageRequest {
				return &stagepb.UpdateStageRequest{
					Data: &stagepb.Stage{
						Id:   concurrentModificationResolver.MustGetString("existingStageId"),
						Name: concurrentModificationResolver.MustGetString("updatedName"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "stage.errors.concurrent_modification",
			Assertions: func(t *testing.T, response *stagepb.UpdateStageResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertError(t, err)
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
			useCase := createUpdateStageTestUseCaseWithAuth(businessType, tc.UseTransaction, tc.UseAuth)

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
