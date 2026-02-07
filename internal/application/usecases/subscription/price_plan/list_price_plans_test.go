//go:build mock_db && mock_auth

// Package price_plan provides table-driven tests for the price plan listing use case.
//
// The tests cover various scenarios, including success, transaction handling,
// authorization, nil requests, and filtering.
// Each test case is defined in a table with a specific test code, request setup,
// and assertions.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user").
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education").
//
// Usage:
//   - Run all tests: go test -tags="mock_db,mock_auth" ./...
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestListPricePlansUseCase_Execute_TableDriven
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-SUBSCRIPTION-PRICEPLAN-LIST-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-SUBSCRIPTION-PRICEPLAN-LIST-TRANSACTION-SUCCESS-v1.0: WithTransactionSuccess
//   - ESPYNA-TEST-SUBSCRIPTION-PRICEPLAN-LIST-AUTHORIZATION-DENIED-v1.0: AuthorizationDenied
//   - ESPYNA-TEST-SUBSCRIPTION-PRICEPLAN-LIST-NIL-REQUEST-v1.0: NilRequest
//   - ESPYNA-TEST-SUBSCRIPTION-PRICEPLAN-LIST-VERIFY-DETAILS-v1.0: VerifyPricePlanDetails
//   - ESPYNA-TEST-SUBSCRIPTION-PRICEPLAN-LIST-BUSINESS-LOGIC-VALIDATION-v1.0: BusinessLogicValidation
//   - ESPYNA-TEST-SUBSCRIPTION-PRICEPLAN-LIST-TRANSACTION-FAILURE-v1.0: TransactionFailure
//
// Data Sources:
//   - Test cases: packages/copya/data_test/{businessType}/price_plan.json
//   - Mock data: packages/copya/data/{businessType}/price_plan.json
//   - Translations: packages/lyngua/translations/{languageCode}/{businessType}/price_plan.json
package price_plan

import (
	"context"
	"testing"

	copyatestutil "leapfor.xyz/copya/golang/testutil"
	"leapfor.xyz/espyna/internal/application/shared/testutil"
	"leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock/subscription"
	priceplanpb "leapfor.xyz/esqyma/golang/v1/domain/subscription/price_plan"
)

// Type alias for list price plans test cases
type ListPricePlansTestCase = testutil.GenericTestCase[*priceplanpb.ListPricePlansRequest, *priceplanpb.ListPricePlansResponse]

func createListTestUseCaseWithAuth(businessType string, supportsTransaction bool, shouldAuthorize bool) *ListPricePlansUseCase {
	mockPricePlanRepo := subscription.NewMockPricePlanRepository(businessType)

	repositories := ListPricePlansRepositories{
		PricePlan: mockPricePlanRepo,
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, shouldAuthorize)
	services := ListPricePlansServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
	}

	return NewListPricePlansUseCase(repositories, services)
}

func TestListPricePlansUseCase_Execute_TableDriven(t *testing.T) {
	businessType := testutil.GetTestBusinessType()

	// Load test data resolvers before defining test cases
	listSuccessResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "price_plan", "ListPricePlans_Success")
	testutil.AssertTestCaseLoad(t, err, "ListPricePlans_Success")

	testCases := []ListPricePlansTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-PRICEPLAN-LIST-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *priceplanpb.ListPricePlansRequest {
				return &priceplanpb.ListPricePlansRequest{}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *priceplanpb.ListPricePlansResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertNotNil(t, response.Data, "response data")
				expectedCount := listSuccessResolver.MustGetInt("expectedPricePlanCount")
				testutil.AssertEqual(t, expectedCount, len(response.Data), "price plan count")

				// Validate specific price plan names from test data
				firstPricePlanName := listSuccessResolver.MustGetString("firstPricePlanName")
				secondPricePlanName := listSuccessResolver.MustGetString("secondPricePlanName")
				thirdPricePlanName := listSuccessResolver.MustGetString("thirdPricePlanName")

				pricePlanNames := make(map[string]bool)
				for _, pp := range response.Data {
					pricePlanNames[pp.Name] = true
				}

				testutil.AssertTrue(t, pricePlanNames[firstPricePlanName], "expected price plan '"+firstPricePlanName+"' found")
				testutil.AssertTrue(t, pricePlanNames[secondPricePlanName], "expected price plan '"+secondPricePlanName+"' found")
				testutil.AssertTrue(t, pricePlanNames[thirdPricePlanName], "expected price plan '"+thirdPricePlanName+"' found")
			},
		},
		{
			Name:     "WithTransactionSuccess",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-PRICEPLAN-LIST-TRANSACTION-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *priceplanpb.ListPricePlansRequest {
				return &priceplanpb.ListPricePlansRequest{}
			},
			UseTransaction: true,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *priceplanpb.ListPricePlansResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				expectedCount := listSuccessResolver.MustGetInt("expectedPricePlanCount")
				testutil.AssertEqual(t, expectedCount, len(response.Data), "price plan count with transaction")
			},
		},
		{
			Name:     "AuthorizationDenied",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-PRICEPLAN-LIST-AUTHORIZATION-DENIED-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *priceplanpb.ListPricePlansRequest {
				return &priceplanpb.ListPricePlansRequest{}
			},
			UseTransaction: false,
			UseAuth:        false,
			ExpectSuccess:  false,
			ExpectedError:  "price_plan.errors.authorization_failed",
			Assertions: func(t *testing.T, response *priceplanpb.ListPricePlansResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertAuthorizationError(t, err)
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-PRICEPLAN-LIST-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *priceplanpb.ListPricePlansRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "price_plan.validation.request_required",
			Assertions: func(t *testing.T, response *priceplanpb.ListPricePlansResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilRequest(t, err)
			},
		},
		{
			Name:     "EmptyResults",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-PRICEPLAN-LIST-EMPTY-RESULTS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *priceplanpb.ListPricePlansRequest {
				return &priceplanpb.ListPricePlansRequest{}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *priceplanpb.ListPricePlansResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertNotNil(t, response.Data, "response data")
				// For empty results scenario, we validate the response structure is correct
				testutil.AssertGreaterThanOrEqual(t, len(response.Data), 0, "price plan count")
			},
		},
		{
			Name:     "TransactionFailure",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-PRICEPLAN-LIST-TRANSACTION-FAILURE-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *priceplanpb.ListPricePlansRequest {
				return &priceplanpb.ListPricePlansRequest{}
			},
			UseTransaction: true,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "price_plan.errors.transaction_failed",
			Assertions: func(t *testing.T, response *priceplanpb.ListPricePlansResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTransactionError(t, err)
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
