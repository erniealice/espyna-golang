//go:build mock_db && mock_auth

// Package price_product provides table-driven tests for the price product updating use case.
//
// The tests cover various scenarios, including success, transaction handling,
// authorization, nil requests, validation errors, and not found conditions.
// Each test case is defined in a table with a specific test code, request setup,
// and assertions.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user").
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education").
//
// Usage:
//   - Run all tests: go test -tags="mock_db,mock_auth" ./...
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestUpdatePriceProductUseCase_Execute_TableDriven
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-PRODUCT-PRICEPRODUCT-UPDATE-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-PRODUCT-PRICEPRODUCT-UPDATE-TRANSACTION-v1.0: WithTransaction
//   - ESPYNA-TEST-PRODUCT-PRICEPRODUCT-UPDATE-AUTHORIZATION-v1.0: Unauthorized
//   - ESPYNA-TEST-PRODUCT-PRICEPRODUCT-UPDATE-NIL-REQUEST-v1.0: NilRequest
//   - ESPYNA-TEST-PRODUCT-PRICEPRODUCT-UPDATE-NIL-DATA-v1.0: NilData
//   - ESPYNA-TEST-PRODUCT-PRICEPRODUCT-UPDATE-NOT-FOUND-v1.0: NotFound
//   - ESPYNA-TEST-PRODUCT-PRICEPRODUCT-UPDATE-TRANSACTION-FAILURE-v1.0: TransactionFailure
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
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/testutil"
	mockAuth "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/auth/mock"
	mockDb "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock"
	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock/product"
	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/translation"
	priceproductpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/price_product"
)

// Type alias for update price product test cases
type UpdatePriceProductTestCase = testutil.GenericTestCase[*priceproductpb.UpdatePriceProductRequest, *priceproductpb.UpdatePriceProductResponse]

// updateTestUseCaseWithAuth is a helper to create the UpdatePriceProductUseCase with mock dependencies and auth control.
func updateTestUseCaseWithAuth(businessType string, supportsTransaction bool, shouldAuthorize bool) *UpdatePriceProductUseCase {
	// Use the default repository which will be pre-populated from copya data
	mockRepo := product.NewMockPriceProductRepository(businessType)

	// This use case might need a Product repository as well for validation.
	// We will add it here proactively based on the Create use case.
	mockProductRepo := product.NewMockProductRepository(businessType)

	repositories := UpdatePriceProductRepositories{
		PriceProduct: mockRepo,
		Product:      mockProductRepo,
	}

	var authService ports.AuthorizationService
	if shouldAuthorize {
		authService = mockAuth.NewAllowAllAuth()
	} else {
		authService = mockAuth.NewDenyAllAuth()
	}

	services := UpdatePriceProductServices{
		AuthorizationService: authService,
		TransactionService:   mockDb.NewMockTransactionService(supportsTransaction),
		TranslationService:   translation.NewLynguaTranslationService(),
	}

	return NewUpdatePriceProductUseCase(repositories, services)
}

