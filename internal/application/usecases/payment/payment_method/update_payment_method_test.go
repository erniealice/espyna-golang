//go:build mock_db && mock_auth

// Package payment_method provides table-driven tests for the payment method update use case.
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
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestUpdatePaymentMethodUseCase_Execute_TableDriven
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-PAYMENT-PAYMENT-METHOD-UPDATE-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-PAYMENT-PAYMENT-METHOD-UPDATE-TRANSACTION-v1.0: WithTransaction
//   - ESPYNA-TEST-PAYMENT-PAYMENT-METHOD-UPDATE-AUTHORIZATION-v1.0: Unauthorized
//   - ESPYNA-TEST-PAYMENT-PAYMENT-METHOD-UPDATE-NIL-REQUEST-v1.0: NilRequest
//   - ESPYNA-TEST-PAYMENT-PAYMENT-METHOD-UPDATE-NIL-DATA-v1.0: NilData
//   - ESPYNA-TEST-PAYMENT-PAYMENT-METHOD-UPDATE-VALIDATION-EMPTY-ID-v1.0: EmptyId
//   - ESPYNA-TEST-PAYMENT-PAYMENT-METHOD-UPDATE-VALIDATION-EMPTY-NAME-v1.0: EmptyName
//   - ESPYNA-TEST-PAYMENT-PAYMENT-METHOD-UPDATE-VALIDATION-NON-EXISTENT-v1.0: NonExistentPaymentMethod
//   - ESPYNA-TEST-PAYMENT-PAYMENT-METHOD-UPDATE-VALIDATION-INVALID-CARD-TYPE-v1.0: InvalidCardType
//   - ESPYNA-TEST-PAYMENT-PAYMENT-METHOD-UPDATE-ENRICHMENT-v1.0: DataEnrichment
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
	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock/payment"
	paymentmethodpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/payment/payment_method"
)

// Type alias for update payment method test cases
type UpdatePaymentMethodTestCase = testutil.GenericTestCase[*paymentmethodpb.UpdatePaymentMethodRequest, *paymentmethodpb.UpdatePaymentMethodResponse]

// Test helper to create use case with real services
func createUpdateTestUseCase(businessType string, supportsTransaction bool) *UpdatePaymentMethodUseCase {
	return createUpdateTestUseCaseWithAuth(businessType, supportsTransaction, true)
}

func createUpdateTestUseCaseWithAuth(businessType string, supportsTransaction bool, shouldAuthorize bool) *UpdatePaymentMethodUseCase {
	paymentMethodRepo := payment.NewMockPaymentMethodRepository(businessType)

	repositories := UpdatePaymentMethodRepositories{
		PaymentMethod: paymentMethodRepo,
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, shouldAuthorize)
	services := UpdatePaymentMethodServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
	}

	return NewUpdatePaymentMethodUseCase(repositories, services)
}

