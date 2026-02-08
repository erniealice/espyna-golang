//go:build mock_db && mock_auth

// Package subscription provides table-driven tests for the subscription listing use case.
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
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestListSubscriptionsUseCase_Execute_TableDriven
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-SUBSCRIPTION-SUBSCRIPTION-LIST-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-SUBSCRIPTION-SUBSCRIPTION-LIST-TRANSACTION-v1.0: WithTransaction
//   - ESPYNA-TEST-SUBSCRIPTION-SUBSCRIPTION-LIST-NIL-REQUEST-v1.0: NilRequest
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
	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock/subscription"
	subscriptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription"
)

// Type alias for list subscriptions test cases
type ListSubscriptionsTestCase = testutil.GenericTestCase[*subscriptionpb.ListSubscriptionsRequest, *subscriptionpb.ListSubscriptionsResponse]

func createTestListSubscriptionsUseCase(businessType string, supportsTransaction bool, shouldAuthorize bool) *ListSubscriptionsUseCase {
	mockSubscriptionRepo := subscription.NewMockSubscriptionRepository(businessType)

	repositories := ListSubscriptionsRepositories{
		Subscription: mockSubscriptionRepo,
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, shouldAuthorize)
	services := ListSubscriptionsServices{
		TransactionService: standardServices.TransactionService,
		TranslationService: standardServices.TranslationService,
	}

	return NewListSubscriptionsUseCase(repositories, services)
}

func TestListSubscriptionsUseCase_Execute_TableDriven(t *testing.T) {
	businessType := testutil.GetTestBusinessType()

	// Load test data resolvers before defining test cases
	listSuccessResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "subscription", "ListSubscriptions_Success")
	testutil.AssertTestCaseLoad(t, err, "ListSubscriptions_Success")

	testCases := []ListSubscriptionsTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-SUBSCRIPTION-LIST-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *subscriptionpb.ListSubscriptionsRequest {
				return &subscriptionpb.ListSubscriptionsRequest{}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *subscriptionpb.ListSubscriptionsResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				expectedCount := listSuccessResolver.MustGetInt("expectedSubscriptionCount")
				testutil.AssertEqual(t, expectedCount, len(response.Data), "response data length")

				// Verify that all subscriptions have required fields
				for _, sub := range response.Data {
					testutil.AssertNonEmptyString(t, sub.Id, "subscription ID")
					testutil.AssertNonEmptyString(t, sub.Name, "subscription name")
					testutil.AssertNonEmptyString(t, sub.ClientId, "client ID")
					testutil.AssertNonEmptyString(t, sub.PricePlanId, "price plan ID")
					testutil.AssertFieldSet(t, sub.DateCreated, "DateCreated")
					testutil.AssertFieldSet(t, sub.DateCreatedString, "DateCreatedString")
				}
			},
		},
		{
			Name:     "WithTransaction",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-SUBSCRIPTION-LIST-TRANSACTION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *subscriptionpb.ListSubscriptionsRequest {
				return &subscriptionpb.ListSubscriptionsRequest{}
			},
			UseTransaction: true,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *subscriptionpb.ListSubscriptionsResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				expectedCount := listSuccessResolver.MustGetInt("expectedSubscriptionCount")
				testutil.AssertEqual(t, expectedCount, len(response.Data), "response data length")
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-SUBSCRIPTION-LIST-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *subscriptionpb.ListSubscriptionsRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "subscription.validation.request_required",
			Assertions: func(t *testing.T, response *subscriptionpb.ListSubscriptionsResponse, err error, useCase interface{}, ctx context.Context) {
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
			useCase := createTestListSubscriptionsUseCase(businessType, tc.UseTransaction, tc.UseAuth)

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
