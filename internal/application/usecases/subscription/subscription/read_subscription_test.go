//go:build mock_db && mock_auth

// Package subscription provides table-driven tests for the subscription reading use case.
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
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestReadSubscriptionUseCase_Execute_TableDriven
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-SUBSCRIPTION-SUBSCRIPTION-READ-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-SUBSCRIPTION-SUBSCRIPTION-READ-TRANSACTION-v1.0: WithTransaction
//   - ESPYNA-TEST-SUBSCRIPTION-SUBSCRIPTION-READ-NIL-REQUEST-v1.0: NilRequest
//   - ESPYNA-TEST-SUBSCRIPTION-SUBSCRIPTION-READ-NIL-DATA-v1.0: NilData
//   - ESPYNA-TEST-SUBSCRIPTION-SUBSCRIPTION-READ-EMPTY-ID-v1.0: EmptyId
//   - ESPYNA-TEST-SUBSCRIPTION-SUBSCRIPTION-READ-ID-TOO-SHORT-v1.0: IdTooShort
//   - ESPYNA-TEST-SUBSCRIPTION-SUBSCRIPTION-READ-NOT-FOUND-v1.0: NotFound
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
	"leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock/subscription"
	subscriptionpb "leapfor.xyz/esqyma/golang/v1/domain/subscription/subscription"
)

// Type alias for read subscription test cases
type ReadSubscriptionTestCase = testutil.GenericTestCase[*subscriptionpb.ReadSubscriptionRequest, *subscriptionpb.ReadSubscriptionResponse]

func createTestReadSubscriptionUseCase(businessType string, supportsTransaction bool, shouldAuthorize bool) *ReadSubscriptionUseCase {
	mockSubscriptionRepo := subscription.NewMockSubscriptionRepository(businessType)

	repositories := ReadSubscriptionRepositories{
		Subscription: mockSubscriptionRepo,
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, shouldAuthorize)
	services := ReadSubscriptionServices{
		TransactionService: standardServices.TransactionService,
		TranslationService: standardServices.TranslationService,
	}

	return NewReadSubscriptionUseCase(repositories, services)
}

func TestReadSubscriptionUseCase_Execute_TableDriven(t *testing.T) {
	businessType := testutil.GetTestBusinessType()

	// Load test data resolvers before defining test cases
	readSuccessResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "subscription", "ReadSubscription_Success")
	testutil.AssertTestCaseLoad(t, err, "ReadSubscription_Success")

	readNotFoundResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "subscription", "ReadSubscription_NotFound")
	testutil.AssertTestCaseLoad(t, err, "ReadSubscription_NotFound")

	testCases := []ReadSubscriptionTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-SUBSCRIPTION-READ-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *subscriptionpb.ReadSubscriptionRequest {
				return &subscriptionpb.ReadSubscriptionRequest{
					Data: &subscriptionpb.Subscription{
						Id: readSuccessResolver.MustGetString("existingSubscriptionId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *subscriptionpb.ReadSubscriptionResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertEqual(t, 1, len(response.Data), "response data length")
				readSubscription := response.Data[0]
				testutil.AssertStringEqual(t, readSuccessResolver.MustGetString("existingSubscriptionId"), readSubscription.Id, "subscription ID")
				testutil.AssertStringEqual(t, readSuccessResolver.MustGetString("expectedSubscriptionName"), readSubscription.Name, "subscription name")
				testutil.AssertStringEqual(t, readSuccessResolver.MustGetString("expectedClientId"), readSubscription.ClientId, "client ID")
				testutil.AssertStringEqual(t, readSuccessResolver.MustGetString("expectedPricePlanId"), readSubscription.PricePlanId, "price plan ID")
				testutil.AssertFieldSet(t, readSubscription.DateCreated, "DateCreated")
				testutil.AssertFieldSet(t, readSubscription.DateCreatedString, "DateCreatedString")
			},
		},
		{
			Name:     "WithTransaction",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-SUBSCRIPTION-READ-TRANSACTION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *subscriptionpb.ReadSubscriptionRequest {
				return &subscriptionpb.ReadSubscriptionRequest{
					Data: &subscriptionpb.Subscription{
						Id: readSuccessResolver.MustGetString("existingSubscriptionId"),
					},
				}
			},
			UseTransaction: true,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *subscriptionpb.ReadSubscriptionResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertEqual(t, 1, len(response.Data), "response data length")
				readSubscription := response.Data[0]
				testutil.AssertStringEqual(t, readSuccessResolver.MustGetString("existingSubscriptionId"), readSubscription.Id, "subscription ID")
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-SUBSCRIPTION-READ-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *subscriptionpb.ReadSubscriptionRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "subscription.validation.request_required",
			Assertions: func(t *testing.T, response *subscriptionpb.ReadSubscriptionResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilRequest(t, err)
			},
		},
		{
			Name:     "NilData",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-SUBSCRIPTION-READ-NIL-DATA-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *subscriptionpb.ReadSubscriptionRequest {
				return &subscriptionpb.ReadSubscriptionRequest{Data: nil}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "subscription.validation.data_required",
			Assertions: func(t *testing.T, response *subscriptionpb.ReadSubscriptionResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "nil data")
			},
		},
		{
			Name:     "EmptyId",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-SUBSCRIPTION-READ-EMPTY-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *subscriptionpb.ReadSubscriptionRequest {
				return &subscriptionpb.ReadSubscriptionRequest{
					Data: &subscriptionpb.Subscription{
						Id: "",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "subscription.validation.id_required",
			Assertions: func(t *testing.T, response *subscriptionpb.ReadSubscriptionResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty ID")
			},
		},
		{
			Name:     "IdTooShort",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-SUBSCRIPTION-READ-ID-TOO-SHORT-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *subscriptionpb.ReadSubscriptionRequest {
				return &subscriptionpb.ReadSubscriptionRequest{
					Data: &subscriptionpb.Subscription{
						Id: "ab",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "subscription.validation.id_too_short",
			Assertions: func(t *testing.T, response *subscriptionpb.ReadSubscriptionResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "ID too short")
			},
		},
		{
			Name:     "NotFound",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-SUBSCRIPTION-READ-NOT-FOUND-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *subscriptionpb.ReadSubscriptionRequest {
				return &subscriptionpb.ReadSubscriptionRequest{
					Data: &subscriptionpb.Subscription{
						Id: readNotFoundResolver.MustGetString("nonExistentSubscriptionId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "subscription.errors.not_found",
			Assertions: func(t *testing.T, response *subscriptionpb.ReadSubscriptionResponse, err error, useCase interface{}, ctx context.Context) {
				uc := useCase.(*ReadSubscriptionUseCase)
				testutil.AssertTranslatedError(t, err, "subscription.errors.not_found", uc.services.TranslationService, ctx)
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
			useCase := createTestReadSubscriptionUseCase(businessType, tc.UseTransaction, tc.UseAuth)

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
