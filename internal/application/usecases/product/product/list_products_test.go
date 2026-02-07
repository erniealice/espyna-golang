//go:build mock_db && mock_auth

// Package product provides table-driven tests for the product listing use case.
//
// The tests cover various scenarios, including success, authorization, nil requests,
// validation, and business logic verification. Each test case is defined in a table
// with a specific test code, request setup, and assertions.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user").
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education").
//
// Usage:
//   - Run all tests: go test -tags="mock_db,mock_auth" ./...
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestListProductsUseCase_Execute_TableDriven
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-PRODUCT-PRODUCT-LIST-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-PRODUCT-PRODUCT-LIST-AUTHORIZATION-v1.0: Unauthorized
//   - ESPYNA-TEST-PRODUCT-PRODUCT-LIST-NIL-REQUEST-v1.0: NilRequest
//   - ESPYNA-TEST-PRODUCT-PRODUCT-LIST-VERIFY-DETAILS-v1.0: VerifyDetails
//   - ESPYNA-TEST-PRODUCT-PRODUCT-LIST-BUSINESS-LOGIC-v1.0: BusinessLogic
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
	"leapfor.xyz/espyna/internal/application/shared/testutil"
	mockProduct "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock/product"
	productpb "leapfor.xyz/esqyma/golang/v1/domain/product/product"
)

// Type alias for list products test cases
type ListProductsTestCase = testutil.GenericTestCase[*productpb.ListProductsRequest, *productpb.ListProductsResponse]

func createListTestUseCaseWithAuth(businessType string, shouldAuthorize bool) *ListProductsUseCase {
	mockProductRepo := mockProduct.NewMockProductRepository(businessType)

	repositories := ListProductsRepositories{
		Product: mockProductRepo,
	}

	standardServices := testutil.CreateStandardServices(false, shouldAuthorize)
	services := ListProductsServices{
		AuthorizationService: standardServices.AuthorizationService,
		TranslationService:   standardServices.TranslationService,
	}

	return NewListProductsUseCase(repositories, services)
}

func TestListProductsUseCase_Execute_TableDriven(t *testing.T) {
	businessType := testutil.GetTestBusinessType()

	// Load test data resolvers before defining test cases
	listSuccessResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "product", "ListProducts_Success")
	testutil.AssertTestCaseLoad(t, err, "ListProducts_Success")

	testCases := []ListProductsTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-LIST-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productpb.ListProductsRequest {
				return &productpb.ListProductsRequest{}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *productpb.ListProductsResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertNotNil(t, response.Data, "response data")
				expectedCount := listSuccessResolver.MustGetInt("expectedProductCount")
				testutil.AssertEqual(t, expectedCount, len(response.Data), "product count")
			},
		},
		{
			Name:     "Unauthorized",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-LIST-AUTHORIZATION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productpb.ListProductsRequest {
				return &productpb.ListProductsRequest{}
			},
			UseTransaction: false,
			UseAuth:        false,
			ExpectSuccess:  false,
			ExpectedError:  "product.errors.authorization_failed",
			Assertions: func(t *testing.T, response *productpb.ListProductsResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertAuthorizationError(t, err)
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-LIST-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productpb.ListProductsRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "product.validation.request_required",
			Assertions: func(t *testing.T, response *productpb.ListProductsResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilRequest(t, err)
			},
		},
		{
			Name:     "VerifyDetails",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-LIST-VERIFY-DETAILS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productpb.ListProductsRequest {
				return &productpb.ListProductsRequest{}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *productpb.ListProductsResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertNotNil(t, response.Data, "response data")

				// Verify specific products exist - check first few products
				if len(response.Data) >= 2 {
					// Check first product
					firstProduct := response.Data[0]
					testutil.AssertNonEmptyString(t, firstProduct.Id, "first product ID")
					testutil.AssertNonEmptyString(t, firstProduct.Name, "first product name")
					testutil.AssertTrue(t, firstProduct.Active, "first product active")

					// Check second product
					secondProduct := response.Data[1]
					testutil.AssertNonEmptyString(t, secondProduct.Id, "second product ID")
					testutil.AssertNonEmptyString(t, secondProduct.Name, "second product name")
					testutil.AssertTrue(t, secondProduct.Active, "second product active")
				}
			},
		},
		{
			Name:     "BusinessLogic",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-LIST-BUSINESS-LOGIC-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productpb.ListProductsRequest {
				return &productpb.ListProductsRequest{}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *productpb.ListProductsResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertNotNil(t, response.Data, "response data")

				// Verify all products are active (business rule)
				for _, product := range response.Data {
					testutil.AssertTrue(t, product.Active, "product active: "+product.Id)
					testutil.AssertNonEmptyString(t, product.Id, "product ID")
					testutil.AssertNonEmptyString(t, product.Name, "product name")
				}
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
			useCase := createListTestUseCaseWithAuth(businessType, tc.UseAuth)

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
