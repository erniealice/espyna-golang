//go:build mock_db && mock_auth

// Package payment_profile provides table-driven tests for the payment profile creation use case.
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
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestCreatePaymentProfileUseCase_Execute_TableDriven
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-PAYMENT-PAYMENT-PROFILE-CREATE-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-PAYMENT-PAYMENT-PROFILE-CREATE-TRANSACTION-v1.0: WithTransaction
//   - ESPYNA-TEST-PAYMENT-PAYMENT-PROFILE-CREATE-AUTHORIZATION-v1.0: Unauthorized
//   - ESPYNA-TEST-PAYMENT-PAYMENT-PROFILE-CREATE-NIL-REQUEST-v1.0: NilRequest
//   - ESPYNA-TEST-PAYMENT-PAYMENT-PROFILE-CREATE-NIL-DATA-v1.0: NilData
//   - ESPYNA-TEST-PAYMENT-PAYMENT-PROFILE-CREATE-VALIDATION-EMPTY-CLIENT-ID-v1.0: EmptyClientId
//   - ESPYNA-TEST-PAYMENT-PAYMENT-PROFILE-CREATE-VALIDATION-EMPTY-PAYMENT-METHOD-ID-v1.0: EmptyPaymentMethodId
//   - ESPYNA-TEST-PAYMENT-PAYMENT-PROFILE-CREATE-VALIDATION-INVALID-CLIENT-ID-v1.0: InvalidClientId
//   - ESPYNA-TEST-PAYMENT-PAYMENT-PROFILE-CREATE-VALIDATION-INVALID-PAYMENT-METHOD-ID-v1.0: InvalidPaymentMethodId
//   - ESPYNA-TEST-PAYMENT-PAYMENT-PROFILE-CREATE-ENRICHMENT-v1.0: DataEnrichment
//   - ESPYNA-TEST-PAYMENT-PAYMENT-PROFILE-CREATE-TRANSACTION-FAILURE-v1.0: TransactionFailure
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
	"leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock/entity"
	"leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock/payment"
	paymentprofilepb "leapfor.xyz/esqyma/golang/v1/domain/payment/payment_profile"
)

// Type alias for create payment profile test cases
type CreatePaymentProfileTestCase = testutil.GenericTestCase[*paymentprofilepb.CreatePaymentProfileRequest, *paymentprofilepb.CreatePaymentProfileResponse]

func createCreateTestUseCaseWithAuth(businessType string, supportsTransaction bool, shouldAuthorize bool) *CreatePaymentProfileUseCase {
	paymentProfileRepo := payment.NewMockPaymentProfileRepository(businessType)
	clientRepo := entity.NewMockClientRepository(businessType)
	paymentMethodRepo := payment.NewMockPaymentMethodRepository(businessType)

	repositories := CreatePaymentProfileRepositories{
		PaymentProfile: paymentProfileRepo,
		Client:         clientRepo,
		PaymentMethod:  paymentMethodRepo,
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, shouldAuthorize)
	services := CreatePaymentProfileServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
		IDService:            standardServices.IDService,
	}

	return NewCreatePaymentProfileUseCase(repositories, services)
}

