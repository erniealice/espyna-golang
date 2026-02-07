//go:build mock_db && mock_auth

// Package payment_profile provides table-driven tests for the payment profile deletion use case.
//
// The tests cover various scenarios, including success, transaction handling,
// authorization, nil requests, validation errors, and boundary conditions.
// Each test case is defined in a table with a specific test code, request setup,
// and assertions.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user").
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education").
//
// Usage:
//   - Run all tests: go test -tags="mock_db,mock_auth" ./...
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestDeletePaymentProfileUseCase_Execute_TableDriven
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-PAYMENT-PAYMENT-PROFILE-DELETE-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-PAYMENT-PAYMENT-PROFILE-DELETE-TRANSACTION-v1.0: WithTransaction
//   - ESPYNA-TEST-PAYMENT-PAYMENT-PROFILE-DELETE-AUTHORIZATION-v1.0: Unauthorized
//   - ESPYNA-TEST-PAYMENT-PAYMENT-PROFILE-DELETE-NIL-REQUEST-v1.0: NilRequest
//   - ESPYNA-TEST-PAYMENT-PAYMENT-PROFILE-DELETE-NIL-DATA-v1.0: NilData
//   - ESPYNA-TEST-PAYMENT-PAYMENT-PROFILE-DELETE-VALIDATION-EMPTY-ID-v1.0: EmptyId
//   - ESPYNA-TEST-PAYMENT-PAYMENT-PROFILE-DELETE-VALIDATION-INVALID-ID-v1.0: InvalidId
//   - ESPYNA-TEST-PAYMENT-PAYMENT-PROFILE-DELETE-VALIDATION-NON-EXISTENT-v1.0: NonExistent
//   - ESPYNA-TEST-PAYMENT-PAYMENT-PROFILE-DELETE-TRANSACTION-FAILURE-v1.0: TransactionFailure
//
// Data Sources:
//   - Test cases: packages/copya/data_test/{businessType}/payment_profile.json
//   - Mock data: packages/copya/data/{businessType}/payment_profile.json
//   - Translations: packages/lyngua/translations/{languageCode}/{businessType}/payment_profile.json

package payment_profile

import (
	"context"
	"testing"

	copyatestutil "leapfor.xyz/copya/golang/testutil"
	"leapfor.xyz/espyna/internal/application/shared/testutil"
	mockDb "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock"
	"leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock/payment"
	paymentprofilepb "leapfor.xyz/esqyma/golang/v1/domain/payment/payment_profile"
)

// Type alias for delete payment profile test cases
type DeletePaymentProfileTestCase = testutil.GenericTestCase[*paymentprofilepb.DeletePaymentProfileRequest, *paymentprofilepb.DeletePaymentProfileResponse]

func createDeleteTestUseCaseWithAuth(businessType string, supportsTransaction bool, shouldAuthorize bool) *DeletePaymentProfileUseCase {
	paymentProfileRepo := payment.NewMockPaymentProfileRepository(businessType)

	repositories := DeletePaymentProfileRepositories{
		PaymentProfile: paymentProfileRepo,
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, shouldAuthorize)
	services := DeletePaymentProfileServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
	}

	return NewDeletePaymentProfileUseCase(repositories, services)
}

