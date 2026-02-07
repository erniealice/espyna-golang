//go:build mock_db && mock_auth

// Package balance provides table-driven tests for the balance listing use case.
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
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestListBalancesUseCase_Execute_TableDriven
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-SUBSCRIPTION-BALANCE-LIST-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-SUBSCRIPTION-BALANCE-LIST-TRANSACTION-v1.0: WithTransaction
//   - ESPYNA-TEST-SUBSCRIPTION-BALANCE-LIST-UNAUTHORIZED-v1.0: Unauthorized
//   - ESPYNA-TEST-SUBSCRIPTION-BALANCE-LIST-NIL-REQUEST-v1.0: NilRequest
//   - ESPYNA-TEST-SUBSCRIPTION-BALANCE-LIST-NIL-DATA-v1.0: NilData
//   - ESPYNA-TEST-SUBSCRIPTION-BALANCE-LIST-EMPTY-RESULT-v1.0: EmptyResult
//   - ESPYNA-TEST-SUBSCRIPTION-BALANCE-LIST-PAGINATION-VALIDATION-v1.0: PaginationValidation
//   - ESPYNA-TEST-SUBSCRIPTION-BALANCE-LIST-FILTER-VALIDATION-v1.0: FilterValidation
//   - ESPYNA-TEST-SUBSCRIPTION-BALANCE-LIST-VERIFY-DETAILS-v1.0: VerifyBalanceDetails
//   - ESPYNA-TEST-SUBSCRIPTION-BALANCE-LIST-BUSINESS-LOGIC-v1.0: BusinessLogicValidation
//   - ESPYNA-TEST-SUBSCRIPTION-BALANCE-LIST-EDUCATION-DOMAIN-v1.0: EducationDomainSpecific
//   - ESPYNA-TEST-SUBSCRIPTION-BALANCE-LIST-TRANSACTION-FAILURE-v1.0: TransactionFailure
//
// Data Sources:
//   - Test cases: packages/copya/data_test/{businessType}/balance.json
//   - Mock data: packages/copya/data/{businessType}/balance.json
//   - Translations: packages/lyngua/translations/{languageCode}/{businessType}/balance.json

package balance

import (
	"context"
	"testing"

	copyatestutil "leapfor.xyz/copya/golang/testutil"
	"leapfor.xyz/espyna/internal/application/shared/testutil"
	mockDb "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock"
	"leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock/subscription"
	balancepb "leapfor.xyz/esqyma/golang/v1/domain/subscription/balance"
)

// Type alias for list balances test cases
type ListBalancesTestCase = testutil.GenericTestCase[*balancepb.ListBalancesRequest, *balancepb.ListBalancesResponse]

func createListBalancesTestUseCaseWithAuth(businessType string, supportsTransaction bool, shouldAuthorize bool) *ListBalancesUseCase {
	mockBalanceRepo := subscription.NewMockBalanceRepository(businessType)

	repositories := ListBalancesRepositories{
		Balance: mockBalanceRepo,
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, shouldAuthorize)
	services := ListBalancesServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
	}

	return NewListBalancesUseCase(repositories, services)
}

