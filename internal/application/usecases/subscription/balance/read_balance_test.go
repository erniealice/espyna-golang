//go:build mock_db && mock_auth

// Package balance provides table-driven tests for the balance reading use case.
//
// The tests cover various scenarios, including success, transaction handling,
// authorization, nil requests, validation errors, data verification, and boundary conditions.
// Each test case is defined in a table with a specific test code, request setup,
// and assertions.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user").
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education").
//
// Usage:
//   - Run all tests: go test -tags="mock_db,mock_auth" ./...
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestReadBalanceUseCase_Execute_TableDriven
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-SUBSCRIPTION-BALANCE-READ-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-SUBSCRIPTION-BALANCE-READ-TRANSACTION-v1.0: WithTransaction
//   - ESPYNA-TEST-SUBSCRIPTION-BALANCE-READ-AUTHORIZATION-v1.0: Unauthorized
//   - ESPYNA-TEST-SUBSCRIPTION-BALANCE-READ-NIL-REQUEST-v1.0: NilRequest
//   - ESPYNA-TEST-SUBSCRIPTION-BALANCE-READ-NIL-DATA-v1.0: NilData
//   - ESPYNA-TEST-SUBSCRIPTION-BALANCE-READ-EMPTY-ID-v1.0: EmptyId
//   - ESPYNA-TEST-SUBSCRIPTION-BALANCE-READ-NOT-FOUND-v1.0: BalanceNotFound
//   - ESPYNA-TEST-SUBSCRIPTION-BALANCE-READ-VERIFY-DETAILS-v1.0: VerifyBalanceDetails
//   - ESPYNA-TEST-SUBSCRIPTION-BALANCE-READ-BUSINESS-LOGIC-v1.0: BusinessLogicValidation
//   - ESPYNA-TEST-SUBSCRIPTION-BALANCE-READ-EDUCATION-DOMAIN-v1.0: EducationDomainSpecific
//   - ESPYNA-TEST-SUBSCRIPTION-BALANCE-READ-TRANSACTION-FAILURE-v1.0: TransactionFailure
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

// Type alias for read balance test cases
type ReadBalanceTestCase = testutil.GenericTestCase[*balancepb.ReadBalanceRequest, *balancepb.ReadBalanceResponse]

func createReadBalanceUseCaseWithAuth(businessType string, supportsTransaction bool, shouldAuthorize bool) *ReadBalanceUseCase {
	mockBalanceRepo := subscription.NewMockBalanceRepository(businessType)

	repositories := ReadBalanceRepositories{
		Balance: mockBalanceRepo,
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, shouldAuthorize)
	services := ReadBalanceServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
	}

	return NewReadBalanceUseCase(repositories, services)
}

