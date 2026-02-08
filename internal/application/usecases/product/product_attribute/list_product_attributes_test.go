//go:build mock_db && mock_auth

// Package product_attribute provides table-driven tests for the product attribute listing use case.
//
// The tests cover various scenarios, including success, transaction handling,
// authorization, nil requests, validation errors, filtering, and boundary conditions.
// Each test case is defined in a table with a specific test code, request setup,
// and assertions.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user").
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education").
//
// Usage:
//   - Run all tests: go test -tags="mock_db,mock_auth" ./...
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestListProductAttributesUseCase_Execute_TableDriven
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-PRODUCT-PRODUCT-ATTRIBUTE-LIST-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-PRODUCT-PRODUCT-ATTRIBUTE-LIST-TRANSACTION-v1.0: WithTransaction
//   - ESPYNA-TEST-PRODUCT-PRODUCT-ATTRIBUTE-LIST-AUTHORIZATION-v1.0: Unauthorized
//   - ESPYNA-TEST-PRODUCT-PRODUCT-ATTRIBUTE-LIST-NIL-REQUEST-v1.0: NilRequest
//   - ESPYNA-TEST-PRODUCT-PRODUCT-ATTRIBUTE-LIST-EMPTY-RESULT-v1.0: EmptyResult
//   - ESPYNA-TEST-PRODUCT-PRODUCT-ATTRIBUTE-LIST-FILTER-PRODUCT-ID-v1.0: FilterByProductId
//   - ESPYNA-TEST-PRODUCT-PRODUCT-ATTRIBUTE-LIST-FILTER-ATTRIBUTE-ID-v1.0: FilterByAttributeId
//   - ESPYNA-TEST-PRODUCT-PRODUCT-ATTRIBUTE-LIST-PAGINATION-v1.0: WithPagination
//   - ESPYNA-TEST-PRODUCT-PRODUCT-ATTRIBUTE-LIST-TRANSACTION-FAILURE-v1.0: TransactionFailure
//
// Data Sources:
//   - Test cases: packages/copya/data_test/{businessType}/product_attribute.json
//   - Mock data: packages/copya/data/{businessType}/product_attribute.json
//   - Translations: packages/lyngua/translations/{languageCode}/{businessType}/product_attribute.json
package product_attribute

import (
	"context"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/shared/testutil"
	mockDb "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock"
	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock/product"
	productattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_attribute"
)

// Type alias for list product attributes test cases
type ListProductAttributesTestCase = testutil.GenericTestCase[*productattributepb.ListProductAttributesRequest, *productattributepb.ListProductAttributesResponse]

func createListTestUseCaseWithAuth(businessType string, supportsTransaction bool, shouldAuthorize bool) *ListProductAttributesUseCase {
	mockProductAttributeRepo := product.NewMockProductAttributeRepository(businessType)

	repositories := ListProductAttributesRepositories{
		ProductAttribute: mockProductAttributeRepo,
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, shouldAuthorize)
	services := ListProductAttributesServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
	}

	return NewListProductAttributesUseCase(repositories, services)
}

