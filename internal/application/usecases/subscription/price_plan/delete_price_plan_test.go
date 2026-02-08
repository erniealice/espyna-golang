//go:build mock_db && mock_auth

// Package price_plan provides table-driven tests for the price plan deletion use case.
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
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestDeletePricePlanUseCase_Execute_TableDriven
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-SUBSCRIPTION-PRICEPLAN-DELETE-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-SUBSCRIPTION-PRICEPLAN-DELETE-NOT-FOUND-v1.0: NonExistentPricePlan
//   - ESPYNA-TEST-SUBSCRIPTION-PRICEPLAN-DELETE-VALIDATION-EMPTY-ID-v1.0: EmptyPricePlanId
//   - ESPYNA-TEST-SUBSCRIPTION-PRICEPLAN-DELETE-NIL-REQUEST-v1.0: NilRequest
//   - ESPYNA-TEST-SUBSCRIPTION-PRICEPLAN-DELETE-NIL-DATA-v1.0: NilData
//   - ESPYNA-TEST-SUBSCRIPTION-PRICEPLAN-DELETE-TRANSACTION-SUCCESS-v1.0: WithTransactionSuccess
//   - ESPYNA-TEST-SUBSCRIPTION-PRICEPLAN-DELETE-AUTHORIZATION-DENIED-v1.0: AuthorizationDenied
//   - ESPYNA-TEST-SUBSCRIPTION-PRICEPLAN-DELETE-TRANSACTION-FAILURE-v1.0: WithTransactionFailure
//   - ESPYNA-TEST-SUBSCRIPTION-PRICEPLAN-DELETE-INTEGRATION-v1.0: MultipleValidPricePlans
//   - ESPYNA-TEST-SUBSCRIPTION-PRICEPLAN-DELETE-BUSINESS-LOGIC-v1.0: BusinessLogicValidation
//
// Data Sources:
//   - Test cases: packages/copya/data_test/{businessType}/price_plan.json
//   - Mock data: packages/copya/data/{businessType}/price_plan.json
//   - Translations: packages/lyngua/translations/{languageCode}/{businessType}/price_plan.json
package price_plan

import (
	"context"
	"fmt"
	"strings"
	"testing"

	copyatestutil "leapfor.xyz/copya/golang/testutil"
	"github.com/erniealice/espyna-golang/internal/application/shared/testutil"
	mockDb "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock"
	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock/subscription"
	priceplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_plan"
)

// Type alias for delete price plan test cases
type DeletePricePlanTestCase = testutil.GenericTestCase[*priceplanpb.DeletePricePlanRequest, *priceplanpb.DeletePricePlanResponse]

func createDeleteTestUseCaseWithAuth(businessType string, supportsTransaction bool, shouldAuthorize bool) *DeletePricePlanUseCase {
	mockPricePlanRepo := subscription.NewMockPricePlanRepository(businessType)

	repositories := DeletePricePlanRepositories{
		PricePlan: mockPricePlanRepo,
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, shouldAuthorize)
	services := DeletePricePlanServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
	}

	return NewDeletePricePlanUseCase(repositories, services)
}

func createDeleteTestUseCaseWithFailingTransaction(businessType string, shouldAuthorize bool) *DeletePricePlanUseCase {
	mockPricePlanRepo := subscription.NewMockPricePlanRepository(businessType)

	repositories := DeletePricePlanRepositories{
		PricePlan: mockPricePlanRepo,
	}

	standardServices := testutil.CreateStandardServices(false, shouldAuthorize)
	// Override with failing transaction service
	standardServices.TransactionService = mockDb.NewFailingMockTransactionService()
	services := DeletePricePlanServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
	}

	return NewDeletePricePlanUseCase(repositories, services)
}

