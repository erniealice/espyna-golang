//go:build mock_db && mock_auth

// Package balance provides table-driven tests for the balance creation use case.
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
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestCreateBalanceUseCase_Execute_TableDriven
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-SUBSCRIPTION-BALANCE-CREATE-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-SUBSCRIPTION-BALANCE-CREATE-TRANSACTION-v1.0: WithTransaction
//   - ESPYNA-TEST-SUBSCRIPTION-BALANCE-CREATE-AUTHORIZATION-v1.0: Unauthorized
//   - ESPYNA-TEST-SUBSCRIPTION-BALANCE-CREATE-NIL-REQUEST-v1.0: NilRequest
//   - ESPYNA-TEST-SUBSCRIPTION-BALANCE-CREATE-NIL-DATA-v1.0: NilData
//   - ESPYNA-TEST-SUBSCRIPTION-BALANCE-CREATE-VALIDATION-EMPTY-AMOUNT-v1.0: EmptyAmount
//   - ESPYNA-TEST-SUBSCRIPTION-BALANCE-CREATE-VALIDATION-INVALID-AMOUNT-v1.0: InvalidAmount
//   - ESPYNA-TEST-SUBSCRIPTION-BALANCE-CREATE-ENRICHMENT-v1.0: DataEnrichment
//   - ESPYNA-TEST-SUBSCRIPTION-BALANCE-CREATE-VALIDATION-MINIMAL-v1.0: MinimalValidData
//   - ESPYNA-TEST-SUBSCRIPTION-BALANCE-CREATE-VALIDATION-MAXIMAL-v1.0: MaxValidData
//   - ESPYNA-TEST-SUBSCRIPTION-BALANCE-CREATE-EDUCATION-DOMAIN-v1.0: EducationDomainSpecific
//   - ESPYNA-TEST-SUBSCRIPTION-BALANCE-CREATE-TRANSACTION-FAILURE-v1.0: TransactionFailure
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
	"github.com/erniealice/espyna-golang/internal/application/shared/testutil"
	mockDb "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock"
	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock/subscription"
	balancepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/balance"
)

// Type alias for create balance test cases
type CreateBalanceTestCase = testutil.GenericTestCase[*balancepb.CreateBalanceRequest, *balancepb.CreateBalanceResponse]

func createCreateBalanceTestUseCaseWithAuth(businessType string, supportsTransaction bool, shouldAuthorize bool) *CreateBalanceUseCase {
	mockBalanceRepo := subscription.NewMockBalanceRepository(businessType)

	repositories := CreateBalanceRepositories{
		Balance: mockBalanceRepo,
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, shouldAuthorize)
	services := CreateBalanceServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
		IDService:            standardServices.IDService,
	}

	return NewCreateBalanceUseCase(repositories, services)
}

