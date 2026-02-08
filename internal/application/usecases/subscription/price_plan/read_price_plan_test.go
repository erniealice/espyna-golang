//go:build mock_db && mock_auth

// Package price_plan provides table-driven tests for the price plan read use case.
//
// The tests cover various scenarios, including success, transaction handling,
// nil requests, validation errors, and not-found cases.
// Each test case is defined in a table with a specific test code, request setup,
// and assertions.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user").
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education").
//
// Usage:
//   - Run all tests: go test -tags="mock_db,mock_auth" ./...
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestReadPricePlanUseCase_Execute_TableDriven
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-SUBSCRIPTION-PRICEPLAN-READ-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-SUBSCRIPTION-PRICEPLAN-READ-TRANSACTION-v1.0: WithTransaction
//   - ESPYNA-TEST-SUBSCRIPTION-PRICEPLAN-READ-UNAUTHORIZED-v1.0: Unauthorized
//   - ESPYNA-TEST-SUBSCRIPTION-PRICEPLAN-READ-NIL-REQUEST-v1.0: NilRequest
//   - ESPYNA-TEST-SUBSCRIPTION-PRICEPLAN-READ-NIL-DATA-v1.0: NilData
//   - ESPYNA-TEST-SUBSCRIPTION-PRICEPLAN-READ-VALIDATION-EMPTY-ID-v1.0: EmptyId
//   - ESPYNA-TEST-SUBSCRIPTION-PRICEPLAN-READ-VALIDATION-INVALID-ID-v1.0: InvalidId
//   - ESPYNA-TEST-SUBSCRIPTION-PRICEPLAN-READ-NOT-FOUND-v1.0: NotFound
//   - ESPYNA-TEST-SUBSCRIPTION-PRICEPLAN-READ-VALIDATION-MINIMAL-ID-v1.0: ValidMinimalId
//   - ESPYNA-TEST-SUBSCRIPTION-PRICEPLAN-READ-INTEGRATION-v1.0: RealisticDomainPricePlan
//   - ESPYNA-TEST-SUBSCRIPTION-PRICEPLAN-READ-STRUCTURE-VALIDATION-v1.0: PricePlanStructureValidation
//   - ESPYNA-TEST-SUBSCRIPTION-PRICEPLAN-READ-TRANSACTION-FAILURE-v1.0: TransactionFailure
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
	"github.com/erniealice/espyna-golang/internal/application/shared/testutil"
	mockDb "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock"
	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock/subscription"
	priceplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_plan"
)

// Type alias for read price plan test cases
type ReadPricePlanTestCase = testutil.GenericTestCase[*priceplanpb.ReadPricePlanRequest, *priceplanpb.ReadPricePlanResponse]

func createReadTestUseCaseWithAuth(businessType string, supportsTransaction bool, shouldAuthorize bool) *ReadPricePlanUseCase {
	mockPricePlanRepo := subscription.NewMockPricePlanRepository(businessType)

	repositories := ReadPricePlanRepositories{
		PricePlan: mockPricePlanRepo,
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, shouldAuthorize)
	services := ReadPricePlanServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
	}

	return NewReadPricePlanUseCase(repositories, services)
}

