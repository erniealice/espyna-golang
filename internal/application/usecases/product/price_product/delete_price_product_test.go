//go:build mock_db && mock_auth

// Package price_product provides table-driven tests for the price product deletion use case.
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
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestDeletePriceProductUseCase_Execute_TableDriven
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-PRODUCT-PRICEPRODUCT-DELETE-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-PRODUCT-PRICEPRODUCT-DELETE-TRANSACTION-v1.0: WithTransaction
//   - ESPYNA-TEST-PRODUCT-PRICEPRODUCT-DELETE-AUTHORIZATION-v1.0: Unauthorized
//   - ESPYNA-TEST-PRODUCT-PRICEPRODUCT-DELETE-NIL-REQUEST-v1.0: NilRequest
//   - ESPYNA-TEST-PRODUCT-PRICEPRODUCT-DELETE-NIL-DATA-v1.0: NilData
//   - ESPYNA-TEST-PRODUCT-PRICEPRODUCT-DELETE-NOT-FOUND-v1.0: NotFound
//   - ESPYNA-TEST-PRODUCT-PRICEPRODUCT-DELETE-TRANSACTION-FAILURE-v1.0: TransactionFailure
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

// Type alias for delete price product test cases
type DeletePriceProductTestCase = testutil.GenericTestCase[*priceproductpb.DeletePriceProductRequest, *priceproductpb.DeletePriceProductResponse]

// deleteTestUseCaseWithAuth is a helper to create the DeletePriceProductUseCase with mock dependencies and auth control.
func deleteTestUseCaseWithAuth(businessType string, supportsTransaction bool, shouldAuthorize bool) *DeletePriceProductUseCase {
	// Use the default repository which will be pre-populated from copya data
	mockRepo := product.NewMockPriceProductRepository(businessType)

	repositories := DeletePriceProductRepositories{
		PriceProduct: mockRepo,
	}

	var authService ports.AuthorizationService
	if shouldAuthorize {
		authService = mockAuth.NewAllowAllAuth()
	} else {
		authService = mockAuth.NewDenyAllAuth()
	}

	services := DeletePriceProductServices{
		AuthorizationService: authService,
		TransactionService:   mockDb.NewMockTransactionService(supportsTransaction),
		TranslationService:   translation.NewLynguaTranslationService(),
	}

	return NewDeletePriceProductUseCase(repositories, services)
}

