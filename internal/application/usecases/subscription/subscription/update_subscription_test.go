//go:build mock_db && mock_auth

// Package subscription provides table-driven tests for the subscription updating use case.
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
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestUpdateSubscriptionUseCase_Execute_TableDriven
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-SUBSCRIPTION-SUBSCRIPTION-UPDATE-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-SUBSCRIPTION-SUBSCRIPTION-UPDATE-TRANSACTION-v1.0: WithTransaction
//   - ESPYNA-TEST-SUBSCRIPTION-SUBSCRIPTION-UPDATE-NIL-REQUEST-v1.0: NilRequest
//   - ESPYNA-TEST-SUBSCRIPTION-SUBSCRIPTION-UPDATE-NIL-DATA-v1.0: NilData
//   - ESPYNA-TEST-SUBSCRIPTION-SUBSCRIPTION-UPDATE-EMPTY-ID-v1.0: EmptyId
//   - ESPYNA-TEST-SUBSCRIPTION-SUBSCRIPTION-UPDATE-EMPTY-NAME-v1.0: EmptyName
//   - ESPYNA-TEST-SUBSCRIPTION-SUBSCRIPTION-UPDATE-EMPTY-CLIENT-ID-v1.0: EmptyClientId
//   - ESPYNA-TEST-SUBSCRIPTION-SUBSCRIPTION-UPDATE-EMPTY-PRICE-PLAN-ID-v1.0: EmptyPricePlanId
//   - ESPYNA-TEST-SUBSCRIPTION-SUBSCRIPTION-UPDATE-NOT-FOUND-v1.0: NotFound
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
	"github.com/erniealice/espyna-golang/internal/application/shared/testutil"
	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock/entity"
	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock/subscription"
	subscriptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription"
)

// Type alias for update subscription test cases
type UpdateSubscriptionTestCase = testutil.GenericTestCase[*subscriptionpb.UpdateSubscriptionRequest, *subscriptionpb.UpdateSubscriptionResponse]

func createTestUpdateSubscriptionUseCase(businessType string, supportsTransaction bool, shouldAuthorize bool) *UpdateSubscriptionUseCase {
	mockSubscriptionRepo := subscription.NewMockSubscriptionRepository(businessType)
	mockClientRepo := entity.NewMockClientRepository(businessType)
	mockPricePlanRepo := subscription.NewMockPricePlanRepository(businessType)

	repositories := UpdateSubscriptionRepositories{
		Subscription: mockSubscriptionRepo,
		Client:       mockClientRepo,
		PricePlan:    mockPricePlanRepo,
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, shouldAuthorize)
	services := UpdateSubscriptionServices{
		TransactionService: standardServices.TransactionService,
		TranslationService: standardServices.TranslationService,
	}

	return NewUpdateSubscriptionUseCase(repositories, services)
}

