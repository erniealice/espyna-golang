//go:build mock_db && mock_auth

// Package product provides table-driven tests for the product creation use case.
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
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestCreateProductUseCase_Execute_TableDriven
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-PRODUCT-PRODUCT-CREATE-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-PRODUCT-PRODUCT-CREATE-TRANSACTION-v1.0: WithTransaction
//   - ESPYNA-TEST-PRODUCT-PRODUCT-CREATE-AUTHORIZATION-v1.0: Unauthorized
//   - ESPYNA-TEST-PRODUCT-PRODUCT-CREATE-NIL-REQUEST-v1.0: NilRequest
//   - ESPYNA-TEST-PRODUCT-PRODUCT-CREATE-NIL-DATA-v1.0: NilData
//   - ESPYNA-TEST-PRODUCT-PRODUCT-CREATE-VALIDATION-EMPTY-NAME-v1.0: EmptyName
//   - ESPYNA-TEST-PRODUCT-PRODUCT-CREATE-VALIDATION-NAME-TOO-SHORT-v1.0: NameTooShort
//   - ESPYNA-TEST-PRODUCT-PRODUCT-CREATE-VALIDATION-NAME-TOO-LONG-v1.0: NameTooLong
//   - ESPYNA-TEST-PRODUCT-PRODUCT-CREATE-VALIDATION-DESCRIPTION-TOO-LONG-v1.0: DescriptionTooLong
//   - ESPYNA-TEST-PRODUCT-PRODUCT-CREATE-ENRICHMENT-v1.0: DataEnrichment
//   - ESPYNA-TEST-PRODUCT-PRODUCT-CREATE-VALIDATION-MINIMAL-v1.0: MinimalValidData
//   - ESPYNA-TEST-PRODUCT-PRODUCT-CREATE-VALIDATION-MAXIMAL-v1.0: MaxValidData
//   - ESPYNA-TEST-PRODUCT-PRODUCT-CREATE-TRANSACTION-FAILURE-v1.0: TransactionFailure
//
// Data Sources:
//   - Test cases: packages/copya/data_test/{businessType}/product.json
//   - Mock data: packages/copya/data/{businessType}/product.json
//   - Translations: packages/lyngua/translations/{languageCode}/{businessType}/product.json
package product

import (
	"context"
	"testing"

	copyatestutil "leapfor.xyz/copya/golang/testutil"
	"github.com/erniealice/espyna-golang/internal/application/shared/testutil"
	mockDb "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock"
	mockProduct "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock/product"
	productpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product"
)

// Type alias for create product test cases
type CreateProductTestCase = testutil.GenericTestCase[*productpb.CreateProductRequest, *productpb.CreateProductResponse]

func createTestUseCaseWithAuth(businessType string, supportsTransaction bool, shouldAuthorize bool) *CreateProductUseCase {
	mockProductRepo := mockProduct.NewMockProductRepository(businessType)

	repositories := CreateProductRepositories{
		Product: mockProductRepo,
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, shouldAuthorize)
	services := CreateProductServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
		IDService:            standardServices.IDService,
	}

	return NewCreateProductUseCase(repositories, services)
}

func createTestUseCaseWithFailingTransaction(businessType string, shouldAuthorize bool) *CreateProductUseCase {
	mockProductRepo := mockProduct.NewMockProductRepository(businessType)

	repositories := CreateProductRepositories{
		Product: mockProductRepo,
	}

	standardServices := testutil.CreateStandardServices(false, shouldAuthorize)
	services := CreateProductServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   mockDb.NewFailingMockTransactionService(),
		TranslationService:   standardServices.TranslationService,
		IDService:            standardServices.IDService,
	}

	return NewCreateProductUseCase(repositories, services)
}

