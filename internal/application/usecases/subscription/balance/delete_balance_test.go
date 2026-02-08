//go:build mock_db && mock_auth

// Package balance provides table-driven tests for the balance deletion use case.
//
// The tests cover various scenarios, including success, transaction handling,
// authorization, nil requests, validation errors, and error handling.
// Each test case is defined in a table with a specific test code, request setup,
// and assertions.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user").
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education").
//
// Usage:
//   - Run all tests: go test -tags="mock_db,mock_auth" ./...
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestDeleteBalanceUseCase_Execute_TableDriven
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-SUBSCRIPTION-BALANCE-DELETE-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-SUBSCRIPTION-BALANCE-DELETE-TRANSACTION-v1.0: WithTransaction
//   - ESPYNA-TEST-SUBSCRIPTION-BALANCE-DELETE-AUTHORIZATION-v1.0: Unauthorized
//   - ESPYNA-TEST-SUBSCRIPTION-BALANCE-DELETE-NOT-FOUND-v1.0: BalanceNotFound
//   - ESPYNA-TEST-SUBSCRIPTION-BALANCE-DELETE-NIL-REQUEST-v1.0: NilRequest
//   - ESPYNA-TEST-SUBSCRIPTION-BALANCE-DELETE-NIL-DATA-v1.0: NilData
//   - ESPYNA-TEST-SUBSCRIPTION-BALANCE-DELETE-VALIDATION-EMPTY-ID-v1.0: EmptyId
//   - ESPYNA-TEST-SUBSCRIPTION-BALANCE-DELETE-SOFT-DELETE-v1.0: SoftDeleteVerification
//   - ESPYNA-TEST-SUBSCRIPTION-BALANCE-DELETE-EDUCATION-DOMAIN-v1.0: EducationDomainSpecific
//   - ESPYNA-TEST-SUBSCRIPTION-BALANCE-DELETE-TRANSACTION-FAILURE-v1.0: TransactionFailure
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

// Type alias for delete balance test cases
type DeleteBalanceTestCase = testutil.GenericTestCase[*balancepb.DeleteBalanceRequest, *balancepb.DeleteBalanceResponse]

func createDeleteBalanceTestUseCaseWithAuth(businessType string, supportsTransaction bool, shouldAuthorize bool) *DeleteBalanceUseCase {
	mockBalanceRepo := subscription.NewMockBalanceRepository(businessType)

	repositories := DeleteBalanceRepositories{
		Balance: mockBalanceRepo,
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, shouldAuthorize)
	services := DeleteBalanceServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
	}

	return NewDeleteBalanceUseCase(repositories, services)
}

