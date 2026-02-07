//go:build mock_db && mock_auth

// Package payment_method provides table-driven tests for the payment method deletion use case.
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
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestDeletePaymentMethodUseCase_Execute_TableDriven
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-PAYMENT-PAYMENT-METHOD-DELETE-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-PAYMENT-PAYMENT-METHOD-DELETE-TRANSACTION-v1.0: WithTransaction
//   - ESPYNA-TEST-PAYMENT-PAYMENT-METHOD-DELETE-AUTHORIZATION-v1.0: Unauthorized
//   - ESPYNA-TEST-PAYMENT-PAYMENT-METHOD-DELETE-NIL-REQUEST-v1.0: NilRequest
//   - ESPYNA-TEST-PAYMENT-PAYMENT-METHOD-DELETE-NIL-DATA-v1.0: NilData
//   - ESPYNA-TEST-PAYMENT-PAYMENT-METHOD-DELETE-VALIDATION-EMPTY-ID-v1.0: EmptyId
//   - ESPYNA-TEST-PAYMENT-PAYMENT-METHOD-DELETE-VALIDATION-INVALID-ID-v1.0: InvalidId
//   - ESPYNA-TEST-PAYMENT-PAYMENT-METHOD-DELETE-TRANSACTION-FAILURE-v1.0: TransactionFailure
//
// Data Sources:
//   - Test cases: packages/copya/data_test/{businessType}/payment_method.json
//   - Mock data: packages/copya/data/{businessType}/payment_method.json
//   - Translations: packages/lyngua/translations/{languageCode}/{businessType}/payment_method.json

package payment_method

import (
	"context"
	"testing"

	copyatestutil "leapfor.xyz/copya/golang/testutil"
	"leapfor.xyz/espyna/internal/application/shared/testutil"
	mockDb "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock"
	"leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock/payment"
	paymentmethodpb "leapfor.xyz/esqyma/golang/v1/domain/payment/payment_method"
)

// Type alias for delete payment method test cases
type DeletePaymentMethodTestCase = testutil.GenericTestCase[*paymentmethodpb.DeletePaymentMethodRequest, *paymentmethodpb.DeletePaymentMethodResponse]

// Test helper to create use case with real services
func createDeleteTestUseCase(businessType string, supportsTransaction bool) *DeletePaymentMethodUseCase {
	return createDeleteTestUseCaseWithAuth(businessType, supportsTransaction, true)
}

func createDeleteTestUseCaseWithAuth(businessType string, supportsTransaction bool, shouldAuthorize bool) *DeletePaymentMethodUseCase {
	paymentMethodRepo := payment.NewMockPaymentMethodRepository(businessType)

	repositories := DeletePaymentMethodRepositories{
		PaymentMethod: paymentMethodRepo,
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, shouldAuthorize)
	services := DeletePaymentMethodServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
	}

	return NewDeletePaymentMethodUseCase(repositories, services)
}

