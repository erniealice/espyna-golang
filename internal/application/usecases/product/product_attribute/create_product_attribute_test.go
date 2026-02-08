//go:build mock_db && mock_auth

// Package product_attribute provides table-driven tests for the product attribute creation use case.
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
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestCreateProductAttributeUseCase_Execute_TableDriven
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-PRODUCT-PRODUCT-ATTRIBUTE-CREATE-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-PRODUCT-PRODUCT-ATTRIBUTE-CREATE-TRANSACTION-v1.0: WithTransaction
//   - ESPYNA-TEST-PRODUCT-PRODUCT-ATTRIBUTE-CREATE-AUTHORIZATION-v1.0: Unauthorized
//   - ESPYNA-TEST-PRODUCT-PRODUCT-ATTRIBUTE-CREATE-NIL-REQUEST-v1.0: NilRequest
//   - ESPYNA-TEST-PRODUCT-PRODUCT-ATTRIBUTE-CREATE-NIL-DATA-v1.0: NilData
//   - ESPYNA-TEST-PRODUCT-PRODUCT-ATTRIBUTE-CREATE-VALIDATION-EMPTY-PRODUCT-ID-v1.0: EmptyProductId
//   - ESPYNA-TEST-PRODUCT-PRODUCT-ATTRIBUTE-CREATE-VALIDATION-EMPTY-ATTRIBUTE-ID-v1.0: EmptyAttributeId
//   - ESPYNA-TEST-PRODUCT-PRODUCT-ATTRIBUTE-CREATE-VALIDATION-EMPTY-VALUE-v1.0: EmptyValue
//   - ESPYNA-TEST-PRODUCT-PRODUCT-ATTRIBUTE-CREATE-VALIDATION-INVALID-PRODUCT-ID-v1.0: InvalidProductId
//   - ESPYNA-TEST-PRODUCT-PRODUCT-ATTRIBUTE-CREATE-VALIDATION-INVALID-ATTRIBUTE-ID-v1.0: InvalidAttributeId
//   - ESPYNA-TEST-PRODUCT-PRODUCT-ATTRIBUTE-CREATE-ENRICHMENT-v1.0: DataEnrichment
//   - ESPYNA-TEST-PRODUCT-PRODUCT-ATTRIBUTE-CREATE-TRANSACTION-FAILURE-v1.0: TransactionFailure
//
// Data Sources:
//   - Test cases: packages/copya/data_test/{businessType}/product_attribute.json
//   - Mock data: packages/copya/data/{businessType}/product_attribute.json
//   - Translations: packages/lyngua/translations/{languageCode}/{businessType}/product_attribute.json
package product_attribute

import (
	"context"
	"testing"

	copyatestutil "leapfor.xyz/copya/golang/testutil"
	"github.com/erniealice/espyna-golang/internal/application/shared/testutil"
	mockDb "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock"
	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock/common"
	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock/product"
	productattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_attribute"
)

// Type alias for create product attribute test cases
type CreateProductAttributeTestCase = testutil.GenericTestCase[*productattributepb.CreateProductAttributeRequest, *productattributepb.CreateProductAttributeResponse]

func createTestUseCaseWithAuth(businessType string, supportsTransaction bool, shouldAuthorize bool) *CreateProductAttributeUseCase {
	mockProductAttributeRepo := product.NewMockProductAttributeRepository(businessType)
	mockProductRepo := product.NewMockProductRepository(businessType)
	mockAttributeRepo := common.NewMockAttributeRepository(businessType)

	repositories := CreateProductAttributeRepositories{
		ProductAttribute: mockProductAttributeRepo,
		Product:          mockProductRepo,
		Attribute:        mockAttributeRepo,
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, shouldAuthorize)
	services := CreateProductAttributeServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
		IDService:            standardServices.IDService,
	}

	return NewCreateProductAttributeUseCase(repositories, services)
}