func TestUpdateSubscriptionUseCase_Execute_TableDriven(t *testing.T) {
	businessType := testutil.GetTestBusinessType()

	// Load test data resolvers before defining test cases
	updateSuccessResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "subscription", "UpdateSubscription_Success")
	testutil.AssertTestCaseLoad(t, err, "UpdateSubscription_Success")

	updateNotFoundResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "subscription", "UpdateSubscription_NotFound")
	testutil.AssertTestCaseLoad(t, err, "UpdateSubscription_NotFound")

	testCases := []UpdateSubscriptionTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-SUBSCRIPTION-UPDATE-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *subscriptionpb.UpdateSubscriptionRequest {
				return &subscriptionpb.UpdateSubscriptionRequest{
					Data: &subscriptionpb.Subscription{
						Id:          updateSuccessResolver.MustGetString("existingSubscriptionId"),
						Name:        updateSuccessResolver.MustGetString("updatedSubscriptionName"),
						ClientId:    updateSuccessResolver.MustGetString("expectedClientId"),
						PricePlanId: updateSuccessResolver.MustGetString("expectedPricePlanId"),
						Active:      true,
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *subscriptionpb.UpdateSubscriptionResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertEqual(t, 1, len(response.Data), "response data length")
				updatedSubscription := response.Data[0]
				testutil.AssertStringEqual(t, updateSuccessResolver.MustGetString("existingSubscriptionId"), updatedSubscription.Id, "subscription ID")
				testutil.AssertStringEqual(t, updateSuccessResolver.MustGetString("updatedSubscriptionName"), updatedSubscription.Name, "subscription name")
				testutil.AssertFieldSet(t, updatedSubscription.DateModified, "DateModified")
				testutil.AssertFieldSet(t, updatedSubscription.DateModifiedString, "DateModifiedString")
			},
		},
		{
			Name:     "WithTransaction",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-SUBSCRIPTION-UPDATE-TRANSACTION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *subscriptionpb.UpdateSubscriptionRequest {
				return &subscriptionpb.UpdateSubscriptionRequest{
					Data: &subscriptionpb.Subscription{
						Id:          updateSuccessResolver.MustGetString("existingSubscriptionId"),
						Name:        updateSuccessResolver.MustGetString("updatedSubscriptionName"),
						ClientId:    updateSuccessResolver.MustGetString("expectedClientId"),
						PricePlanId: updateSuccessResolver.MustGetString("expectedPricePlanId"),
						Active:      true,
					},
				}
			},
			UseTransaction: true,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *subscriptionpb.UpdateSubscriptionResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertEqual(t, 1, len(response.Data), "response data length")
				updatedSubscription := response.Data[0]
				testutil.AssertStringEqual(t, updateSuccessResolver.MustGetString("updatedSubscriptionName"), updatedSubscription.Name, "subscription name")
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-SUBSCRIPTION-UPDATE-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *subscriptionpb.UpdateSubscriptionRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "subscription.validation.request_required",
			Assertions: func(t *testing.T, response *subscriptionpb.UpdateSubscriptionResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilRequest(t, err)
			},
		},
		{
			Name:     "NilData",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-SUBSCRIPTION-UPDATE-NIL-DATA-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *subscriptionpb.UpdateSubscriptionRequest {
				return &subscriptionpb.UpdateSubscriptionRequest{Data: nil}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "subscription.validation.data_required",
			Assertions: func(t *testing.T, response *subscriptionpb.UpdateSubscriptionResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "nil data")
			},
		},
		{
			Name:     "EmptyId",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-SUBSCRIPTION-UPDATE-EMPTY-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *subscriptionpb.UpdateSubscriptionRequest {
				return &subscriptionpb.UpdateSubscriptionRequest{
					Data: &subscriptionpb.Subscription{
						Id:          "",
						Name:        updateSuccessResolver.MustGetString("updatedSubscriptionName"),
						ClientId:    updateSuccessResolver.MustGetString("expectedClientId"),
						PricePlanId: updateSuccessResolver.MustGetString("expectedPricePlanId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "subscription.validation.id_required",
			Assertions: func(t *testing.T, response *subscriptionpb.UpdateSubscriptionResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty ID")
			},
		},
		{
			Name:     "EmptyName",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-SUBSCRIPTION-UPDATE-EMPTY-NAME-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *subscriptionpb.UpdateSubscriptionRequest {
				return &subscriptionpb.UpdateSubscriptionRequest{
					Data: &subscriptionpb.Subscription{
						Id:          updateSuccessResolver.MustGetString("existingSubscriptionId"),
						Name:        "",
						ClientId:    updateSuccessResolver.MustGetString("expectedClientId"),
						PricePlanId: updateSuccessResolver.MustGetString("expectedPricePlanId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "subscription.validation.name_required",
			Assertions: func(t *testing.T, response *subscriptionpb.UpdateSubscriptionResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty name")
			},
		},
		{
			Name:     "EmptyClientId",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-SUBSCRIPTION-UPDATE-EMPTY-CLIENT-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *subscriptionpb.UpdateSubscriptionRequest {
				return &subscriptionpb.UpdateSubscriptionRequest{
					Data: &subscriptionpb.Subscription{
						Id:          updateSuccessResolver.MustGetString("existingSubscriptionId"),
						Name:        updateSuccessResolver.MustGetString("updatedSubscriptionName"),
						ClientId:    "",
						PricePlanId: updateSuccessResolver.MustGetString("expectedPricePlanId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "subscription.validation.client_id_required",
			Assertions: func(t *testing.T, response *subscriptionpb.UpdateSubscriptionResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty client ID")
			},
		},
		{
			Name:     "EmptyPricePlanId",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-SUBSCRIPTION-UPDATE-EMPTY-PRICE-PLAN-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *subscriptionpb.UpdateSubscriptionRequest {
				return &subscriptionpb.UpdateSubscriptionRequest{
					Data: &subscriptionpb.Subscription{
						Id:          updateSuccessResolver.MustGetString("existingSubscriptionId"),
						Name:        updateSuccessResolver.MustGetString("updatedSubscriptionName"),
						ClientId:    updateSuccessResolver.MustGetString("expectedClientId"),
						PricePlanId: "",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "subscription.validation.price_plan_id_required",
			Assertions: func(t *testing.T, response *subscriptionpb.UpdateSubscriptionResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty price plan ID")
			},
		},
		{
			Name:     "NotFound",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-SUBSCRIPTION-UPDATE-NOT-FOUND-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *subscriptionpb.UpdateSubscriptionRequest {
				return &subscriptionpb.UpdateSubscriptionRequest{
					Data: &subscriptionpb.Subscription{
						Id:          updateNotFoundResolver.MustGetString("nonExistentSubscriptionId"),
						Name:        updateNotFoundResolver.MustGetString("updatedSubscriptionName"),
						ClientId:    updateNotFoundResolver.MustGetString("expectedClientId"),
						PricePlanId: updateNotFoundResolver.MustGetString("expectedPricePlanId"),
						Active:      true,
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "subscription.errors.not_found",
			ErrorTags:      map[string]any{"subscriptionId": updateNotFoundResolver.MustGetString("nonExistentSubscriptionId")},
			Assertions: func(t *testing.T, response *subscriptionpb.UpdateSubscriptionResponse, err error, useCase interface{}, ctx context.Context) {
				uc := useCase.(*UpdateSubscriptionUseCase)
				testutil.AssertTranslatedErrorWithTags(t, err, "subscription.errors.not_found", map[string]any{"subscriptionId": updateNotFoundResolver.MustGetString("nonExistentSubscriptionId")}, uc.services.TranslationService, ctx)
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
			useCase := createTestUpdateSubscriptionUseCase(businessType, tc.UseTransaction, tc.UseAuth)

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