func TestCreateBalanceUseCase_Execute_TableDriven(t *testing.T) {
	businessType := testutil.GetTestBusinessType()

	// Load test data resolvers before defining test cases
	createSuccessResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "balance", "CreateBalance_Success")
	testutil.AssertTestCaseLoad(t, err, "CreateBalance_Success")

	testCases := []CreateBalanceTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-BALANCE-CREATE-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *balancepb.CreateBalanceRequest {
				return &balancepb.CreateBalanceRequest{
					Data: &balancepb.Balance{
						Amount: createSuccessResolver.MustGetFloat64("currentBalance"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *balancepb.CreateBalanceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertEqual(t, 1, len(response.Data), "response data length")
				createdBalance := response.Data[0]
				testutil.AssertEqual(t, createSuccessResolver.MustGetFloat64("currentBalance"), createdBalance.Amount, "balance amount")
				testutil.AssertNonEmptyString(t, createdBalance.Id, "balance ID")
				testutil.AssertTrue(t, createdBalance.Active, "balance active status")
				testutil.AssertFieldSet(t, createdBalance.DateCreated, "DateCreated")
				testutil.AssertFieldSet(t, createdBalance.DateCreatedString, "DateCreatedString")
			},
		},
		{
			Name:     "WithTransaction",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-BALANCE-CREATE-TRANSACTION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *balancepb.CreateBalanceRequest {
				return &balancepb.CreateBalanceRequest{
					Data: &balancepb.Balance{
						Amount: createSuccessResolver.MustGetFloat64("currentBalance"),
					},
				}
			},
			UseTransaction: true,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *balancepb.CreateBalanceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				createdBalance := response.Data[0]
				testutil.AssertEqual(t, createSuccessResolver.MustGetFloat64("currentBalance"), createdBalance.Amount, "balance amount")
			},
		},
		{
			Name:     "Unauthorized",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-BALANCE-CREATE-AUTHORIZATION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *balancepb.CreateBalanceRequest {
				return &balancepb.CreateBalanceRequest{
					Data: &balancepb.Balance{
						Amount: 100.0,
					},
				}
			},
			UseTransaction: false,
			UseAuth:        false,
			ExpectSuccess:  false,
			ExpectedError:  "balance.errors.authorization_failed",
			Assertions: func(t *testing.T, response *balancepb.CreateBalanceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertAuthorizationError(t, err)
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-BALANCE-CREATE-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *balancepb.CreateBalanceRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "balance.validation.request_required",
			Assertions: func(t *testing.T, response *balancepb.CreateBalanceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilRequest(t, err)
			},
		},
		{
			Name:     "NilData",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-BALANCE-CREATE-NIL-DATA-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *balancepb.CreateBalanceRequest {
				return &balancepb.CreateBalanceRequest{Data: nil}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "balance.validation.data_required",
			Assertions: func(t *testing.T, response *balancepb.CreateBalanceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilData(t, err)
			},
		},
		{
			Name:     "InvalidAmount",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-BALANCE-CREATE-VALIDATION-INVALID-AMOUNT-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *balancepb.CreateBalanceRequest {
				return &balancepb.CreateBalanceRequest{
					Data: &balancepb.Balance{
						Amount: -100.00, // Invalid negative amount
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "balance.validation.amount_invalid",
			Assertions: func(t *testing.T, response *balancepb.CreateBalanceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "invalid amount")
			},
		},
		{
			Name:     "DataEnrichment",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-BALANCE-CREATE-ENRICHMENT-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *balancepb.CreateBalanceRequest {
				return &balancepb.CreateBalanceRequest{
					Data: &balancepb.Balance{
						Amount: 500.00,
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *balancepb.CreateBalanceResponse, err error, useCase interface{}, ctx context.Context) {
				createdBalance := response.Data[0]
				testutil.AssertNonEmptyString(t, createdBalance.Id, "generated ID")
				testutil.AssertFieldSet(t, createdBalance.DateCreated, "DateCreated")
				testutil.AssertFieldSet(t, createdBalance.DateCreatedString, "DateCreatedString")
				testutil.AssertTrue(t, createdBalance.Active, "Active")
			},
		},
		{
			Name:     "MinimalValidData",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-BALANCE-CREATE-VALIDATION-MINIMAL-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *balancepb.CreateBalanceRequest {
				return &balancepb.CreateBalanceRequest{
					Data: &balancepb.Balance{
						Amount: 0.00, // Minimal valid amount
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *balancepb.CreateBalanceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				createdBalance := response.Data[0]
				testutil.AssertEqual(t, 0.00, createdBalance.Amount, "balance amount")
			},
		},
		{
			Name:     "MaxValidData",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-BALANCE-CREATE-VALIDATION-MAXIMAL-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *balancepb.CreateBalanceRequest {
				return &balancepb.CreateBalanceRequest{
					Data: &balancepb.Balance{
						Amount: 99999999.99, // Large valid amount
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *balancepb.CreateBalanceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				createdBalance := response.Data[0]
				testutil.AssertEqual(t, 99999999.99, createdBalance.Amount, "balance amount")
			},
		},
		{
			Name:     "EducationDomainSpecific",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-BALANCE-CREATE-EDUCATION-DOMAIN-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *balancepb.CreateBalanceRequest {
				return &balancepb.CreateBalanceRequest{
					Data: &balancepb.Balance{
						Amount: 2500.00, // Typical education balance amount
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *balancepb.CreateBalanceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				createdBalance := response.Data[0]
				testutil.AssertEqual(t, 2500.00, createdBalance.Amount, "balance amount")
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
			useCase := createCreateBalanceTestUseCaseWithAuth(businessType, tc.UseTransaction, tc.UseAuth)

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

func TestCreateBalanceUseCase_Execute_TransactionFailure(t *testing.T) {
	testCode := "ESPYNA-TEST-SUBSCRIPTION-BALANCE-CREATE-TRANSACTION-FAILURE-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "TransactionFailure", false)

	// Create a use case with a failing transaction service
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	mockBalanceRepo := subscription.NewMockBalanceRepository(businessType)

	repositories := CreateBalanceRepositories{
		Balance: mockBalanceRepo,
	}

	standardServices := testutil.CreateStandardServices(false, true)
	services := CreateBalanceServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   mockDb.NewFailingMockTransactionService(),
		TranslationService:   standardServices.TranslationService,
		IDService:            standardServices.IDService,
	}

	useCase := NewCreateBalanceUseCase(repositories, services)

	// Load test case using centralized TestCaseResolver
	resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "balance", "CreateBalance_Success")
	testutil.AssertTestCaseLoad(t, err, "CreateBalance_Success")

	req := &balancepb.CreateBalanceRequest{
		Data: &balancepb.Balance{
			Amount: resolver.MustGetFloat64("currentBalance"),
		},
	}

	_, err = useCase.Execute(ctx, req)

	// Verify that a transaction error occurred
	testutil.AssertTransactionError(t, err)

	// Log completion of transaction failure test
	testutil.LogTestResult(t, testCode, "TransactionFailure", false, err)
}