func TestUpdatePaymentMethodUseCase_Execute_TableDriven(t *testing.T) {
	businessType := testutil.GetTestBusinessType()

	// Load test data resolvers before defining test cases
	updateSuccessResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "payment_method", "UpdatePaymentMethod_Success")
	testutil.AssertTestCaseLoad(t, err, "UpdatePaymentMethod_Success")

	authorizationUnauthorizedResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "payment_method", "Authorization_Unauthorized")
	testutil.AssertTestCaseLoad(t, err, "Authorization_Unauthorized")

	validationErrorEmptyIdResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "payment_method", "ValidationError_EmptyId")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_EmptyId")

	validationErrorEmptyNameResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "payment_method", "ValidationError_EmptyName")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_EmptyName")

	validationErrorNonExistentResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "payment_method", "ValidationError_NonExistent")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_NonExistent")

	validationErrorInvalidCardTypeResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "payment_method", "ValidationError_InvalidCardType")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_InvalidCardType")

	testCases := []UpdatePaymentMethodTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-PAYMENT-PAYMENT-METHOD-UPDATE-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *paymentmethodpb.UpdatePaymentMethodRequest {
				return &paymentmethodpb.UpdatePaymentMethodRequest{
					Data: &paymentmethodpb.PaymentMethod{
						Id:   updateSuccessResolver.MustGetString("existingPaymentMethodId"),
						Name: updateSuccessResolver.MustGetString("updatedPaymentMethodName"),
						MethodDetails: &paymentmethodpb.PaymentMethod_Card{
							Card: &paymentmethodpb.CardDetails{
								CardType:       updateSuccessResolver.MustGetString("updatedCardType"),
								LastFourDigits: updateSuccessResolver.MustGetString("updatedLastFourDigits"),
								ExpiryMonth:    int32(updateSuccessResolver.MustGetInt("updatedExpiryMonth")),
								ExpiryYear:     int32(updateSuccessResolver.MustGetInt("updatedExpiryYear")),
							},
						},
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *paymentmethodpb.UpdatePaymentMethodResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertEqual(t, 1, len(response.Data), "response data length")
				updatedPaymentMethod := response.Data[0]
				testutil.AssertStringEqual(t, updateSuccessResolver.MustGetString("existingPaymentMethodId"), updatedPaymentMethod.Id, "payment method ID")
				testutil.AssertStringEqual(t, updateSuccessResolver.MustGetString("updatedPaymentMethodName"), updatedPaymentMethod.Name, "payment method name")
				testutil.AssertFieldSet(t, updatedPaymentMethod.DateModified, "DateModified")
				testutil.AssertFieldSet(t, updatedPaymentMethod.DateModifiedString, "DateModifiedString")
			},
		},
		{
			Name:     "WithTransaction",
			TestCode: "ESPYNA-TEST-PAYMENT-PAYMENT-METHOD-UPDATE-TRANSACTION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *paymentmethodpb.UpdatePaymentMethodRequest {
				return &paymentmethodpb.UpdatePaymentMethodRequest{
					Data: &paymentmethodpb.PaymentMethod{
						Id:   updateSuccessResolver.MustGetString("existingPaymentMethodId"),
						Name: updateSuccessResolver.MustGetString("updatedPaymentMethodName"),
						MethodDetails: &paymentmethodpb.PaymentMethod_Card{
							Card: &paymentmethodpb.CardDetails{
								CardType:       updateSuccessResolver.MustGetString("updatedCardType"),
								LastFourDigits: updateSuccessResolver.MustGetString("updatedLastFourDigits"),
								ExpiryMonth:    int32(updateSuccessResolver.MustGetInt("updatedExpiryMonth")),
								ExpiryYear:     int32(updateSuccessResolver.MustGetInt("updatedExpiryYear")),
							},
						},
					},
				}
			},
			UseTransaction: true,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *paymentmethodpb.UpdatePaymentMethodResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				updatedPaymentMethod := response.Data[0]
				testutil.AssertStringEqual(t, updateSuccessResolver.MustGetString("existingPaymentMethodId"), updatedPaymentMethod.Id, "payment method ID")
			},
		},
		{
			Name:     "Unauthorized",
			TestCode: "ESPYNA-TEST-PAYMENT-PAYMENT-METHOD-UPDATE-AUTHORIZATION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *paymentmethodpb.UpdatePaymentMethodRequest {
				return &paymentmethodpb.UpdatePaymentMethodRequest{
					Data: &paymentmethodpb.PaymentMethod{
						Id:   authorizationUnauthorizedResolver.MustGetString("unauthorizedPaymentMethodId"),
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
			Assertions: func(t *testing.T, response *paymentmethodpb.UpdatePaymentMethodResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertAuthorizationError(t, err)
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-PAYMENT-PAYMENT-METHOD-UPDATE-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *paymentmethodpb.UpdatePaymentMethodRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "payment_method.validation.request_required",
			Assertions: func(t *testing.T, response *paymentmethodpb.UpdatePaymentMethodResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilRequest(t, err)
			},
		},
		{
			Name:     "NilData",
			TestCode: "ESPYNA-TEST-PAYMENT-PAYMENT-METHOD-UPDATE-NIL-DATA-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *paymentmethodpb.UpdatePaymentMethodRequest {
				return &paymentmethodpb.UpdatePaymentMethodRequest{Data: nil}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "payment_method.validation.data_required",
			Assertions: func(t *testing.T, response *paymentmethodpb.UpdatePaymentMethodResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilData(t, err)
			},
		},
		{
			Name:     "EmptyId",
			TestCode: "ESPYNA-TEST-PAYMENT-PAYMENT-METHOD-UPDATE-VALIDATION-EMPTY-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *paymentmethodpb.UpdatePaymentMethodRequest {
				return &paymentmethodpb.UpdatePaymentMethodRequest{
					Data: &paymentmethodpb.PaymentMethod{
						Id:   validationErrorEmptyIdResolver.MustGetString("emptyId"),
						Name: validationErrorEmptyIdResolver.MustGetString("validName"),
						MethodDetails: &paymentmethodpb.PaymentMethod_Card{
							Card: &paymentmethodpb.CardDetails{
								CardType:       validationErrorEmptyIdResolver.MustGetString("validCardType"),
								LastFourDigits: validationErrorEmptyIdResolver.MustGetString("validLastFourDigits"),
								ExpiryMonth:    int32(validationErrorEmptyIdResolver.MustGetInt("validExpiryMonth")),
								ExpiryYear:     int32(validationErrorEmptyIdResolver.MustGetInt("validExpiryYear")),
							},
						},
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "payment_method.validation.id_required",
			Assertions: func(t *testing.T, response *paymentmethodpb.UpdatePaymentMethodResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty ID")
			},
		},
		{
			Name:     "EmptyName",
			TestCode: "ESPYNA-TEST-PAYMENT-PAYMENT-METHOD-UPDATE-VALIDATION-EMPTY-NAME-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *paymentmethodpb.UpdatePaymentMethodRequest {
				return &paymentmethodpb.UpdatePaymentMethodRequest{
					Data: &paymentmethodpb.PaymentMethod{
						Id:   validationErrorEmptyNameResolver.MustGetString("validId"),
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
			Assertions: func(t *testing.T, response *paymentmethodpb.UpdatePaymentMethodResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty name")
			},
		},
		{
			Name:     "NonExistentPaymentMethod",
			TestCode: "ESPYNA-TEST-PAYMENT-PAYMENT-METHOD-UPDATE-VALIDATION-NON-EXISTENT-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *paymentmethodpb.UpdatePaymentMethodRequest {
				return &paymentmethodpb.UpdatePaymentMethodRequest{
					Data: &paymentmethodpb.PaymentMethod{
						Id:   validationErrorNonExistentResolver.MustGetString("nonExistentPaymentMethodId"),
						Name: validationErrorNonExistentResolver.MustGetString("validName"),
						MethodDetails: &paymentmethodpb.PaymentMethod_Card{
							Card: &paymentmethodpb.CardDetails{
								CardType:       validationErrorNonExistentResolver.MustGetString("validCardType"),
								LastFourDigits: validationErrorNonExistentResolver.MustGetString("validLastFourDigits"),
								ExpiryMonth:    int32(validationErrorNonExistentResolver.MustGetInt("validExpiryMonth")),
								ExpiryYear:     int32(validationErrorNonExistentResolver.MustGetInt("validExpiryYear")),
							},
						},
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "payment_method.errors.not_found",
			ErrorTags:      map[string]any{"paymentMethodId": validationErrorNonExistentResolver.MustGetString("nonExistentPaymentMethodId")},
			Assertions: func(t *testing.T, response *paymentmethodpb.UpdatePaymentMethodResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "non-existent payment method")
			},
		},
		{
			Name:     "InvalidCardType",
			TestCode: "ESPYNA-TEST-PAYMENT-PAYMENT-METHOD-UPDATE-VALIDATION-INVALID-CARD-TYPE-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *paymentmethodpb.UpdatePaymentMethodRequest {
				return &paymentmethodpb.UpdatePaymentMethodRequest{
					Data: &paymentmethodpb.PaymentMethod{
						Id:   validationErrorInvalidCardTypeResolver.MustGetString("validId"),
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
			Assertions: func(t *testing.T, response *paymentmethodpb.UpdatePaymentMethodResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "invalid card type")
			},
		},
		{
			Name:     "DataEnrichment",
			TestCode: "ESPYNA-TEST-PAYMENT-PAYMENT-METHOD-UPDATE-ENRICHMENT-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *paymentmethodpb.UpdatePaymentMethodRequest {
				return &paymentmethodpb.UpdatePaymentMethodRequest{
					Data: &paymentmethodpb.PaymentMethod{
						Id:   "payment-method-001",
						Name: "Data Enrichment Test Update",
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
			Assertions: func(t *testing.T, response *paymentmethodpb.UpdatePaymentMethodResponse, err error, useCase interface{}, ctx context.Context) {
				updatedPaymentMethod := response.Data[0]
				testutil.AssertStringEqual(t, "payment-method-001", updatedPaymentMethod.Id, "payment method ID")
				testutil.AssertFieldSet(t, updatedPaymentMethod.DateModified, "DateModified")
				testutil.AssertFieldSet(t, updatedPaymentMethod.DateModifiedString, "DateModifiedString")
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
			useCase := createUpdateTestUseCaseWithAuth(businessType, tc.UseTransaction, tc.UseAuth)

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
