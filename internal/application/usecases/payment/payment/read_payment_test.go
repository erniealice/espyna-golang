//go:build mock_db && mock_auth

// Package payment provides table-driven tests for the payment reading use case.
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
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestReadPaymentUseCase_Execute_TableDriven
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-PAYMENT-PAYMENT-READ-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-PAYMENT-PAYMENT-READ-TRANSACTION-v1.0: WithTransaction
//   - ESPYNA-TEST-PAYMENT-PAYMENT-READ-AUTHORIZATION-v1.0: Unauthorized
//   - ESPYNA-TEST-PAYMENT-PAYMENT-READ-NIL-REQUEST-v1.0: NilRequest
//   - ESPYNA-TEST-PAYMENT-PAYMENT-READ-NIL-DATA-v1.0: NilData
//   - ESPYNA-TEST-PAYMENT-PAYMENT-READ-VALIDATION-EMPTY-ID-v1.0: EmptyId
//   - ESPYNA-TEST-PAYMENT-PAYMENT-READ-VALIDATION-INVALID-ID-v1.0: InvalidId
//   - ESPYNA-TEST-PAYMENT-PAYMENT-READ-TRANSACTION-FAILURE-v1.0: TransactionFailure
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
	paymentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/payment/payment"
)

// Type alias for read payment test cases
type ReadPaymentTestCase = testutil.GenericTestCase[*paymentpb.ReadPaymentRequest, *paymentpb.ReadPaymentResponse]

// Test helper to create use case with real services
func createReadTestUseCase(businessType string, supportsTransaction bool) *ReadPaymentUseCase {
	return createReadTestUseCaseWithAuth(businessType, supportsTransaction, true)
}

func createReadTestUseCaseWithAuth(businessType string, supportsTransaction bool, shouldAuthorize bool) *ReadPaymentUseCase {
	paymentRepo := payment.NewMockPaymentRepository(businessType)

	repositories := ReadPaymentRepositories{
		Payment: paymentRepo,
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, shouldAuthorize)
	services := ReadPaymentServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
	}

	return NewReadPaymentUseCase(repositories, services)
}

