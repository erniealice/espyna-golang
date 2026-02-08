//go:build mock_db && mock_auth

// Package product_attribute provides table-driven tests for the product attribute updating use case.
//
// The tests cover various scenarios, including success, transaction handling,
// authorization, nil requests, validation errors, not found errors, and boundary conditions.
// Each test case is defined in a table with a specific test code, request setup,
// and assertions.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user").
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education").
//
// Usage:
//   - Run all tests: go test -tags="mock_db,mock_auth" ./...
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestUpdateProductAttributeUseCase_Execute_TableDriven
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-PRODUCT-PRODUCT-ATTRIBUTE-UPDATE-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-PRODUCT-PRODUCT-ATTRIBUTE-UPDATE-TRANSACTION-v1.0: WithTransaction
//   - ESPYNA-TEST-PRODUCT-PRODUCT-ATTRIBUTE-UPDATE-AUTHORIZATION-v1.0: Unauthorized
//   - ESPYNA-TEST-PRODUCT-PRODUCT-ATTRIBUTE-UPDATE-NIL-REQUEST-v1.0: NilRequest
//   - ESPYNA-TEST-PRODUCT-PRODUCT-ATTRIBUTE-UPDATE-NIL-DATA-v1.0: NilData
//   - ESPYNA-TEST-PRODUCT-PRODUCT-ATTRIBUTE-UPDATE-NOT-FOUND-v1.0: NotFound
//   - ESPYNA-TEST-PRODUCT-PRODUCT-ATTRIBUTE-UPDATE-VALIDATION-EMPTY-ID-v1.0: EmptyId
//   - ESPYNA-TEST-PRODUCT-PRODUCT-ATTRIBUTE-UPDATE-VALIDATION-EMPTY-VALUE-v1.0: EmptyValue
//   - ESPYNA-TEST-PRODUCT-PRODUCT-ATTRIBUTE-UPDATE-VALIDATION-INVALID-PRODUCT-ID-v1.0: InvalidProductId
//   - ESPYNA-TEST-PRODUCT-PRODUCT-ATTRIBUTE-UPDATE-VALIDATION-INVALID-ATTRIBUTE-ID-v1.0: InvalidAttributeId
//   - ESPYNA-TEST-PRODUCT-PRODUCT-ATTRIBUTE-UPDATE-ENRICHMENT-v1.0: DataEnrichment
//   - ESPYNA-TEST-PRODUCT-PRODUCT-ATTRIBUTE-UPDATE-TRANSACTION-FAILURE-v1.0: TransactionFailure
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

// Type alias for update product attribute test cases
type UpdateProductAttributeTestCase = testutil.GenericTestCase[*productattributepb.UpdateProductAttributeRequest, *productattributepb.UpdateProductAttributeResponse]

func createUpdateTestUseCaseWithAuth(businessType string, supportsTransaction bool, shouldAuthorize bool) *UpdateProductAttributeUseCase {
	mockProductAttributeRepo := product.NewMockProductAttributeRepository(businessType)
	mockProductRepo := product.NewMockProductRepository(businessType)
	mockAttributeRepo := common.NewMockAttributeRepository(businessType)

	repositories := UpdateProductAttributeRepositories{
		ProductAttribute: mockProductAttributeRepo,
		Product:          mockProductRepo,
		Attribute:        mockAttributeRepo,
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, shouldAuthorize)
	services := UpdateProductAttributeServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
	}

	return NewUpdateProductAttributeUseCase(repositories, services)
}

