//go:build mock_db && mock_auth

// Package delegate provides table-driven tests for the delegate updating use case.
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
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestUpdateDelegateUseCase_Execute_TableDriven
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-ENTITY-DELEGATE-UPDATE-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-ENTITY-DELEGATE-UPDATE-TRANSACTION-v1.0: WithTransaction
//   - ESPYNA-TEST-ENTITY-DELEGATE-UPDATE-AUTHORIZATION-v1.0: Unauthorized
//   - ESPYNA-TEST-ENTITY-DELEGATE-UPDATE-NIL-REQUEST-v1.0: NilRequest
//   - ESPYNA-TEST-ENTITY-DELEGATE-UPDATE-NIL-DATA-v1.0: NilData
//   - ESPYNA-TEST-ENTITY-DELEGATE-UPDATE-VALIDATION-EMPTY-ID-v1.0: EmptyId
//   - ESPYNA-TEST-ENTITY-DELEGATE-UPDATE-VALIDATION-INVALID-EMAIL-v1.0: InvalidEmail
//   - ESPYNA-TEST-ENTITY-DELEGATE-UPDATE-NOT-FOUND-v1.0: NotFound
//   - ESPYNA-TEST-ENTITY-DELEGATE-UPDATE-TRANSACTION-FAILURE-v1.0: TransactionFailure
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
	userpb "leapfor.xyz/esqyma/golang/v1/domain/entity/user"
)

// Type alias for update delegate test cases
type UpdateDelegateTestCase = testutil.GenericTestCase[*delegatepb.UpdateDelegateRequest, *delegatepb.UpdateDelegateResponse]

func createTestUpdateUseCaseWithAuth(businessType string, supportsTransaction bool, shouldAuthorize bool) *UpdateDelegateUseCase {
	mockDelegateRepo := entity.NewMockDelegateRepository(businessType)

	repositories := UpdateDelegateRepositories{
		Delegate: mockDelegateRepo,
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, shouldAuthorize)
	services := UpdateDelegateServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
	}

	return NewUpdateDelegateUseCase(repositories, services)
}