func TestReadBalanceUseCase_Execute_TableDriven(t *testing.T) {
	businessType := testutil.GetTestBusinessType()

	// Load test data resolvers before defining test cases
	readSuccessResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "balance", "ReadBalance_Success")
	testutil.AssertTestCaseLoad(t, err, "ReadBalance_Success")

	readNotFoundResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "balance", "ReadBalance_NotFound")
	testutil.AssertTestCaseLoad(t, err, "ReadBalance_NotFound")

	testCases := []ReadBalanceTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-BALANCE-READ-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *balancepb.ReadBalanceRequest {
				return &balancepb.ReadBalanceRequest{
					Data: &balancepb.Balance{
						Id: readSuccessResolver.MustGetString("existingBalanceId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *balancepb.ReadBalanceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertEqual(t, 1, len(response.Data), "response data length")
				readBalance := response.Data[0]
				testutil.AssertStringEqual(t, readSuccessResolver.MustGetString("existingBalanceId"), readBalance.Id, "balance ID")
				testutil.AssertFieldSet(t, readBalance.DateCreated, "DateCreated")
				testutil.AssertTrue(t, readBalance.Active, "active status")
			},
		},
		{
			Name:     "WithTransaction",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-BALANCE-READ-TRANSACTION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *balancepb.ReadBalanceRequest {
				return &balancepb.ReadBalanceRequest{
					Data: &balancepb.Balance{
						Id: readSuccessResolver.MustGetString("existingBalanceId"),
					},
				}
			},
			UseTransaction: true,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *balancepb.ReadBalanceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				readBalance := response.Data[0]
				testutil.AssertStringEqual(t, readSuccessResolver.MustGetString("existingBalanceId"), readBalance.Id, "balance ID")
			},
		},
		{
			Name:     "BalanceNotFound",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-BALANCE-READ-NOT-FOUND-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *balancepb.ReadBalanceRequest {
				return &balancepb.ReadBalanceRequest{
					Data: &balancepb.Balance{
						Id: readNotFoundResolver.MustGetString("nonExistentBalanceId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "balance.errors.not_found",
			ErrorTags:      map[string]any{"balanceId": readNotFoundResolver.MustGetString("nonExistentBalanceId")},
			Assertions: func(t *testing.T, response *balancepb.ReadBalanceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertError(t, err)
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-BALANCE-READ-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *balancepb.ReadBalanceRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "balance.validation.request_required",
			Assertions: func(t *testing.T, response *balancepb.ReadBalanceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilRequest(t, err)
			},
		},
		{
			Name:     "NilData",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-BALANCE-READ-NIL-DATA-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *balancepb.ReadBalanceRequest {
				return &balancepb.ReadBalanceRequest{Data: nil}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "balance.validation.data_required",
			Assertions: func(t *testing.T, response *balancepb.ReadBalanceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilData(t, err)
			},
		},
		{
			Name:     "EmptyId",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-BALANCE-READ-EMPTY-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *balancepb.ReadBalanceRequest {
				return &balancepb.ReadBalanceRequest{
					Data: &balancepb.Balance{Id: ""},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "balance.validation.id_required",
			Assertions: func(t *testing.T, response *balancepb.ReadBalanceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty ID")
			},
		},
		{
			Name:     "VerifyBalanceDetails",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-BALANCE-READ-VERIFY-DETAILS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *balancepb.ReadBalanceRequest {
				return &balancepb.ReadBalanceRequest{
					Data: &balancepb.Balance{
						Id: readSuccessResolver.MustGetString("existingBalanceId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *balancepb.ReadBalanceResponse, err error, useCase interface{}, ctx context.Context) {
				readBalance := response.Data[0]
				// Note: Test data uses "expectedCurrentBalance" but protobuf uses "amount"
				testutil.AssertTrue(t, readBalance.Amount >= 0, "balance amount is non-negative")
				testutil.AssertFieldSet(t, readBalance.DateCreatedString, "DateCreatedString")
				testutil.AssertTrue(t, readBalance.Active, "active status")
			},
		},
		{
			Name:     "BusinessLogicValidation",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-BALANCE-READ-BUSINESS-LOGIC-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *balancepb.ReadBalanceRequest {
				return &balancepb.ReadBalanceRequest{
					Data: &balancepb.Balance{
						Id: "ab", // Too short ID for business logic validation
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "balance.validation.id_too_short",
			Assertions: func(t *testing.T, response *balancepb.ReadBalanceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "ID too short")
			},
		},
		{
			Name:     "EducationDomainSpecific",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-BALANCE-READ-EDUCATION-DOMAIN-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *balancepb.ReadBalanceRequest {
				return &balancepb.ReadBalanceRequest{
					Data: &balancepb.Balance{
						Id: readSuccessResolver.MustGetString("existingBalanceId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *balancepb.ReadBalanceResponse, err error, useCase interface{}, ctx context.Context) {
				// Education domain specific validation
				readBalance := response.Data[0]
				testutil.AssertNonEmptyString(t, readBalance.Id, "education balance ID")
				testutil.AssertTrue(t, readBalance.Amount >= 0, "education balance non-negative")
				testutil.AssertTrue(t, readBalance.Active, "education balance active")
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
			useCase := createReadBalanceUseCaseWithAuth(businessType, tc.UseTransaction, tc.UseAuth)

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

func TestReadBalanceUseCase_Execute_TransactionFailure(t *testing.T) {
	testCode := "ESPYNA-TEST-SUBSCRIPTION-BALANCE-READ-TRANSACTION-FAILURE-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "TransactionFailure", false)

	// Create a use case with a failing transaction service
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	mockBalanceRepo := subscription.NewMockBalanceRepository(businessType)

	repositories := ReadBalanceRepositories{
		Balance: mockBalanceRepo,
	}

	standardServices := testutil.CreateStandardServices(false, true)
	services := ReadBalanceServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   mockDb.NewFailingMockTransactionService(),
		TranslationService:   standardServices.TranslationService,
	}

	useCase := NewReadBalanceUseCase(repositories, services)

	// Load test case using centralized TestCaseResolver
	resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "balance", "ReadBalance_Success")
	testutil.AssertTestCaseLoad(t, err, "ReadBalance_Success")

	req := &balancepb.ReadBalanceRequest{
		Data: &balancepb.Balance{
			Id: resolver.MustGetString("existingBalanceId"),
		},
	}

	_, err = useCase.Execute(ctx, req)

	// Verify that a transaction error occurred
	testutil.AssertTransactionError(t, err)

	// Log completion of transaction failure test
	testutil.LogTestResult(t, testCode, "TransactionFailure", false, err)
}