func TestCreatePaymentProfileUseCase_Execute_TableDriven(t *testing.T) {
	businessType := testutil.GetTestBusinessType()

	// Load test data resolvers before defining test cases
	createSuccessResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "payment_profile", "CreatePaymentProfile_Success")
	testutil.AssertTestCaseLoad(t, err, "CreatePaymentProfile_Success")

	authorizationUnauthorizedResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "payment_profile", "Authorization_Unauthorized")
	testutil.AssertTestCaseLoad(t, err, "Authorization_Unauthorized")

	validationErrorEmptyClientIdResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "payment_profile", "ValidationError_EmptyClientId")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_EmptyClientId")

	validationErrorEmptyPaymentMethodIdResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "payment_profile", "ValidationError_EmptyPaymentMethodId")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_EmptyPaymentMethodId")

	validationErrorInvalidClientIdResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "payment_profile", "ValidationError_InvalidClientId")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_InvalidClientId")

	validationErrorInvalidPaymentMethodIdResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "payment_profile", "ValidationError_InvalidPaymentMethodId")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_InvalidPaymentMethodId")

	testCases := []CreatePaymentProfileTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-PAYMENT-PAYMENT-PROFILE-CREATE-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *paymentprofilepb.CreatePaymentProfileRequest {
				return &paymentprofilepb.CreatePaymentProfileRequest{
					Data: &paymentprofilepb.PaymentProfile{
						ClientId:        createSuccessResolver.MustGetString("validClientId"),
						PaymentMethodId: createSuccessResolver.MustGetString("validPaymentMethodId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *paymentprofilepb.CreatePaymentProfileResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertEqual(t, 1, len(response.Data), "response data length")
				createdPaymentProfile := response.Data[0]
				testutil.AssertStringEqual(t, createSuccessResolver.MustGetString("validClientId"), createdPaymentProfile.ClientId, "client ID")
				testutil.AssertStringEqual(t, createSuccessResolver.MustGetString("validPaymentMethodId"), createdPaymentProfile.PaymentMethodId, "payment method ID")
				testutil.AssertNonEmptyString(t, createdPaymentProfile.Id, "payment profile ID")
				testutil.AssertTrue(t, createdPaymentProfile.Active, "payment profile active status")
				testutil.AssertFieldSet(t, createdPaymentProfile.DateCreated, "DateCreated")
				testutil.AssertFieldSet(t, createdPaymentProfile.DateCreatedString, "DateCreatedString")
			},
		},
		{
			Name:     "WithTransaction",
			TestCode: "ESPYNA-TEST-PAYMENT-PAYMENT-PROFILE-CREATE-TRANSACTION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *paymentprofilepb.CreatePaymentProfileRequest {
				return &paymentprofilepb.CreatePaymentProfileRequest{
					Data: &paymentprofilepb.PaymentProfile{
						ClientId:        createSuccessResolver.MustGetString("validClientId"),
						PaymentMethodId: createSuccessResolver.MustGetString("validPaymentMethodId"),
					},
				}
			},
			UseTransaction: true,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *paymentprofilepb.CreatePaymentProfileResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				createdPaymentProfile := response.Data[0]
				testutil.AssertStringEqual(t, createSuccessResolver.MustGetString("validClientId"), createdPaymentProfile.ClientId, "client ID")
			},
		},
		{
			Name:     "Unauthorized",
			TestCode: "ESPYNA-TEST-PAYMENT-PAYMENT-PROFILE-CREATE-AUTHORIZATION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *paymentprofilepb.CreatePaymentProfileRequest {
				return &paymentprofilepb.CreatePaymentProfileRequest{
					Data: &paymentprofilepb.PaymentProfile{
						ClientId:        authorizationUnauthorizedResolver.MustGetString("unauthorizedClientId"),
						PaymentMethodId: authorizationUnauthorizedResolver.MustGetString("unauthorizedPaymentMethodId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        false,
			ExpectSuccess:  false,
			ExpectedError:  "payment_profile.errors.authorization_failed",
			Assertions: func(t *testing.T, response *paymentprofilepb.CreatePaymentProfileResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertAuthorizationError(t, err)
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-PAYMENT-PAYMENT-PROFILE-CREATE-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *paymentprofilepb.CreatePaymentProfileRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "payment_profile.validation.request_required",
			Assertions: func(t *testing.T, response *paymentprofilepb.CreatePaymentProfileResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilRequest(t, err)
			},
		},
		{
			Name:     "NilData",
			TestCode: "ESPYNA-TEST-PAYMENT-PAYMENT-PROFILE-CREATE-NIL-DATA-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *paymentprofilepb.CreatePaymentProfileRequest {
				return &paymentprofilepb.CreatePaymentProfileRequest{Data: nil}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "payment_profile.validation.data_required",
			Assertions: func(t *testing.T, response *paymentprofilepb.CreatePaymentProfileResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilData(t, err)
			},
		},
		{
			Name:     "EmptyClientId",
			TestCode: "ESPYNA-TEST-PAYMENT-PAYMENT-PROFILE-CREATE-VALIDATION-EMPTY-CLIENT-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *paymentprofilepb.CreatePaymentProfileRequest {
				return &paymentprofilepb.CreatePaymentProfileRequest{
					Data: &paymentprofilepb.PaymentProfile{
						ClientId:        validationErrorEmptyClientIdResolver.MustGetString("emptyClientId"),
						PaymentMethodId: validationErrorEmptyClientIdResolver.MustGetString("validPaymentMethodId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "payment_profile.validation.client_id_required",
			Assertions: func(t *testing.T, response *paymentprofilepb.CreatePaymentProfileResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty client ID")
			},
		},
		{
			Name:     "EmptyPaymentMethodId",
			TestCode: "ESPYNA-TEST-PAYMENT-PAYMENT-PROFILE-CREATE-VALIDATION-EMPTY-PAYMENT-METHOD-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *paymentprofilepb.CreatePaymentProfileRequest {
				return &paymentprofilepb.CreatePaymentProfileRequest{
					Data: &paymentprofilepb.PaymentProfile{
						ClientId:        validationErrorEmptyPaymentMethodIdResolver.MustGetString("validClientId"),
						PaymentMethodId: validationErrorEmptyPaymentMethodIdResolver.MustGetString("emptyPaymentMethodId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "payment_profile.validation.payment_method_id_required",
			Assertions: func(t *testing.T, response *paymentprofilepb.CreatePaymentProfileResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty payment method ID")
			},
		},
		{
			Name:     "InvalidClientId",
			TestCode: "ESPYNA-TEST-PAYMENT-PAYMENT-PROFILE-CREATE-VALIDATION-INVALID-CLIENT-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *paymentprofilepb.CreatePaymentProfileRequest {
				return &paymentprofilepb.CreatePaymentProfileRequest{
					Data: &paymentprofilepb.PaymentProfile{
						ClientId:        validationErrorInvalidClientIdResolver.MustGetString("invalidClientId"),
						PaymentMethodId: validationErrorInvalidClientIdResolver.MustGetString("validPaymentMethodId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "payment_profile.validation.client_id_invalid",
			Assertions: func(t *testing.T, response *paymentprofilepb.CreatePaymentProfileResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "invalid client ID")
			},
		},
		{
			Name:     "InvalidPaymentMethodId",
			TestCode: "ESPYNA-TEST-PAYMENT-PAYMENT-PROFILE-CREATE-VALIDATION-INVALID-PAYMENT-METHOD-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *paymentprofilepb.CreatePaymentProfileRequest {
				return &paymentprofilepb.CreatePaymentProfileRequest{
					Data: &paymentprofilepb.PaymentProfile{
						ClientId:        validationErrorInvalidPaymentMethodIdResolver.MustGetString("validClientId"),
						PaymentMethodId: validationErrorInvalidPaymentMethodIdResolver.MustGetString("invalidPaymentMethodId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "payment_profile.validation.payment_method_id_invalid",
			Assertions: func(t *testing.T, response *paymentprofilepb.CreatePaymentProfileResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "invalid payment method ID")
			},
		},
		{
			Name:     "DataEnrichment",
			TestCode: "ESPYNA-TEST-PAYMENT-PAYMENT-PROFILE-CREATE-ENRICHMENT-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *paymentprofilepb.CreatePaymentProfileRequest {
				return &paymentprofilepb.CreatePaymentProfileRequest{
					Data: &paymentprofilepb.PaymentProfile{
						ClientId:        "student-001",
						PaymentMethodId: "payment-method-001",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *paymentprofilepb.CreatePaymentProfileResponse, err error, useCase interface{}, ctx context.Context) {
				createdPaymentProfile := response.Data[0]
				testutil.AssertNonEmptyString(t, createdPaymentProfile.Id, "generated ID")
				testutil.AssertFieldSet(t, createdPaymentProfile.DateCreated, "DateCreated")
				testutil.AssertFieldSet(t, createdPaymentProfile.DateCreatedString, "DateCreatedString")
				testutil.AssertTrue(t, createdPaymentProfile.Active, "Active")
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
			useCase := createCreateTestUseCaseWithAuth(businessType, tc.UseTransaction, tc.UseAuth)

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

func TestCreatePaymentProfileUseCase_Execute_TransactionFailure(t *testing.T) {
	testCode := "ESPYNA-TEST-PAYMENT-PAYMENT-PROFILE-CREATE-TRANSACTION-FAILURE-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "TransactionFailure", false)

	// Create a use case with a failing transaction service
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	paymentProfileRepo := payment.NewMockPaymentProfileRepository(businessType)
	clientRepo := entity.NewMockClientRepository(businessType)
	paymentMethodRepo := payment.NewMockPaymentMethodRepository(businessType)

	repositories := CreatePaymentProfileRepositories{
		PaymentProfile: paymentProfileRepo,
		Client:         clientRepo,
		PaymentMethod:  paymentMethodRepo,
	}

	standardServices := testutil.CreateStandardServices(false, true)
	services := CreatePaymentProfileServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   mockDb.NewFailingMockTransactionService(),
		TranslationService:   standardServices.TranslationService,
		IDService:            standardServices.IDService,
	}

	useCase := NewCreatePaymentProfileUseCase(repositories, services)

	// Load test case using centralized TestCaseResolver
	resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "payment_profile", "CreatePaymentProfile_Success")
	testutil.AssertTestCaseLoad(t, err, "CreatePaymentProfile_Success")

	req := &paymentprofilepb.CreatePaymentProfileRequest{
		Data: &paymentprofilepb.PaymentProfile{
			ClientId:        resolver.MustGetString("validClientId"),
			PaymentMethodId: resolver.MustGetString("validPaymentMethodId"),
		},
	}

	_, err = useCase.Execute(ctx, req)

	// Verify that a transaction error occurred
	testutil.AssertTransactionError(t, err)

	// Log completion of transaction failure test
	testutil.LogTestResult(t, testCode, "TransactionFailure", false, err)
}