func TestDeletePaymentMethodUseCase_Execute_TableDriven(t *testing.T) {
	businessType := testutil.GetTestBusinessType()

	// Load test data resolvers before defining test cases
	deleteSuccessResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "payment_method", "DeletePaymentMethod_Success")
	testutil.AssertTestCaseLoad(t, err, "DeletePaymentMethod_Success")

	authorizationUnauthorizedResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "payment_method", "Authorization_Unauthorized")
	testutil.AssertTestCaseLoad(t, err, "Authorization_Unauthorized")

	validationErrorEmptyIdResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "payment_method", "ValidationError_EmptyId")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_EmptyId")

	validationErrorInvalidIdResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "payment_method", "ValidationError_InvalidId")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_InvalidId")

	testCases := []DeletePaymentMethodTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-PAYMENT-PAYMENT-METHOD-DELETE-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *paymentmethodpb.DeletePaymentMethodRequest {
				return &paymentmethodpb.DeletePaymentMethodRequest{
					Data: &paymentmethodpb.PaymentMethod{
						Id: deleteSuccessResolver.MustGetString("validPaymentMethodId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *paymentmethodpb.DeletePaymentMethodResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
			},
		},
		{
			Name:     "WithTransaction",
			TestCode: "ESPYNA-TEST-PAYMENT-PAYMENT-METHOD-DELETE-TRANSACTION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *paymentmethodpb.DeletePaymentMethodRequest {
				return &paymentmethodpb.DeletePaymentMethodRequest{
					Data: &paymentmethodpb.PaymentMethod{
						Id: deleteSuccessResolver.MustGetString("validPaymentMethodId"),
					},
				}
			},
			UseTransaction: true,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *paymentmethodpb.DeletePaymentMethodResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
			},
		},
		{
			Name:     "Unauthorized",
			TestCode: "ESPYNA-TEST-PAYMENT-PAYMENT-METHOD-DELETE-AUTHORIZATION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *paymentmethodpb.DeletePaymentMethodRequest {
				return &paymentmethodpb.DeletePaymentMethodRequest{
					Data: &paymentmethodpb.PaymentMethod{
						Id: authorizationUnauthorizedResolver.MustGetString("unauthorizedPaymentMethodId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        false,
			ExpectSuccess:  false,
			ExpectedError:  "payment_method.errors.authorization_failed",
			Assertions: func(t *testing.T, response *paymentmethodpb.DeletePaymentMethodResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertAuthorizationError(t, err)
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-PAYMENT-PAYMENT-METHOD-DELETE-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *paymentmethodpb.DeletePaymentMethodRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "payment_method.validation.request_required",
			Assertions: func(t *testing.T, response *paymentmethodpb.DeletePaymentMethodResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilRequest(t, err)
			},
		},
		{
			Name:     "NilData",
			TestCode: "ESPYNA-TEST-PAYMENT-PAYMENT-METHOD-DELETE-NIL-DATA-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *paymentmethodpb.DeletePaymentMethodRequest {
				return &paymentmethodpb.DeletePaymentMethodRequest{Data: nil}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "payment_method.validation.data_required",
			Assertions: func(t *testing.T, response *paymentmethodpb.DeletePaymentMethodResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilData(t, err)
			},
		},
		{
			Name:     "EmptyId",
			TestCode: "ESPYNA-TEST-PAYMENT-PAYMENT-METHOD-DELETE-VALIDATION-EMPTY-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *paymentmethodpb.DeletePaymentMethodRequest {
				return &paymentmethodpb.DeletePaymentMethodRequest{
					Data: &paymentmethodpb.PaymentMethod{
						Id: validationErrorEmptyIdResolver.MustGetString("emptyId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "payment_method.validation.id_required",
			Assertions: func(t *testing.T, response *paymentmethodpb.DeletePaymentMethodResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty ID")
			},
		},
		{
			Name:     "InvalidId",
			TestCode: "ESPYNA-TEST-PAYMENT-PAYMENT-METHOD-DELETE-VALIDATION-INVALID-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *paymentmethodpb.DeletePaymentMethodRequest {
				return &paymentmethodpb.DeletePaymentMethodRequest{
					Data: &paymentmethodpb.PaymentMethod{
						Id: validationErrorInvalidIdResolver.MustGetString("invalidId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "payment_method.validation.id_invalid",
			Assertions: func(t *testing.T, response *paymentmethodpb.DeletePaymentMethodResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "invalid ID")
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

func TestDeletePaymentMethodUseCase_Execute_TransactionFailure(t *testing.T) {
	testCode := "ESPYNA-TEST-PAYMENT-PAYMENT-METHOD-DELETE-TRANSACTION-FAILURE-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "TransactionFailure", false)

	// Create a use case with a failing transaction service
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	mockPaymentMethodRepo := payment.NewMockPaymentMethodRepository(businessType)

	repositories := DeletePaymentMethodRepositories{
		PaymentMethod: mockPaymentMethodRepo,
	}

	standardServices := testutil.CreateStandardServices(false, true)
	services := DeletePaymentMethodServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   mockDb.NewFailingMockTransactionService(),
		TranslationService:   standardServices.TranslationService,
	}

	useCase := NewDeletePaymentMethodUseCase(repositories, services)

	// Load test case using centralized TestCaseResolver
	resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "payment_method", "DeletePaymentMethod_Success")
	testutil.AssertTestCaseLoad(t, err, "DeletePaymentMethod_Success")

	req := &paymentmethodpb.DeletePaymentMethodRequest{
		Data: &paymentmethodpb.PaymentMethod{
			Id: resolver.MustGetString("validPaymentMethodId"),
		},
	}

	_, err = useCase.Execute(ctx, req)

	// Verify that a transaction error occurred
	testutil.AssertTransactionError(t, err)

	// Log completion of transaction failure test
	testutil.LogTestResult(t, testCode, "TransactionFailure", false, err)
}
