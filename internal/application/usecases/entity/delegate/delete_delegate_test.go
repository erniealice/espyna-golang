//go:build mock_db && mock_auth

// Package delegate provides table-driven tests for the delegate deletion use case.
//
// The tests cover various scenarios, including success, transaction handling,
// authorization, nil requests, validation errors, and not found conditions.
// Each test case is defined in a table with a specific test code, request setup,
// and assertions.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user").
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education").
//
// Usage:
//   - Run all tests: go test -tags="mock_db,mock_auth" ./...
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestDeleteDelegateUseCase_Execute_TableDriven
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-ENTITY-DELEGATE-DELETE-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-ENTITY-DELEGATE-DELETE-TRANSACTION-v1.0: WithTransaction
//   - ESPYNA-TEST-ENTITY-DELEGATE-DELETE-AUTHORIZATION-v1.0: Unauthorized
//   - ESPYNA-TEST-ENTITY-DELEGATE-DELETE-NIL-REQUEST-v1.0: NilRequest
//   - ESPYNA-TEST-ENTITY-DELEGATE-DELETE-NIL-DATA-v1.0: NilData
//   - ESPYNA-TEST-ENTITY-DELEGATE-DELETE-VALIDATION-EMPTY-ID-v1.0: EmptyId
//   - ESPYNA-TEST-ENTITY-DELEGATE-DELETE-NOT-FOUND-v1.0: NotFound
//   - ESPYNA-TEST-ENTITY-DELEGATE-DELETE-TRANSACTION-FAILURE-v1.0: TransactionFailure
//
// Data Sources:
//   - Test cases: packages/copya/data_test/{businessType}/delegate.json
//   - Mock data: packages/copya/data/{businessType}/delegate.json
//   - Translations: packages/lyngua/translations/{languageCode}/{businessType}/delegate.json
package delegate

import (
	"context"
	"testing"

	copyatestutil "leapfor.xyz/copya/golang/testutil"
	"leapfor.xyz/espyna/internal/application/shared/testutil"
	mockDb "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock"
	"leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock/entity"
	delegatepb "leapfor.xyz/esqyma/golang/v1/domain/entity/delegate"
)

// Type alias for delete delegate test cases
type DeleteDelegateTestCase = testutil.GenericTestCase[*delegatepb.DeleteDelegateRequest, *delegatepb.DeleteDelegateResponse]

func createTestDeleteUseCaseWithAuth(businessType string, supportsTransaction bool, shouldAuthorize bool) *DeleteDelegateUseCase {
	mockDelegateRepo := entity.NewMockDelegateRepository(businessType)

	repositories := DeleteDelegateRepositories{
		Delegate: mockDelegateRepo,
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, shouldAuthorize)
	services := DeleteDelegateServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
	}

	return NewDeleteDelegateUseCase(repositories, services)
}

