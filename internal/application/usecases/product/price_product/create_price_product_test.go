//go:build mock_db && mock_auth && google && uuidv7

// Package price_product provides table-driven tests for the price product creation use case.
//
// The tests cover various scenarios, including success, transaction handling,
// authorization, nil requests, validation errors, and boundary conditions.
// Each test case is defined in a table with a specific test code, request setup,
// and assertions.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user").
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education").
//
// Usage:
//   - Run all tests: go test -tags="mock_db,mock_auth" ./...
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestCreatePriceProductUseCase_Execute_TableDriven
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-PRODUCT-PRICEPRODUCT-CREATE-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-PRODUCT-PRICEPRODUCT-CREATE-TRANSACTION-v1.0: WithTransaction
//   - ESPYNA-TEST-PRODUCT-PRICEPRODUCT-CREATE-AUTHORIZATION-v1.0: Unauthorized
//   - ESPYNA-TEST-PRODUCT-PRICEPRODUCT-CREATE-NIL-REQUEST-v1.0: NilRequest
//   - ESPYNA-TEST-PRODUCT-PRICEPRODUCT-CREATE-NIL-DATA-v1.0: NilData
//   - ESPYNA-TEST-PRODUCT-PRICEPRODUCT-CREATE-VALIDATION-EMPTY-NAME-v1.0: EmptyName
//   - ESPYNA-TEST-PRODUCT-PRICEPRODUCT-CREATE-VALIDATION-EMPTY-PRODUCT-ID-v1.0: EmptyProductId
//   - ESPYNA-TEST-PRODUCT-PRICEPRODUCT-CREATE-VALIDATION-INVALID-PRODUCT-ID-v1.0: InvalidProductId
//   - ESPYNA-TEST-PRODUCT-PRICEPRODUCT-CREATE-VALIDATION-NEGATIVE-AMOUNT-v1.0: NegativeAmount
//   - ESPYNA-TEST-PRODUCT-PRICEPRODUCT-CREATE-VALIDATION-EMPTY-CURRENCY-v1.0: EmptyCurrency
//   - ESPYNA-TEST-PRODUCT-PRICEPRODUCT-CREATE-ENRICHMENT-v1.0: DataEnrichment
//   - ESPYNA-TEST-PRODUCT-PRICEPRODUCT-CREATE-VALIDATION-MINIMAL-v1.0: MinimalValidData
//   - ESPYNA-TEST-PRODUCT-PRICEPRODUCT-CREATE-VALIDATION-MAXIMAL-v1.0: MaxValidData
//   - ESPYNA-TEST-PRODUCT-PRICEPRODUCT-CREATE-TRANSACTION-FAILURE-v1.0: TransactionFailure
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
	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/id/uuidv7"
	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/translation"
	priceproductpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/price_product"
)

// Type alias for create price product test cases
type CreatePriceProductTestCase = testutil.GenericTestCase[*priceproductpb.CreatePriceProductRequest, *priceproductpb.CreatePriceProductResponse]

// createTestUseCaseWithAuth is a helper to create the CreatePriceProductUseCase with mock dependencies and auth control.
func createTestUseCaseWithAuth(businessType string, supportsTransaction bool, shouldAuthorize bool) *CreatePriceProductUseCase {
	// This use case requires a mock repository for PriceProduct itself,
	// and also for the Product entity to validate references.
	mockPriceProductRepo := product.NewMockPriceProductRepository(businessType, product.WithoutPriceProductInitialData())
	mockProductRepo := product.NewMockProductRepository(businessType)

	repositories := CreatePriceProductRepositories{
		PriceProduct: mockPriceProductRepo,
		Product:      mockProductRepo,
	}

	var authService ports.AuthorizationService
	if shouldAuthorize {
		authService = mockAuth.NewAllowAllAuth()
	} else {
		authService = mockAuth.NewDenyAllAuth()
	}

	services := CreatePriceProductServices{
		AuthorizationService: authService,
		TransactionService:   mockDb.NewMockTransactionService(supportsTransaction),
		TranslationService:   translation.NewLynguaTranslationService(),
		IDService:            uuidv7.NewGoogleUUIDv7Service(),
	}

	return NewCreatePriceProductUseCase(repositories, services)
}

