//go:build mock_db && mock_auth

// Package balance provides table-driven tests for the balance update use case.
//
// The tests cover various scenarios, including success, transaction handling,
// nil requests, validation errors, data enrichment, and boundary conditions.
// Each test case is defined in a table with a specific test code, request setup,
// and assertions.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user").
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education").
//
// Usage:
//   - Run all tests: go test -tags="mock_db,mock_auth" ./...
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestUpdateBalanceUseCase_Execute_TableDriven
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-SUBSCRIPTION-BALANCE-UPDATE-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-SUBSCRIPTION-BALANCE-UPDATE-TRANSACTION-v1.0: WithTransaction
//   - ESPYNA-TEST-SUBSCRIPTION-BALANCE-UPDATE-UNAUTHORIZED-v1.0: Unauthorized
//   - ESPYNA-TEST-SUBSCRIPTION-BALANCE-UPDATE-NIL-REQUEST-v1.0: NilRequest
//   - ESPYNA-TEST-SUBSCRIPTION-BALANCE-UPDATE-NIL-DATA-v1.0: NilData
//   - ESPYNA-TEST-SUBSCRIPTION-BALANCE-UPDATE-VALIDATION-EMPTY-ID-v1.0: EmptyId
//   - ESPYNA-TEST-SUBSCRIPTION-BALANCE-UPDATE-NOT-FOUND-v1.0: NotFound
//   - ESPYNA-TEST-SUBSCRIPTION-BALANCE-UPDATE-VALIDATION-INVALID-CURRENT-BALANCE-v1.0: InvalidCurrentBalance
//   - ESPYNA-TEST-SUBSCRIPTION-BALANCE-UPDATE-ENRICHMENT-v1.0: DataEnrichment
//   - ESPYNA-TEST-SUBSCRIPTION-BALANCE-UPDATE-MINIMAL-VALID-v1.0: MinimalValidData
//   - ESPYNA-TEST-SUBSCRIPTION-BALANCE-UPDATE-EDUCATION-DOMAIN-v1.0: EducationDomainSpecific
//   - ESPYNA-TEST-SUBSCRIPTION-BALANCE-UPDATE-TRANSACTION-FAILURE-v1.0: TransactionFailure
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

// Type alias for update balance test cases
type UpdateBalanceTestCase = testutil.GenericTestCase[*balancepb.UpdateBalanceRequest, *balancepb.UpdateBalanceResponse]

func createTestUpdateBalanceUseCaseWithAuth(businessType string, supportsTransaction bool, shouldAuthorize bool) *UpdateBalanceUseCase {
	mockBalanceRepo := subscription.NewMockBalanceRepository(businessType)

	repositories := UpdateBalanceRepositories{
		Balance: mockBalanceRepo,
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, shouldAuthorize)
	services := UpdateBalanceServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
	}

	return NewUpdateBalanceUseCase(repositories, services)
}

