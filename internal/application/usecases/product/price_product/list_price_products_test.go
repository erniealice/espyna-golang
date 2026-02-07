//go:build mock_db && mock_auth

// Package price_product provides table-driven tests for the price product listing use case.
//
// The tests cover various scenarios, including success, transaction handling,
// authorization, nil requests, validation errors, and empty list conditions.
// Each test case is defined in a table with a specific test code, request setup,
// and assertions.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user").
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education").
//
// Usage:
//   - Run all tests: go test -tags="mock_db,mock_auth" ./...
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestListPriceProductsUseCase_Execute_TableDriven
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-PRODUCT-PRICEPRODUCT-LIST-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-PRODUCT-PRICEPRODUCT-LIST-TRANSACTION-v1.0: WithTransaction
//   - ESPYNA-TEST-PRODUCT-PRICEPRODUCT-LIST-AUTHORIZATION-v1.0: Unauthorized
//   - ESPYNA-TEST-PRODUCT-PRICEPRODUCT-LIST-NIL-REQUEST-v1.0: NilRequest
//   - ESPYNA-TEST-PRODUCT-PRICEPRODUCT-LIST-EMPTY-v1.0: EmptyList
//
// Data Sources:
//   - Test cases: packages/copya/data_test/{businessType}/price_product.json
//   - Mock data: packages/copya/data/{businessType}/price-product.json
//   - Translations: packages/lyngua/translations/{languageCode}/{businessType}/price_product.json

package price_product

import (
	"context"
	"testing"

	copyatestutil "leapfor.xyz/copya/golang/testutil"
	"leapfor.xyz/espyna/internal/application/ports"
	"leapfor.xyz/espyna/internal/application/shared/testutil"
	mockAuth "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/auth/mock"
	mockDb "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock"
	"leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock/product"
	"leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/translation"
	priceproductpb "leapfor.xyz/esqyma/golang/v1/domain/product/price_product"
)

// Type alias for list price products test cases
type ListPriceProductsTestCase = testutil.GenericTestCase[*priceproductpb.ListPriceProductsRequest, *priceproductpb.ListPriceProductsResponse]

// listTestUseCaseWithAuth is a helper to create the ListPriceProductsUseCase with mock dependencies and auth control.
func listTestUseCaseWithAuth(businessType string, supportsTransaction bool, shouldAuthorize bool, repoOptions ...product.PriceProductRepositoryOption) *ListPriceProductsUseCase {
	mockRepo := product.NewMockPriceProductRepository(businessType, repoOptions...)

	repositories := ListPriceProductsRepositories{
		PriceProduct: mockRepo,
	}

	var authService ports.AuthorizationService
	if shouldAuthorize {
		authService = mockAuth.NewAllowAllAuth()
	} else {
		authService = mockAuth.NewDenyAllAuth()
	}

	services := ListPriceProductsServices{
		AuthorizationService: authService,
		TransactionService:   mockDb.NewMockTransactionService(supportsTransaction),
		TranslationService:   translation.NewLynguaTranslationService(),
	}

	return NewListPriceProductsUseCase(repositories, services)
}

