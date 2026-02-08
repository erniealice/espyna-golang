//go:build mock_db && mock_auth

// Package payment_profile provides table-driven tests for the payment profile update use case.
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
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestUpdatePaymentProfileUseCase_Execute_TableDriven
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-PAYMENT-PAYMENT-PROFILE-UPDATE-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-PAYMENT-PAYMENT-PROFILE-UPDATE-TRANSACTION-v1.0: WithTransaction
//   - ESPYNA-TEST-PAYMENT-PAYMENT-PROFILE-UPDATE-AUTHORIZATION-v1.0: Unauthorized
//   - ESPYNA-TEST-PAYMENT-PAYMENT-PROFILE-UPDATE-NIL-REQUEST-v1.0: NilRequest
//   - ESPYNA-TEST-PAYMENT-PAYMENT-PROFILE-UPDATE-NIL-DATA-v1.0: NilData
//   - ESPYNA-TEST-PAYMENT-PAYMENT-PROFILE-UPDATE-VALIDATION-EMPTY-ID-v1.0: EmptyId
//   - ESPYNA-TEST-PAYMENT-PAYMENT-PROFILE-UPDATE-VALIDATION-INVALID-ID-v1.0: InvalidId
//   - ESPYNA-TEST-PAYMENT-PAYMENT-PROFILE-UPDATE-VALIDATION-NON-EXISTENT-v1.0: NonExistent
//   - ESPYNA-TEST-PAYMENT-PAYMENT-PROFILE-UPDATE-VALIDATION-EMPTY-CLIENT-ID-v1.0: EmptyClientId
//   - ESPYNA-TEST-PAYMENT-PAYMENT-PROFILE-UPDATE-VALIDATION-EMPTY-PAYMENT-METHOD-ID-v1.0: EmptyPaymentMethodId
//   - ESPYNA-TEST-PAYMENT-PAYMENT-PROFILE-UPDATE-VALIDATION-INVALID-CLIENT-ID-v1.0: InvalidClientId
//   - ESPYNA-TEST-PAYMENT-PAYMENT-PROFILE-UPDATE-VALIDATION-INVALID-PAYMENT-METHOD-ID-v1.0: InvalidPaymentMethodId
//   - ESPYNA-TEST-PAYMENT-PAYMENT-PROFILE-UPDATE-TRANSACTION-FAILURE-v1.0: TransactionFailure
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
	"github.com/erniealice/espyna-golang/internal/application/shared/testutil"
	mockDb "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock"
	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock/entity"
	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock/payment"
	paymentprofilepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/payment/payment_profile"
)

// Type alias for update payment profile test cases
type UpdatePaymentProfileTestCase = testutil.GenericTestCase[*paymentprofilepb.UpdatePaymentProfileRequest, *paymentprofilepb.UpdatePaymentProfileResponse]

func createUpdateTestUseCaseWithAuth(businessType string, supportsTransaction bool, shouldAuthorize bool) *UpdatePaymentProfileUseCase {
	paymentProfileRepo := payment.NewMockPaymentProfileRepository(businessType)
	clientRepo := entity.NewMockClientRepository(businessType)
	paymentMethodRepo := payment.NewMockPaymentMethodRepository(businessType)

	repositories := UpdatePaymentProfileRepositories{
		PaymentProfile: paymentProfileRepo,
		Client:         clientRepo,
		PaymentMethod:  paymentMethodRepo,
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, shouldAuthorize)
	services := UpdatePaymentProfileServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
	}

	return NewUpdatePaymentProfileUseCase(repositories, services)
}

