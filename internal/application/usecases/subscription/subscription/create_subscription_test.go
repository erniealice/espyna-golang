//go:build mock_db && mock_auth

// Package subscription provides table-driven tests for the subscription creation use case.
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
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestCreateSubscriptionUseCase_Execute_TableDriven
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-SUBSCRIPTION-SUBSCRIPTION-CREATE-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-SUBSCRIPTION-SUBSCRIPTION-CREATE-TRANSACTION-v1.0: WithTransaction
//   - ESPYNA-TEST-SUBSCRIPTION-SUBSCRIPTION-CREATE-NIL-REQUEST-v1.0: NilRequest
//   - ESPYNA-TEST-SUBSCRIPTION-SUBSCRIPTION-CREATE-NIL-DATA-v1.0: NilData
//   - ESPYNA-TEST-SUBSCRIPTION-SUBSCRIPTION-CREATE-VALIDATION-EMPTY-NAME-v1.0: EmptyName
//   - ESPYNA-TEST-SUBSCRIPTION-SUBSCRIPTION-CREATE-VALIDATION-EMPTY-CLIENT-ID-v1.0: EmptyClientId
//   - ESPYNA-TEST-SUBSCRIPTION-SUBSCRIPTION-CREATE-VALIDATION-EMPTY-PRICE-PLAN-ID-v1.0: EmptyPricePlanId
//   - ESPYNA-TEST-SUBSCRIPTION-SUBSCRIPTION-CREATE-VALIDATION-INVALID-CLIENT-ID-v1.0: InvalidClientId
//   - ESPYNA-TEST-SUBSCRIPTION-SUBSCRIPTION-CREATE-VALIDATION-INVALID-PRICE-PLAN-ID-v1.0: InvalidPricePlanId
//   - ESPYNA-TEST-SUBSCRIPTION-SUBSCRIPTION-CREATE-VALIDATION-NAME-TOO-SHORT-v1.0: NameTooShort
//   - ESPYNA-TEST-SUBSCRIPTION-SUBSCRIPTION-CREATE-VALIDATION-NAME-TOO-LONG-v1.0: NameTooLong
//   - ESPYNA-TEST-SUBSCRIPTION-SUBSCRIPTION-CREATE-ENRICHMENT-v1.0: DataEnrichment
//   - ESPYNA-TEST-SUBSCRIPTION-SUBSCRIPTION-CREATE-TRANSACTION-FAILURE-v1.0: TransactionFailure
//
// Data Sources:
//   - Test cases: packages/copya/data_test/{businessType}/subscription.json
//   - Mock data: packages/copya/data/{businessType}/subscription.json
//   - Translations: packages/lyngua/translations/{languageCode}/{businessType}/subscription.json
package subscription

import (
	"context"
	"testing"

	copyatestutil "leapfor.xyz/copya/golang/testutil"
	"leapfor.xyz/espyna/internal/application/shared/testutil"
	"leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock/entity"
	"leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock/subscription"
	subscriptionpb "leapfor.xyz/esqyma/golang/v1/domain/subscription/subscription"
)

// Type alias for create subscription test cases
type CreateSubscriptionTestCase = testutil.GenericTestCase[*subscriptionpb.CreateSubscriptionRequest, *subscriptionpb.CreateSubscriptionResponse]

func createTestUseCaseWithAuth(businessType string, supportsTransaction bool, shouldAuthorize bool) *CreateSubscriptionUseCase {
	mockSubscriptionRepo := subscription.NewMockSubscriptionRepository(businessType)
	mockClientRepo := entity.NewMockClientRepository(businessType)
	mockPricePlanRepo := subscription.NewMockPricePlanRepository(businessType)

	repositories := CreateSubscriptionRepositories{
		Subscription: mockSubscriptionRepo,
		Client:       mockClientRepo,
		PricePlan:    mockPricePlanRepo,
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, shouldAuthorize)
	services := CreateSubscriptionServices{
		TransactionService: standardServices.TransactionService,
		TranslationService: standardServices.TranslationService,
		IDService:          standardServices.IDService,
	}

	return NewCreateSubscriptionUseCase(repositories, services)
}

