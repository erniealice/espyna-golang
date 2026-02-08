//go:build mock_db && mock_auth

// Package payment provides table-driven tests for the payment updating use case.
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
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestUpdatePaymentUseCase_Execute_TableDriven
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-PAYMENT-PAYMENT-UPDATE-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-PAYMENT-PAYMENT-UPDATE-TRANSACTION-v1.0: WithTransaction
//   - ESPYNA-TEST-PAYMENT-PAYMENT-UPDATE-AUTHORIZATION-v1.0: Unauthorized
//   - ESPYNA-TEST-PAYMENT-PAYMENT-UPDATE-NIL-REQUEST-v1.0: NilRequest
//   - ESPYNA-TEST-PAYMENT-PAYMENT-UPDATE-NIL-DATA-v1.0: NilData
//   - ESPYNA-TEST-PAYMENT-PAYMENT-UPDATE-VALIDATION-EMPTY-ID-v1.0: EmptyId
//   - ESPYNA-TEST-PAYMENT-PAYMENT-UPDATE-VALIDATION-INVALID-ID-v1.0: InvalidId
//   - ESPYNA-TEST-PAYMENT-PAYMENT-UPDATE-VALIDATION-EMPTY-NAME-v1.0: EmptyName
//   - ESPYNA-TEST-PAYMENT-PAYMENT-UPDATE-VALIDATION-NAME-TOO-SHORT-v1.0: NameTooShort
//   - ESPYNA-TEST-PAYMENT-PAYMENT-UPDATE-VALIDATION-NAME-TOO-LONG-v1.0: NameTooLong
//   - ESPYNA-TEST-PAYMENT-PAYMENT-UPDATE-TRANSACTION-FAILURE-v1.0: TransactionFailure
//
// Data Sources:
//   - Test cases: packages/copya/data_test/{businessType}/payment.json
//   - Mock data: packages/copya/data/{businessType}/payment.json
//   - Translations: packages/lyngua/translations/{languageCode}/{businessType}/payment.json

package payment

import (
	"context"
	"testing"

	copyatestutil "leapfor.xyz/copya/golang/testutil"
	"github.com/erniealice/espyna-golang/internal/application/shared/testutil"
	mockDb "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock"
	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock/payment"
	subscriptionMock "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock/subscription"
	paymentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/payment/payment"
)

// Type alias for update payment test cases
type UpdatePaymentTestCase = testutil.GenericTestCase[*paymentpb.UpdatePaymentRequest, *paymentpb.UpdatePaymentResponse]

// Test helper to create use case with real services and foreign key validation
func createUpdateTestUseCase(businessType string, supportsTransaction bool) *UpdatePaymentUseCase {
	return createUpdateTestUseCaseWithAuth(businessType, supportsTransaction, true)
}

func createUpdateTestUseCaseWithAuth(businessType string, supportsTransaction bool, shouldAuthorize bool) *UpdatePaymentUseCase {
	paymentRepo := payment.NewMockPaymentRepository(businessType)
	subscriptionRepo := subscriptionMock.NewMockSubscriptionRepository(businessType) // For foreign key validation

	repositories := UpdatePaymentRepositories{
		Payment:      paymentRepo,
		Subscription: subscriptionRepo, // Foreign key validation
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, shouldAuthorize)
	services := UpdatePaymentServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
	}

	return NewUpdatePaymentUseCase(repositories, services)
}