func TestUpdateProductAttributeUseCase_Execute_TableDriven(t *testing.T) {
	businessType := testutil.GetTestBusinessType()

	// Load test data resolvers before defining test cases
	updateSuccessResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "product_attribute", "UpdateProductAttribute_Success")
	testutil.AssertTestCaseLoad(t, err, "UpdateProductAttribute_Success")

	updateNotFoundResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "product_attribute", "UpdateProductAttribute_NotFound")
	testutil.AssertTestCaseLoad(t, err, "UpdateProductAttribute_NotFound")

	authorizationUnauthorizedResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "product_attribute", "Authorization_Unauthorized")
	testutil.AssertTestCaseLoad(t, err, "Authorization_Unauthorized")

	testCases := []UpdateProductAttributeTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-ATTRIBUTE-UPDATE-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productattributepb.UpdateProductAttributeRequest {
				return &productattributepb.UpdateProductAttributeRequest{
					Data: &productattributepb.ProductAttribute{
						Id:          updateSuccessResolver.MustGetString("validProductAttributeId"),
						ProductId:   updateSuccessResolver.MustGetString("validProductId"),
						AttributeId: updateSuccessResolver.MustGetString("validAttributeId"),
						Value:       updateSuccessResolver.MustGetString("updatedValue"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *productattributepb.UpdateProductAttributeResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertEqual(t, 1, len(response.Data), "response data length")

				updated := response.Data[0]
				testutil.AssertStringEqual(t, updateSuccessResolver.MustGetString("validProductAttributeId"), updated.Id, "product attribute ID")
				testutil.AssertStringEqual(t, updateSuccessResolver.MustGetString("validProductId"), updated.ProductId, "product ID")
				testutil.AssertStringEqual(t, updateSuccessResolver.MustGetString("validAttributeId"), updated.AttributeId, "attribute ID")
				testutil.AssertStringEqual(t, updateSuccessResolver.MustGetString("updatedValue"), updated.Value, "updated value")
				testutil.AssertFieldSet(t, updated.DateCreated, "DateCreated")
				testutil.AssertFieldSet(t, updated.DateCreatedString, "DateCreatedString")
				testutil.AssertFieldSet(t, updated.DateModified, "DateModified")
				testutil.AssertFieldSet(t, updated.DateModifiedString, "DateModifiedString")
			},
		},
		{
			Name:     "WithTransaction",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-ATTRIBUTE-UPDATE-TRANSACTION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productattributepb.UpdateProductAttributeRequest {
				return &productattributepb.UpdateProductAttributeRequest{
					Data: &productattributepb.ProductAttribute{
						Id:          updateSuccessResolver.MustGetString("validProductAttributeId"),
						ProductId:   updateSuccessResolver.MustGetString("validProductId"),
						AttributeId: updateSuccessResolver.MustGetString("validAttributeId"),
						Value:       updateSuccessResolver.MustGetString("updatedValue"),
					},
				}
			},
			UseTransaction: true,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *productattributepb.UpdateProductAttributeResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				updated := response.Data[0]
				testutil.AssertStringEqual(t, updateSuccessResolver.MustGetString("updatedValue"), updated.Value, "updated value")
			},
		},
		{
			Name:     "Unauthorized",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-ATTRIBUTE-UPDATE-AUTHORIZATION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productattributepb.UpdateProductAttributeRequest {
				return &productattributepb.UpdateProductAttributeRequest{
					Data: &productattributepb.ProductAttribute{
						Id:          updateSuccessResolver.MustGetString("validProductAttributeId"),
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
			Assertions: func(t *testing.T, response *productattributepb.UpdateProductAttributeResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertAuthorizationError(t, err)
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-ATTRIBUTE-UPDATE-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productattributepb.UpdateProductAttributeRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "product_attribute.validation.request_required",
			Assertions: func(t *testing.T, response *productattributepb.UpdateProductAttributeResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilRequest(t, err)
			},
		},
		{
			Name:     "NilData",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-ATTRIBUTE-UPDATE-NIL-DATA-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productattributepb.UpdateProductAttributeRequest {
				return &productattributepb.UpdateProductAttributeRequest{Data: nil}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "product_attribute.validation.data_required",
			Assertions: func(t *testing.T, response *productattributepb.UpdateProductAttributeResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilData(t, err)
			},
		},
		{
			Name:     "NotFound",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-ATTRIBUTE-UPDATE-NOT-FOUND-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productattributepb.UpdateProductAttributeRequest {
				return &productattributepb.UpdateProductAttributeRequest{
					Data: &productattributepb.ProductAttribute{
						Id:          updateNotFoundResolver.MustGetString("invalidProductAttributeId"),
						ProductId:   updateNotFoundResolver.MustGetString("validProductId"),
						AttributeId: updateNotFoundResolver.MustGetString("validAttributeId"),
						Value:       updateNotFoundResolver.MustGetString("updatedValue"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "product_attribute.errors.not_found",
			ErrorTags:      map[string]any{"productAttributeId": updateNotFoundResolver.MustGetString("invalidProductAttributeId")},
			Assertions: func(t *testing.T, response *productattributepb.UpdateProductAttributeResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "not found")
			},
		},
		{
			Name:     "EmptyId",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-ATTRIBUTE-UPDATE-VALIDATION-EMPTY-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productattributepb.UpdateProductAttributeRequest {
				return &productattributepb.UpdateProductAttributeRequest{
					Data: &productattributepb.ProductAttribute{
						Id:          "",
						ProductId:   updateSuccessResolver.MustGetString("validProductId"),
						AttributeId: updateSuccessResolver.MustGetString("validAttributeId"),
						Value:       updateSuccessResolver.MustGetString("updatedValue"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "product_attribute.validation.id_required",
			Assertions: func(t *testing.T, response *productattributepb.UpdateProductAttributeResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty ID")
			},
		},
		{
			Name:     "EmptyValue",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-ATTRIBUTE-UPDATE-VALIDATION-EMPTY-VALUE-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productattributepb.UpdateProductAttributeRequest {
				return &productattributepb.UpdateProductAttributeRequest{
					Data: &productattributepb.ProductAttribute{
						Id:          updateSuccessResolver.MustGetString("validProductAttributeId"),
						ProductId:   updateSuccessResolver.MustGetString("validProductId"),
						AttributeId: updateSuccessResolver.MustGetString("validAttributeId"),
						Value:       "",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "product_attribute.validation.value_required",
			Assertions: func(t *testing.T, response *productattributepb.UpdateProductAttributeResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty value")
			},
		},
		{
			Name:     "InvalidProductId",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-ATTRIBUTE-UPDATE-VALIDATION-INVALID-PRODUCT-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productattributepb.UpdateProductAttributeRequest {
				return &productattributepb.UpdateProductAttributeRequest{
					Data: &productattributepb.ProductAttribute{
						Id:          updateSuccessResolver.MustGetString("validProductAttributeId"),
						ProductId:   "nonexistent-product-id",
						AttributeId: updateSuccessResolver.MustGetString("validAttributeId"),
						Value:       updateSuccessResolver.MustGetString("updatedValue"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "product_attribute.errors.product_not_found",
			Assertions: func(t *testing.T, response *productattributepb.UpdateProductAttributeResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "invalid product ID")
			},
		},
		{
			Name:     "InvalidAttributeId",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-ATTRIBUTE-UPDATE-VALIDATION-INVALID-ATTRIBUTE-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productattributepb.UpdateProductAttributeRequest {
				return &productattributepb.UpdateProductAttributeRequest{
					Data: &productattributepb.ProductAttribute{
						Id:          updateSuccessResolver.MustGetString("validProductAttributeId"),
						ProductId:   updateSuccessResolver.MustGetString("validProductId"),
						AttributeId: "nonexistent-attribute-id",
						Value:       updateSuccessResolver.MustGetString("updatedValue"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "product_attribute.errors.attribute_not_found",
			Assertions: func(t *testing.T, response *productattributepb.UpdateProductAttributeResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "invalid attribute ID")
			},
		},
		{
			Name:     "DataEnrichment",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-ATTRIBUTE-UPDATE-ENRICHMENT-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productattributepb.UpdateProductAttributeRequest {
				return &productattributepb.UpdateProductAttributeRequest{
					Data: &productattributepb.ProductAttribute{
						Id:          updateSuccessResolver.MustGetString("validProductAttributeId"),
						ProductId:   updateSuccessResolver.MustGetString("validProductId"),
						AttributeId: updateSuccessResolver.MustGetString("validAttributeId"),
						Value:       "Data Enrichment Test Value",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *productattributepb.UpdateProductAttributeResponse, err error, useCase interface{}, ctx context.Context) {
				updated := response.Data[0]
				testutil.AssertStringEqual(t, "Data Enrichment Test Value", updated.Value, "enriched value")
				testutil.AssertFieldSet(t, updated.DateModified, "DateModified")
				testutil.AssertFieldSet(t, updated.DateModifiedString, "DateModifiedString")
				testutil.AssertFieldSet(t, updated.DateCreated, "DateCreated")
				testutil.AssertFieldSet(t, updated.DateCreatedString, "DateCreatedString")
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
			useCase := createUpdateTestUseCaseWithAuth(businessType, tc.UseTransaction, tc.UseAuth)

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

func TestUpdateProductAttributeUseCase_Execute_TransactionFailure(t *testing.T) {
	testCode := "ESPYNA-TEST-PRODUCT-PRODUCT-ATTRIBUTE-UPDATE-TRANSACTION-FAILURE-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "TransactionFailure", false)

	// Create a use case with a failing transaction service
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	mockProductAttributeRepo := product.NewMockProductAttributeRepository(businessType)
	mockProductRepo := product.NewMockProductRepository(businessType)
	mockAttributeRepo := common.NewMockAttributeRepository(businessType)

	repositories := UpdateProductAttributeRepositories{
		ProductAttribute: mockProductAttributeRepo,
		Product:          mockProductRepo,
		Attribute:        mockAttributeRepo,
	}

	standardServices := testutil.CreateStandardServices(false, true)
	services := UpdateProductAttributeServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   mockDb.NewFailingMockTransactionService(),
		TranslationService:   standardServices.TranslationService,
	}

	useCase := NewUpdateProductAttributeUseCase(repositories, services)

	// Load test case using centralized TestCaseResolver
	resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "product_attribute", "UpdateProductAttribute_Success")
	testutil.AssertTestCaseLoad(t, err, "UpdateProductAttribute_Success")

	req := &productattributepb.UpdateProductAttributeRequest{
		Data: &productattributepb.ProductAttribute{
			Id:          resolver.MustGetString("validProductAttributeId"),
			ProductId:   resolver.MustGetString("validProductId"),
			AttributeId: resolver.MustGetString("validAttributeId"),
			Value:       resolver.MustGetString("updatedValue"),
		},
	}

	_, err = useCase.Execute(ctx, req)

	// Verify that a transaction error occurred
	testutil.AssertTransactionError(t, err)

	// Log completion of transaction failure test
	testutil.LogTestResult(t, testCode, "TransactionFailure", false, err)
}