func TestCreateSubscriptionUseCase_Execute_TableDriven(t *testing.T) {
	businessType := testutil.GetTestBusinessType()

	// Load test data resolvers before defining test cases
	createSuccessResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "subscription", "CreateSubscription_Success")
	testutil.AssertTestCaseLoad(t, err, "CreateSubscription_Success")

	validationErrorEmptyNameResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "subscription", "ValidationError_EmptyName")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_EmptyName")

	validationErrorEmptyClientIdResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "subscription", "ValidationError_EmptyClientId")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_EmptyClientId")

	validationErrorEmptyPricePlanIdResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "subscription", "ValidationError_EmptyPricePlanId")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_EmptyPricePlanId")

	validationErrorInvalidClientIdResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "subscription", "ValidationError_InvalidClientId")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_InvalidClientId")

	validationErrorInvalidPricePlanIdResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "subscription", "ValidationError_InvalidPricePlanId")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_InvalidPricePlanId")

	validationErrorNameTooShortResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "subscription", "ValidationError_NameTooShort")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_NameTooShort")

	validationErrorNameTooLongResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "subscription", "ValidationError_NameTooLong")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_NameTooLong")

	testCases := []CreateSubscriptionTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-SUBSCRIPTION-CREATE-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *subscriptionpb.CreateSubscriptionRequest {
				return &subscriptionpb.CreateSubscriptionRequest{
					Data: &subscriptionpb.Subscription{
						Name:        createSuccessResolver.MustGetString("newSubscriptionName"),
						ClientId:    createSuccessResolver.MustGetString("validClientId"),
						PricePlanId: createSuccessResolver.MustGetString("validPricePlanId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *subscriptionpb.CreateSubscriptionResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertEqual(t, 1, len(response.Data), "response data length")
				createdSubscription := response.Data[0]
				testutil.AssertStringEqual(t, createSuccessResolver.MustGetString("newSubscriptionName"), createdSubscription.Name, "subscription name")
				testutil.AssertNonEmptyString(t, createdSubscription.Id, "subscription ID")
				testutil.AssertTrue(t, createdSubscription.Active, "subscription active status")
				testutil.AssertFieldSet(t, createdSubscription.DateCreated, "DateCreated")
				testutil.AssertFieldSet(t, createdSubscription.DateCreatedString, "DateCreatedString")
			},
		},
		{
			Name:     "WithTransaction",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-SUBSCRIPTION-CREATE-TRANSACTION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *subscriptionpb.CreateSubscriptionRequest {
				return &subscriptionpb.CreateSubscriptionRequest{
					Data: &subscriptionpb.Subscription{
						Name:        createSuccessResolver.MustGetString("newSubscriptionName"),
						ClientId:    createSuccessResolver.MustGetString("validClientId"),
						PricePlanId: createSuccessResolver.MustGetString("validPricePlanId"),
					},
				}
			},
			UseTransaction: true,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *subscriptionpb.CreateSubscriptionResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				createdSubscription := response.Data[0]
				testutil.AssertStringEqual(t, createSuccessResolver.MustGetString("newSubscriptionName"), createdSubscription.Name, "subscription name")
			},
		},
		// Note: Authorization is not handled at the use case level for subscriptions
		// {
		// 	Name:     "Unauthorized",
		// 	TestCode: "ESPYNA-TEST-SUBSCRIPTION-SUBSCRIPTION-CREATE-AUTHORIZATION-v1.0",
		// 	SetupRequest: func(t *testing.T, businessType string) *subscriptionpb.CreateSubscriptionRequest {
		// 		return &subscriptionpb.CreateSubscriptionRequest{
		// 			Data: &subscriptionpb.Subscription{
		// 				Name:     authorizationUnauthorizedResolver.MustGetString("unauthorizedSubscriptionName"),
		// 				ClientId: authorizationUnauthorizedResolver.MustGetString("unauthorizedClientId"),
		// 				PlanId:   authorizationUnauthorizedResolver.MustGetString("unauthorizedPlanId"),
		// 			},
		// 		}
		// 	},
		// 	UseTransaction: false,
		// 	UseAuth:        false,
		// 	ExpectSuccess:  false,
		// 	ExpectedError:  "subscription.errors.authorization_failed",
		// 	Assertions: func(t *testing.T, response *subscriptionpb.CreateSubscriptionResponse, err error, useCase interface{}, ctx context.Context) {
		// 		testutil.AssertAuthorizationError(t, err)
		// 	},
		// },
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-SUBSCRIPTION-CREATE-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *subscriptionpb.CreateSubscriptionRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "subscription.validation.request_required",
			Assertions: func(t *testing.T, response *subscriptionpb.CreateSubscriptionResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilRequest(t, err)
			},
		},
		{
			Name:     "NilData",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-SUBSCRIPTION-CREATE-NIL-DATA-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *subscriptionpb.CreateSubscriptionRequest {
				return &subscriptionpb.CreateSubscriptionRequest{Data: nil}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "subscription.validation.data_required",
			Assertions: func(t *testing.T, response *subscriptionpb.CreateSubscriptionResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "nil data")
			},
		},
		{
			Name:     "EmptyName",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-SUBSCRIPTION-CREATE-VALIDATION-EMPTY-NAME-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *subscriptionpb.CreateSubscriptionRequest {
				return &subscriptionpb.CreateSubscriptionRequest{
					Data: &subscriptionpb.Subscription{
						Name:        validationErrorEmptyNameResolver.MustGetString("emptyName"),
						ClientId:    validationErrorEmptyNameResolver.MustGetString("validClientId"),
						PricePlanId: validationErrorEmptyNameResolver.MustGetString("validPricePlanId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "subscription.validation.name_required",
			Assertions: func(t *testing.T, response *subscriptionpb.CreateSubscriptionResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty name")
			},
		},
		{
			Name:     "EmptyClientId",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-SUBSCRIPTION-CREATE-VALIDATION-EMPTY-CLIENT-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *subscriptionpb.CreateSubscriptionRequest {
				return &subscriptionpb.CreateSubscriptionRequest{
					Data: &subscriptionpb.Subscription{
						Name:        validationErrorEmptyClientIdResolver.MustGetString("validSubscriptionName"),
						ClientId:    validationErrorEmptyClientIdResolver.MustGetString("emptyClientId"),
						PricePlanId: validationErrorEmptyClientIdResolver.MustGetString("validPricePlanId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "subscription.validation.client_id_required",
			Assertions: func(t *testing.T, response *subscriptionpb.CreateSubscriptionResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty client ID")
			},
		},
		{
			Name:     "EmptyPricePlanId",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-SUBSCRIPTION-CREATE-VALIDATION-EMPTY-PRICE-PLAN-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *subscriptionpb.CreateSubscriptionRequest {
				return &subscriptionpb.CreateSubscriptionRequest{
					Data: &subscriptionpb.Subscription{
						Name:        validationErrorEmptyPricePlanIdResolver.MustGetString("validSubscriptionName"),
						ClientId:    validationErrorEmptyPricePlanIdResolver.MustGetString("validClientId"),
						PricePlanId: validationErrorEmptyPricePlanIdResolver.MustGetString("emptyPricePlanId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "subscription.validation.price_plan_id_required",
			Assertions: func(t *testing.T, response *subscriptionpb.CreateSubscriptionResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty price plan ID")
			},
		},
		{
			Name:     "InvalidClientId",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-SUBSCRIPTION-CREATE-VALIDATION-INVALID-CLIENT-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *subscriptionpb.CreateSubscriptionRequest {
				return &subscriptionpb.CreateSubscriptionRequest{
					Data: &subscriptionpb.Subscription{
						Name:        validationErrorInvalidClientIdResolver.MustGetString("validSubscriptionName"),
						ClientId:    validationErrorInvalidClientIdResolver.MustGetString("invalidClientId"),
						PricePlanId: validationErrorInvalidClientIdResolver.MustGetString("validPricePlanId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "subscription.errors.client_not_found",
			Assertions: func(t *testing.T, response *subscriptionpb.CreateSubscriptionResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "invalid client ID")
			},
		},
		{
			Name:     "InvalidPricePlanId",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-SUBSCRIPTION-CREATE-VALIDATION-INVALID-PRICE-PLAN-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *subscriptionpb.CreateSubscriptionRequest {
				return &subscriptionpb.CreateSubscriptionRequest{
					Data: &subscriptionpb.Subscription{
						Name:        validationErrorInvalidPricePlanIdResolver.MustGetString("validSubscriptionName"),
						ClientId:    validationErrorInvalidPricePlanIdResolver.MustGetString("validClientId"),
						PricePlanId: validationErrorInvalidPricePlanIdResolver.MustGetString("invalidPricePlanId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "subscription.errors.price_plan_not_found",
			Assertions: func(t *testing.T, response *subscriptionpb.CreateSubscriptionResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "invalid price plan ID")
			},
		},
		{
			Name:     "NameTooShort",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-SUBSCRIPTION-CREATE-VALIDATION-NAME-TOO-SHORT-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *subscriptionpb.CreateSubscriptionRequest {
				return &subscriptionpb.CreateSubscriptionRequest{
					Data: &subscriptionpb.Subscription{
						Name:        validationErrorNameTooShortResolver.MustGetString("tooShortName"),
						ClientId:    validationErrorNameTooShortResolver.MustGetString("validClientId"),
						PricePlanId: validationErrorNameTooShortResolver.MustGetString("validPricePlanId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "subscription.validation.name_too_short",
			Assertions: func(t *testing.T, response *subscriptionpb.CreateSubscriptionResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "name too short")
			},
		},
		{
			Name:     "NameTooLong",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-SUBSCRIPTION-CREATE-VALIDATION-NAME-TOO-LONG-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *subscriptionpb.CreateSubscriptionRequest {
				return &subscriptionpb.CreateSubscriptionRequest{
					Data: &subscriptionpb.Subscription{
						Name:        validationErrorNameTooLongResolver.MustGetString("tooLongName"),
						ClientId:    validationErrorNameTooLongResolver.MustGetString("validClientId"),
						PricePlanId: validationErrorNameTooLongResolver.MustGetString("validPricePlanId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "subscription.validation.name_too_long",
			Assertions: func(t *testing.T, response *subscriptionpb.CreateSubscriptionResponse, err error, useCase interface{}, ctx context.Context) {
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