func TestCreatePriceProductUseCase_Execute_TableDriven(t *testing.T) {
	businessType := testutil.GetTestBusinessType()

	// Load test data resolvers before defining test cases
	createSuccessResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "price_product", "CreatePriceProduct_Success")
	testutil.AssertTestCaseLoad(t, err, "CreatePriceProduct_Success")

	authorizationUnauthorizedResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "price_product", "Authorization_Unauthorized")
	testutil.AssertTestCaseLoad(t, err, "Authorization_Unauthorized")

	validationErrorEmptyNameResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "price_product", "ValidationError_EmptyName")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_EmptyName")

	validationErrorInvalidProductIdResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "price_product", "ValidationError_InvalidProductId")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_InvalidProductId")

	testCases := []CreatePriceProductTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-PRODUCT-PRICEPRODUCT-CREATE-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *priceproductpb.CreatePriceProductRequest {
				return &priceproductpb.CreatePriceProductRequest{
					Data: &priceproductpb.PriceProduct{
						ProductId: createSuccessResolver.MustGetString("validProductId"),
						Name:      createSuccessResolver.MustGetString("newPriceProductName"),
						Amount:    int64(createSuccessResolver.MustGetInt("newPriceProductAmount")),
						Currency:  createSuccessResolver.MustGetString("newPriceProductCurrency"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *priceproductpb.CreatePriceProductResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertEqual(t, 1, len(response.Data), "response data length")
				createdPriceProduct := response.Data[0]
				testutil.AssertStringEqual(t, createSuccessResolver.MustGetString("newPriceProductName"), createdPriceProduct.Name, "price product name")
				testutil.AssertNonEmptyString(t, createdPriceProduct.Id, "price product ID")
				testutil.AssertTrue(t, createdPriceProduct.Active, "price product active status")
				testutil.AssertFieldSet(t, createdPriceProduct.DateCreated, "DateCreated")
				testutil.AssertFieldSet(t, createdPriceProduct.DateCreatedString, "DateCreatedString")
				testutil.AssertEqual(t, int64(createSuccessResolver.MustGetInt("newPriceProductAmount")), createdPriceProduct.Amount, "price product amount")
				testutil.AssertStringEqual(t, createSuccessResolver.MustGetString("newPriceProductCurrency"), createdPriceProduct.Currency, "price product currency")
			},
		},
		{
			Name:     "WithTransaction",
			TestCode: "ESPYNA-TEST-PRODUCT-PRICEPRODUCT-CREATE-TRANSACTION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *priceproductpb.CreatePriceProductRequest {
				return &priceproductpb.CreatePriceProductRequest{
					Data: &priceproductpb.PriceProduct{
						ProductId: createSuccessResolver.MustGetString("validProductId"),
						Name:      createSuccessResolver.MustGetString("newPriceProductName"),
						Amount:    int64(createSuccessResolver.MustGetInt("newPriceProductAmount")),
						Currency:  createSuccessResolver.MustGetString("newPriceProductCurrency"),
					},
				}
			},
			UseTransaction: true,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *priceproductpb.CreatePriceProductResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				createdPriceProduct := response.Data[0]
				testutil.AssertStringEqual(t, createSuccessResolver.MustGetString("newPriceProductName"), createdPriceProduct.Name, "price product name")
			},
		},
		{
			Name:     "Unauthorized",
			TestCode: "ESPYNA-TEST-PRODUCT-PRICEPRODUCT-CREATE-AUTHORIZATION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *priceproductpb.CreatePriceProductRequest {
				return &priceproductpb.CreatePriceProductRequest{
					Data: &priceproductpb.PriceProduct{
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
			Assertions: func(t *testing.T, response *priceproductpb.CreatePriceProductResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertAuthorizationError(t, err)
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-PRODUCT-PRICEPRODUCT-CREATE-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *priceproductpb.CreatePriceProductRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "price_product.validation.request_required",
			Assertions: func(t *testing.T, response *priceproductpb.CreatePriceProductResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilRequest(t, err)
			},
		},
		{
			Name:     "NilData",
			TestCode: "ESPYNA-TEST-PRODUCT-PRICEPRODUCT-CREATE-NIL-DATA-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *priceproductpb.CreatePriceProductRequest {
				return &priceproductpb.CreatePriceProductRequest{
					Data: nil,
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "price_product.validation.data_required",
			Assertions: func(t *testing.T, response *priceproductpb.CreatePriceProductResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "nil data")
			},
		},
		{
			Name:     "EmptyName",
			TestCode: "ESPYNA-TEST-PRODUCT-PRICEPRODUCT-CREATE-VALIDATION-EMPTY-NAME-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *priceproductpb.CreatePriceProductRequest {
				return &priceproductpb.CreatePriceProductRequest{
					Data: &priceproductpb.PriceProduct{
						ProductId: validationErrorEmptyNameResolver.MustGetString("validProductId"),
						Name:      validationErrorEmptyNameResolver.MustGetString("emptyName"),
						Amount:    int64(validationErrorEmptyNameResolver.MustGetInt("validPriceProductAmount")),
						Currency:  validationErrorEmptyNameResolver.MustGetString("validPriceProductCurrency"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "price_product.validation.name_required",
			Assertions: func(t *testing.T, response *priceproductpb.CreatePriceProductResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty name")
			},
		},
		{
			Name:     "EmptyProductId",
			TestCode: "ESPYNA-TEST-PRODUCT-PRICEPRODUCT-CREATE-VALIDATION-EMPTY-PRODUCT-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *priceproductpb.CreatePriceProductRequest {
				return &priceproductpb.CreatePriceProductRequest{
					Data: &priceproductpb.PriceProduct{
						ProductId: "",
						Name:      "Test Tuition",
						Amount:    500,
						Currency:  "USD",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "price_product.validation.product_id_required",
			Assertions: func(t *testing.T, response *priceproductpb.CreatePriceProductResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty product ID")
			},
		},
		{
			Name:     "InvalidProductId",
			TestCode: "ESPYNA-TEST-PRODUCT-PRICEPRODUCT-CREATE-VALIDATION-INVALID-PRODUCT-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *priceproductpb.CreatePriceProductRequest {
				return &priceproductpb.CreatePriceProductRequest{
					Data: &priceproductpb.PriceProduct{
						ProductId: validationErrorInvalidProductIdResolver.MustGetString("invalidProductId"),
						Name:      validationErrorInvalidProductIdResolver.MustGetString("validPriceProductName"),
						Amount:    int64(validationErrorInvalidProductIdResolver.MustGetInt("validPriceProductAmount")),
						Currency:  validationErrorInvalidProductIdResolver.MustGetString("validPriceProductCurrency"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "price_product.validation.product_id_invalid",
			Assertions: func(t *testing.T, response *priceproductpb.CreatePriceProductResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "invalid product ID")
			},
		},
		{
			Name:     "NegativeAmount",
			TestCode: "ESPYNA-TEST-PRODUCT-PRICEPRODUCT-CREATE-VALIDATION-NEGATIVE-AMOUNT-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *priceproductpb.CreatePriceProductRequest {
				return &priceproductpb.CreatePriceProductRequest{
					Data: &priceproductpb.PriceProduct{
						ProductId: "subject-math",
						Name:      "Negative Amount Tuition",
						Amount:    -100,
						Currency:  "USD",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "price_product.validation.amount_invalid",
			Assertions: func(t *testing.T, response *priceproductpb.CreatePriceProductResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "negative amount")
			},
		},
		{
			Name:     "EmptyCurrency",
			TestCode: "ESPYNA-TEST-PRODUCT-PRICEPRODUCT-CREATE-VALIDATION-EMPTY-CURRENCY-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *priceproductpb.CreatePriceProductRequest {
				return &priceproductpb.CreatePriceProductRequest{
					Data: &priceproductpb.PriceProduct{
						ProductId: "subject-math",
						Name:      "No Currency Tuition",
						Amount:    550,
						Currency:  "",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "price_product.validation.currency_required",
			Assertions: func(t *testing.T, response *priceproductpb.CreatePriceProductResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty currency")
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
			useCase := createTestUseCaseWithAuth(businessType, tc.UseTransaction, tc.UseAuth)

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
func TestCreatePriceProductUseCase_Execute_Success(t *testing.T) {
	testCode := "ESPYNA-TEST-PRODUCT-PRICEPRODUCT-CREATE-SUCCESS-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "Success", true)

	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestUseCaseWithAuth(businessType, false, true)

	// Load test case using centralized TestCaseResolver
	resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "price_product", "CreatePriceProduct_Success")
	testutil.AssertTestCaseLoad(t, err, "CreatePriceProduct_Success")

	req := &priceproductpb.CreatePriceProductRequest{
		Data: &priceproductpb.PriceProduct{
			ProductId: resolver.MustGetString("validProductId"),
			Name:      resolver.MustGetString("newPriceProductName"),
			Amount:    int64(resolver.MustGetInt("newPriceProductAmount")),
			Currency:  resolver.MustGetString("newPriceProductCurrency"),
		},
	}

	response, err := useCase.Execute(ctx, req)

	testutil.AssertNoError(t, err)
	testutil.AssertNotNil(t, response, "response")
	testutil.AssertTrue(t, response.Success, "success")
	testutil.AssertEqual(t, 1, len(response.Data), "response data length")

	createdPriceProduct := response.Data[0]
	testutil.AssertStringEqual(t, resolver.MustGetString("newPriceProductName"), createdPriceProduct.Name, "price product name")
	testutil.AssertNonEmptyString(t, createdPriceProduct.Id, "price product ID")
	testutil.AssertTrue(t, createdPriceProduct.Active, "price product active status")
	testutil.AssertFieldSet(t, createdPriceProduct.DateCreated, "DateCreated")
	testutil.AssertFieldSet(t, createdPriceProduct.DateCreatedString, "DateCreatedString")
	testutil.AssertEqual(t, int64(resolver.MustGetInt("newPriceProductAmount")), createdPriceProduct.Amount, "price product amount")
	testutil.AssertStringEqual(t, resolver.MustGetString("newPriceProductCurrency"), createdPriceProduct.Currency, "price product currency")

	// Log test completion with result
	testutil.LogTestResult(t, testCode, "Success", true, nil)
}