func TestDeletePaymentProfileUseCase_Execute_TableDriven(t *testing.T) {
	businessType := testutil.GetTestBusinessType()

	// Load test data resolvers before defining test cases
	deleteSuccessResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "payment_profile", "DeletePaymentProfile_Success")
	testutil.AssertTestCaseLoad(t, err, "DeletePaymentProfile_Success")

	authorizationUnauthorizedResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "payment_profile", "Authorization_Unauthorized")
	testutil.AssertTestCaseLoad(t, err, "Authorization_Unauthorized")

	validationErrorEmptyIdResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "payment_profile", "ValidationError_EmptyId")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_EmptyId")

	validationErrorInvalidIdResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "payment_profile", "ValidationError_InvalidId")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_InvalidId")

	validationErrorNonExistentResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "payment_profile", "ValidationError_NonExistent")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_NonExistent")

	testCases := []DeletePaymentProfileTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-PAYMENT-PAYMENT-PROFILE-DELETE-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *paymentprofilepb.DeletePaymentProfileRequest {
				return &paymentprofilepb.DeletePaymentProfileRequest{
					Data: &paymentprofilepb.PaymentProfile{
						Id: deleteSuccessResolver.MustGetString("validPaymentProfileId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *paymentprofilepb.DeletePaymentProfileResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
			},
		},
		{
			Name:     "WithTransaction",
			TestCode: "ESPYNA-TEST-PAYMENT-PAYMENT-PROFILE-DELETE-TRANSACTION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *paymentprofilepb.DeletePaymentProfileRequest {
				return &paymentprofilepb.DeletePaymentProfileRequest{
					Data: &paymentprofilepb.PaymentProfile{
						Id: deleteSuccessResolver.MustGetString("validPaymentProfileId"),
					},
				}
			},
			UseTransaction: true,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *paymentprofilepb.DeletePaymentProfileResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
			},
		},
		{
			Name:     "Unauthorized",
			TestCode: "ESPYNA-TEST-PAYMENT-PAYMENT-PROFILE-DELETE-AUTHORIZATION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *paymentprofilepb.DeletePaymentProfileRequest {
				return &paymentprofilepb.DeletePaymentProfileRequest{
					Data: &paymentprofilepb.PaymentProfile{
						Id: authorizationUnauthorizedResolver.MustGetString("unauthorizedPaymentProfileId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        false,
			ExpectSuccess:  false,
			ExpectedError:  "payment_profile.errors.authorization_failed",
			Assertions: func(t *testing.T, response *paymentprofilepb.DeletePaymentProfileResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertAuthorizationError(t, err)
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-PAYMENT-PAYMENT-PROFILE-DELETE-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *paymentprofilepb.DeletePaymentProfileRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "payment_profile.validation.request_required",
			Assertions: func(t *testing.T, response *paymentprofilepb.DeletePaymentProfileResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilRequest(t, err)
			},
		},
		{
			Name:     "NilData",
			TestCode: "ESPYNA-TEST-PAYMENT-PAYMENT-PROFILE-DELETE-NIL-DATA-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *paymentprofilepb.DeletePaymentProfileRequest {
				return &paymentprofilepb.DeletePaymentProfileRequest{Data: nil}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "payment_profile.validation.data_required",
			Assertions: func(t *testing.T, response *paymentprofilepb.DeletePaymentProfileResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilData(t, err)
			},
		},
		{
			Name:     "EmptyId",
			TestCode: "ESPYNA-TEST-PAYMENT-PAYMENT-PROFILE-DELETE-VALIDATION-EMPTY-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *paymentprofilepb.DeletePaymentProfileRequest {
				return &paymentprofilepb.DeletePaymentProfileRequest{
					Data: &paymentprofilepb.PaymentProfile{
						Id: validationErrorEmptyIdResolver.MustGetString("emptyId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "payment_profile.validation.id_required",
			Assertions: func(t *testing.T, response *paymentprofilepb.DeletePaymentProfileResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty ID")
			},
		},
		{
			Name:     "InvalidId",
			TestCode: "ESPYNA-TEST-PAYMENT-PAYMENT-PROFILE-DELETE-VALIDATION-INVALID-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *paymentprofilepb.DeletePaymentProfileRequest {
				return &paymentprofilepb.DeletePaymentProfileRequest{
					Data: &paymentprofilepb.PaymentProfile{
						Id: validationErrorInvalidIdResolver.MustGetString("invalidId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "payment_profile.validation.id_invalid",
			Assertions: func(t *testing.T, response *paymentprofilepb.DeletePaymentProfileResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "invalid ID")
			},
		},
		{
			Name:     "NonExistent",
			TestCode: "ESPYNA-TEST-PAYMENT-PAYMENT-PROFILE-DELETE-VALIDATION-NON-EXISTENT-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *paymentprofilepb.DeletePaymentProfileRequest {
				return &paymentprofilepb.DeletePaymentProfileRequest{
					Data: &paymentprofilepb.PaymentProfile{
						Id: validationErrorNonExistentResolver.MustGetString("nonExistentPaymentProfileId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "payment_profile.validation.not_found",
			Assertions: func(t *testing.T, response *paymentprofilepb.DeletePaymentProfileResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "non-existent payment profile")
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

func TestDeletePaymentProfileUseCase_Execute_TransactionFailure(t *testing.T) {
	testCode := "ESPYNA-TEST-PAYMENT-PAYMENT-PROFILE-DELETE-TRANSACTION-FAILURE-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "TransactionFailure", false)

	// Create a use case with a failing transaction service
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	paymentProfileRepo := payment.NewMockPaymentProfileRepository(businessType)

	repositories := DeletePaymentProfileRepositories{
		PaymentProfile: paymentProfileRepo,
	}

	standardServices := testutil.CreateStandardServices(false, true)
	services := DeletePaymentProfileServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   mockDb.NewFailingMockTransactionService(),
		TranslationService:   standardServices.TranslationService,
	}

	useCase := NewDeletePaymentProfileUseCase(repositories, services)

	// Load test case using centralized TestCaseResolver
	resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "payment_profile", "DeletePaymentProfile_Success")
	testutil.AssertTestCaseLoad(t, err, "DeletePaymentProfile_Success")

	req := &paymentprofilepb.DeletePaymentProfileRequest{
		Data: &paymentprofilepb.PaymentProfile{
			Id: resolver.MustGetString("validPaymentProfileId"),
		},
	}

	_, err = useCase.Execute(ctx, req)

	// Verify that a transaction error occurred
	testutil.AssertTransactionError(t, err)

	// Log completion of transaction failure test
	testutil.LogTestResult(t, testCode, "TransactionFailure", false, err)
}