func TestCreateProductAttributeUseCase_Execute_TableDriven(t *testing.T) {
	businessType := testutil.GetTestBusinessType()

	// Load test data resolvers before defining test cases
	createSuccessResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "product_attribute", "CreateProductAttribute_Success")
	testutil.AssertTestCaseLoad(t, err, "CreateProductAttribute_Success")

	authorizationUnauthorizedResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "product_attribute", "Authorization_Unauthorized")
	testutil.AssertTestCaseLoad(t, err, "Authorization_Unauthorized")

	validationErrorEmptyProductIdResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "product_attribute", "ValidationError_EmptyProductId")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_EmptyProductId")

	validationErrorEmptyAttributeIdResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "product_attribute", "ValidationError_EmptyAttributeId")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_EmptyAttributeId")

	validationErrorEmptyValueResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "product_attribute", "ValidationError_EmptyValue")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_EmptyValue")

	testCases := []CreateProductAttributeTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-ATTRIBUTE-CREATE-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productattributepb.CreateProductAttributeRequest {
				return &productattributepb.CreateProductAttributeRequest{
					Data: &productattributepb.ProductAttribute{
						ProductId:   createSuccessResolver.MustGetString("newProductAttributeProductId"),
						AttributeId: createSuccessResolver.MustGetString("newProductAttributeAttributeId"),
						Value:       createSuccessResolver.MustGetString("newProductAttributeValue"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *productattributepb.CreateProductAttributeResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertEqual(t, 1, len(response.Data), "response data length")
				createdProductAttribute := response.Data[0]
				testutil.AssertNonEmptyString(t, createdProductAttribute.Id, "product attribute ID")
				testutil.AssertStringEqual(t, createSuccessResolver.MustGetString("newProductAttributeProductId"), createdProductAttribute.ProductId, "product ID")
				testutil.AssertStringEqual(t, createSuccessResolver.MustGetString("newProductAttributeAttributeId"), createdProductAttribute.AttributeId, "attribute ID")
				testutil.AssertStringEqual(t, createSuccessResolver.MustGetString("newProductAttributeValue"), createdProductAttribute.Value, "value")
				testutil.AssertFieldSet(t, createdProductAttribute.DateCreated, "DateCreated")
				testutil.AssertFieldSet(t, createdProductAttribute.DateCreatedString, "DateCreatedString")
			},
		},
		{
			Name:     "WithTransaction",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-ATTRIBUTE-CREATE-TRANSACTION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productattributepb.CreateProductAttributeRequest {
				return &productattributepb.CreateProductAttributeRequest{
					Data: &productattributepb.ProductAttribute{
						ProductId:   createSuccessResolver.MustGetString("newProductAttributeProductId"),
						AttributeId: createSuccessResolver.MustGetString("newProductAttributeAttributeId"),
						Value:       createSuccessResolver.MustGetString("newProductAttributeValue"),
					},
				}
			},
			UseTransaction: true,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *productattributepb.CreateProductAttributeResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				createdProductAttribute := response.Data[0]
				testutil.AssertStringEqual(t, createSuccessResolver.MustGetString("newProductAttributeProductId"), createdProductAttribute.ProductId, "product ID")
			},
		},
		{
			Name:     "Unauthorized",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-ATTRIBUTE-CREATE-AUTHORIZATION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productattributepb.CreateProductAttributeRequest {
				return &productattributepb.CreateProductAttributeRequest{
					Data: &productattributepb.ProductAttribute{
						ProductId:   authorizationUnauthorizedResolver.MustGetString("unauthorizedProductId"),
						AttributeId: authorizationUnauthorizedResolver.MustGetString("unauthorizedAttributeId"),
						Value:       authorizationUnauthorizedResolver.MustGetString("unauthorizedValue"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        false,
			ExpectSuccess:  false,
			ExpectedError:  "product_attribute.errors.authorization_failed",
			Assertions: func(t *testing.T, response *productattributepb.CreateProductAttributeResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertAuthorizationError(t, err)
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-ATTRIBUTE-CREATE-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productattributepb.CreateProductAttributeRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "product_attribute.validation.request_required",
			Assertions: func(t *testing.T, response *productattributepb.CreateProductAttributeResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilRequest(t, err)
			},
		},
		{
			Name:     "NilData",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-ATTRIBUTE-CREATE-NIL-DATA-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productattributepb.CreateProductAttributeRequest {
				return &productattributepb.CreateProductAttributeRequest{Data: nil}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "product_attribute.validation.data_required",
			Assertions: func(t *testing.T, response *productattributepb.CreateProductAttributeResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilData(t, err)
			},
		},
		{
			Name:     "EmptyProductId",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-ATTRIBUTE-CREATE-VALIDATION-EMPTY-PRODUCT-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productattributepb.CreateProductAttributeRequest {
				return &productattributepb.CreateProductAttributeRequest{
					Data: &productattributepb.ProductAttribute{
						ProductId:   validationErrorEmptyProductIdResolver.MustGetString("emptyProductId"),
						AttributeId: validationErrorEmptyProductIdResolver.MustGetString("validAttributeId"),
						Value:       validationErrorEmptyProductIdResolver.MustGetString("validValue"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "product_attribute.validation.product_id_required",
			Assertions: func(t *testing.T, response *productattributepb.CreateProductAttributeResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty product ID")
			},
		},
		{
			Name:     "EmptyAttributeId",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-ATTRIBUTE-CREATE-VALIDATION-EMPTY-ATTRIBUTE-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productattributepb.CreateProductAttributeRequest {
				return &productattributepb.CreateProductAttributeRequest{
					Data: &productattributepb.ProductAttribute{
						ProductId:   validationErrorEmptyAttributeIdResolver.MustGetString("validProductId"),
						AttributeId: validationErrorEmptyAttributeIdResolver.MustGetString("emptyAttributeId"),
						Value:       validationErrorEmptyAttributeIdResolver.MustGetString("validValue"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "product_attribute.validation.attribute_id_required",
			Assertions: func(t *testing.T, response *productattributepb.CreateProductAttributeResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty attribute ID")
			},
		},
		{
			Name:     "EmptyValue",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-ATTRIBUTE-CREATE-VALIDATION-EMPTY-VALUE-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productattributepb.CreateProductAttributeRequest {
				return &productattributepb.CreateProductAttributeRequest{
					Data: &productattributepb.ProductAttribute{
						ProductId:   validationErrorEmptyValueResolver.MustGetString("validProductId"),
						AttributeId: validationErrorEmptyValueResolver.MustGetString("validAttributeId"),
						Value:       validationErrorEmptyValueResolver.MustGetString("emptyValue"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "product_attribute.validation.value_required",
			Assertions: func(t *testing.T, response *productattributepb.CreateProductAttributeResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty value")
			},
		},
		{
			Name:     "InvalidProductId",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-ATTRIBUTE-CREATE-VALIDATION-INVALID-PRODUCT-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productattributepb.CreateProductAttributeRequest {
				return &productattributepb.CreateProductAttributeRequest{
					Data: &productattributepb.ProductAttribute{
						ProductId:   "nonexistent-product-id",
						AttributeId: createSuccessResolver.MustGetString("newProductAttributeAttributeId"),
						Value:       createSuccessResolver.MustGetString("newProductAttributeValue"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "product_attribute.errors.product_not_found",
			Assertions: func(t *testing.T, response *productattributepb.CreateProductAttributeResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "invalid product ID")
			},
		},
		{
			Name:     "InvalidAttributeId",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-ATTRIBUTE-CREATE-VALIDATION-INVALID-ATTRIBUTE-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productattributepb.CreateProductAttributeRequest {
				return &productattributepb.CreateProductAttributeRequest{
					Data: &productattributepb.ProductAttribute{
						ProductId:   createSuccessResolver.MustGetString("newProductAttributeProductId"),
						AttributeId: "nonexistent-attribute-id",
						Value:       createSuccessResolver.MustGetString("newProductAttributeValue"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "product_attribute.errors.attribute_not_found",
			Assertions: func(t *testing.T, response *productattributepb.CreateProductAttributeResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "invalid attribute ID")
			},
		},
		{
			Name:     "DataEnrichment",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-ATTRIBUTE-CREATE-ENRICHMENT-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productattributepb.CreateProductAttributeRequest {
				return &productattributepb.CreateProductAttributeRequest{
					Data: &productattributepb.ProductAttribute{
						ProductId:   createSuccessResolver.MustGetString("newProductAttributeProductId"),
						AttributeId: createSuccessResolver.MustGetString("newProductAttributeAttributeId"),
						Value:       "Data Enrichment Test",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *productattributepb.CreateProductAttributeResponse, err error, useCase interface{}, ctx context.Context) {
				createdProductAttribute := response.Data[0]
				testutil.AssertNonEmptyString(t, createdProductAttribute.Id, "generated ID")
				testutil.AssertFieldSet(t, createdProductAttribute.DateCreated, "DateCreated")
				testutil.AssertFieldSet(t, createdProductAttribute.DateCreatedString, "DateCreatedString")
				testutil.AssertFieldSet(t, createdProductAttribute.DateModified, "DateModified")
				testutil.AssertFieldSet(t, createdProductAttribute.DateModifiedString, "DateModifiedString")
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

func TestCreateProductAttributeUseCase_Execute_TransactionFailure(t *testing.T) {
	testCode := "ESPYNA-TEST-PRODUCT-PRODUCT-ATTRIBUTE-CREATE-TRANSACTION-FAILURE-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "TransactionFailure", false)

	// Create a use case with a failing transaction service
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	mockProductAttributeRepo := product.NewMockProductAttributeRepository(businessType)
	mockProductRepo := product.NewMockProductRepository(businessType)
	mockAttributeRepo := common.NewMockAttributeRepository(businessType)

	repositories := CreateProductAttributeRepositories{
		ProductAttribute: mockProductAttributeRepo,
		Product:          mockProductRepo,
		Attribute:        mockAttributeRepo,
	}

	standardServices := testutil.CreateStandardServices(false, true)
	services := CreateProductAttributeServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   mockDb.NewFailingMockTransactionService(),
		TranslationService:   standardServices.TranslationService,
		IDService:            standardServices.IDService,
	}

	useCase := NewCreateProductAttributeUseCase(repositories, services)

	// Load test case using centralized TestCaseResolver
	resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "product_attribute", "CreateProductAttribute_Success")
	testutil.AssertTestCaseLoad(t, err, "CreateProductAttribute_Success")

	req := &productattributepb.CreateProductAttributeRequest{
		Data: &productattributepb.ProductAttribute{
			ProductId:   resolver.MustGetString("newProductAttributeProductId"),
			AttributeId: resolver.MustGetString("newProductAttributeAttributeId"),
			Value:       resolver.MustGetString("newProductAttributeValue"),
		},
	}

	_, err = useCase.Execute(ctx, req)

	// Verify that a transaction error occurred
	testutil.AssertTransactionError(t, err)

	// Log completion of transaction failure test
	testutil.LogTestResult(t, testCode, "TransactionFailure", false, err)
}