func TestReadPaymentUseCase_Execute_TableDriven(t *testing.T) {
	businessType := testutil.GetTestBusinessType()

	// Load test data resolvers before defining test cases
	readSuccessResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "payment", "ReadPayment_Success")
	testutil.AssertTestCaseLoad(t, err, "ReadPayment_Success")

	authorizationUnauthorizedResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "payment", "Authorization_Unauthorized")
	testutil.AssertTestCaseLoad(t, err, "Authorization_Unauthorized")

	validationErrorEmptyIdResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "payment", "ValidationError_EmptyId")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_EmptyId")

	validationErrorInvalidIdResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "payment", "ValidationError_InvalidId")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_InvalidId")

	testCases := []ReadPaymentTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-PAYMENT-PAYMENT-READ-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *paymentpb.ReadPaymentRequest {
				return &paymentpb.ReadPaymentRequest{
					Data: &paymentpb.Payment{
						Id: readSuccessResolver.MustGetString("validPaymentId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *paymentpb.ReadPaymentResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertEqual(t, 1, len(response.Data), "response data length")
				readPayment := response.Data[0]
				testutil.AssertStringEqual(t, readSuccessResolver.MustGetString("validPaymentId"), readPayment.Id, "payment ID")
				testutil.AssertNonEmptyString(t, readPayment.Name, "payment name")
			},
		},
		{
			Name:     "WithTransaction",
			TestCode: "ESPYNA-TEST-PAYMENT-PAYMENT-READ-TRANSACTION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *paymentpb.ReadPaymentRequest {
				return &paymentpb.ReadPaymentRequest{
					Data: &paymentpb.Payment{
						Id: readSuccessResolver.MustGetString("validPaymentId"),
					},
				}
			},
			UseTransaction: true,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *paymentpb.ReadPaymentResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				readPayment := response.Data[0]
				testutil.AssertStringEqual(t, readSuccessResolver.MustGetString("validPaymentId"), readPayment.Id, "payment ID")
			},
		},
		{
			Name:     "Unauthorized",
			TestCode: "ESPYNA-TEST-PAYMENT-PAYMENT-READ-AUTHORIZATION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *paymentpb.ReadPaymentRequest {
				return &paymentpb.ReadPaymentRequest{
					Data: &paymentpb.Payment{
						Id: authorizationUnauthorizedResolver.MustGetString("unauthorizedPaymentId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        false,
			ExpectSuccess:  false,
			ExpectedError:  "payment.errors.authorization_failed",
			Assertions: func(t *testing.T, response *paymentpb.ReadPaymentResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertAuthorizationError(t, err)
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-PAYMENT-PAYMENT-READ-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *paymentpb.ReadPaymentRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "payment.validation.request_required",
			Assertions: func(t *testing.T, response *paymentpb.ReadPaymentResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilRequest(t, err)
			},
		},
		{
			Name:     "NilData",
			TestCode: "ESPYNA-TEST-PAYMENT-PAYMENT-READ-NIL-DATA-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *paymentpb.ReadPaymentRequest {
				return &paymentpb.ReadPaymentRequest{Data: nil}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "payment.validation.data_required",
			Assertions: func(t *testing.T, response *paymentpb.ReadPaymentResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilData(t, err)
			},
		},
		{
			Name:     "EmptyId",
			TestCode: "ESPYNA-TEST-PAYMENT-PAYMENT-READ-VALIDATION-EMPTY-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *paymentpb.ReadPaymentRequest {
				return &paymentpb.ReadPaymentRequest{
					Data: &paymentpb.Payment{
						Id: validationErrorEmptyIdResolver.MustGetString("emptyId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "payment.validation.id_required",
			Assertions: func(t *testing.T, response *paymentpb.ReadPaymentResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty ID")
			},
		},
		{
			Name:     "InvalidId",
			TestCode: "ESPYNA-TEST-PAYMENT-PAYMENT-READ-VALIDATION-INVALID-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *paymentpb.ReadPaymentRequest {
				return &paymentpb.ReadPaymentRequest{
					Data: &paymentpb.Payment{
						Id: validationErrorInvalidIdResolver.MustGetString("invalidId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "payment.validation.id_invalid",
			Assertions: func(t *testing.T, response *paymentpb.ReadPaymentResponse, err error, useCase interface{}, ctx context.Context) {
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

func TestReadPaymentUseCase_Execute_TransactionFailure(t *testing.T) {
	testCode := "ESPYNA-TEST-PAYMENT-PAYMENT-READ-TRANSACTION-FAILURE-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "TransactionFailure", false)

	// Create a use case with a failing transaction service
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	mockPaymentRepo := payment.NewMockPaymentRepository(businessType)

	repositories := ReadPaymentRepositories{
		Payment: mockPaymentRepo,
	}

	standardServices := testutil.CreateStandardServices(false, true)
	services := ReadPaymentServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   mockDb.NewFailingMockTransactionService(),
		TranslationService:   standardServices.TranslationService,
	}

	useCase := NewReadPaymentUseCase(repositories, services)

	// Load test case using centralized TestCaseResolver
	resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "payment", "ReadPayment_Success")
	testutil.AssertTestCaseLoad(t, err, "ReadPayment_Success")

	req := &paymentpb.ReadPaymentRequest{
		Data: &paymentpb.Payment{
			Id: resolver.MustGetString("validPaymentId"),
		},
	}

	_, err = useCase.Execute(ctx, req)

	// Verify that a transaction error occurred
	testutil.AssertTransactionError(t, err)

	// Log completion of transaction failure test
	testutil.LogTestResult(t, testCode, "TransactionFailure", false, err)
}