func TestDeleteDelegateUseCase_Execute_TableDriven(t *testing.T) {
	businessType := testutil.GetTestBusinessType()

	// Load test data resolvers before defining test cases
	deleteSuccessResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "delegate", "DeleteDelegate_Success")
	testutil.AssertTestCaseLoad(t, err, "DeleteDelegate_Success")

	deleteNotFoundResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "delegate", "DeleteDelegate_NotFound")
	testutil.AssertTestCaseLoad(t, err, "DeleteDelegate_NotFound")

	authorizationUnauthorizedResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "delegate", "Authorization_Unauthorized")
	testutil.AssertTestCaseLoad(t, err, "Authorization_Unauthorized")

	testCases := []DeleteDelegateTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-ENTITY-DELEGATE-DELETE-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *delegatepb.DeleteDelegateRequest {
				return &delegatepb.DeleteDelegateRequest{
					Data: &delegatepb.Delegate{
						Id: deleteSuccessResolver.MustGetString("targetDelegateId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *delegatepb.DeleteDelegateResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
			},
		},
		{
			Name:     "WithTransaction",
			TestCode: "ESPYNA-TEST-ENTITY-DELEGATE-DELETE-TRANSACTION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *delegatepb.DeleteDelegateRequest {
				return &delegatepb.DeleteDelegateRequest{
					Data: &delegatepb.Delegate{
						Id: deleteSuccessResolver.MustGetString("deletableId"),
					},
				}
			},
			UseTransaction: true,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *delegatepb.DeleteDelegateResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
			},
		},
		{
			Name:     "Unauthorized",
			TestCode: "ESPYNA-TEST-ENTITY-DELEGATE-DELETE-AUTHORIZATION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *delegatepb.DeleteDelegateRequest {
				return &delegatepb.DeleteDelegateRequest{
					Data: &delegatepb.Delegate{
						Id: authorizationUnauthorizedResolver.MustGetString("targetDelegateId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        false,
			ExpectSuccess:  false,
			ExpectedError:  "delegate.errors.authorization_failed",
			Assertions: func(t *testing.T, response *delegatepb.DeleteDelegateResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertAuthorizationError(t, err)
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-ENTITY-DELEGATE-DELETE-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *delegatepb.DeleteDelegateRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "delegate.validation.request_required",
			Assertions: func(t *testing.T, response *delegatepb.DeleteDelegateResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilRequest(t, err)
			},
		},
		{
			Name:     "NilData",
			TestCode: "ESPYNA-TEST-ENTITY-DELEGATE-DELETE-NIL-DATA-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *delegatepb.DeleteDelegateRequest {
				return &delegatepb.DeleteDelegateRequest{Data: nil}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "delegate.validation.data_required",
			Assertions: func(t *testing.T, response *delegatepb.DeleteDelegateResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilData(t, err)
			},
		},
		{
			Name:     "EmptyId",
			TestCode: "ESPYNA-TEST-ENTITY-DELEGATE-DELETE-VALIDATION-EMPTY-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *delegatepb.DeleteDelegateRequest {
				return &delegatepb.DeleteDelegateRequest{
					Data: &delegatepb.Delegate{Id: ""},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "delegate.validation.id_required",
			Assertions: func(t *testing.T, response *delegatepb.DeleteDelegateResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty ID")
			},
		},
		{
			Name:     "NotFound",
			TestCode: "ESPYNA-TEST-ENTITY-DELEGATE-DELETE-NOT-FOUND-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *delegatepb.DeleteDelegateRequest {
				return &delegatepb.DeleteDelegateRequest{
					Data: &delegatepb.Delegate{
						Id: deleteNotFoundResolver.MustGetString("nonExistentDelegateId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "delegate.errors.delete_failed",
			Assertions: func(t *testing.T, response *delegatepb.DeleteDelegateResponse, err error, useCase interface{}, ctx context.Context) {
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
			useCase := createTestDeleteUseCaseWithAuth(businessType, tc.UseTransaction, tc.UseAuth)

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

func TestDeleteDelegateUseCase_Execute_TransactionFailure(t *testing.T) {
	testCode := "ESPYNA-TEST-ENTITY-DELEGATE-DELETE-TRANSACTION-FAILURE-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "TransactionFailure", false)

	// Create a use case with a failing transaction service
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	mockDelegateRepo := entity.NewMockDelegateRepository(businessType)

	repositories := DeleteDelegateRepositories{
		Delegate: mockDelegateRepo,
	}

	standardServices := testutil.CreateStandardServices(false, true)
	services := DeleteDelegateServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   mockDb.NewFailingMockTransactionService(),
		TranslationService:   standardServices.TranslationService,
	}

	useCase := NewDeleteDelegateUseCase(repositories, services)

	// Load test case using centralized TestCaseResolver
	resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "delegate", "DeleteDelegate_Success")
	testutil.AssertTestCaseLoad(t, err, "DeleteDelegate_Success")

	req := &delegatepb.DeleteDelegateRequest{
		Data: &delegatepb.Delegate{
			Id: resolver.MustGetString("targetDelegateId"),
		},
	}

	_, err = useCase.Execute(ctx, req)

	// Verify that a transaction error occurred
	testutil.AssertTransactionError(t, err)

	// Log completion of transaction failure test
	testutil.LogTestResult(t, testCode, "TransactionFailure", false, err)
}