func TestListPriceProductsUseCase_Execute_TableDriven(t *testing.T) {
	businessType := testutil.GetTestBusinessType()

	// Load test data resolvers before defining test cases
	listSuccessResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "price_product", "ListPriceProducts_Success")
	testutil.AssertTestCaseLoad(t, err, "ListPriceProducts_Success")

	testCases := []ListPriceProductsTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-PRODUCT-PRICEPRODUCT-LIST-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *priceproductpb.ListPriceProductsRequest {
				return &priceproductpb.ListPriceProductsRequest{}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *priceproductpb.ListPriceProductsResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")

				// Validate expected count from test data
				expectedCount := listSuccessResolver.MustGetInt("expectedPriceProductCount")
				testutil.AssertEqual(t, expectedCount, len(response.Data), "expected price product count")

				// Validate specific IDs if available
				if listSuccessResolver.HasKey("firstPriceProductId") {
					foundIds := make(map[string]bool)
					for _, priceProduct := range response.Data {
						foundIds[priceProduct.Id] = true
					}
					testutil.AssertTrue(t, foundIds[listSuccessResolver.MustGetString("firstPriceProductId")], "first price product found")
					testutil.AssertTrue(t, foundIds[listSuccessResolver.MustGetString("secondPriceProductId")], "second price product found")
					testutil.AssertTrue(t, foundIds[listSuccessResolver.MustGetString("thirdPriceProductId")], "third price product found")
				}
			},
		},
		{
			Name:     "WithTransaction",
			TestCode: "ESPYNA-TEST-PRODUCT-PRICEPRODUCT-LIST-TRANSACTION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *priceproductpb.ListPriceProductsRequest {
				return &priceproductpb.ListPriceProductsRequest{}
			},
			UseTransaction: true,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *priceproductpb.ListPriceProductsResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertTrue(t, len(response.Data) >= 0, "data should be present or empty")
			},
		},
		{
			Name:     "Unauthorized",
			TestCode: "ESPYNA-TEST-PRODUCT-PRICEPRODUCT-LIST-AUTHORIZATION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *priceproductpb.ListPriceProductsRequest {
				return &priceproductpb.ListPriceProductsRequest{}
			},
			UseTransaction: false,
			UseAuth:        false,
			ExpectSuccess:  false,
			ExpectedError:  "price_product.errors.authorization_failed",
			Assertions: func(t *testing.T, response *priceproductpb.ListPriceProductsResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertAuthorizationError(t, err)
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-PRODUCT-PRICEPRODUCT-LIST-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *priceproductpb.ListPriceProductsRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "price_product.validation.request_required",
			Assertions: func(t *testing.T, response *priceproductpb.ListPriceProductsResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilRequest(t, err)
			},
		},
		{
			Name:     "EmptyList",
			TestCode: "ESPYNA-TEST-PRODUCT-PRICEPRODUCT-LIST-EMPTY-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *priceproductpb.ListPriceProductsRequest {
				return &priceproductpb.ListPriceProductsRequest{}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *priceproductpb.ListPriceProductsResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success for empty list")
				testutil.AssertEqual(t, 0, len(response.Data), "empty list should have 0 items")
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

			// Special handling for EmptyList test case - use empty repository
			var useCase *ListPriceProductsUseCase
			if tc.Name == "EmptyList" {
				useCase = listTestUseCaseWithAuth(businessType, tc.UseTransaction, tc.UseAuth, product.WithoutPriceProductInitialData())
			} else {
				useCase = listTestUseCaseWithAuth(businessType, tc.UseTransaction, tc.UseAuth)
			}

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

// Individual test function for backward compatibility and specialized scenarios

func TestListPriceProductsUseCase_Execute_Success(t *testing.T) {
	testCode := "ESPYNA-TEST-PRODUCT-PRICEPRODUCT-LIST-SUCCESS-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "Success", true)

	// This test relies on the mock repository auto-loading the baseline data.
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := listTestUseCaseWithAuth(businessType, false, true)

	// Load test case using centralized TestCaseResolver
	resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "price_product", "ListPriceProducts_Success")
	testutil.AssertTestCaseLoad(t, err, "ListPriceProducts_Success")

	req := &priceproductpb.ListPriceProductsRequest{}

	res, err := useCase.Execute(ctx, req)
	testutil.AssertNoError(t, err)
	testutil.AssertNotNil(t, res, "response")
	testutil.AssertTrue(t, res.Success, "success")

	// Validate expected count from test data
	expectedCount := resolver.MustGetInt("expectedPriceProductCount")
	testutil.AssertEqual(t, expectedCount, len(res.Data), "expected price product count")

	// Validate specific IDs if available
	if resolver.HasKey("firstPriceProductId") {
		foundIds := make(map[string]bool)
		for _, priceProduct := range res.Data {
			foundIds[priceProduct.Id] = true
		}
		testutil.AssertTrue(t, foundIds[resolver.MustGetString("firstPriceProductId")], "first price product found")
		testutil.AssertTrue(t, foundIds[resolver.MustGetString("secondPriceProductId")], "second price product found")
		testutil.AssertTrue(t, foundIds[resolver.MustGetString("thirdPriceProductId")], "third price product found")
	}

	// Log test completion with result
	testutil.LogTestResult(t, testCode, "Success", true, nil)
}

func TestListPriceProductsUseCase_Execute_EmptyList(t *testing.T) {
	testCode := "ESPYNA-TEST-PRODUCT-PRICEPRODUCT-LIST-EMPTY-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "EmptyList", true)

	// Use the WithoutInitialData option to ensure the repository starts empty.
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := listTestUseCaseWithAuth(businessType, false, true, product.WithoutPriceProductInitialData())

	req := &priceproductpb.ListPriceProductsRequest{}

	res, err := useCase.Execute(ctx, req)
	testutil.AssertNoError(t, err)
	testutil.AssertNotNil(t, res, "response")
	testutil.AssertTrue(t, res.Success, "success for empty list")
	testutil.AssertEqual(t, 0, len(res.Data), "empty list should have 0 items")

	// Log test completion with result
	testutil.LogTestResult(t, testCode, "EmptyList", true, nil)
}