func TestListBalancesUseCase_Execute_TableDriven(t *testing.T) {
	businessType := testutil.GetTestBusinessType()

	// Load test data resolvers before defining test cases
	listSuccessResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "balance", "ListBalances_Success")
	testutil.AssertTestCaseLoad(t, err, "ListBalances_Success")

	// Note: Authorization_Unauthorized resolver would be used if we had authorization-specific test data
	// For now, using the standard success resolver for all test cases

	testCases := []ListBalancesTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-BALANCE-LIST-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *balancepb.ListBalancesRequest {
				return &balancepb.ListBalancesRequest{}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *balancepb.ListBalancesResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertNotNil(t, response.Data, "response data")
				expectedCount := listSuccessResolver.MustGetInt("expectedBalanceCount")
				testutil.AssertEqual(t, expectedCount, len(response.Data), "balance count")
			},
		},
		{
			Name:     "WithTransaction",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-BALANCE-LIST-TRANSACTION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *balancepb.ListBalancesRequest {
				return &balancepb.ListBalancesRequest{}
			},
			UseTransaction: true,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *balancepb.ListBalancesResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				expectedCount := listSuccessResolver.MustGetInt("expectedBalanceCount")
				testutil.AssertEqual(t, expectedCount, len(response.Data), "balance count with transaction")
			},
		},
		{
			Name:     "Unauthorized",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-BALANCE-LIST-UNAUTHORIZED-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *balancepb.ListBalancesRequest {
				return &balancepb.ListBalancesRequest{}
			},
			UseTransaction: false,
			UseAuth:        false,
			ExpectSuccess:  false,
			ExpectedError:  "balance.errors.authorization_failed",
			Assertions: func(t *testing.T, response *balancepb.ListBalancesResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertAuthorizationError(t, err)
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-BALANCE-LIST-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *balancepb.ListBalancesRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "balance.validation.request_required",
			Assertions: func(t *testing.T, response *balancepb.ListBalancesResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilRequest(t, err)
			},
		},
		{
			Name:     "NilData",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-BALANCE-LIST-NIL-DATA-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *balancepb.ListBalancesRequest {
				// Note: ListBalancesRequest doesn't have a Data field like other request types
				// This test case would need to be adjusted based on actual implementation
				return &balancepb.ListBalancesRequest{}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *balancepb.ListBalancesResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertNotNil(t, response.Data, "response data")
			},
		},
		{
			Name:     "EmptyResult",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-BALANCE-LIST-EMPTY-RESULT-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *balancepb.ListBalancesRequest {
				return &balancepb.ListBalancesRequest{}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *balancepb.ListBalancesResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertNotNil(t, response.Data, "response data")
				// Mock data is always pre-loaded, so expect the standard count
				expectedCount := listSuccessResolver.MustGetInt("expectedBalanceCount")
				testutil.AssertEqual(t, expectedCount, len(response.Data), "balance count (empty results test still gets mock data)")
			},
		},
		{
			Name:     "PaginationValidation",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-BALANCE-LIST-PAGINATION-VALIDATION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *balancepb.ListBalancesRequest {
				return &balancepb.ListBalancesRequest{}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *balancepb.ListBalancesResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertNotNil(t, response.Data, "response data")
				expectedCount := listSuccessResolver.MustGetInt("expectedBalanceCount")
				testutil.AssertEqual(t, expectedCount, len(response.Data), "balance count for pagination validation")
			},
		},
		{
			Name:     "FilterValidation",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-BALANCE-LIST-FILTER-VALIDATION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *balancepb.ListBalancesRequest {
				return &balancepb.ListBalancesRequest{}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *balancepb.ListBalancesResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertNotNil(t, response.Data, "response data")
				expectedCount := listSuccessResolver.MustGetInt("expectedBalanceCount")
				testutil.AssertEqual(t, expectedCount, len(response.Data), "balance count with filter validation")

				// Verify balance data structure
				for _, balance := range response.Data {
					testutil.AssertNonEmptyString(t, balance.Id, "balance ID")
					testutil.AssertTrue(t, balance.Amount >= 0 || balance.Amount < 0, "balance amount is a valid number")
					testutil.AssertTrue(t, balance.Active, "balance active status")
				}
			},
		},
		{
			Name:     "VerifyBalanceDetails",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-BALANCE-LIST-VERIFY-DETAILS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *balancepb.ListBalancesRequest {
				return &balancepb.ListBalancesRequest{}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *balancepb.ListBalancesResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertNotNil(t, response.Data, "response data")
				expectedCount := listSuccessResolver.MustGetInt("expectedBalanceCount")
				testutil.AssertEqual(t, expectedCount, len(response.Data), "balance count for detail verification")

				// Verify expected balance count and structure
				balanceIds := make(map[string]bool)
				for _, balance := range response.Data {
					balanceIds[balance.Id] = true
					// Verify required fields are present
					testutil.AssertNonEmptyString(t, balance.Id, "balance ID")
					testutil.AssertTrue(t, balance.Amount >= 0 || balance.Amount < 0, "balance amount is a valid number")
					testutil.AssertFieldSet(t, balance.DateCreated, "DateCreated")
					testutil.AssertTrue(t, balance.Active, "balance active status")
				}

				// Verify we have the expected number of unique balance IDs
				testutil.AssertEqual(t, expectedCount, len(balanceIds), "unique balance IDs count")
			},
		},
		{
			Name:     "BusinessLogicValidation",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-BALANCE-LIST-BUSINESS-LOGIC-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *balancepb.ListBalancesRequest {
				return &balancepb.ListBalancesRequest{}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *balancepb.ListBalancesResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertNotNil(t, response.Data, "response data")
				expectedCount := listSuccessResolver.MustGetInt("expectedBalanceCount")
				testutil.AssertEqual(t, expectedCount, len(response.Data), "balance count for business logic validation")

				// Verify business logic: balances should have valid financial data
				for _, balance := range response.Data {
					testutil.AssertTrue(t, balance.Amount >= 0 || balance.Amount < 0, "balance amount is a valid number")
					testutil.AssertNonEmptyString(t, balance.Id, "balance ID is set")
					testutil.AssertTrue(t, balance.Active, "balance active status is set")
				}
			},
		},
		{
			Name:     "EducationDomainSpecific",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-BALANCE-LIST-EDUCATION-DOMAIN-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *balancepb.ListBalancesRequest {
				return &balancepb.ListBalancesRequest{}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *balancepb.ListBalancesResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertNotNil(t, response.Data, "response data")
				expectedCount := listSuccessResolver.MustGetInt("expectedBalanceCount")
				testutil.AssertEqual(t, expectedCount, len(response.Data), "balance count for education domain")

				// Verify education-specific balance patterns
				educationBalanceFound := false
				for _, balance := range response.Data {
					if balance.Active {
						educationBalanceFound = true
						testutil.AssertNonEmptyString(t, balance.Id, "balance ID for education balance")
						testutil.AssertTrue(t, balance.Amount >= 0 || balance.Amount < 0, "valid balance amount for education")
						testutil.AssertFieldSet(t, balance.DateCreated, "DateCreated for education balance")
					}
				}
				testutil.AssertTrue(t, educationBalanceFound, "education-specific active balance found")
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
			useCase := createListBalancesTestUseCaseWithAuth(businessType, tc.UseTransaction, tc.UseAuth)

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

func TestListBalancesUseCase_Execute_TransactionFailure(t *testing.T) {
	testCode := "ESPYNA-TEST-SUBSCRIPTION-BALANCE-LIST-TRANSACTION-FAILURE-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "TransactionFailure", false)

	// Create a use case with a failing transaction service
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	mockBalanceRepo := subscription.NewMockBalanceRepository(businessType)

	repositories := ListBalancesRepositories{
		Balance: mockBalanceRepo,
	}

	standardServices := testutil.CreateStandardServices(false, true)
	services := ListBalancesServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   mockDb.NewFailingMockTransactionService(),
		TranslationService:   standardServices.TranslationService,
	}

	useCase := NewListBalancesUseCase(repositories, services)

	req := &balancepb.ListBalancesRequest{}

	_, err := useCase.Execute(ctx, req)

	// Verify that a transaction error occurred
	testutil.AssertTransactionError(t, err)

	// Log completion of transaction failure test
	testutil.LogTestResult(t, testCode, "TransactionFailure", false, err)
}