func TestUpdateDelegateUseCase_Execute_TableDriven(t *testing.T) {
	businessType := testutil.GetTestBusinessType()

	// Load test data resolvers before defining test cases
	updateSuccessResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "delegate", "UpdateDelegate_Success")
	testutil.AssertTestCaseLoad(t, err, "UpdateDelegate_Success")

	updateNotFoundResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "delegate", "UpdateDelegate_NotFound")
	testutil.AssertTestCaseLoad(t, err, "UpdateDelegate_NotFound")

	authorizationUnauthorizedResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "delegate", "Authorization_Unauthorized")
	testutil.AssertTestCaseLoad(t, err, "Authorization_Unauthorized")

	validationErrorInvalidEmailResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "delegate", "ValidationError_InvalidEmail")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_InvalidEmail")

	testCases := []UpdateDelegateTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-ENTITY-DELEGATE-UPDATE-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *delegatepb.UpdateDelegateRequest {
				return &delegatepb.UpdateDelegateRequest{
					Data: &delegatepb.Delegate{
						Id: updateSuccessResolver.MustGetString("targetDelegateId"),
						User: &userpb.User{
							FirstName:    updateSuccessResolver.MustGetString("updatedFirstName"),
							LastName:     updateSuccessResolver.MustGetString("updatedLastName"),
							EmailAddress: updateSuccessResolver.MustGetString("updatedEmailAddress"),
						},
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *delegatepb.UpdateDelegateResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertEqual(t, 1, len(response.Data), "response data length")
				updatedDelegate := response.Data[0]
				testutil.AssertStringEqual(t, updateSuccessResolver.MustGetString("targetDelegateId"), updatedDelegate.Id, "delegate ID")
				testutil.AssertStringEqual(t, updateSuccessResolver.MustGetString("updatedFirstName"), updatedDelegate.User.FirstName, "updated first name")
				testutil.AssertStringEqual(t, updateSuccessResolver.MustGetString("updatedLastName"), updatedDelegate.User.LastName, "updated last name")
				testutil.AssertStringEqual(t, updateSuccessResolver.MustGetString("updatedEmailAddress"), updatedDelegate.User.EmailAddress, "updated email address")
				testutil.AssertFieldSet(t, updatedDelegate.DateModified, "DateModified")
			},
		},
		{
			Name:     "WithTransaction",
			TestCode: "ESPYNA-TEST-ENTITY-DELEGATE-UPDATE-TRANSACTION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *delegatepb.UpdateDelegateRequest {
				return &delegatepb.UpdateDelegateRequest{
					Data: &delegatepb.Delegate{
						Id: updateSuccessResolver.MustGetString("targetDelegateId"),
						User: &userpb.User{
							FirstName:    updateSuccessResolver.MustGetString("updatedFirstName"),
							LastName:     updateSuccessResolver.MustGetString("updatedLastName"),
							EmailAddress: updateSuccessResolver.MustGetString("updatedEmailAddress"),
						},
					},
				}
			},
			UseTransaction: true,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *delegatepb.UpdateDelegateResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				updatedDelegate := response.Data[0]
				testutil.AssertStringEqual(t, updateSuccessResolver.MustGetString("updatedFirstName"), updatedDelegate.User.FirstName, "updated first name")
			},
		},
		{
			Name:     "Unauthorized",
			TestCode: "ESPYNA-TEST-ENTITY-DELEGATE-UPDATE-AUTHORIZATION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *delegatepb.UpdateDelegateRequest {
				return &delegatepb.UpdateDelegateRequest{
					Data: &delegatepb.Delegate{
						Id: authorizationUnauthorizedResolver.MustGetString("targetDelegateId"),
						User: &userpb.User{
							FirstName:    authorizationUnauthorizedResolver.MustGetString("unauthorizedDelegateFirstName"),
							LastName:     authorizationUnauthorizedResolver.MustGetString("unauthorizedDelegateLastName"),
							EmailAddress: authorizationUnauthorizedResolver.MustGetString("unauthorizedDelegateEmailAddress"),
						},
					},
				}
			},
			UseTransaction: false,
			UseAuth:        false,
			ExpectSuccess:  false,
			ExpectedError:  "delegate.errors.authorization_failed",
			Assertions: func(t *testing.T, response *delegatepb.UpdateDelegateResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertAuthorizationError(t, err)
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-ENTITY-DELEGATE-UPDATE-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *delegatepb.UpdateDelegateRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "delegate.validation.request_required",
			Assertions: func(t *testing.T, response *delegatepb.UpdateDelegateResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilRequest(t, err)
			},
		},
		{
			Name:     "NilData",
			TestCode: "ESPYNA-TEST-ENTITY-DELEGATE-UPDATE-NIL-DATA-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *delegatepb.UpdateDelegateRequest {
				return &delegatepb.UpdateDelegateRequest{Data: nil}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "delegate.validation.data_required",
			Assertions: func(t *testing.T, response *delegatepb.UpdateDelegateResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilData(t, err)
			},
		},
		{
			Name:     "EmptyId",
			TestCode: "ESPYNA-TEST-ENTITY-DELEGATE-UPDATE-VALIDATION-EMPTY-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *delegatepb.UpdateDelegateRequest {
				return &delegatepb.UpdateDelegateRequest{
					Data: &delegatepb.Delegate{
						Id: "",
						User: &userpb.User{
							FirstName:    "Valid",
							LastName:     "Name",
							EmailAddress: "valid@test.com",
						},
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "delegate.validation.id_required",
			Assertions: func(t *testing.T, response *delegatepb.UpdateDelegateResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty ID")
			},
		},
		{
			Name:     "InvalidEmail",
			TestCode: "ESPYNA-TEST-ENTITY-DELEGATE-UPDATE-VALIDATION-INVALID-EMAIL-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *delegatepb.UpdateDelegateRequest {
				return &delegatepb.UpdateDelegateRequest{
					Data: &delegatepb.Delegate{
						Id: updateSuccessResolver.MustGetString("targetDelegateId"),
						User: &userpb.User{
							FirstName:    validationErrorInvalidEmailResolver.MustGetString("validFirstName"),
							LastName:     validationErrorInvalidEmailResolver.MustGetString("validLastName"),
							EmailAddress: validationErrorInvalidEmailResolver.MustGetString("invalidEmailAddress"),
						},
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "delegate.validation.email_invalid",
			Assertions: func(t *testing.T, response *delegatepb.UpdateDelegateResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "invalid email")
			},
		},
		{
			Name:     "NotFound",
			TestCode: "ESPYNA-TEST-ENTITY-DELEGATE-UPDATE-NOT-FOUND-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *delegatepb.UpdateDelegateRequest {
				return &delegatepb.UpdateDelegateRequest{
					Data: &delegatepb.Delegate{
						Id: updateNotFoundResolver.MustGetString("nonExistentDelegateId"),
						User: &userpb.User{
							FirstName:    updateNotFoundResolver.MustGetString("updateFirstName"),
							LastName:     updateNotFoundResolver.MustGetString("updateLastName"),
							EmailAddress: updateNotFoundResolver.MustGetString("updateEmailAddress"),
						},
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "delegate.errors.update_failed",
			Assertions: func(t *testing.T, response *delegatepb.UpdateDelegateResponse, err error, useCase interface{}, ctx context.Context) {
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
			useCase := createTestUpdateUseCaseWithAuth(businessType, tc.UseTransaction, tc.UseAuth)

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

func TestUpdateDelegateUseCase_Execute_TransactionFailure(t *testing.T) {
	testCode := "ESPYNA-TEST-ENTITY-DELEGATE-UPDATE-TRANSACTION-FAILURE-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "TransactionFailure", false)

	// Create a use case with a failing transaction service
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	mockDelegateRepo := entity.NewMockDelegateRepository(businessType)

	repositories := UpdateDelegateRepositories{
		Delegate: mockDelegateRepo,
	}

	standardServices := testutil.CreateStandardServices(false, true)
	services := UpdateDelegateServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   mockDb.NewFailingMockTransactionService(),
		TranslationService:   standardServices.TranslationService,
	}

	useCase := NewUpdateDelegateUseCase(repositories, services)

	// Load test case using centralized TestCaseResolver
	resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "delegate", "UpdateDelegate_Success")
	testutil.AssertTestCaseLoad(t, err, "UpdateDelegate_Success")

	req := &delegatepb.UpdateDelegateRequest{
		Data: &delegatepb.Delegate{
			Id: resolver.MustGetString("targetDelegateId"),
			User: &userpb.User{
				FirstName:    resolver.MustGetString("updatedFirstName"),
				LastName:     resolver.MustGetString("updatedLastName"),
				EmailAddress: resolver.MustGetString("updatedEmailAddress"),
			},
		},
	}

	_, err = useCase.Execute(ctx, req)

	// Verify that a transaction error occurred
	testutil.AssertTransactionError(t, err)

	// Log completion of transaction failure test
	testutil.LogTestResult(t, testCode, "TransactionFailure", false, err)
}