func TestUpdatePaymentProfileUseCase_Execute_TableDriven(t *testing.T) {
	businessType := testutil.GetTestBusinessType()

	// Load test data resolvers before defining test cases
	updateSuccessResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "payment_profile", "UpdatePaymentProfile_Success")
	testutil.AssertTestCaseLoad(t, err, "UpdatePaymentProfile_Success")

	authorizationUnauthorizedResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "payment_profile", "Authorization_Unauthorized")
	testutil.AssertTestCaseLoad(t, err, "Authorization_Unauthorized")

	validationErrorEmptyIdResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "payment_profile", "ValidationError_EmptyId")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_EmptyId")

	validationErrorInvalidIdResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "payment_profile", "ValidationError_InvalidId")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_InvalidId")

	validationErrorNonExistentResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "payment_profile", "ValidationError_NonExistent")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_NonExistent")

	validationErrorEmptyClientIdResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "payment_profile", "ValidationError_EmptyClientId")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_EmptyClientId")

	validationErrorEmptyPaymentMethodIdResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "payment_profile", "ValidationError_EmptyPaymentMethodId")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_EmptyPaymentMethodId")

	validationErrorInvalidClientIdResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "payment_profile", "ValidationError_InvalidClientId")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_InvalidClientId")

	validationErrorInvalidPaymentMethodIdResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "payment_profile", "ValidationError_InvalidPaymentMethodId")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_InvalidPaymentMethodId")

	testCases := []UpdatePaymentProfileTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-PAYMENT-PAYMENT-PROFILE-UPDATE-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *paymentprofilepb.UpdatePaymentProfileRequest {
				return &paymentprofilepb.UpdatePaymentProfileRequest{
					Data: &paymentprofilepb.PaymentProfile{
						Id:              updateSuccessResolver.MustGetString("existingPaymentProfileId"),
						ClientId:        updateSuccessResolver.MustGetString("validClientId"),
						PaymentMethodId: updateSuccessResolver.MustGetString("updatedPaymentMethodId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *paymentprofilepb.UpdatePaymentProfileResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertEqual(t, 1, len(response.Data), "response data length")
				updatedPaymentProfile := response.Data[0]
				testutil.AssertStringEqual(t, updateSuccessResolver.MustGetString("existingPaymentProfileId"), updatedPaymentProfile.Id, "payment profile ID")
				testutil.AssertStringEqual(t, updateSuccessResolver.MustGetString("validClientId"), updatedPaymentProfile.ClientId, "client ID")
				testutil.AssertStringEqual(t, updateSuccessResolver.MustGetString("updatedPaymentMethodId"), updatedPaymentProfile.PaymentMethodId, "payment method ID")
			},
		},
		{
			Name:     "WithTransaction",
			TestCode: "ESPYNA-TEST-PAYMENT-PAYMENT-PROFILE-UPDATE-TRANSACTION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *paymentprofilepb.UpdatePaymentProfileRequest {
				return &paymentprofilepb.UpdatePaymentProfileRequest{
					Data: &paymentprofilepb.PaymentProfile{
						Id:              updateSuccessResolver.MustGetString("existingPaymentProfileId"),
						ClientId:        updateSuccessResolver.MustGetString("validClientId"),
						PaymentMethodId: updateSuccessResolver.MustGetString("updatedPaymentMethodId"),
					},
				}
			},
			UseTransaction: true,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *paymentprofilepb.UpdatePaymentProfileResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				updatedPaymentProfile := response.Data[0]
				testutil.AssertStringEqual(t, updateSuccessResolver.MustGetString("existingPaymentProfileId"), updatedPaymentProfile.Id, "payment profile ID")
			},
		},
		{
			Name:     "Unauthorized",
			TestCode: "ESPYNA-TEST-PAYMENT-PAYMENT-PROFILE-UPDATE-AUTHORIZATION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *paymentprofilepb.UpdatePaymentProfileRequest {
				return &paymentprofilepb.UpdatePaymentProfileRequest{
					Data: &paymentprofilepb.PaymentProfile{
						Id:              authorizationUnauthorizedResolver.MustGetString("unauthorizedPaymentProfileId"),
						ClientId:        authorizationUnauthorizedResolver.MustGetString("unauthorizedClientId"),
						PaymentMethodId: authorizationUnauthorizedResolver.MustGetString("unauthorizedPaymentMethodId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        false,
			ExpectSuccess:  false,
			ExpectedError:  "payment_profile.errors.authorization_failed",
			Assertions: func(t *testing.T, response *paymentprofilepb.UpdatePaymentProfileResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertAuthorizationError(t, err)
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-PAYMENT-PAYMENT-PROFILE-UPDATE-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *paymentprofilepb.UpdatePaymentProfileRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "payment_profile.validation.request_required",
			Assertions: func(t *testing.T, response *paymentprofilepb.UpdatePaymentProfileResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilRequest(t, err)
			},
		},
		{
			Name:     "NilData",
			TestCode: "ESPYNA-TEST-PAYMENT-PAYMENT-PROFILE-UPDATE-NIL-DATA-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *paymentprofilepb.UpdatePaymentProfileRequest {
				return &paymentprofilepb.UpdatePaymentProfileRequest{Data: nil}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "payment_profile.validation.data_required",
			Assertions: func(t *testing.T, response *paymentprofilepb.UpdatePaymentProfileResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilData(t, err)
			},
		},
		{
			Name:     "EmptyId",
			TestCode: "ESPYNA-TEST-PAYMENT-PAYMENT-PROFILE-UPDATE-VALIDATION-EMPTY-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *paymentprofilepb.UpdatePaymentProfileRequest {
				return &paymentprofilepb.UpdatePaymentProfileRequest{
					Data: &paymentprofilepb.PaymentProfile{
						Id:              validationErrorEmptyIdResolver.MustGetString("emptyId"),
						ClientId:        validationErrorEmptyIdResolver.MustGetString("validClientId"),
						PaymentMethodId: validationErrorEmptyIdResolver.MustGetString("validPaymentMethodId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "payment_profile.validation.id_required",
			Assertions: func(t *testing.T, response *paymentprofilepb.UpdatePaymentProfileResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty ID")
			},
		},
		{
			Name:     "InvalidId",
			TestCode: "ESPYNA-TEST-PAYMENT-PAYMENT-PROFILE-UPDATE-VALIDATION-INVALID-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *paymentprofilepb.UpdatePaymentProfileRequest {
				return &paymentprofilepb.UpdatePaymentProfileRequest{
					Data: &paymentprofilepb.PaymentProfile{
						Id:              validationErrorInvalidIdResolver.MustGetString("invalidId"),
						ClientId:        validationErrorInvalidIdResolver.MustGetString("validClientId"),
						PaymentMethodId: validationErrorInvalidIdResolver.MustGetString("validPaymentMethodId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "payment_profile.validation.id_invalid",
			Assertions: func(t *testing.T, response *paymentprofilepb.UpdatePaymentProfileResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "invalid ID")
			},
		},
		{
			Name:     "NonExistent",
			TestCode: "ESPYNA-TEST-PAYMENT-PAYMENT-PROFILE-UPDATE-VALIDATION-NON-EXISTENT-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *paymentprofilepb.UpdatePaymentProfileRequest {
				return &paymentprofilepb.UpdatePaymentProfileRequest{
					Data: &paymentprofilepb.PaymentProfile{
						Id:              validationErrorNonExistentResolver.MustGetString("nonExistentPaymentProfileId"),
						ClientId:        validationErrorNonExistentResolver.MustGetString("validClientId"),
						PaymentMethodId: validationErrorNonExistentResolver.MustGetString("validPaymentMethodId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "payment_profile.validation.not_found",
			Assertions: func(t *testing.T, response *paymentprofilepb.UpdatePaymentProfileResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "non-existent payment profile")
			},
		},
		{
			Name:     "EmptyClientId",
			TestCode: "ESPYNA-TEST-PAYMENT-PAYMENT-PROFILE-UPDATE-VALIDATION-EMPTY-CLIENT-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *paymentprofilepb.UpdatePaymentProfileRequest {
				return &paymentprofilepb.UpdatePaymentProfileRequest{
					Data: &paymentprofilepb.PaymentProfile{
						Id:              "payment-profile-001",
						ClientId:        validationErrorEmptyClientIdResolver.MustGetString("emptyClientId"),
						PaymentMethodId: validationErrorEmptyClientIdResolver.MustGetString("validPaymentMethodId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "payment_profile.validation.client_id_required",
			Assertions: func(t *testing.T, response *paymentprofilepb.UpdatePaymentProfileResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty client ID")
			},
		},
		{
			Name:     "EmptyPaymentMethodId",
			TestCode: "ESPYNA-TEST-PAYMENT-PAYMENT-PROFILE-UPDATE-VALIDATION-EMPTY-PAYMENT-METHOD-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *paymentprofilepb.UpdatePaymentProfileRequest {
				return &paymentprofilepb.UpdatePaymentProfileRequest{
					Data: &paymentprofilepb.PaymentProfile{
						Id:              "payment-profile-001",
						ClientId:        validationErrorEmptyPaymentMethodIdResolver.MustGetString("validClientId"),
						PaymentMethodId: validationErrorEmptyPaymentMethodIdResolver.MustGetString("emptyPaymentMethodId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "payment_profile.validation.payment_method_id_required",
			Assertions: func(t *testing.T, response *paymentprofilepb.UpdatePaymentProfileResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty payment method ID")
			},
		},
		{
			Name:     "InvalidClientId",
			TestCode: "ESPYNA-TEST-PAYMENT-PAYMENT-PROFILE-UPDATE-VALIDATION-INVALID-CLIENT-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *paymentprofilepb.UpdatePaymentProfileRequest {
				return &paymentprofilepb.UpdatePaymentProfileRequest{
					Data: &paymentprofilepb.PaymentProfile{
						Id:              "payment-profile-001",
						ClientId:        validationErrorInvalidClientIdResolver.MustGetString("invalidClientId"),
						PaymentMethodId: validationErrorInvalidClientIdResolver.MustGetString("validPaymentMethodId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "payment_profile.validation.client_id_invalid",
			Assertions: func(t *testing.T, response *paymentprofilepb.UpdatePaymentProfileResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "invalid client ID")
			},
		},
		{
			Name:     "InvalidPaymentMethodId",
			TestCode: "ESPYNA-TEST-PAYMENT-PAYMENT-PROFILE-UPDATE-VALIDATION-INVALID-PAYMENT-METHOD-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *paymentprofilepb.UpdatePaymentProfileRequest {
				return &paymentprofilepb.UpdatePaymentProfileRequest{
					Data: &paymentprofilepb.PaymentProfile{
						Id:              "payment-profile-001",
						ClientId:        validationErrorInvalidPaymentMethodIdResolver.MustGetString("validClientId"),
						PaymentMethodId: validationErrorInvalidPaymentMethodIdResolver.MustGetString("invalidPaymentMethodId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "payment_profile.validation.payment_method_id_invalid",
			Assertions: func(t *testing.T, response *paymentprofilepb.UpdatePaymentProfileResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "invalid payment method ID")
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

func TestUpdatePaymentProfileUseCase_Execute_TransactionFailure(t *testing.T) {
	testCode := "ESPYNA-TEST-PAYMENT-PAYMENT-PROFILE-UPDATE-TRANSACTION-FAILURE-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "TransactionFailure", false)

	// Create a use case with a failing transaction service
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	paymentProfileRepo := payment.NewMockPaymentProfileRepository(businessType)
	clientRepo := entity.NewMockClientRepository(businessType)
	paymentMethodRepo := payment.NewMockPaymentMethodRepository(businessType)

	repositories := UpdatePaymentProfileRepositories{
		PaymentProfile: paymentProfileRepo,
		Client:         clientRepo,
		PaymentMethod:  paymentMethodRepo,
	}

	standardServices := testutil.CreateStandardServices(false, true)
	services := UpdatePaymentProfileServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   mockDb.NewFailingMockTransactionService(),
		TranslationService:   standardServices.TranslationService,
	}

	useCase := NewUpdatePaymentProfileUseCase(repositories, services)

	// Load test case using centralized TestCaseResolver
	resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "payment_profile", "UpdatePaymentProfile_Success")
	testutil.AssertTestCaseLoad(t, err, "UpdatePaymentProfile_Success")

	req := &paymentprofilepb.UpdatePaymentProfileRequest{
		Data: &paymentprofilepb.PaymentProfile{
			Id:              resolver.MustGetString("existingPaymentProfileId"),
			ClientId:        resolver.MustGetString("validClientId"),
			PaymentMethodId: resolver.MustGetString("updatedPaymentMethodId"),
		},
	}

	_, err = useCase.Execute(ctx, req)

	// Verify that a transaction error occurred
	testutil.AssertTransactionError(t, err)

	// Log completion of transaction failure test
	testutil.LogTestResult(t, testCode, "TransactionFailure", false, err)
}
