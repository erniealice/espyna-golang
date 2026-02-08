//go:build mock_db && mock_auth

// Package payment provides table-driven tests for the payment creation use case.
//
// The tests cover various scenarios, including success, transaction handling,
// authorization, nil requests, validation errors, data enrichment, and boundary conditions.
// Each test case is defined in a table with a specific test code, request setup,
// and assertions.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user").
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education").
//
// Usage:
//   - Run all tests: go test -tags="mock_db,mock_auth" ./...
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestCreatePaymentUseCase_Execute_TableDriven
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-PAYMENT-PAYMENT-CREATE-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-PAYMENT-PAYMENT-CREATE-TRANSACTION-v1.0: WithTransaction
//   - ESPYNA-TEST-PAYMENT-PAYMENT-CREATE-AUTHORIZATION-v1.0: Unauthorized
//   - ESPYNA-TEST-PAYMENT-PAYMENT-CREATE-NIL-REQUEST-v1.0: NilRequest
//   - ESPYNA-TEST-PAYMENT-PAYMENT-CREATE-NIL-DATA-v1.0: NilData
//   - ESPYNA-TEST-PAYMENT-PAYMENT-CREATE-VALIDATION-EMPTY-NAME-v1.0: EmptyName
//   - ESPYNA-TEST-PAYMENT-PAYMENT-CREATE-VALIDATION-EMPTY-SUBSCRIPTION-ID-v1.0: EmptySubscriptionId
//   - ESPYNA-TEST-PAYMENT-PAYMENT-CREATE-VALIDATION-INVALID-SUBSCRIPTION-ID-v1.0: InvalidSubscriptionId
//   - ESPYNA-TEST-PAYMENT-PAYMENT-CREATE-VALIDATION-NAME-TOO-SHORT-v1.0: NameTooShort
//   - ESPYNA-TEST-PAYMENT-PAYMENT-CREATE-VALIDATION-NAME-TOO-LONG-v1.0: NameTooLong
//   - ESPYNA-TEST-PAYMENT-PAYMENT-CREATE-ENRICHMENT-v1.0: DataEnrichment
//   - ESPYNA-TEST-PAYMENT-PAYMENT-CREATE-VALIDATION-MINIMAL-v1.0: MinimalValidData
//   - ESPYNA-TEST-PAYMENT-PAYMENT-CREATE-VALIDATION-MAXIMAL-v1.0: MaxValidData
//   - ESPYNA-TEST-PAYMENT-PAYMENT-CREATE-TRANSACTION-FAILURE-v1.0: TransactionFailure
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

// Type alias for create payment test cases
type CreatePaymentTestCase = testutil.GenericTestCase[*paymentpb.CreatePaymentRequest, *paymentpb.CreatePaymentResponse]

// Test helper to create use case with real services and foreign key validation
func createTestUseCase(businessType string, supportsTransaction bool) *CreatePaymentUseCase {
	return createTestUseCaseWithAuth(businessType, supportsTransaction, true)
}

func createTestUseCaseWithAuth(businessType string, supportsTransaction bool, shouldAuthorize bool) *CreatePaymentUseCase {
	mockPaymentRepo := payment.NewMockPaymentRepository(businessType)
	mockSubscriptionRepo := subscriptionMock.NewMockSubscriptionRepository(businessType) // For foreign key validation

	repositories := CreatePaymentRepositories{
		Payment:      mockPaymentRepo,
		Subscription: mockSubscriptionRepo, // Foreign key validation
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, shouldAuthorize)
	services := CreatePaymentServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
		IDService:            standardServices.IDService,
	}

	return NewCreatePaymentUseCase(repositories, services)
}