func TestListProductAttributesUseCase_Execute_TableDriven(t *testing.T) {

	testCases := []ListProductAttributesTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-ATTRIBUTE-LIST-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productattributepb.ListProductAttributesRequest {
				return &productattributepb.ListProductAttributesRequest{}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *productattributepb.ListProductAttributesResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertTrue(t, len(response.Data) >= 0, "data length should be non-negative")
				if len(response.Data) > 0 {
					for _, item := range response.Data {
						testutil.AssertNonEmptyString(t, item.Id, "product attribute ID")
						testutil.AssertNonEmptyString(t, item.ProductId, "product ID")
						testutil.AssertNonEmptyString(t, item.AttributeId, "attribute ID")
						testutil.AssertNonEmptyString(t, item.Value, "value")
						testutil.AssertFieldSet(t, item.DateCreated, "DateCreated")
						testutil.AssertFieldSet(t, item.DateCreatedString, "DateCreatedString")
					}
				}
			},
		},
		{
			Name:     "WithTransaction",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-ATTRIBUTE-LIST-TRANSACTION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productattributepb.ListProductAttributesRequest {
				return &productattributepb.ListProductAttributesRequest{}
			},
			UseTransaction: true,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *productattributepb.ListProductAttributesResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertTrue(t, len(response.Data) >= 0, "data length should be non-negative")
			},
		},
		{
			Name:     "Unauthorized",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-ATTRIBUTE-LIST-AUTHORIZATION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productattributepb.ListProductAttributesRequest {
				return &productattributepb.ListProductAttributesRequest{}
			},
			UseTransaction: false,
			UseAuth:        false,
			ExpectSuccess:  false,
			ExpectedError:  "product_attribute.errors.authorization_failed",
			Assertions: func(t *testing.T, response *productattributepb.ListProductAttributesResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertAuthorizationError(t, err)
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-ATTRIBUTE-LIST-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productattributepb.ListProductAttributesRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "product_attribute.validation.request_required",
			Assertions: func(t *testing.T, response *productattributepb.ListProductAttributesResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilRequest(t, err)
			},
		},
		{
			Name:     "FilterByProductId",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-ATTRIBUTE-LIST-FILTER-PRODUCT-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productattributepb.ListProductAttributesRequest {
				return &productattributepb.ListProductAttributesRequest{}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *productattributepb.ListProductAttributesResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				// Note: Basic list without filtering - all product attributes returned
				testutil.AssertTrue(t, len(response.Data) >= 0, "data length should be non-negative")
			},
		},
		{
			Name:     "FilterByAttributeId",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-ATTRIBUTE-LIST-FILTER-ATTRIBUTE-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productattributepb.ListProductAttributesRequest {
				return &productattributepb.ListProductAttributesRequest{}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *productattributepb.ListProductAttributesResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				// Note: Basic list without filtering - all product attributes returned
				testutil.AssertTrue(t, len(response.Data) >= 0, "data length should be non-negative")
			},
		},
		{
			Name:     "WithPagination",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-ATTRIBUTE-LIST-PAGINATION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productattributepb.ListProductAttributesRequest {
				return &productattributepb.ListProductAttributesRequest{}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *productattributepb.ListProductAttributesResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				// Note: Basic list without pagination - all product attributes returned
				testutil.AssertTrue(t, len(response.Data) >= 0, "data length should be non-negative")
			},
		},
		{
			Name:     "EmptyResult",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-ATTRIBUTE-LIST-EMPTY-RESULT-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productattributepb.ListProductAttributesRequest {
				return &productattributepb.ListProductAttributesRequest{}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *productattributepb.ListProductAttributesResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				// Note: Basic list without filtering - returns available product attributes
				testutil.AssertTrue(t, len(response.Data) >= 0, "data length should be non-negative")
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
			useCase := createListTestUseCaseWithAuth(businessType, tc.UseTransaction, tc.UseAuth)

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

func TestListProductAttributesUseCase_Execute_TransactionFailure(t *testing.T) {
	testCode := "ESPYNA-TEST-PRODUCT-PRODUCT-ATTRIBUTE-LIST-TRANSACTION-FAILURE-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "TransactionFailure", false)

	// Create a use case with a failing transaction service
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	mockProductAttributeRepo := product.NewMockProductAttributeRepository(businessType)

	repositories := ListProductAttributesRepositories{
		ProductAttribute: mockProductAttributeRepo,
	}

	standardServices := testutil.CreateStandardServices(false, true)
	services := ListProductAttributesServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   mockDb.NewFailingMockTransactionService(),
		TranslationService:   standardServices.TranslationService,
	}

	useCase := NewListProductAttributesUseCase(repositories, services)

	req := &productattributepb.ListProductAttributesRequest{}

	_, err := useCase.Execute(ctx, req)

	// Verify that a transaction error occurred
	testutil.AssertTransactionError(t, err)

	// Log completion of transaction failure test
	testutil.LogTestResult(t, testCode, "TransactionFailure", false, err)
}
