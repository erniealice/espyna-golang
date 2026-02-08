//go:build mock_db && mock_auth

// Package payment_method provides table-driven tests for the payment method creation use case.
//
// The tests cover various scenarios, including success, transaction handling,
// authorization, nil requests, validation errors, payment method details validation,
// and boundary conditions. Each test case is defined in a table with a specific test code,
// request setup, and assertions.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user").
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education").
//
// Usage:
//   - Run all tests: go test -tags="mock_db,mock_auth" ./...
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestCreatePaymentMethodUseCase_Execute_TableDriven
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-PAYMENT-PAYMENT-METHOD-CREATE-SUCCESS-CREDIT-CARD-v1.0: Success with credit card
//   - ESPYNA-TEST-PAYMENT-PAYMENT-METHOD-CREATE-SUCCESS-BANK-ACCOUNT-v1.0: Success with bank account
//   - ESPYNA-TEST-PAYMENT-PAYMENT-METHOD-CREATE-TRANSACTION-v1.0: WithTransaction
//   - ESPYNA-TEST-PAYMENT-PAYMENT-METHOD-CREATE-AUTHORIZATION-v1.0: Unauthorized
//   - ESPYNA-TEST-PAYMENT-PAYMENT-METHOD-CREATE-NIL-REQUEST-v1.0: NilRequest
//   - ESPYNA-TEST-PAYMENT-PAYMENT-METHOD-CREATE-NIL-DATA-v1.0: NilData
//   - ESPYNA-TEST-PAYMENT-PAYMENT-METHOD-CREATE-VALIDATION-EMPTY-NAME-v1.0: EmptyName
//   - ESPYNA-TEST-PAYMENT-PAYMENT-METHOD-CREATE-VALIDATION-MISSING-DETAILS-v1.0: MissingPaymentMethodDetails
//   - ESPYNA-TEST-PAYMENT-PAYMENT-METHOD-CREATE-VALIDATION-INVALID-CARD-TYPE-v1.0: InvalidCardType
//   - ESPYNA-TEST-PAYMENT-PAYMENT-METHOD-CREATE-VALIDATION-INVALID-EXPIRY-MONTH-v1.0: InvalidExpiryMonth
//   - ESPYNA-TEST-PAYMENT-PAYMENT-METHOD-CREATE-VALIDATION-INVALID-EXPIRY-YEAR-v1.0: InvalidExpiryYear
//   - ESPYNA-TEST-PAYMENT-PAYMENT-METHOD-CREATE-VALIDATION-INVALID-BANK-NAME-v1.0: InvalidBankName
//   - ESPYNA-TEST-PAYMENT-PAYMENT-METHOD-CREATE-ENRICHMENT-v1.0: DataEnrichment
//   - ESPYNA-TEST-PAYMENT-PAYMENT-METHOD-CREATE-TRANSACTION-FAILURE-v1.0: TransactionFailure
//
// Data Sources:
//   - Test cases: packages/copya/data_test/{businessType}/payment_method.json
//   - Mock data: packages/copya/data/{businessType}/payment_method.json
//   - Translations: packages/lyngua/translations/{languageCode}/{businessType}/payment_method.json

package payment_method

import (
	"context"
	"testing"
	"time"

	copyatestutil "leapfor.xyz/copya/golang/testutil"
	"github.com/erniealice/espyna-golang/internal/application/shared/testutil"
	mockDb "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock"
	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock/payment"
	paymentmethodpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/payment/payment_method"
)

// Type alias for create payment method test cases
type CreatePaymentMethodTestCase = testutil.GenericTestCase[*paymentmethodpb.CreatePaymentMethodRequest, *paymentmethodpb.CreatePaymentMethodResponse]

// Test helper to create use case with real services
func createTestUseCase(businessType string, supportsTransaction bool) *CreatePaymentMethodUseCase {
	return createTestUseCaseWithAuth(businessType, supportsTransaction, true)
}

func createTestUseCaseWithAuth(businessType string, supportsTransaction bool, shouldAuthorize bool) *CreatePaymentMethodUseCase {
	mockPaymentMethodRepo := payment.NewMockPaymentMethodRepository(businessType)

	repositories := CreatePaymentMethodRepositories{
		PaymentMethod: mockPaymentMethodRepo,
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, shouldAuthorize)
	services := CreatePaymentMethodServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
		IDService:            standardServices.IDService,
	}

	return NewCreatePaymentMethodUseCase(repositories, services)
}

