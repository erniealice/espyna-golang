//go:build mock_db && mock_auth

// Package payment_method provides table-driven tests for the payment method read use case.
//
// The tests cover various scenarios, including success, transaction handling,
// authorization, nil requests, validation errors, and non-existent payment methods.
// Each test case is defined in a table with a specific test code, request setup,
// and assertions.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user").
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education").
//
// Usage:
//   - Run all tests: go test -tags="mock_db,mock_auth" ./...
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestReadPaymentMethodUseCase_Execute_TableDriven
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-PAYMENT-PAYMENT-METHOD-READ-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-PAYMENT-PAYMENT-METHOD-READ-TRANSACTION-v1.0: WithTransaction
//   - ESPYNA-TEST-PAYMENT-PAYMENT-METHOD-READ-AUTHORIZATION-v1.0: Unauthorized
//   - ESPYNA-TEST-PAYMENT-PAYMENT-METHOD-READ-NIL-REQUEST-v1.0: NilRequest
//   - ESPYNA-TEST-PAYMENT-PAYMENT-METHOD-READ-NIL-DATA-v1.0: NilData
//   - ESPYNA-TEST-PAYMENT-PAYMENT-METHOD-READ-VALIDATION-EMPTY-ID-v1.0: EmptyId
//   - ESPYNA-TEST-PAYMENT-PAYMENT-METHOD-READ-VALIDATION-NON-EXISTENT-v1.0: NonExistentPaymentMethod
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
	"leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock/payment"
	paymentmethodpb "leapfor.xyz/esqyma/golang/v1/domain/payment/payment_method"
)

// Type alias for read payment method test cases
type ReadPaymentMethodTestCase = testutil.GenericTestCase[*paymentmethodpb.ReadPaymentMethodRequest, *paymentmethodpb.ReadPaymentMethodResponse]

// Test helper to create use case with real services
func createReadTestUseCase(businessType string, supportsTransaction bool) *ReadPaymentMethodUseCase {
	return createReadTestUseCaseWithAuth(businessType, supportsTransaction, true)
}

func createReadTestUseCaseWithAuth(businessType string, supportsTransaction bool, shouldAuthorize bool) *ReadPaymentMethodUseCase {
	paymentMethodRepo := payment.NewMockPaymentMethodRepository(businessType)

	repositories := ReadPaymentMethodRepositories{
		PaymentMethod: paymentMethodRepo,
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, shouldAuthorize)
	services := ReadPaymentMethodServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
	}

	return NewReadPaymentMethodUseCase(repositories, services)
}

