//go:build mock_db && mock_auth

// Package payment provides table-driven tests for the payment listing use case.
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
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestListPaymentsUseCase_Execute_TableDriven
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-PAYMENT-PAYMENT-LIST-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-PAYMENT-PAYMENT-LIST-TRANSACTION-v1.0: WithTransaction
//   - ESPYNA-TEST-PAYMENT-PAYMENT-LIST-AUTHORIZATION-v1.0: Unauthorized
//   - ESPYNA-TEST-PAYMENT-PAYMENT-LIST-NIL-REQUEST-v1.0: NilRequest
//   - ESPYNA-TEST-PAYMENT-PAYMENT-LIST-TRANSACTION-FAILURE-v1.0: TransactionFailure
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

// Type alias for list payments test cases
type ListPaymentsTestCase = testutil.GenericTestCase[*paymentpb.ListPaymentsRequest, *paymentpb.ListPaymentsResponse]

// Test helper to create use case with real services
func createListTestUseCase(businessType string, supportsTransaction bool) *ListPaymentsUseCase {
	return createListTestUseCaseWithAuth(businessType, supportsTransaction, true)
}

func createListTestUseCaseWithAuth(businessType string, supportsTransaction bool, shouldAuthorize bool) *ListPaymentsUseCase {
	paymentRepo := payment.NewMockPaymentRepository(businessType)

	repositories := ListPaymentsRepositories{
		Payment: paymentRepo,
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, shouldAuthorize)
	services := ListPaymentsServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
	}

	return NewListPaymentsUseCase(repositories, services)
}

func TestListPaymentsUseCase_Execute_TableDriven(t *testing.T) {
	businessType := testutil.GetTestBusinessType()

	// Load test data resolvers before defining test cases
	listSuccessResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "payment", "ListPayments_Success")
	testutil.AssertTestCaseLoad(t, err, "ListPayments_Success")

	testCases := []ListPaymentsTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-PAYMENT-PAYMENT-LIST-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *paymentpb.ListPaymentsRequest {
				return &paymentpb.ListPaymentsRequest{}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *paymentpb.ListPaymentsResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				expectedCount := listSuccessResolver.MustGetInt("expectedCount")
				testutil.AssertEqual(t, expectedCount, len(response.Data), "expected count")
			},
		},
		{
			Name:     "WithTransaction",
			TestCode: "ESPYNA-TEST-PAYMENT-PAYMENT-LIST-TRANSACTION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *paymentpb.ListPaymentsRequest {
				return &paymentpb.ListPaymentsRequest{}
			},
			UseTransaction: true,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *paymentpb.ListPaymentsResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				expectedCount := listSuccessResolver.MustGetInt("expectedCount")
				testutil.AssertEqual(t, expectedCount, len(response.Data), "expected count")
			},
		},
		{
			Name:     "Unauthorized",
			TestCode: "ESPYNA-TEST-PAYMENT-PAYMENT-LIST-AUTHORIZATION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *paymentpb.ListPaymentsRequest {
				return &paymentpb.ListPaymentsRequest{}
			},
			UseTransaction: false,
			UseAuth:        false,
			ExpectSuccess:  false,
			ExpectedError:  "payment.errors.authorization_failed",
			Assertions: func(t *testing.T, response *paymentpb.ListPaymentsResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertAuthorizationError(t, err)
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-PAYMENT-PAYMENT-LIST-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *paymentpb.ListPaymentsRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "payment.validation.request_required",
			Assertions: func(t *testing.T, response *paymentpb.ListPaymentsResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilRequest(t, err)
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
			useCase := createListTestUseCaseWithAuth(businessType, tc.UseTransaction, tc.UseAuth)

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

func TestListPaymentsUseCase_Execute_TransactionFailure(t *testing.T) {
	testCode := "ESPYNA-TEST-PAYMENT-PAYMENT-LIST-TRANSACTION-FAILURE-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "TransactionFailure", false)

	// Create a use case with a failing transaction service
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	mockPaymentRepo := payment.NewMockPaymentRepository(businessType)

	repositories := ListPaymentsRepositories{
		Payment: mockPaymentRepo,
	}

	standardServices := testutil.CreateStandardServices(false, true)
	services := ListPaymentsServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   mockDb.NewFailingMockTransactionService(),
		TranslationService:   standardServices.TranslationService,
	}

	useCase := NewListPaymentsUseCase(repositories, services)

	req := &paymentpb.ListPaymentsRequest{}

	_, err := useCase.Execute(ctx, req)

	// Verify that a transaction error occurred
	testutil.AssertTransactionError(t, err)

	// Log completion of transaction failure test
	testutil.LogTestResult(t, testCode, "TransactionFailure", false, err)
}