func TestUpdatePaymentUseCase_Execute_TableDriven(t *testing.T) {
	businessType := testutil.GetTestBusinessType()

	// Load test data resolvers before defining test cases
	updateSuccessResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "payment", "UpdatePayment_Success")
	testutil.AssertTestCaseLoad(t, err, "UpdatePayment_Success")

	authorizationUnauthorizedResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "payment", "Authorization_Unauthorized")
	testutil.AssertTestCaseLoad(t, err, "Authorization_Unauthorized")

	validationErrorEmptyIdResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "payment", "ValidationError_EmptyId")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_EmptyId")

	validationErrorInvalidIdResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "payment", "ValidationError_InvalidId")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_InvalidId")

	validationErrorEmptyNameResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "payment", "ValidationError_EmptyName")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_EmptyName")

	testCases := []UpdatePaymentTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-PAYMENT-PAYMENT-UPDATE-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *paymentpb.UpdatePaymentRequest {
				return &paymentpb.UpdatePaymentRequest{
					Data: &paymentpb.Payment{
						Id:             updateSuccessResolver.MustGetString("validPaymentId"),
						Name:           updateSuccessResolver.MustGetString("updatedPaymentName"),
						SubscriptionId: updateSuccessResolver.MustGetString("validSubscriptionId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *paymentpb.UpdatePaymentResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertEqual(t, 1, len(response.Data), "response data length")
				updatedPayment := response.Data[0]
				testutil.AssertStringEqual(t, updateSuccessResolver.MustGetString("updatedPaymentName"), updatedPayment.Name, "payment name")
				testutil.AssertStringEqual(t, updateSuccessResolver.MustGetString("validPaymentId"), updatedPayment.Id, "payment ID")
				testutil.AssertFieldSet(t, updatedPayment.DateModified, "DateModified")
				testutil.AssertFieldSet(t, updatedPayment.DateModifiedString, "DateModifiedString")
			},
		},
		{
			Name:     "WithTransaction",
			TestCode: "ESPYNA-TEST-PAYMENT-PAYMENT-UPDATE-TRANSACTION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *paymentpb.UpdatePaymentRequest {
				return &paymentpb.UpdatePaymentRequest{
					Data: &paymentpb.Payment{
						Id:             updateSuccessResolver.MustGetString("validPaymentId"),
						Name:           updateSuccessResolver.MustGetString("updatedPaymentName"),
						SubscriptionId: updateSuccessResolver.MustGetString("validSubscriptionId"),
					},
				}
			},
			UseTransaction: true,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *paymentpb.UpdatePaymentResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				updatedPayment := response.Data[0]
				testutil.AssertStringEqual(t, updateSuccessResolver.MustGetString("updatedPaymentName"), updatedPayment.Name, "payment name")
			},
		},
		{
			Name:     "Unauthorized",
			TestCode: "ESPYNA-TEST-PAYMENT-PAYMENT-UPDATE-AUTHORIZATION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *paymentpb.UpdatePaymentRequest {
				return &paymentpb.UpdatePaymentRequest{
					Data: &paymentpb.Payment{
						Id:             authorizationUnauthorizedResolver.MustGetString("unauthorizedPaymentId"),
						Name:           authorizationUnauthorizedResolver.MustGetString("unauthorizedPaymentName"),
						SubscriptionId: authorizationUnauthorizedResolver.MustGetString("unauthorizedSubscriptionId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        false,
			ExpectSuccess:  false,
			ExpectedError:  "payment.errors.authorization_failed",
			Assertions: func(t *testing.T, response *paymentpb.UpdatePaymentResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertAuthorizationError(t, err)
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-PAYMENT-PAYMENT-UPDATE-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *paymentpb.UpdatePaymentRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "payment.validation.request_required",
			Assertions: func(t *testing.T, response *paymentpb.UpdatePaymentResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilRequest(t, err)
			},
		},
		{
			Name:     "NilData",
			TestCode: "ESPYNA-TEST-PAYMENT-PAYMENT-UPDATE-NIL-DATA-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *paymentpb.UpdatePaymentRequest {
				return &paymentpb.UpdatePaymentRequest{Data: nil}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "payment.validation.data_required",
			Assertions: func(t *testing.T, response *paymentpb.UpdatePaymentResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilData(t, err)
			},
		},
		{
			Name:     "EmptyId",
			TestCode: "ESPYNA-TEST-PAYMENT-PAYMENT-UPDATE-VALIDATION-EMPTY-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *paymentpb.UpdatePaymentRequest {
				return &paymentpb.UpdatePaymentRequest{
					Data: &paymentpb.Payment{
						Id:             validationErrorEmptyIdResolver.MustGetString("emptyId"),
						Name:           validationErrorEmptyIdResolver.MustGetString("validName"),
						SubscriptionId: validationErrorEmptyIdResolver.MustGetString("validSubscriptionId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "payment.validation.id_required",
			Assertions: func(t *testing.T, response *paymentpb.UpdatePaymentResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty ID")
			},
		},
		{
			Name:     "InvalidId",
			TestCode: "ESPYNA-TEST-PAYMENT-PAYMENT-UPDATE-VALIDATION-INVALID-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *paymentpb.UpdatePaymentRequest {
				return &paymentpb.UpdatePaymentRequest{
					Data: &paymentpb.Payment{
						Id:             validationErrorInvalidIdResolver.MustGetString("invalidId"),
						Name:           validationErrorInvalidIdResolver.MustGetString("validName"),
						SubscriptionId: validationErrorInvalidIdResolver.MustGetString("validSubscriptionId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "payment.validation.id_invalid",
			Assertions: func(t *testing.T, response *paymentpb.UpdatePaymentResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "invalid ID")
			},
		},
		{
			Name:     "EmptyName",
			TestCode: "ESPYNA-TEST-PAYMENT-PAYMENT-UPDATE-VALIDATION-EMPTY-NAME-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *paymentpb.UpdatePaymentRequest {
				return &paymentpb.UpdatePaymentRequest{
					Data: &paymentpb.Payment{
						Id:             updateSuccessResolver.MustGetString("validPaymentId"),
						Name:           validationErrorEmptyNameResolver.MustGetString("emptyName"),
						SubscriptionId: validationErrorEmptyNameResolver.MustGetString("validSubscriptionId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "payment.validation.name_required",
			Assertions: func(t *testing.T, response *paymentpb.UpdatePaymentResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty name")
			},
		},
		{
			Name:     "NameTooShort",
			TestCode: "ESPYNA-TEST-PAYMENT-PAYMENT-UPDATE-VALIDATION-NAME-TOO-SHORT-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *paymentpb.UpdatePaymentRequest {
				return &paymentpb.UpdatePaymentRequest{
					Data: &paymentpb.Payment{
						Id:             updateSuccessResolver.MustGetString("validPaymentId"),
						Name:           "AB",
						SubscriptionId: updateSuccessResolver.MustGetString("validSubscriptionId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "payment.validation.name_too_short",
			Assertions: func(t *testing.T, response *paymentpb.UpdatePaymentResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "name too short")
			},
		},
		{
			Name:     "NameTooLong",
			TestCode: "ESPYNA-TEST-PAYMENT-PAYMENT-UPDATE-VALIDATION-NAME-TOO-LONG-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *paymentpb.UpdatePaymentRequest {
				resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "payment", "ValidationError_NameTooLongGenerated")
				testutil.AssertTestCaseLoad(t, err, "ValidationError_NameTooLongGenerated")
				return &paymentpb.UpdatePaymentRequest{
					Data: &paymentpb.Payment{
						Id:             resolver.MustGetString("validPaymentId"),
						Name:           resolver.MustGetString("tooLongNameGenerated"),
						SubscriptionId: resolver.MustGetString("validSubscriptionId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "payment.validation.name_too_long",
			Assertions: func(t *testing.T, response *paymentpb.UpdatePaymentResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "name too long")
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

func TestUpdatePaymentUseCase_Execute_TransactionFailure(t *testing.T) {
	testCode := "ESPYNA-TEST-PAYMENT-PAYMENT-UPDATE-TRANSACTION-FAILURE-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "TransactionFailure", false)

	// Create a use case with a failing transaction service
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	mockPaymentRepo := payment.NewMockPaymentRepository(businessType)
	mockSubscriptionRepo := subscriptionMock.NewMockSubscriptionRepository(businessType)

	repositories := UpdatePaymentRepositories{
		Payment:      mockPaymentRepo,
		Subscription: mockSubscriptionRepo,
	}

	standardServices := testutil.CreateStandardServices(false, true)
	services := UpdatePaymentServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   mockDb.NewFailingMockTransactionService(),
		TranslationService:   standardServices.TranslationService,
	}

	useCase := NewUpdatePaymentUseCase(repositories, services)

	// Load test case using centralized TestCaseResolver
	resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "payment", "UpdatePayment_Success")
	testutil.AssertTestCaseLoad(t, err, "UpdatePayment_Success")

	req := &paymentpb.UpdatePaymentRequest{
		Data: &paymentpb.Payment{
			Id:             resolver.MustGetString("validPaymentId"),
			Name:           resolver.MustGetString("updatedPaymentName"),
			SubscriptionId: resolver.MustGetString("validSubscriptionId"),
		},
	}

	_, err = useCase.Execute(ctx, req)

	// Verify that a transaction error occurred
	testutil.AssertTransactionError(t, err)

	// Log completion of transaction failure test
	testutil.LogTestResult(t, testCode, "TransactionFailure", false, err)
}