func TestReadPaymentMethodUseCase_Execute_TableDriven(t *testing.T) {
	businessType := testutil.GetTestBusinessType()

	// Load test data resolvers before defining test cases
	readSuccessResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "payment_method", "ReadPaymentMethod_Success")
	testutil.AssertTestCaseLoad(t, err, "ReadPaymentMethod_Success")

	authorizationUnauthorizedResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "payment_method", "Authorization_Unauthorized")
	testutil.AssertTestCaseLoad(t, err, "Authorization_Unauthorized")

	validationErrorEmptyIdResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "payment_method", "ValidationError_EmptyId")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_EmptyId")

	validationErrorNonExistentResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "payment_method", "ValidationError_NonExistent")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_NonExistent")

	testCases := []ReadPaymentMethodTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-PAYMENT-PAYMENT-METHOD-READ-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *paymentmethodpb.ReadPaymentMethodRequest {
				return &paymentmethodpb.ReadPaymentMethodRequest{
					Data: &paymentmethodpb.PaymentMethod{
						Id: readSuccessResolver.MustGetString("existingPaymentMethodId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *paymentmethodpb.ReadPaymentMethodResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertEqual(t, 1, len(response.Data), "response data length")
				readPaymentMethod := response.Data[0]
				testutil.AssertStringEqual(t, readSuccessResolver.MustGetString("existingPaymentMethodId"), readPaymentMethod.Id, "payment method ID")
				testutil.AssertNonEmptyString(t, readPaymentMethod.Name, "payment method name")
				testutil.AssertFieldSet(t, readPaymentMethod.DateCreated, "DateCreated")
				testutil.AssertFieldSet(t, readPaymentMethod.DateCreatedString, "DateCreatedString")
			},
		},
		{
			Name:     "WithTransaction",
			TestCode: "ESPYNA-TEST-PAYMENT-PAYMENT-METHOD-READ-TRANSACTION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *paymentmethodpb.ReadPaymentMethodRequest {
				return &paymentmethodpb.ReadPaymentMethodRequest{
					Data: &paymentmethodpb.PaymentMethod{
						Id: readSuccessResolver.MustGetString("existingPaymentMethodId"),
					},
				}
			},
			UseTransaction: true,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *paymentmethodpb.ReadPaymentMethodResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				readPaymentMethod := response.Data[0]
				testutil.AssertStringEqual(t, readSuccessResolver.MustGetString("existingPaymentMethodId"), readPaymentMethod.Id, "payment method ID")
			},
		},
		{
			Name:     "Unauthorized",
			TestCode: "ESPYNA-TEST-PAYMENT-PAYMENT-METHOD-READ-AUTHORIZATION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *paymentmethodpb.ReadPaymentMethodRequest {
				return &paymentmethodpb.ReadPaymentMethodRequest{
					Data: &paymentmethodpb.PaymentMethod{
						Id: authorizationUnauthorizedResolver.MustGetString("unauthorizedPaymentMethodId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        false,
			ExpectSuccess:  false,
			ExpectedError:  "payment_method.errors.authorization_failed",
			Assertions: func(t *testing.T, response *paymentmethodpb.ReadPaymentMethodResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertAuthorizationError(t, err)
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-PAYMENT-PAYMENT-METHOD-READ-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *paymentmethodpb.ReadPaymentMethodRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "payment_method.validation.request_required",
			Assertions: func(t *testing.T, response *paymentmethodpb.ReadPaymentMethodResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilRequest(t, err)
			},
		},
		{
			Name:     "NilData",
			TestCode: "ESPYNA-TEST-PAYMENT-PAYMENT-METHOD-READ-NIL-DATA-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *paymentmethodpb.ReadPaymentMethodRequest {
				return &paymentmethodpb.ReadPaymentMethodRequest{Data: nil}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "payment_method.validation.data_required",
			Assertions: func(t *testing.T, response *paymentmethodpb.ReadPaymentMethodResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilData(t, err)
			},
		},
		{
			Name:     "EmptyId",
			TestCode: "ESPYNA-TEST-PAYMENT-PAYMENT-METHOD-READ-VALIDATION-EMPTY-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *paymentmethodpb.ReadPaymentMethodRequest {
				return &paymentmethodpb.ReadPaymentMethodRequest{
					Data: &paymentmethodpb.PaymentMethod{
						Id: validationErrorEmptyIdResolver.MustGetString("emptyId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "payment_method.validation.id_required",
			Assertions: func(t *testing.T, response *paymentmethodpb.ReadPaymentMethodResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty ID")
			},
		},
		{
			Name:     "NonExistentPaymentMethod",
			TestCode: "ESPYNA-TEST-PAYMENT-PAYMENT-METHOD-READ-VALIDATION-NON-EXISTENT-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *paymentmethodpb.ReadPaymentMethodRequest {
				return &paymentmethodpb.ReadPaymentMethodRequest{
					Data: &paymentmethodpb.PaymentMethod{
						Id: validationErrorNonExistentResolver.MustGetString("nonExistentPaymentMethodId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "payment_method.errors.not_found",
			ErrorTags:      map[string]any{"paymentMethodId": validationErrorNonExistentResolver.MustGetString("nonExistentPaymentMethodId")},
			Assertions: func(t *testing.T, response *paymentmethodpb.ReadPaymentMethodResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "non-existent payment method")
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
			useCase := createReadTestUseCaseWithAuth(businessType, tc.UseTransaction, tc.UseAuth)

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
