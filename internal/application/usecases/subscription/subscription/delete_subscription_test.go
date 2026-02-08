//go:build mock_db && mock_auth

// Package subscription provides table-driven tests for the subscription deletion use case.
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
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestDeleteSubscriptionUseCase_Execute_TableDriven
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-SUBSCRIPTION-SUBSCRIPTION-DELETE-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-SUBSCRIPTION-SUBSCRIPTION-DELETE-TRANSACTION-v1.0: WithTransaction
//   - ESPYNA-TEST-SUBSCRIPTION-SUBSCRIPTION-DELETE-NIL-REQUEST-v1.0: NilRequest
//   - ESPYNA-TEST-SUBSCRIPTION-SUBSCRIPTION-DELETE-NIL-DATA-v1.0: NilData
//   - ESPYNA-TEST-SUBSCRIPTION-SUBSCRIPTION-DELETE-EMPTY-ID-v1.0: EmptyId
//   - ESPYNA-TEST-SUBSCRIPTION-SUBSCRIPTION-DELETE-ID-TOO-SHORT-v1.0: IdTooShort
//   - ESPYNA-TEST-SUBSCRIPTION-SUBSCRIPTION-DELETE-NOT-FOUND-v1.0: NotFound
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

// Type alias for delete subscription test cases
type DeleteSubscriptionTestCase = testutil.GenericTestCase[*subscriptionpb.DeleteSubscriptionRequest, *subscriptionpb.DeleteSubscriptionResponse]

func createTestDeleteSubscriptionUseCase(businessType string, supportsTransaction bool, shouldAuthorize bool) *DeleteSubscriptionUseCase {
	mockSubscriptionRepo := subscription.NewMockSubscriptionRepository(businessType)

	repositories := DeleteSubscriptionRepositories{
		Subscription: mockSubscriptionRepo,
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, shouldAuthorize)
	services := DeleteSubscriptionServices{
		TransactionService: standardServices.TransactionService,
		TranslationService: standardServices.TranslationService,
	}

	return NewDeleteSubscriptionUseCase(repositories, services)
}

func TestDeleteSubscriptionUseCase_Execute_TableDriven(t *testing.T) {
	businessType := testutil.GetTestBusinessType()

	// Load test data resolvers before defining test cases
	deleteSuccessResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "subscription", "DeleteSubscription_Success")
	testutil.AssertTestCaseLoad(t, err, "DeleteSubscription_Success")

	deleteNotFoundResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "subscription", "DeleteSubscription_NotFound")
	testutil.AssertTestCaseLoad(t, err, "DeleteSubscription_NotFound")

	testCases := []DeleteSubscriptionTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-SUBSCRIPTION-DELETE-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *subscriptionpb.DeleteSubscriptionRequest {
				return &subscriptionpb.DeleteSubscriptionRequest{
					Data: &subscriptionpb.Subscription{
						Id: deleteSuccessResolver.MustGetString("existingSubscriptionId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *subscriptionpb.DeleteSubscriptionResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
			},
		},
		{
			Name:     "WithTransaction",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-SUBSCRIPTION-DELETE-TRANSACTION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *subscriptionpb.DeleteSubscriptionRequest {
				return &subscriptionpb.DeleteSubscriptionRequest{
					Data: &subscriptionpb.Subscription{
						Id: deleteSuccessResolver.MustGetString("existingSubscriptionId"),
					},
				}
			},
			UseTransaction: true,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *subscriptionpb.DeleteSubscriptionResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-SUBSCRIPTION-DELETE-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *subscriptionpb.DeleteSubscriptionRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "subscription.validation.request_required",
			Assertions: func(t *testing.T, response *subscriptionpb.DeleteSubscriptionResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilRequest(t, err)
			},
		},
		{
			Name:     "NilData",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-SUBSCRIPTION-DELETE-NIL-DATA-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *subscriptionpb.DeleteSubscriptionRequest {
				return &subscriptionpb.DeleteSubscriptionRequest{Data: nil}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "subscription.validation.data_required",
			Assertions: func(t *testing.T, response *subscriptionpb.DeleteSubscriptionResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "nil data")
			},
		},
		{
			Name:     "EmptyId",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-SUBSCRIPTION-DELETE-EMPTY-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *subscriptionpb.DeleteSubscriptionRequest {
				return &subscriptionpb.DeleteSubscriptionRequest{
					Data: &subscriptionpb.Subscription{
						Id: "",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "subscription.validation.id_required",
			Assertions: func(t *testing.T, response *subscriptionpb.DeleteSubscriptionResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty ID")
			},
		},
		{
			Name:     "IdTooShort",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-SUBSCRIPTION-DELETE-ID-TOO-SHORT-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *subscriptionpb.DeleteSubscriptionRequest {
				return &subscriptionpb.DeleteSubscriptionRequest{
					Data: &subscriptionpb.Subscription{
						Id: "ab",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "subscription.validation.id_too_short",
			Assertions: func(t *testing.T, response *subscriptionpb.DeleteSubscriptionResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "ID too short")
			},
		},
		{
			Name:     "NotFound",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-SUBSCRIPTION-DELETE-NOT-FOUND-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *subscriptionpb.DeleteSubscriptionRequest {
				return &subscriptionpb.DeleteSubscriptionRequest{
					Data: &subscriptionpb.Subscription{
						Id: deleteNotFoundResolver.MustGetString("nonExistentSubscriptionId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "subscription.errors.not_found",
			ErrorTags:      map[string]any{"subscriptionId": deleteNotFoundResolver.MustGetString("nonExistentSubscriptionId")},
			Assertions: func(t *testing.T, response *subscriptionpb.DeleteSubscriptionResponse, err error, useCase interface{}, ctx context.Context) {
				uc := useCase.(*DeleteSubscriptionUseCase)
				testutil.AssertTranslatedErrorWithTags(t, err, "subscription.errors.not_found", map[string]any{"subscriptionId": deleteNotFoundResolver.MustGetString("nonExistentSubscriptionId")}, uc.services.TranslationService, ctx)
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
			useCase := createTestDeleteSubscriptionUseCase(businessType, tc.UseTransaction, tc.UseAuth)

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