func TestDeletePricePlanUseCase_Execute_TableDriven(t *testing.T) {
	businessType := testutil.GetTestBusinessType()

	// Load test data resolvers before defining test cases
	commonDataResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "price_plan", "CreatePricePlan_Success")
	testutil.AssertTestCaseLoad(t, err, "CreatePricePlan_Success")

	deleteSuccessResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "price_plan", "DeletePricePlan_Success")
	testutil.AssertTestCaseLoad(t, err, "DeletePricePlan_Success")

	testCases := []DeletePricePlanTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-PRICEPLAN-DELETE-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *priceplanpb.DeletePricePlanRequest {
				return &priceplanpb.DeletePricePlanRequest{
					Data: &priceplanpb.PricePlan{
						Id: deleteSuccessResolver.MustGetString("existingPricePlanId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *priceplanpb.DeletePricePlanResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "successful deletion")
			},
		},
		{
			Name:     "NonExistentPricePlan",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-PRICEPLAN-DELETE-NOT-FOUND-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *priceplanpb.DeletePricePlanRequest {
				return &priceplanpb.DeletePricePlanRequest{
					Data: &priceplanpb.PricePlan{
						Id: commonDataResolver.MustGetString("nonExistentId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "price_plan.errors.not_found",
			ErrorTags:      map[string]any{"pricePlanId": commonDataResolver.MustGetString("nonExistentId")},
			Assertions: func(t *testing.T, response *priceplanpb.DeletePricePlanResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertError(t, err)
				testutil.AssertNil(t, response, "response for non-existent price plan")
			},
		},
		{
			Name:     "EmptyPricePlanId",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-PRICEPLAN-DELETE-VALIDATION-EMPTY-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *priceplanpb.DeletePricePlanRequest {
				return &priceplanpb.DeletePricePlanRequest{
					Data: &priceplanpb.PricePlan{
						Id: "",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "price_plan.validation.id_required",
			Assertions: func(t *testing.T, response *priceplanpb.DeletePricePlanResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty price plan ID")
				testutil.AssertNil(t, response, "response for invalid input")
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-PRICEPLAN-DELETE-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *priceplanpb.DeletePricePlanRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "price_plan.validation.request_required",
			Assertions: func(t *testing.T, response *priceplanpb.DeletePricePlanResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilRequest(t, err)
				testutil.AssertNil(t, response, "response for nil request")
			},
		},
		{
			Name:     "NilData",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-PRICEPLAN-DELETE-NIL-DATA-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *priceplanpb.DeletePricePlanRequest {
				return &priceplanpb.DeletePricePlanRequest{Data: nil}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "price_plan.validation.data_required",
			Assertions: func(t *testing.T, response *priceplanpb.DeletePricePlanResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilData(t, err)
				testutil.AssertNil(t, response, "response for nil data")
			},
		},
		{
			Name:     "WithTransactionSuccess",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-PRICEPLAN-DELETE-TRANSACTION-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *priceplanpb.DeletePricePlanRequest {
				return &priceplanpb.DeletePricePlanRequest{
					Data: &priceplanpb.PricePlan{
						Id: commonDataResolver.MustGetString("secondaryPricePlanId"),
					},
				}
			},
			UseTransaction: true,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *priceplanpb.DeletePricePlanResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "successful deletion with transaction")
			},
		},
		{
			Name:     "AuthorizationDenied",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-PRICEPLAN-DELETE-AUTHORIZATION-DENIED-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *priceplanpb.DeletePricePlanRequest {
				return &priceplanpb.DeletePricePlanRequest{
					Data: &priceplanpb.PricePlan{
						Id: commonDataResolver.MustGetString("businessRulesPricePlanId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        false,
			ExpectSuccess:  false,
			ExpectedError:  "price_plan.errors.authorization_failed",
			Assertions: func(t *testing.T, response *priceplanpb.DeletePricePlanResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertAuthorizationError(t, err)
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
			useCase := createDeleteTestUseCaseWithAuth(businessType, tc.UseTransaction, tc.UseAuth)

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

func TestDeletePricePlanUseCase_Execute_WithTransaction_Failure(t *testing.T) {
	testCode := "ESPYNA-TEST-SUBSCRIPTION-PRICEPLAN-DELETE-TRANSACTION-FAILURE-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "WithTransactionFailure", false)

	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createDeleteTestUseCaseWithFailingTransaction(businessType, true)

	// Load test case using centralized TestCaseResolver
	resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "price_plan", "CreatePricePlan_Success")
	testutil.AssertTestCaseLoad(t, err, "CreatePricePlan_Success")

	req := &priceplanpb.DeletePricePlanRequest{
		Data: &priceplanpb.PricePlan{
			Id: resolver.MustGetString("thirdPricePlanId"),
		},
	}

	_, err2 := useCase.Execute(ctx, req)

	testutil.AssertTransactionError(t, err2)

	expectedError := "Transaction execution failed: transaction error [TRANSACTION_GENERAL] during run_in_transaction: mock run in transaction failed"
	testutil.AssertStringEqual(t, expectedError, err2.Error(), "error message")

	// Log test completion with result
	testutil.LogTestResult(t, testCode, "WithTransactionFailure", false, err2)
}

func TestDeletePricePlanUseCase_Execute_MultipleValidPricePlans(t *testing.T) {
	testCode := "ESPYNA-TEST-SUBSCRIPTION-PRICEPLAN-DELETE-INTEGRATION-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "MultipleValidPricePlans", true)

	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()

	// Load test case using centralized TestCaseResolver
	resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "price_plan", "CreatePricePlan_Success")
	testutil.AssertTestCaseLoad(t, err, "CreatePricePlan_Success")

	pricePlanIds := []string{
		resolver.MustGetString("primaryPricePlanId"),
		resolver.MustGetString("secondaryPricePlanId"),
		resolver.MustGetString("thirdPricePlanId"),
	}

	// Create test cases dynamically based on available price plan IDs
	testCases := make([]struct {
		name        string
		pricePlanId string
		expectError bool
	}, len(pricePlanIds))

	for i, pricePlanId := range pricePlanIds {
		testCases[i] = struct {
			name        string
			pricePlanId string
			expectError bool
		}{
			name:        fmt.Sprintf("Delete price plan %d (%s)", i+1, pricePlanId),
			pricePlanId: pricePlanId,
			expectError: false,
		}
	}

	useCase := createDeleteTestUseCaseWithAuth(businessType, false, true)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			req := &priceplanpb.DeletePricePlanRequest{
				Data: &priceplanpb.PricePlan{
					Id: tc.pricePlanId,
				},
			}

			response, err := useCase.Execute(ctx, req)

			if tc.expectError {
				testutil.AssertError(t, err)
			} else {
				testutil.AssertNoError(t, err)

				testutil.AssertNotNil(t, response, "response")

				testutil.AssertTrue(t, response.Success, "successful deletion")
			}
		})
	}

	// Log test completion with result
	testutil.LogTestResult(t, testCode, "MultipleValidPricePlans", true, nil)
}

func TestDeletePricePlanUseCase_Execute_BusinessLogicValidation(t *testing.T) {
	testCode := "ESPYNA-TEST-SUBSCRIPTION-PRICEPLAN-DELETE-BUSINESS-LOGIC-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "BusinessLogicValidation", true)

	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createDeleteTestUseCaseWithAuth(businessType, false, true)

	// Load test case using centralized TestCaseResolver
	resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "price_plan", "CreatePricePlan_Success")
	testutil.AssertTestCaseLoad(t, err, "CreatePricePlan_Success")

	tests := []struct {
		name          string
		pricePlanData *priceplanpb.PricePlan
		expectError   bool
		expectedError string
	}{
		{
			name: "Valid price plan ID",
			pricePlanData: &priceplanpb.PricePlan{
				Id: resolver.MustGetString("primaryPricePlanId"),
			},
			expectError: false,
		},
		{
			name: "Invalid price plan ID format",
			pricePlanData: &priceplanpb.PricePlan{
				Id: "invalid-id-format",
			},
			expectError: true,
		},
		{
			name: "Extremely long price plan ID",
			pricePlanData: &priceplanpb.PricePlan{
				Id: fmt.Sprintf("price-plan-%s", strings.Repeat("A", 300)), // Create long string for validation test
			},
			expectError: true,
		},
		{
			name: "Short price plan ID",
			pricePlanData: &priceplanpb.PricePlan{
				Id: "ab", // Less than 3 characters
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			req := &priceplanpb.DeletePricePlanRequest{
				Data: tt.pricePlanData,
			}

			response, err := useCase.Execute(ctx, req)

			if tt.expectError {
				testutil.AssertError(t, err)
				if tt.expectedError != "" {
					testutil.AssertStringEqual(t, tt.expectedError, err.Error(), "error message")
				}
				testutil.AssertNil(t, response, "response")
			} else {
				testutil.AssertNoError(t, err)
				testutil.AssertNotNil(t, response, "response")
			}
		})
	}

	// Log test completion with result
	testutil.LogTestResult(t, testCode, "BusinessLogicValidation", true, nil)
}