func TestUpdatePriceProductUseCase_Execute_TableDriven(t *testing.T) {
	businessType := testutil.GetTestBusinessType()

	// Load test data resolvers before defining test cases
	updateSuccessResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "price_product", "UpdatePriceProduct_Success")
	testutil.AssertTestCaseLoad(t, err, "UpdatePriceProduct_Success")

	authorizationUnauthorizedResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "price_product", "Authorization_Unauthorized")
	testutil.AssertTestCaseLoad(t, err, "Authorization_Unauthorized")

	testCases := []UpdatePriceProductTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-PRODUCT-PRICEPRODUCT-UPDATE-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *priceproductpb.UpdatePriceProductRequest {
				return &priceproductpb.UpdatePriceProductRequest{
					Data: &priceproductpb.PriceProduct{
						Id:        updateSuccessResolver.MustGetString("existingPriceProductId"),
						ProductId: updateSuccessResolver.MustGetString("existingProductId"),
						Name:      updateSuccessResolver.MustGetString("updatedPriceProductName"),
						Amount:    int64(updateSuccessResolver.MustGetInt("updatedPriceProductAmount")),
						Currency:  updateSuccessResolver.MustGetString("updatedPriceProductCurrency"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *priceproductpb.UpdatePriceProductResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertEqual(t, 1, len(response.Data), "response data length")
				updated := response.Data[0]
				testutil.AssertStringEqual(t, updateSuccessResolver.MustGetString("updatedPriceProductName"), updated.Name, "updated price product name")
				testutil.AssertEqual(t, int64(updateSuccessResolver.MustGetInt("updatedPriceProductAmount")), updated.Amount, "updated price product amount")
				testutil.AssertStringEqual(t, updateSuccessResolver.MustGetString("updatedPriceProductCurrency"), updated.Currency, "updated price product currency")
			},
		},
		{
			Name:     "WithTransaction",
			TestCode: "ESPYNA-TEST-PRODUCT-PRICEPRODUCT-UPDATE-TRANSACTION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *priceproductpb.UpdatePriceProductRequest {
				return &priceproductpb.UpdatePriceProductRequest{
					Data: &priceproductpb.PriceProduct{
						Id:        updateSuccessResolver.MustGetString("existingPriceProductId"),
						ProductId: updateSuccessResolver.MustGetString("existingProductId"),
						Name:      updateSuccessResolver.MustGetString("updatedPriceProductName"),
						Amount:    int64(updateSuccessResolver.MustGetInt("updatedPriceProductAmount")),
						Currency:  updateSuccessResolver.MustGetString("updatedPriceProductCurrency"),
					},
				}
			},
			UseTransaction: true,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *priceproductpb.UpdatePriceProductResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				updated := response.Data[0]
				testutil.AssertStringEqual(t, updateSuccessResolver.MustGetString("updatedPriceProductName"), updated.Name, "updated price product name")
			},
		},
		{
			Name:     "Unauthorized",
			TestCode: "ESPYNA-TEST-PRODUCT-PRICEPRODUCT-UPDATE-AUTHORIZATION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *priceproductpb.UpdatePriceProductRequest {
				return &priceproductpb.UpdatePriceProductRequest{
					Data: &priceproductpb.PriceProduct{
						Id:        authorizationUnauthorizedResolver.MustGetString("targetPriceProductId"),
						ProductId: authorizationUnauthorizedResolver.MustGetString("unauthorizedProductId"),
						Name:      authorizationUnauthorizedResolver.MustGetString("unauthorizedPriceProductName"),
						Amount:    int64(authorizationUnauthorizedResolver.MustGetInt("unauthorizedPriceProductAmount")),
						Currency:  authorizationUnauthorizedResolver.MustGetString("unauthorizedPriceProductCurrency"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        false,
			ExpectSuccess:  false,
			ExpectedError:  "price_product.errors.authorization_failed",
			Assertions: func(t *testing.T, response *priceproductpb.UpdatePriceProductResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertAuthorizationError(t, err)
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-PRODUCT-PRICEPRODUCT-UPDATE-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *priceproductpb.UpdatePriceProductRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "price_product.validation.request_required",
			Assertions: func(t *testing.T, response *priceproductpb.UpdatePriceProductResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilRequest(t, err)
			},
		},
		{
			Name:     "NilData",
			TestCode: "ESPYNA-TEST-PRODUCT-PRICEPRODUCT-UPDATE-NIL-DATA-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *priceproductpb.UpdatePriceProductRequest {
				return &priceproductpb.UpdatePriceProductRequest{
					Data: nil,
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "price_product.validation.data_required",
			Assertions: func(t *testing.T, response *priceproductpb.UpdatePriceProductResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "nil data")
			},
		},
		{
			Name:     "NotFound",
			TestCode: "ESPYNA-TEST-PRODUCT-PRICEPRODUCT-UPDATE-NOT-FOUND-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *priceproductpb.UpdatePriceProductRequest {
				return &priceproductpb.UpdatePriceProductRequest{
					Data: &priceproductpb.PriceProduct{
						Id:        "price-non-existent",
						ProductId: "subject-math",
						Name:      "Updated Test Tuition",
						Amount:    600,
						Currency:  "USD",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			Assertions: func(t *testing.T, response *priceproductpb.UpdatePriceProductResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertError(t, err)
				// Error should indicate entity not found - checking exact message in development only
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
			useCase := updateTestUseCaseWithAuth(businessType, tc.UseTransaction, tc.UseAuth)

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

func TestUpdatePriceProductUseCase_Execute_Success(t *testing.T) {
	testCode := "ESPYNA-TEST-PRODUCT-PRICEPRODUCT-UPDATE-SUCCESS-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "Success", true)

	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := updateTestUseCaseWithAuth(businessType, false, true)

	// Load test case using centralized TestCaseResolver
	resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "price_product", "UpdatePriceProduct_Success")
	testutil.AssertTestCaseLoad(t, err, "UpdatePriceProduct_Success")

	req := &priceproductpb.UpdatePriceProductRequest{
		Data: &priceproductpb.PriceProduct{
			Id:        resolver.MustGetString("existingPriceProductId"),
			ProductId: resolver.MustGetString("existingProductId"),
			Name:      resolver.MustGetString("updatedPriceProductName"),
			Amount:    int64(resolver.MustGetInt("updatedPriceProductAmount")),
			Currency:  resolver.MustGetString("updatedPriceProductCurrency"),
		},
	}

	res, err := useCase.Execute(ctx, req)
	testutil.AssertNoError(t, err)
	testutil.AssertNotNil(t, res, "response")
	testutil.AssertTrue(t, res.Success, "success")
	testutil.AssertEqual(t, 1, len(res.Data), "response data length")

	updated := res.Data[0]
	testutil.AssertStringEqual(t, resolver.MustGetString("updatedPriceProductName"), updated.Name, "updated price product name")
	testutil.AssertEqual(t, int64(resolver.MustGetInt("updatedPriceProductAmount")), updated.Amount, "updated price product amount")
	testutil.AssertStringEqual(t, resolver.MustGetString("updatedPriceProductCurrency"), updated.Currency, "updated price product currency")

	// Log test completion with result
	testutil.LogTestResult(t, testCode, "Success", true, nil)
}