func TestCreatePaymentUseCase_Execute_TableDriven(t *testing.T) {
	businessType := testutil.GetTestBusinessType()

	// Load test data resolvers before defining test cases
	createSuccessResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "payment", "CreatePayment_Success")
	testutil.AssertTestCaseLoad(t, err, "CreatePayment_Success")

	authorizationUnauthorizedResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "payment", "Authorization_Unauthorized")
	testutil.AssertTestCaseLoad(t, err, "Authorization_Unauthorized")

	validationErrorEmptyNameResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "payment", "ValidationError_EmptyName")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_EmptyName")

	validationErrorEmptySubscriptionIdResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "payment", "ValidationError_EmptySubscriptionId")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_EmptySubscriptionId")

	validationErrorInvalidSubscriptionIdResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "payment", "ValidationError_InvalidSubscriptionId")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_InvalidSubscriptionId")

	testCases := []CreatePaymentTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-PAYMENT-PAYMENT-CREATE-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *paymentpb.CreatePaymentRequest {
				return &paymentpb.CreatePaymentRequest{
					Data: &paymentpb.Payment{
						Name:           createSuccessResolver.MustGetString("newPaymentName"),
						SubscriptionId: createSuccessResolver.MustGetString("validSubscriptionId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *paymentpb.CreatePaymentResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertEqual(t, 1, len(response.Data), "response data length")
				createdPayment := response.Data[0]
				testutil.AssertStringEqual(t, createSuccessResolver.MustGetString("newPaymentName"), createdPayment.Name, "payment name")
				testutil.AssertNonEmptyString(t, createdPayment.Id, "payment ID")
				testutil.AssertTrue(t, createdPayment.Active, "payment active status")
				testutil.AssertFieldSet(t, createdPayment.DateCreated, "DateCreated")
				testutil.AssertFieldSet(t, createdPayment.DateCreatedString, "DateCreatedString")
			},
		},
		{
			Name:     "WithTransaction",
			TestCode: "ESPYNA-TEST-PAYMENT-PAYMENT-CREATE-TRANSACTION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *paymentpb.CreatePaymentRequest {
				return &paymentpb.CreatePaymentRequest{
					Data: &paymentpb.Payment{
						Name:           createSuccessResolver.MustGetString("newPaymentName"),
						SubscriptionId: createSuccessResolver.MustGetString("validSubscriptionId"),
					},
				}
			},
			UseTransaction: true,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *paymentpb.CreatePaymentResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				createdPayment := response.Data[0]
				testutil.AssertStringEqual(t, createSuccessResolver.MustGetString("newPaymentName"), createdPayment.Name, "payment name")
			},
		},
		{
			Name:     "Unauthorized",
			TestCode: "ESPYNA-TEST-PAYMENT-PAYMENT-CREATE-AUTHORIZATION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *paymentpb.CreatePaymentRequest {
				return &paymentpb.CreatePaymentRequest{
					Data: &paymentpb.Payment{
						Name:           authorizationUnauthorizedResolver.MustGetString("unauthorizedPaymentName"),
						SubscriptionId: authorizationUnauthorizedResolver.MustGetString("unauthorizedSubscriptionId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        false,
			ExpectSuccess:  false,
			ExpectedError:  "payment.errors.authorization_failed",
			Assertions: func(t *testing.T, response *paymentpb.CreatePaymentResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertAuthorizationError(t, err)
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-PAYMENT-PAYMENT-CREATE-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *paymentpb.CreatePaymentRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "payment.validation.request_required",
			Assertions: func(t *testing.T, response *paymentpb.CreatePaymentResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilRequest(t, err)
			},
		},
		{
			Name:     "NilData",
			TestCode: "ESPYNA-TEST-PAYMENT-PAYMENT-CREATE-NIL-DATA-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *paymentpb.CreatePaymentRequest {
				return &paymentpb.CreatePaymentRequest{Data: nil}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "payment.validation.data_required",
			Assertions: func(t *testing.T, response *paymentpb.CreatePaymentResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilData(t, err)
			},
		},
		{
			Name:     "EmptyName",
			TestCode: "ESPYNA-TEST-PAYMENT-PAYMENT-CREATE-VALIDATION-EMPTY-NAME-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *paymentpb.CreatePaymentRequest {
				return &paymentpb.CreatePaymentRequest{
					Data: &paymentpb.Payment{
						Name:           validationErrorEmptyNameResolver.MustGetString("emptyName"),
						SubscriptionId: validationErrorEmptyNameResolver.MustGetString("validSubscriptionId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "payment.validation.name_required",
			Assertions: func(t *testing.T, response *paymentpb.CreatePaymentResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty name")
			},
		},
		{
			Name:     "EmptySubscriptionId",
			TestCode: "ESPYNA-TEST-PAYMENT-PAYMENT-CREATE-VALIDATION-EMPTY-SUBSCRIPTION-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *paymentpb.CreatePaymentRequest {
				return &paymentpb.CreatePaymentRequest{
					Data: &paymentpb.Payment{
						Name:           validationErrorEmptySubscriptionIdResolver.MustGetString("validName"),
						SubscriptionId: validationErrorEmptySubscriptionIdResolver.MustGetString("emptySubscriptionId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "payment.validation.subscription_id_required",
			Assertions: func(t *testing.T, response *paymentpb.CreatePaymentResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty subscription ID")
			},
		},
		{
			Name:     "InvalidSubscriptionId",
			TestCode: "ESPYNA-TEST-PAYMENT-PAYMENT-CREATE-VALIDATION-INVALID-SUBSCRIPTION-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *paymentpb.CreatePaymentRequest {
				return &paymentpb.CreatePaymentRequest{
					Data: &paymentpb.Payment{
						Name:           validationErrorInvalidSubscriptionIdResolver.MustGetString("validName"),
						SubscriptionId: validationErrorInvalidSubscriptionIdResolver.MustGetString("invalidSubscriptionId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "payment.validation.subscription_id_invalid",
			Assertions: func(t *testing.T, response *paymentpb.CreatePaymentResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "invalid subscription ID")
			},
		},
		{
			Name:     "NameTooShort",
			TestCode: "ESPYNA-TEST-PAYMENT-PAYMENT-CREATE-VALIDATION-NAME-TOO-SHORT-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *paymentpb.CreatePaymentRequest {
				return &paymentpb.CreatePaymentRequest{
					Data: &paymentpb.Payment{
						Name:           "AB",
						SubscriptionId: createSuccessResolver.MustGetString("validSubscriptionId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "payment.validation.name_too_short",
			Assertions: func(t *testing.T, response *paymentpb.CreatePaymentResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "name too short")
			},
		},
		{
			Name:     "NameTooLong",
			TestCode: "ESPYNA-TEST-PAYMENT-PAYMENT-CREATE-VALIDATION-NAME-TOO-LONG-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *paymentpb.CreatePaymentRequest {
				resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "payment", "ValidationError_NameTooLongGenerated")
				testutil.AssertTestCaseLoad(t, err, "ValidationError_NameTooLongGenerated")
				return &paymentpb.CreatePaymentRequest{
					Data: &paymentpb.Payment{
						Name:           resolver.MustGetString("tooLongNameGenerated"),
						SubscriptionId: resolver.MustGetString("validSubscriptionId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "payment.validation.name_too_long",
			Assertions: func(t *testing.T, response *paymentpb.CreatePaymentResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "name too long")
			},
		},
		{
			Name:     "DataEnrichment",
			TestCode: "ESPYNA-TEST-PAYMENT-PAYMENT-CREATE-ENRICHMENT-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *paymentpb.CreatePaymentRequest {
				resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "payment", "CreatePayment_Success")
				testutil.AssertTestCaseLoad(t, err, "CreatePayment_Success")
				return &paymentpb.CreatePaymentRequest{
					Data: &paymentpb.Payment{
						Name:           "Data Enrichment Test",
						SubscriptionId: resolver.MustGetString("validSubscriptionId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *paymentpb.CreatePaymentResponse, err error, useCase interface{}, ctx context.Context) {
				createdPayment := response.Data[0]
				testutil.AssertNonEmptyString(t, createdPayment.Id, "generated ID")
				testutil.AssertFieldSet(t, createdPayment.DateCreated, "DateCreated")
				testutil.AssertFieldSet(t, createdPayment.DateCreatedString, "DateCreatedString")
				testutil.AssertTrue(t, createdPayment.Active, "Active")
			},
		},
		{
			Name:     "MinimalValidData",
			TestCode: "ESPYNA-TEST-PAYMENT-PAYMENT-CREATE-VALIDATION-MINIMAL-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *paymentpb.CreatePaymentRequest {
				resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "payment", "BoundaryTest_MinimalValid")
				testutil.AssertTestCaseLoad(t, err, "BoundaryTest_MinimalValid")
				return &paymentpb.CreatePaymentRequest{
					Data: &paymentpb.Payment{
						Name:           resolver.MustGetString("minValidName"),
						SubscriptionId: resolver.MustGetString("validSubscriptionId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *paymentpb.CreatePaymentResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				createdPayment := response.Data[0]
				resolver, _ := copyatestutil.LoadTestCaseFromBusinessType(t, testutil.GetTestBusinessType(), "payment", "BoundaryTest_MinimalValid")
				testutil.AssertStringEqual(t, resolver.MustGetString("minValidName"), createdPayment.Name, "payment name")
			},
		},
		{
			Name:     "MaxValidData",
			TestCode: "ESPYNA-TEST-PAYMENT-PAYMENT-CREATE-VALIDATION-MAXIMAL-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *paymentpb.CreatePaymentRequest {
				resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "payment", "BoundaryTest_MaximalValid")
				testutil.AssertTestCaseLoad(t, err, "BoundaryTest_MaximalValid")
				return &paymentpb.CreatePaymentRequest{
					Data: &paymentpb.Payment{
						Name:           resolver.MustGetString("maxValidNameExact100"),
						SubscriptionId: resolver.MustGetString("validSubscriptionId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *paymentpb.CreatePaymentResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				createdPayment := response.Data[0]
				testutil.AssertEqual(t, 100, len(createdPayment.Name), "name length")
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

func TestCreatePaymentUseCase_Execute_TransactionFailure(t *testing.T) {
	testCode := "ESPYNA-TEST-PAYMENT-PAYMENT-CREATE-TRANSACTION-FAILURE-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "TransactionFailure", false)

	// Create a use case with a failing transaction service
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	mockPaymentRepo := payment.NewMockPaymentRepository(businessType)
	mockSubscriptionRepo := subscriptionMock.NewMockSubscriptionRepository(businessType)

	repositories := CreatePaymentRepositories{
		Payment:      mockPaymentRepo,
		Subscription: mockSubscriptionRepo,
	}

	standardServices := testutil.CreateStandardServices(false, true)
	services := CreatePaymentServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   mockDb.NewFailingMockTransactionService(),
		TranslationService:   standardServices.TranslationService,
		IDService:            standardServices.IDService,
	}

	useCase := NewCreatePaymentUseCase(repositories, services)

	// Load test case using centralized TestCaseResolver
	resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "payment", "CreatePayment_Success")
	testutil.AssertTestCaseLoad(t, err, "CreatePayment_Success")

	req := &paymentpb.CreatePaymentRequest{
		Data: &paymentpb.Payment{
			Name:           resolver.MustGetString("newPaymentName"),
			SubscriptionId: resolver.MustGetString("validSubscriptionId"),
		},
	}

	_, err = useCase.Execute(ctx, req)

	// Verify that a transaction error occurred
	testutil.AssertTransactionError(t, err)

	// Log completion of transaction failure test
	testutil.LogTestResult(t, testCode, "TransactionFailure", false, err)
}