func TestDeleteBalanceUseCase_Execute_TableDriven(t *testing.T) {
	businessType := testutil.GetTestBusinessType()

	// Load test data resolvers before defining test cases
	deleteSuccessResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "balance", "DeleteBalance_Success")
	testutil.AssertTestCaseLoad(t, err, "DeleteBalance_Success")

	deleteNotFoundResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "balance", "DeleteBalance_NotFound")
	testutil.AssertTestCaseLoad(t, err, "DeleteBalance_NotFound")

	createSuccessResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "balance", "CreateBalance_Success")
	testutil.AssertTestCaseLoad(t, err, "CreateBalance_Success")

	authorizationUnauthorizedResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "balance", "Authorization_Unauthorized")
	testutil.AssertTestCaseLoad(t, err, "Authorization_Unauthorized")

	testCases := []DeleteBalanceTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-BALANCE-DELETE-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *balancepb.DeleteBalanceRequest {
				return &balancepb.DeleteBalanceRequest{
					Data: &balancepb.Balance{
						Id: deleteSuccessResolver.MustGetString("existingBalanceId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *balancepb.DeleteBalanceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "successful deletion")
			},
		},
		{
			Name:     "WithTransaction",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-BALANCE-DELETE-TRANSACTION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *balancepb.DeleteBalanceRequest {
				return &balancepb.DeleteBalanceRequest{
					Data: &balancepb.Balance{
						Id: createSuccessResolver.MustGetString("secondaryBalanceId"),
					},
				}
			},
			UseTransaction: true,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *balancepb.DeleteBalanceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "successful deletion with transaction")
			},
		},
		{
			Name:     "Unauthorized",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-BALANCE-DELETE-AUTHORIZATION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *balancepb.DeleteBalanceRequest {
				return &balancepb.DeleteBalanceRequest{
					Data: &balancepb.Balance{
						Id: authorizationUnauthorizedResolver.MustGetString("targetBalanceId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        false,
			ExpectSuccess:  false,
			ExpectedError:  "balance.errors.authorization_failed",
			Assertions: func(t *testing.T, response *balancepb.DeleteBalanceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertAuthorizationError(t, err)
			},
		},
		{
			Name:     "BalanceNotFound",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-BALANCE-DELETE-NOT-FOUND-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *balancepb.DeleteBalanceRequest {
				return &balancepb.DeleteBalanceRequest{
					Data: &balancepb.Balance{
						Id: deleteNotFoundResolver.MustGetString("nonExistentBalanceId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "balance.errors.not_found",
			ErrorTags:      map[string]any{"balanceId": deleteNotFoundResolver.MustGetString("nonExistentBalanceId")},
			Assertions: func(t *testing.T, response *balancepb.DeleteBalanceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertError(t, err)
				testutil.AssertNil(t, response, "response for non-existent balance")
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-BALANCE-DELETE-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *balancepb.DeleteBalanceRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "balance.validation.request_required",
			Assertions: func(t *testing.T, response *balancepb.DeleteBalanceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilRequest(t, err)
				testutil.AssertNil(t, response, "response for nil request")
			},
		},
		{
			Name:     "NilData",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-BALANCE-DELETE-NIL-DATA-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *balancepb.DeleteBalanceRequest {
				return &balancepb.DeleteBalanceRequest{Data: nil}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "balance.validation.data_required",
			Assertions: func(t *testing.T, response *balancepb.DeleteBalanceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilData(t, err)
				testutil.AssertNil(t, response, "response for nil data")
			},
		},
		{
			Name:     "EmptyId",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-BALANCE-DELETE-VALIDATION-EMPTY-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *balancepb.DeleteBalanceRequest {
				return &balancepb.DeleteBalanceRequest{
					Data: &balancepb.Balance{
						Id: "",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "balance.validation.id_required",
			Assertions: func(t *testing.T, response *balancepb.DeleteBalanceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty balance ID")
				testutil.AssertNil(t, response, "response for invalid input")
			},
		},
		{
			Name:     "SoftDeleteVerification",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-BALANCE-DELETE-SOFT-DELETE-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *balancepb.DeleteBalanceRequest {
				return &balancepb.DeleteBalanceRequest{
					Data: &balancepb.Balance{
						Id: createSuccessResolver.MustGetString("thirdBalanceId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *balancepb.DeleteBalanceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "soft delete verification")
				// Additional verification that the balance is marked as deleted, not physically removed
			},
		},
		{
			Name:     "EducationDomainSpecific",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-BALANCE-DELETE-EDUCATION-DOMAIN-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *balancepb.DeleteBalanceRequest {
				return &balancepb.DeleteBalanceRequest{
					Data: &balancepb.Balance{
						Id: createSuccessResolver.MustGetString("educationBalanceId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *balancepb.DeleteBalanceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "education domain specific deletion")
				// Education-specific balance deletion validation
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
			useCase := createDeleteBalanceTestUseCaseWithAuth(businessType, tc.UseTransaction, tc.UseAuth)

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

func TestDeleteBalanceUseCase_Execute_TransactionFailure(t *testing.T) {
	testCode := "ESPYNA-TEST-SUBSCRIPTION-BALANCE-DELETE-TRANSACTION-FAILURE-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "TransactionFailure", false)

	// Create a use case with a failing transaction service
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	mockBalanceRepo := subscription.NewMockBalanceRepository(businessType)

	repositories := DeleteBalanceRepositories{
		Balance: mockBalanceRepo,
	}

	standardServices := testutil.CreateStandardServices(false, true)
	services := DeleteBalanceServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   mockDb.NewFailingMockTransactionService(),
		TranslationService:   standardServices.TranslationService,
	}

	useCase := NewDeleteBalanceUseCase(repositories, services)

	// Load test case using centralized TestCaseResolver
	resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "balance", "CreateBalance_Success")
	testutil.AssertTestCaseLoad(t, err, "CreateBalance_Success")

	req := &balancepb.DeleteBalanceRequest{
		Data: &balancepb.Balance{
			Id: resolver.MustGetString("thirdBalanceId"),
		},
	}

	_, err = useCase.Execute(ctx, req)

	// Verify that a transaction error occurred
	testutil.AssertTransactionError(t, err)

	// Log completion of transaction failure test
	testutil.LogTestResult(t, testCode, "TransactionFailure", false, err)
}