func TestUpdateBalanceUseCase_Execute_TableDriven(t *testing.T) {
	businessType := testutil.GetTestBusinessType()

	// Load test data resolvers before defining test cases
	updateSuccessResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "balance", "UpdateBalance_Success")
	testutil.AssertTestCaseLoad(t, err, "UpdateBalance_Success")

	updateNotFoundResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "balance", "UpdateBalance_NotFound")
	testutil.AssertTestCaseLoad(t, err, "UpdateBalance_NotFound")

	createSuccessResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "balance", "CreateBalance_Success")
	testutil.AssertTestCaseLoad(t, err, "CreateBalance_Success")

	validationInvalidCurrentBalanceResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "balance", "ValidationError_InvalidCurrentBalance")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_InvalidCurrentBalance")

	testCases := []UpdateBalanceTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-BALANCE-UPDATE-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *balancepb.UpdateBalanceRequest {
				return &balancepb.UpdateBalanceRequest{
					Data: &balancepb.Balance{
						Id:     updateSuccessResolver.MustGetString("existingBalanceId"),
						Amount: updateSuccessResolver.MustGetFloat64("updatedCurrentBalance"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *balancepb.UpdateBalanceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertEqual(t, 1, len(response.Data), "response data length")
				updatedBalance := response.Data[0]
				testutil.AssertEqual(t, updateSuccessResolver.MustGetFloat64("updatedCurrentBalance"), updatedBalance.Amount, "updated amount")
				testutil.AssertFieldSet(t, updatedBalance.DateModified, "DateModified")
				testutil.AssertFieldSet(t, updatedBalance.DateModifiedString, "DateModifiedString")
			},
		},
		{
			Name:     "WithTransaction",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-BALANCE-UPDATE-TRANSACTION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *balancepb.UpdateBalanceRequest {
				return &balancepb.UpdateBalanceRequest{
					Data: &balancepb.Balance{
						Id:     createSuccessResolver.MustGetString("secondaryBalanceId"),
						Amount: 1800.00,
					},
				}
			},
			UseTransaction: true,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *balancepb.UpdateBalanceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				updatedBalance := response.Data[0]
				testutil.AssertEqual(t, 1800.00, updatedBalance.Amount, "updated amount")
			},
		},
		{
			Name:     "Unauthorized",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-BALANCE-UPDATE-UNAUTHORIZED-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *balancepb.UpdateBalanceRequest {
				return &balancepb.UpdateBalanceRequest{
					Data: &balancepb.Balance{
						Id:     updateSuccessResolver.MustGetString("existingBalanceId"),
						Amount: 1500.00,
					},
				}
			},
			UseTransaction: false,
			UseAuth:        false,
			ExpectSuccess:  false,
			ExpectedError:  "authorization.errors.access_denied",
			Assertions: func(t *testing.T, response *balancepb.UpdateBalanceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertAuthorizationError(t, err)
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-BALANCE-UPDATE-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *balancepb.UpdateBalanceRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "balance.validation.request_required",
			Assertions: func(t *testing.T, response *balancepb.UpdateBalanceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilRequest(t, err)
			},
		},
		{
			Name:     "NilData",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-BALANCE-UPDATE-NIL-DATA-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *balancepb.UpdateBalanceRequest {
				return &balancepb.UpdateBalanceRequest{Data: nil}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "balance.validation.data_required",
			Assertions: func(t *testing.T, response *balancepb.UpdateBalanceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilData(t, err)
			},
		},
		{
			Name:     "EmptyId",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-BALANCE-UPDATE-VALIDATION-EMPTY-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *balancepb.UpdateBalanceRequest {
				return &balancepb.UpdateBalanceRequest{
					Data: &balancepb.Balance{
						Id:     "",
						Amount: 1500.00,
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "balance.validation.id_required",
			Assertions: func(t *testing.T, response *balancepb.UpdateBalanceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty ID")
			},
		},
		{
			Name:     "NotFound",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-BALANCE-UPDATE-NOT-FOUND-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *balancepb.UpdateBalanceRequest {
				return &balancepb.UpdateBalanceRequest{
					Data: &balancepb.Balance{
						Id:     updateNotFoundResolver.MustGetString("nonExistentBalanceId"),
						Amount: 1500.00,
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "balance.errors.not_found",
			ErrorTags:      map[string]any{"balanceId": updateNotFoundResolver.MustGetString("nonExistentBalanceId")},
			Assertions: func(t *testing.T, response *balancepb.UpdateBalanceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertError(t, err)
			},
		},
		{
			Name:     "InvalidCurrentBalance",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-BALANCE-UPDATE-VALIDATION-INVALID-CURRENT-BALANCE-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *balancepb.UpdateBalanceRequest {
				return &balancepb.UpdateBalanceRequest{
					Data: &balancepb.Balance{
						Id:     createSuccessResolver.MustGetString("primaryBalanceId"),
						Amount: validationInvalidCurrentBalanceResolver.MustGetFloat64("invalidCurrentBalance"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "balance.validation.amount_invalid",
			Assertions: func(t *testing.T, response *balancepb.UpdateBalanceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "invalid current balance")
			},
		},
		{
			Name:     "DataEnrichment",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-BALANCE-UPDATE-ENRICHMENT-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *balancepb.UpdateBalanceRequest {
				return &balancepb.UpdateBalanceRequest{
					Data: &balancepb.Balance{
						Id:     createSuccessResolver.MustGetString("thirdBalanceId"),
						Amount: 2000.00,
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *balancepb.UpdateBalanceResponse, err error, useCase interface{}, ctx context.Context) {
				updatedBalance := response.Data[0]
				testutil.AssertFieldSet(t, updatedBalance.DateModified, "DateModified")
				testutil.AssertFieldSet(t, updatedBalance.DateModifiedString, "DateModifiedString")
				testutil.AssertTimestampPositive(t, *updatedBalance.DateModified, "DateModified")
				testutil.AssertTimestampInMilliseconds(t, *updatedBalance.DateModified, "DateModified")
			},
		},
		{
			Name:     "MinimalValidData",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-BALANCE-UPDATE-MINIMAL-VALID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *balancepb.UpdateBalanceRequest {
				return &balancepb.UpdateBalanceRequest{
					Data: &balancepb.Balance{
						Id:     createSuccessResolver.MustGetString("minimalValidId"),
						Amount: 0.01,
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *balancepb.UpdateBalanceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				updatedBalance := response.Data[0]
				testutil.AssertEqual(t, 0.01, updatedBalance.Amount, "minimal valid amount")
			},
		},
		{
			Name:     "EducationDomainSpecific",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-BALANCE-UPDATE-EDUCATION-DOMAIN-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *balancepb.UpdateBalanceRequest {
				return &balancepb.UpdateBalanceRequest{
					Data: &balancepb.Balance{
						Id:     createSuccessResolver.MustGetString("educationBalanceId"),
						Amount: 15000.00,
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *balancepb.UpdateBalanceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				updatedBalance := response.Data[0]
				testutil.AssertEqual(t, 15000.00, updatedBalance.Amount, "education-specific amount")
				testutil.AssertFieldSet(t, updatedBalance.DateModified, "DateModified")
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
			useCase := createTestUpdateBalanceUseCaseWithAuth(businessType, tc.UseTransaction, tc.UseAuth)

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
					if tc.ExactError {
						testutil.AssertStringEqual(t, tc.ExpectedError, err.Error(), "error message")
					} else if tc.ErrorTags != nil {
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

func TestUpdateBalanceUseCase_Execute_TransactionFailure(t *testing.T) {
	testCode := "ESPYNA-TEST-SUBSCRIPTION-BALANCE-UPDATE-TRANSACTION-FAILURE-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "TransactionFailure", false)

	// Create a use case with a failing transaction service
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	mockBalanceRepo := subscription.NewMockBalanceRepository(businessType)

	repositories := UpdateBalanceRepositories{
		Balance: mockBalanceRepo,
	}

	standardServices := testutil.CreateStandardServices(false, true)
	services := UpdateBalanceServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   mockDb.NewFailingMockTransactionService(),
		TranslationService:   standardServices.TranslationService,
	}

	useCase := NewUpdateBalanceUseCase(repositories, services)

	// Load test case using centralized TestCaseResolver
	resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "balance", "UpdateBalance_Success")
	testutil.AssertTestCaseLoad(t, err, "UpdateBalance_Success")

	req := &balancepb.UpdateBalanceRequest{
		Data: &balancepb.Balance{
			Id:     resolver.MustGetString("existingBalanceId"),
			Amount: resolver.MustGetFloat64("updatedCurrentBalance"),
		},
	}

	_, err = useCase.Execute(ctx, req)

	// Verify that a transaction error occurred
	testutil.AssertTransactionError(t, err)

	// Log completion of transaction failure test
	testutil.LogTestResult(t, testCode, "TransactionFailure", false, err)
}