func TestCreateProductUseCase_Execute_TableDriven(t *testing.T) {
	businessType := testutil.GetTestBusinessType()

	// Load test data resolvers before defining test cases
	createSuccessResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "product", "CreateProduct_Success")
	testutil.AssertTestCaseLoad(t, err, "CreateProduct_Success")

	authorizationUnauthorizedResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "product", "Authorization_Unauthorized")
	testutil.AssertTestCaseLoad(t, err, "Authorization_Unauthorized")

	validationErrorEmptyNameResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "product", "ValidationError_EmptyName")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_EmptyName")

	validationErrorNameTooShortResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "product", "ValidationError_NameTooShort")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_NameTooShort")

	validationErrorNameTooLongResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "product", "ValidationError_NameTooLongGenerated")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_NameTooLongGenerated")

	validationErrorDescriptionTooLongResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "product", "ValidationError_DescriptionTooLongGenerated")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_DescriptionTooLongGenerated")

	boundaryMinimalResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "product", "BoundaryTest_MinimalValid")
	testutil.AssertTestCaseLoad(t, err, "BoundaryTest_MinimalValid")

	boundaryMaximalResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "product", "BoundaryTest_MaximalValid")
	testutil.AssertTestCaseLoad(t, err, "BoundaryTest_MaximalValid")

	testCases := []CreateProductTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-CREATE-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productpb.CreateProductRequest {
				return &productpb.CreateProductRequest{
					Data: &productpb.Product{
						Name:        createSuccessResolver.MustGetString("newProductName"),
						Description: &[]string{createSuccessResolver.MustGetString("newProductDescription")}[0],
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *productpb.CreateProductResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertEqual(t, 1, len(response.Data), "response data length")
				createdProduct := response.Data[0]
				testutil.AssertStringEqual(t, createSuccessResolver.MustGetString("newProductName"), createdProduct.Name, "product name")
				testutil.AssertNonEmptyString(t, createdProduct.Id, "product ID")
				testutil.AssertTrue(t, createdProduct.Active, "product active status")
				testutil.AssertFieldSet(t, createdProduct.DateCreated, "DateCreated")
				testutil.AssertFieldSet(t, createdProduct.DateCreatedString, "DateCreatedString")
			},
		},
		{
			Name:     "WithTransaction",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-CREATE-TRANSACTION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productpb.CreateProductRequest {
				return &productpb.CreateProductRequest{
					Data: &productpb.Product{
						Name:        createSuccessResolver.MustGetString("newProductName"),
						Description: &[]string{createSuccessResolver.MustGetString("newProductDescription")}[0],
					},
				}
			},
			UseTransaction: true,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *productpb.CreateProductResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				createdProduct := response.Data[0]
				testutil.AssertStringEqual(t, createSuccessResolver.MustGetString("newProductName"), createdProduct.Name, "product name")
			},
		},
		{
			Name:     "Unauthorized",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-CREATE-AUTHORIZATION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productpb.CreateProductRequest {
				return &productpb.CreateProductRequest{
					Data: &productpb.Product{
						Name:        authorizationUnauthorizedResolver.MustGetString("unauthorizedProductName"),
						Description: &[]string{authorizationUnauthorizedResolver.MustGetString("unauthorizedProductDescription")}[0],
					},
				}
			},
			UseTransaction: false,
			UseAuth:        false,
			ExpectSuccess:  false,
			ExpectedError:  "product.errors.authorization_failed",
			Assertions: func(t *testing.T, response *productpb.CreateProductResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertAuthorizationError(t, err)
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-CREATE-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productpb.CreateProductRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "product.validation.request_required",
			Assertions: func(t *testing.T, response *productpb.CreateProductResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilRequest(t, err)
			},
		},
		{
			Name:     "NilData",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-CREATE-NIL-DATA-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productpb.CreateProductRequest {
				return &productpb.CreateProductRequest{
					Data: nil,
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "product.validation.data_required",
			Assertions: func(t *testing.T, response *productpb.CreateProductResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "nil data")
			},
		},
		{
			Name:     "EmptyName",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-CREATE-VALIDATION-EMPTY-NAME-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productpb.CreateProductRequest {
				return &productpb.CreateProductRequest{
					Data: &productpb.Product{
						Name:        validationErrorEmptyNameResolver.MustGetString("emptyName"),
						Description: &[]string{validationErrorEmptyNameResolver.MustGetString("validDescription")}[0],
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "product.validation.name_required",
			Assertions: func(t *testing.T, response *productpb.CreateProductResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty name")
			},
		},
		{
			Name:     "NameTooShort",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-CREATE-VALIDATION-NAME-TOO-SHORT-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productpb.CreateProductRequest {
				return &productpb.CreateProductRequest{
					Data: &productpb.Product{
						Name:        validationErrorNameTooShortResolver.MustGetString("tooShortName"),
						Description: &[]string{validationErrorNameTooShortResolver.MustGetString("validDescription")}[0],
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "product.validation.name_too_short",
			Assertions: func(t *testing.T, response *productpb.CreateProductResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "name too short")
			},
		},
		{
			Name:     "NameTooLong",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-CREATE-VALIDATION-NAME-TOO-LONG-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productpb.CreateProductRequest {
				return &productpb.CreateProductRequest{
					Data: &productpb.Product{
						Name:        validationErrorNameTooLongResolver.MustGetString("tooLongNameGenerated"),
						Description: &[]string{validationErrorNameTooLongResolver.MustGetString("validDescription")}[0],
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "product.validation.name_too_long",
			Assertions: func(t *testing.T, response *productpb.CreateProductResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "name too long")
			},
		},
		{
			Name:     "DescriptionTooLong",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-CREATE-VALIDATION-DESCRIPTION-TOO-LONG-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productpb.CreateProductRequest {
				return &productpb.CreateProductRequest{
					Data: &productpb.Product{
						Name:        validationErrorDescriptionTooLongResolver.MustGetString("validName"),
						Description: &[]string{validationErrorDescriptionTooLongResolver.MustGetString("tooLongDescriptionGenerated")}[0],
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "product.validation.description_too_long",
			Assertions: func(t *testing.T, response *productpb.CreateProductResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "description too long")
			},
		},
		{
			Name:     "DataEnrichment",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-CREATE-ENRICHMENT-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productpb.CreateProductRequest {
				return &productpb.CreateProductRequest{
					Data: &productpb.Product{
						Name: "Data Enrichment Test",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *productpb.CreateProductResponse, err error, useCase interface{}, ctx context.Context) {
				createdProduct := response.Data[0]
				testutil.AssertNonEmptyString(t, createdProduct.Id, "generated ID")
				testutil.AssertFieldSet(t, createdProduct.DateCreated, "DateCreated")
				testutil.AssertFieldSet(t, createdProduct.DateCreatedString, "DateCreatedString")
				testutil.AssertTrue(t, createdProduct.Active, "Active")
			},
		},
		{
			Name:     "MinimalValidData",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-CREATE-VALIDATION-MINIMAL-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productpb.CreateProductRequest {
				return &productpb.CreateProductRequest{
					Data: &productpb.Product{
						Name: boundaryMinimalResolver.MustGetString("minValidName"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *productpb.CreateProductResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				createdProduct := response.Data[0]
				// Product names are normalized to title case - "ABC" becomes "Abc"
				testutil.AssertStringEqual(t, "Abc", createdProduct.Name, "product name")
			},
		},
		{
			Name:     "MaxValidData",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-CREATE-VALIDATION-MAXIMAL-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productpb.CreateProductRequest {
				return &productpb.CreateProductRequest{
					Data: &productpb.Product{
						Name:        boundaryMaximalResolver.MustGetString("maxValidName"),
						Description: &[]string{boundaryMaximalResolver.MustGetString("maxValidDescription")}[0],
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *productpb.CreateProductResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				createdProduct := response.Data[0]
				// Product names are normalized to title case
				testutil.AssertStringEqual(t, "This Is A Maximally Valid Product Name That Approaches The Character Limit Boundary", createdProduct.Name, "product name")
			},
		},
		{
			Name:     "TransactionFailure",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-CREATE-TRANSACTION-FAILURE-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productpb.CreateProductRequest {
				return &productpb.CreateProductRequest{
					Data: &productpb.Product{
						Name:        createSuccessResolver.MustGetString("newProductName"),
						Description: &[]string{createSuccessResolver.MustGetString("newProductDescription")}[0],
					},
				}
			},
			UseTransaction: true,
			UseAuth:        true,
			ExpectSuccess:  false,
			Assertions: func(t *testing.T, response *productpb.CreateProductResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertError(t, err)
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

			var useCase *CreateProductUseCase
			if tc.Name == "TransactionFailure" {
				useCase = createTestUseCaseWithFailingTransaction(businessType, tc.UseAuth)
			} else {
				useCase = createTestUseCaseWithAuth(businessType, tc.UseTransaction, tc.UseAuth)
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