func TestCreatePaymentMethodUseCase_Execute_TableDriven(t *testing.T) {
	businessType := testutil.GetTestBusinessType()

	// Load test data resolvers before defining test cases
	createSuccessResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "payment_method", "CreatePaymentMethod_Success")
	testutil.AssertTestCaseLoad(t, err, "CreatePaymentMethod_Success")

	authorizationUnauthorizedResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "payment_method", "Authorization_Unauthorized")
	testutil.AssertTestCaseLoad(t, err, "Authorization_Unauthorized")

	validationErrorEmptyNameResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "payment_method", "ValidationError_EmptyName")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_EmptyName")

	validationErrorMissingDetailsResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "payment_method", "ValidationError_MissingDetails")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_MissingDetails")

	validationErrorInvalidCardTypeResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "payment_method", "ValidationError_InvalidCardType")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_InvalidCardType")

	testCases := []CreatePaymentMethodTestCase{
		{
			Name:     "Success_CreditCard",
			TestCode: "ESPYNA-TEST-PAYMENT-PAYMENT-METHOD-CREATE-SUCCESS-CREDIT-CARD-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *paymentmethodpb.CreatePaymentMethodRequest {
				return &paymentmethodpb.CreatePaymentMethodRequest{
					Data: &paymentmethodpb.PaymentMethod{
						Name: createSuccessResolver.MustGetString("newPaymentMethodName"),
						MethodDetails: &paymentmethodpb.PaymentMethod_Card{
							Card: &paymentmethodpb.CardDetails{
								CardType:       createSuccessResolver.MustGetString("validCardType"),
								LastFourDigits: createSuccessResolver.MustGetString("validLastFourDigits"),
								ExpiryMonth:    int32(createSuccessResolver.MustGetInt("validExpiryMonth")),
								ExpiryYear:     int32(createSuccessResolver.MustGetInt("validExpiryYear")),
							},
						},
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *paymentmethodpb.CreatePaymentMethodResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertEqual(t, 1, len(response.Data), "response data length")
				createdPaymentMethod := response.Data[0]
				testutil.AssertStringEqual(t, createSuccessResolver.MustGetString("newPaymentMethodName"), createdPaymentMethod.Name, "payment method name")
				testutil.AssertNonEmptyString(t, createdPaymentMethod.Id, "payment method ID")
				testutil.AssertTrue(t, createdPaymentMethod.Active, "payment method active status")
				testutil.AssertFieldSet(t, createdPaymentMethod.DateCreated, "DateCreated")
				testutil.AssertFieldSet(t, createdPaymentMethod.DateCreatedString, "DateCreatedString")
			},
		},
		{
			Name:     "Success_BankAccount",
			TestCode: "ESPYNA-TEST-PAYMENT-PAYMENT-METHOD-CREATE-SUCCESS-BANK-ACCOUNT-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *paymentmethodpb.CreatePaymentMethodRequest {
				return &paymentmethodpb.CreatePaymentMethodRequest{
					Data: &paymentmethodpb.PaymentMethod{
						Name: createSuccessResolver.MustGetString("newBankAccountName"),
						MethodDetails: &paymentmethodpb.PaymentMethod_BankAccount{
							BankAccount: &paymentmethodpb.BankAccountDetails{
								BankName:       createSuccessResolver.MustGetString("validBankName"),
								LastFourDigits: createSuccessResolver.MustGetString("validBankLastFourDigits"),
							},
						},
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *paymentmethodpb.CreatePaymentMethodResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertEqual(t, 1, len(response.Data), "response data length")
				createdPaymentMethod := response.Data[0]
				testutil.AssertStringEqual(t, createSuccessResolver.MustGetString("newBankAccountName"), createdPaymentMethod.Name, "payment method name")
				testutil.AssertNonEmptyString(t, createdPaymentMethod.Id, "payment method ID")
				testutil.AssertTrue(t, createdPaymentMethod.Active, "payment method active status")
			},
		},
		{
			Name:     "WithTransaction",
			TestCode: "ESPYNA-TEST-PAYMENT-PAYMENT-METHOD-CREATE-TRANSACTION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *paymentmethodpb.CreatePaymentMethodRequest {
				return &paymentmethodpb.CreatePaymentMethodRequest{
					Data: &paymentmethodpb.PaymentMethod{
						Name: createSuccessResolver.MustGetString("newPaymentMethodName"),
						MethodDetails: &paymentmethodpb.PaymentMethod_Card{
							Card: &paymentmethodpb.CardDetails{
								CardType:       createSuccessResolver.MustGetString("validCardType"),
								LastFourDigits: createSuccessResolver.MustGetString("validLastFourDigits"),
								ExpiryMonth:    int32(createSuccessResolver.MustGetInt("validExpiryMonth")),
								ExpiryYear:     int32(createSuccessResolver.MustGetInt("validExpiryYear")),
							},
						},
					},
				}
			},
			UseTransaction: true,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *paymentmethodpb.CreatePaymentMethodResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				createdPaymentMethod := response.Data[0]
				testutil.AssertStringEqual(t, createSuccessResolver.MustGetString("newPaymentMethodName"), createdPaymentMethod.Name, "payment method name")
			},
		},
		{
			Name:     "Unauthorized",
			TestCode: "ESPYNA-TEST-PAYMENT-PAYMENT-METHOD-CREATE-AUTHORIZATION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *paymentmethodpb.CreatePaymentMethodRequest {
				return &paymentmethodpb.CreatePaymentMethodRequest{
					Data: &paymentmethodpb.PaymentMethod{
						Name: authorizationUnauthorizedResolver.MustGetString("unauthorizedPaymentMethodName"),
						MethodDetails: &paymentmethodpb.PaymentMethod_Card{
							Card: &paymentmethodpb.CardDetails{
								CardType:       authorizationUnauthorizedResolver.MustGetString("unauthorizedCardType"),
								LastFourDigits: authorizationUnauthorizedResolver.MustGetString("unauthorizedLastFourDigits"),
								ExpiryMonth:    int32(authorizationUnauthorizedResolver.MustGetInt("unauthorizedExpiryMonth")),
								ExpiryYear:     int32(authorizationUnauthorizedResolver.MustGetInt("unauthorizedExpiryYear")),
							},
						},
					},
				}
			},
			UseTransaction: false,
			UseAuth:        false,
			ExpectSuccess:  false,
			ExpectedError:  "payment_method.errors.authorization_failed",
			Assertions: func(t *testing.T, response *paymentmethodpb.CreatePaymentMethodResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertAuthorizationError(t, err)
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-PAYMENT-PAYMENT-METHOD-CREATE-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *paymentmethodpb.CreatePaymentMethodRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "payment_method.validation.request_required",
			Assertions: func(t *testing.T, response *paymentmethodpb.CreatePaymentMethodResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilRequest(t, err)
			},
		},
		{
			Name:     "NilData",
			TestCode: "ESPYNA-TEST-PAYMENT-PAYMENT-METHOD-CREATE-NIL-DATA-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *paymentmethodpb.CreatePaymentMethodRequest {
				return &paymentmethodpb.CreatePaymentMethodRequest{Data: nil}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "payment_method.validation.data_required",
			Assertions: func(t *testing.T, response *paymentmethodpb.CreatePaymentMethodResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilData(t, err)
			},
		},
		{
			Name:     "EmptyName",
			TestCode: "ESPYNA-TEST-PAYMENT-PAYMENT-METHOD-CREATE-VALIDATION-EMPTY-NAME-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *paymentmethodpb.CreatePaymentMethodRequest {
				return &paymentmethodpb.CreatePaymentMethodRequest{
					Data: &paymentmethodpb.PaymentMethod{
						Name: validationErrorEmptyNameResolver.MustGetString("emptyName"),
						MethodDetails: &paymentmethodpb.PaymentMethod_Card{
							Card: &paymentmethodpb.CardDetails{
								CardType:       validationErrorEmptyNameResolver.MustGetString("validCardType"),
								LastFourDigits: validationErrorEmptyNameResolver.MustGetString("validLastFourDigits"),
								ExpiryMonth:    int32(validationErrorEmptyNameResolver.MustGetInt("validExpiryMonth")),
								ExpiryYear:     int32(validationErrorEmptyNameResolver.MustGetInt("validExpiryYear")),
							},
						},
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "payment_method.validation.name_required",
			Assertions: func(t *testing.T, response *paymentmethodpb.CreatePaymentMethodResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty name")
			},
		},
		{
			Name:     "MissingPaymentMethodDetails",
			TestCode: "ESPYNA-TEST-PAYMENT-PAYMENT-METHOD-CREATE-VALIDATION-MISSING-DETAILS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *paymentmethodpb.CreatePaymentMethodRequest {
				return &paymentmethodpb.CreatePaymentMethodRequest{
					Data: &paymentmethodpb.PaymentMethod{
						Name: validationErrorMissingDetailsResolver.MustGetString("validName"),
						// Missing MethodDetails
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "payment_method.validation.details_required",
			Assertions: func(t *testing.T, response *paymentmethodpb.CreatePaymentMethodResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "missing payment method details")
			},
		},
		{
			Name:     "InvalidCardType",
			TestCode: "ESPYNA-TEST-PAYMENT-PAYMENT-METHOD-CREATE-VALIDATION-INVALID-CARD-TYPE-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *paymentmethodpb.CreatePaymentMethodRequest {
				return &paymentmethodpb.CreatePaymentMethodRequest{
					Data: &paymentmethodpb.PaymentMethod{
						Name: validationErrorInvalidCardTypeResolver.MustGetString("validName"),
						MethodDetails: &paymentmethodpb.PaymentMethod_Card{
							Card: &paymentmethodpb.CardDetails{
								CardType:       validationErrorInvalidCardTypeResolver.MustGetString("invalidCardType"),
								LastFourDigits: validationErrorInvalidCardTypeResolver.MustGetString("validLastFourDigits"),
								ExpiryMonth:    int32(validationErrorInvalidCardTypeResolver.MustGetInt("validExpiryMonth")),
								ExpiryYear:     int32(validationErrorInvalidCardTypeResolver.MustGetInt("validExpiryYear")),
							},
						},
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "payment_method.validation.invalid_card_type",
			Assertions: func(t *testing.T, response *paymentmethodpb.CreatePaymentMethodResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "invalid card type")
			},
		},
		{
			Name:     "InvalidExpiryMonth",
			TestCode: "ESPYNA-TEST-PAYMENT-PAYMENT-METHOD-CREATE-VALIDATION-INVALID-EXPIRY-MONTH-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *paymentmethodpb.CreatePaymentMethodRequest {
				return &paymentmethodpb.CreatePaymentMethodRequest{
					Data: &paymentmethodpb.PaymentMethod{
						Name: "Test Card Invalid Month",
						MethodDetails: &paymentmethodpb.PaymentMethod_Card{
							Card: &paymentmethodpb.CardDetails{
								CardType:       "Visa",
								LastFourDigits: "1234",
								ExpiryMonth:    13, // Invalid month
								ExpiryYear:     int32(time.Now().Year() + 1),
							},
						},
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "payment_method.validation.invalid_expiry_month",
			Assertions: func(t *testing.T, response *paymentmethodpb.CreatePaymentMethodResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "invalid expiry month")
			},
		},
		{
			Name:     "InvalidExpiryYear",
			TestCode: "ESPYNA-TEST-PAYMENT-PAYMENT-METHOD-CREATE-VALIDATION-INVALID-EXPIRY-YEAR-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *paymentmethodpb.CreatePaymentMethodRequest {
				return &paymentmethodpb.CreatePaymentMethodRequest{
					Data: &paymentmethodpb.PaymentMethod{
						Name: "Test Card Invalid Year",
						MethodDetails: &paymentmethodpb.PaymentMethod_Card{
							Card: &paymentmethodpb.CardDetails{
								CardType:       "Visa",
								LastFourDigits: "1234",
								ExpiryMonth:    12,
								ExpiryYear:     int32(time.Now().Year() - 1), // Past year
							},
						},
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "payment_method.validation.invalid_expiry_year",
			Assertions: func(t *testing.T, response *paymentmethodpb.CreatePaymentMethodResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "invalid expiry year")
			},
		},
		{
			Name:     "InvalidBankName",
			TestCode: "ESPYNA-TEST-PAYMENT-PAYMENT-METHOD-CREATE-VALIDATION-INVALID-BANK-NAME-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *paymentmethodpb.CreatePaymentMethodRequest {
				return &paymentmethodpb.CreatePaymentMethodRequest{
					Data: &paymentmethodpb.PaymentMethod{
						Name: "Test Bank Account",
						MethodDetails: &paymentmethodpb.PaymentMethod_BankAccount{
							BankAccount: &paymentmethodpb.BankAccountDetails{
								BankName:       "", // Empty bank name
								LastFourDigits: "5678",
							},
						},
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "payment_method.validation.bank_name_required",
			Assertions: func(t *testing.T, response *paymentmethodpb.CreatePaymentMethodResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty bank name")
			},
		},
		{
			Name:     "DataEnrichment",
			TestCode: "ESPYNA-TEST-PAYMENT-PAYMENT-METHOD-CREATE-ENRICHMENT-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *paymentmethodpb.CreatePaymentMethodRequest {
				return &paymentmethodpb.CreatePaymentMethodRequest{
					Data: &paymentmethodpb.PaymentMethod{
						Name: "Data Enrichment Test",
						MethodDetails: &paymentmethodpb.PaymentMethod_Card{
							Card: &paymentmethodpb.CardDetails{
								CardType:       "Visa",
								LastFourDigits: "1234",
								ExpiryMonth:    12,
								ExpiryYear:     int32(time.Now().Year() + 1),
							},
						},
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *paymentmethodpb.CreatePaymentMethodResponse, err error, useCase interface{}, ctx context.Context) {
				createdPaymentMethod := response.Data[0]
				testutil.AssertNonEmptyString(t, createdPaymentMethod.Id, "generated ID")
				testutil.AssertFieldSet(t, createdPaymentMethod.DateCreated, "DateCreated")
				testutil.AssertFieldSet(t, createdPaymentMethod.DateCreatedString, "DateCreatedString")
				testutil.AssertTrue(t, createdPaymentMethod.Active, "Active")
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
			useCase := createTestUseCaseWithAuth(businessType, tc.UseTransaction, tc.UseAuth)

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

func TestCreatePaymentMethodUseCase_Execute_TransactionFailure(t *testing.T) {
	testCode := "ESPYNA-TEST-PAYMENT-PAYMENT-METHOD-CREATE-TRANSACTION-FAILURE-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "TransactionFailure", false)

	// Create a use case with a failing transaction service
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	mockPaymentMethodRepo := payment.NewMockPaymentMethodRepository(businessType)

	repositories := CreatePaymentMethodRepositories{
		PaymentMethod: mockPaymentMethodRepo,
	}

	standardServices := testutil.CreateStandardServices(false, true)
	services := CreatePaymentMethodServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   mockDb.NewFailingMockTransactionService(),
		TranslationService:   standardServices.TranslationService,
		IDService:            standardServices.IDService,
	}

	useCase := NewCreatePaymentMethodUseCase(repositories, services)

	// Load test case using centralized TestCaseResolver
	resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "payment_method", "CreatePaymentMethod_Success")
	testutil.AssertTestCaseLoad(t, err, "CreatePaymentMethod_Success")

	req := &paymentmethodpb.CreatePaymentMethodRequest{
		Data: &paymentmethodpb.PaymentMethod{
			Name: resolver.MustGetString("newPaymentMethodName"),
			MethodDetails: &paymentmethodpb.PaymentMethod_Card{
				Card: &paymentmethodpb.CardDetails{
					CardType:       resolver.MustGetString("validCardType"),
					LastFourDigits: resolver.MustGetString("validLastFourDigits"),
					ExpiryMonth:    int32(resolver.MustGetInt("validExpiryMonth")),
					ExpiryYear:     int32(resolver.MustGetInt("validExpiryYear")),
				},
			},
		},
	}

	_, err = useCase.Execute(ctx, req)

	// Verify that a transaction error occurred
	testutil.AssertTransactionError(t, err)

	// Log completion of transaction failure test
	testutil.LogTestResult(t, testCode, "TransactionFailure", false, err)
}