func TestDeletePriceProductUseCase_Execute_TableDriven(t *testing.T) {
	businessType := testutil.GetTestBusinessType()

	// Load test data resolvers before defining test cases
	deleteSuccessResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "price_product", "DeletePriceProduct_Success")
	testutil.AssertTestCaseLoad(t, err, "DeletePriceProduct_Success")

	authorizationUnauthorizedResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "price_product", "Authorization_Unauthorized")
	testutil.AssertTestCaseLoad(t, err, "Authorization_Unauthorized")

	testCases := []DeletePriceProductTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-PRODUCT-PRICEPRODUCT-DELETE-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *priceproductpb.DeletePriceProductRequest {
				return &priceproductpb.DeletePriceProductRequest{
					Data: &priceproductpb.PriceProduct{
						Id: deleteSuccessResolver.MustGetString("deletablePriceProductId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *priceproductpb.DeletePriceProductResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")

				// Verify that the item is gone
				useCaseTyped := useCase.(*DeletePriceProductUseCase)
				readReq := &priceproductpb.ReadPriceProductRequest{Data: &priceproductpb.PriceProduct{Id: deleteSuccessResolver.MustGetString("deletablePriceProductId")}}
				_, err = useCaseTyped.repositories.PriceProduct.ReadPriceProduct(ctx, readReq)
				testutil.AssertError(t, err)
				// Error should indicate entity not found (successfully deleted)
			},
		},
		{
			Name:     "WithTransaction",
			TestCode: "ESPYNA-TEST-PRODUCT-PRICEPRODUCT-DELETE-TRANSACTION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *priceproductpb.DeletePriceProductRequest {
				return &priceproductpb.DeletePriceProductRequest{
					Data: &priceproductpb.PriceProduct{
						Id: "price-math-basic", // Use existing ID from mock data
					},
				}
			},
			UseTransaction: true,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *priceproductpb.DeletePriceProductResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
			},
		},
		{
			Name:     "Unauthorized",
			TestCode: "ESPYNA-TEST-PRODUCT-PRICEPRODUCT-DELETE-AUTHORIZATION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *priceproductpb.DeletePriceProductRequest {
				return &priceproductpb.DeletePriceProductRequest{
					Data: &priceproductpb.PriceProduct{
						Id: authorizationUnauthorizedResolver.MustGetString("targetPriceProductId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        false,
			ExpectSuccess:  false,
			ExpectedError:  "price_product.errors.authorization_failed",
			Assertions: func(t *testing.T, response *priceproductpb.DeletePriceProductResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertAuthorizationError(t, err)
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-PRODUCT-PRICEPRODUCT-DELETE-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *priceproductpb.DeletePriceProductRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "price_product.validation.request_required",
			Assertions: func(t *testing.T, response *priceproductpb.DeletePriceProductResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilRequest(t, err)
			},
		},
		{
			Name:     "NilData",
			TestCode: "ESPYNA-TEST-PRODUCT-PRICEPRODUCT-DELETE-NIL-DATA-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *priceproductpb.DeletePriceProductRequest {
				return &priceproductpb.DeletePriceProductRequest{
					Data: nil,
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "price_product.validation.data_required",
			Assertions: func(t *testing.T, response *priceproductpb.DeletePriceProductResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "nil data")
			},
		},
		{
			Name:     "NotFound",
			TestCode: "ESPYNA-TEST-PRODUCT-PRICEPRODUCT-DELETE-NOT-FOUND-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *priceproductpb.DeletePriceProductRequest {
				return &priceproductpb.DeletePriceProductRequest{
					Data: &priceproductpb.PriceProduct{
						Id: "price-non-existent",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "price_product.errors.not_found",
			ErrorTags:      map[string]any{"id": "price-non-existent"},
			Assertions: func(t *testing.T, response *priceproductpb.DeletePriceProductResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertError(t, err)
				// Error message will be validated by ExpectedError and ErrorTags through translation service
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
			useCase := deleteTestUseCaseWithAuth(businessType, tc.UseTransaction, tc.UseAuth)

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

// Individual test functions for backward compatibility and specialized scenarios
func TestDeletePriceProductUseCase_Execute_Success(t *testing.T) {
	testCode := "ESPYNA-TEST-PRODUCT-PRICEPRODUCT-DELETE-SUCCESS-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "Success", true)

	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := deleteTestUseCaseWithAuth(businessType, false, true)

	// Load test case using centralized TestCaseResolver
	resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "price_product", "DeletePriceProduct_Success")
	testutil.AssertTestCaseLoad(t, err, "DeletePriceProduct_Success")

	req := &priceproductpb.DeletePriceProductRequest{
		Data: &priceproductpb.PriceProduct{Id: resolver.MustGetString("deletablePriceProductId")},
	}

	res, err := useCase.Execute(ctx, req)

	testutil.AssertNoError(t, err)
	testutil.AssertNotNil(t, res, "response")
	testutil.AssertTrue(t, res.Success, "success")

	// Verify that the item is gone
	readReq := &priceproductpb.ReadPriceProductRequest{Data: &priceproductpb.PriceProduct{Id: resolver.MustGetString("deletablePriceProductId")}}
	_, err = useCase.repositories.PriceProduct.ReadPriceProduct(ctx, readReq)
	testutil.AssertError(t, err)
	// Error should indicate entity not found (successfully deleted)

	// Log test completion with result
	testutil.LogTestResult(t, testCode, "Success", true, nil)
}

func TestDeletePriceProductUseCase_Execute_NotFound(t *testing.T) {
	testCode := "ESPYNA-TEST-PRODUCT-PRICEPRODUCT-DELETE-NOT-FOUND-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "NotFound", false)

	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := deleteTestUseCaseWithAuth(businessType, false, true)

	req := &priceproductpb.DeletePriceProductRequest{
		Data: &priceproductpb.PriceProduct{Id: "price-non-existent"},
	}

	_, err := useCase.Execute(ctx, req)
	testutil.AssertError(t, err)

	// Error should indicate entity not found

	// Log test completion with result
	testutil.LogTestResult(t, testCode, "NotFound", false, err)
}