func TestReadPricePlanUseCase_Execute_TableDriven(t *testing.T) {
	businessType := testutil.GetTestBusinessType()

	// Load test data resolvers before defining test cases
	commonDataResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "price_plan", "PricePlan_CommonData")
	testutil.AssertTestCaseLoad(t, err, "PricePlan_CommonData")

	readSuccessResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "price_plan", "ReadPricePlan_Success")
	testutil.AssertTestCaseLoad(t, err, "ReadPricePlan_Success")

	testCases := []ReadPricePlanTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-PRICEPLAN-READ-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *priceplanpb.ReadPricePlanRequest {
				return &priceplanpb.ReadPricePlanRequest{
					Data: &priceplanpb.PricePlan{
						Id: readSuccessResolver.MustGetString("existingPricePlanId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *priceplanpb.ReadPricePlanResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertEqual(t, 1, len(response.Data), "response data length")
				pricePlan := response.Data[0]
				testutil.AssertNonEmptyString(t, pricePlan.Name, "price plan name")
				testutil.AssertNonEmptyString(t, pricePlan.PlanId, "plan ID")
				testutil.AssertFieldSet(t, pricePlan.DateCreated, "DateCreated")
				testutil.AssertFieldSet(t, pricePlan.DateCreatedString, "DateCreatedString")
			},
		},
		{
			Name:     "WithTransaction",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-PRICEPLAN-READ-TRANSACTION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *priceplanpb.ReadPricePlanRequest {
				return &priceplanpb.ReadPricePlanRequest{
					Data: &priceplanpb.PricePlan{
						Id: commonDataResolver.MustGetString("secondaryPricePlanId"),
					},
				}
			},
			UseTransaction: true,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *priceplanpb.ReadPricePlanResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				pricePlan := response.Data[0]
				testutil.AssertStringEqual(t, commonDataResolver.MustGetString("secondaryPricePlanId"), pricePlan.Id, "price plan ID")
			},
		},
		{
			Name:     "Unauthorized",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-PRICEPLAN-READ-UNAUTHORIZED-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *priceplanpb.ReadPricePlanRequest {
				return &priceplanpb.ReadPricePlanRequest{
					Data: &priceplanpb.PricePlan{
						Id: commonDataResolver.MustGetString("targetPricePlanId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        false,
			ExpectSuccess:  false,
			ExpectedError:  "auth.errors.unauthorized",
			Assertions: func(t *testing.T, response *priceplanpb.ReadPricePlanResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertError(t, err)
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-PRICEPLAN-READ-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *priceplanpb.ReadPricePlanRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "price_plan.validation.request_required",
			Assertions: func(t *testing.T, response *priceplanpb.ReadPricePlanResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilRequest(t, err)
			},
		},
		{
			Name:     "NilData",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-PRICEPLAN-READ-NIL-DATA-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *priceplanpb.ReadPricePlanRequest {
				return &priceplanpb.ReadPricePlanRequest{Data: nil}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "price_plan.validation.request_required",
			Assertions: func(t *testing.T, response *priceplanpb.ReadPricePlanResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilData(t, err)
			},
		},
		{
			Name:     "EmptyId",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-PRICEPLAN-READ-VALIDATION-EMPTY-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *priceplanpb.ReadPricePlanRequest {
				return &priceplanpb.ReadPricePlanRequest{
					Data: &priceplanpb.PricePlan{Id: ""},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "price_plan.validation.id_required",
			Assertions: func(t *testing.T, response *priceplanpb.ReadPricePlanResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty ID")
			},
		},
		{
			Name:     "InvalidId",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-PRICEPLAN-READ-VALIDATION-INVALID-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *priceplanpb.ReadPricePlanRequest {
				return &priceplanpb.ReadPricePlanRequest{
					Data: &priceplanpb.PricePlan{Id: "a"},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "price_plan.validation.id_invalid",
			Assertions: func(t *testing.T, response *priceplanpb.ReadPricePlanResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "invalid ID")
			},
		},
		{
			Name:     "NotFound",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-PRICEPLAN-READ-NOT-FOUND-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *priceplanpb.ReadPricePlanRequest {
				return &priceplanpb.ReadPricePlanRequest{
					Data: &priceplanpb.PricePlan{Id: commonDataResolver.MustGetString("nonExistentId")},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "price_plan.errors.not_found",
			ErrorTags:      map[string]any{"pricePlanId": commonDataResolver.MustGetString("nonExistentId")},
			Assertions: func(t *testing.T, response *priceplanpb.ReadPricePlanResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertError(t, err)
			},
		},
		{
			Name:     "ValidMinimalId",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-PRICEPLAN-READ-VALIDATION-MINIMAL-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *priceplanpb.ReadPricePlanRequest {
				return &priceplanpb.ReadPricePlanRequest{
					Data: &priceplanpb.PricePlan{Id: commonDataResolver.MustGetString("minimalValidId")},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "price_plan.errors.not_found",
			ErrorTags:      map[string]any{"pricePlanId": commonDataResolver.MustGetString("minimalValidId")},
			Assertions: func(t *testing.T, response *priceplanpb.ReadPricePlanResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertError(t, err)
			},
		},
		{
			Name:     "RealisticDomainPricePlan",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-PRICEPLAN-READ-INTEGRATION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *priceplanpb.ReadPricePlanRequest {
				return &priceplanpb.ReadPricePlanRequest{
					Data: &priceplanpb.PricePlan{
						Id: commonDataResolver.MustGetString("thirdPricePlanId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *priceplanpb.ReadPricePlanResponse, err error, useCase interface{}, ctx context.Context) {
				pricePlan := response.Data[0]
				testutil.AssertNonEmptyString(t, pricePlan.Name, "price plan name")
				testutil.AssertFieldSet(t, pricePlan.Description, "description")
				testutil.AssertNonEmptyString(t, pricePlan.PlanId, "plan ID")
				testutil.AssertNonEmptyString(t, pricePlan.Currency, "currency")
				// Note: BillingPeriod field not available in current protobuf definition
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
			useCase := createReadTestUseCaseWithAuth(businessType, tc.UseTransaction, tc.UseAuth)

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

func TestReadPricePlanUseCase_Execute_PricePlanStructureValidation(t *testing.T) {
	testCode := "ESPYNA-TEST-SUBSCRIPTION-PRICEPLAN-READ-STRUCTURE-VALIDATION-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "PricePlanStructureValidation", true)

	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createReadTestUseCaseWithAuth(businessType, false, true)

	// Note: Using hardcoded price plan IDs from mock data instead of resolver

	// Test with multiple real price plan IDs from mock data
	pricePlanIds := []string{
		"price-plan-student-annual-001",
		"price-plan-student-semester-002",
		"price-plan-student-quarterly-003",
	}

	for _, pricePlanId := range pricePlanIds {
		req := &priceplanpb.ReadPricePlanRequest{
			Data: &priceplanpb.PricePlan{
				Id: pricePlanId,
			},
		}

		response, err := useCase.Execute(ctx, req)

		testutil.AssertNoError(t, err)

		pricePlan := response.Data[0]

		// Validate price plan structure
		testutil.AssertStringEqual(t, pricePlanId, pricePlan.Id, "price plan ID")

		testutil.AssertNonEmptyString(t, pricePlan.Name, "price plan name")

		testutil.AssertTrue(t, pricePlan.Active, "price plan active status")

		testutil.AssertNonEmptyString(t, pricePlan.PlanId, "plan ID")

		testutil.AssertNonEmptyString(t, pricePlan.Currency, "currency")

		// Note: BillingPeriod field not available in current protobuf definition

		// Audit fields
		testutil.AssertFieldSet(t, pricePlan.DateCreated, "DateCreated")

		testutil.AssertFieldSet(t, pricePlan.DateCreatedString, "DateCreatedString")
	}

	// Log completion of structure validation test
	testutil.LogTestResult(t, testCode, "PricePlanStructureValidation", true, nil)
}

func TestReadPricePlanUseCase_Execute_TransactionFailure(t *testing.T) {
	testCode := "ESPYNA-TEST-SUBSCRIPTION-PRICEPLAN-READ-TRANSACTION-FAILURE-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "TransactionFailure", false)

	// Create a use case with a failing transaction service
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	mockPricePlanRepo := subscription.NewMockPricePlanRepository(businessType)

	repositories := ReadPricePlanRepositories{
		PricePlan: mockPricePlanRepo,
	}

	standardServices := testutil.CreateStandardServices(false, true)
	// Override with failing transaction service
	standardServices.TransactionService = mockDb.NewFailingMockTransactionService()
	services := ReadPricePlanServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
	}

	useCase := NewReadPricePlanUseCase(repositories, services)

	// Load test case using centralized TestCaseResolver
	resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "price_plan", "PricePlan_CommonData")
	testutil.AssertTestCaseLoad(t, err, "PricePlan_CommonData")

	req := &priceplanpb.ReadPricePlanRequest{
		Data: &priceplanpb.PricePlan{
			Id: resolver.MustGetString("nonExistentId"),
		},
	}

	// For read operations, transaction failure should not affect the operation
	// since read operations typically don't use transactions
	response, err := useCase.Execute(ctx, req)

	// This should either work (no transaction used) or fail gracefully
	if err != nil {
		// If it fails, verify it's due to the price plan not existing, not transaction failure
		testutil.AssertTranslatedErrorWithTags(t, err, "price_plan.errors.not_found",
			map[string]any{"pricePlanId": resolver.MustGetString("nonExistentId")}, useCase.services.TranslationService, ctx)
	} else {
		// If it succeeds, verify we get a proper response
		testutil.AssertNotNil(t, response, "response")
	}

	// Log completion of transaction failure test
	testutil.LogTestResult(t, testCode, "TransactionFailure", err == nil, err)
}
