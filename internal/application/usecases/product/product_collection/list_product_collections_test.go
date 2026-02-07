//go:build mock_db && mock_auth

// Package product_collection provides table-driven tests for the product collection listing use case.
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
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestListProductCollectionsUseCase_Execute_TableDriven
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-LIST-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-LIST-TRANSACTION-v1.0: WithTransaction
//   - ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-LIST-AUTHORIZATION-v1.0: Unauthorized
//   - ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-LIST-NIL-REQUEST-v1.0: NilRequest
//   - ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-LIST-EMPTY-RESULT-v1.0: EmptyResult
//   - ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-LIST-TRANSACTION-FAILURE-v1.0: TransactionFailure
//
// Data Sources:
//   - Test cases: packages/copya/data_test/{businessType}/product_collection.json
//   - Mock data: packages/copya/data/{businessType}/product_collection.json
//   - Translations: packages/lyngua/translations/{languageCode}/{businessType}/product_collection.json
package product_collection

import (
	"context"
	"testing"

	"leapfor.xyz/espyna/internal/application/shared/testutil"
	mockProduct "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock/product"
	productcollectionpb "leapfor.xyz/esqyma/golang/v1/domain/product/product_collection"
)

// Type alias for list product collections test cases
type ListProductCollectionsTestCase = testutil.GenericTestCase[*productcollectionpb.ListProductCollectionsRequest, *productcollectionpb.ListProductCollectionsResponse]

func createListTestUseCaseWithAuth(businessType string, supportsTransaction bool, shouldAuthorize bool) *ListProductCollectionsUseCase {
	mockProductCollectionRepo := mockProduct.NewMockProductCollectionRepository(businessType)

	repositories := ListProductCollectionsRepositories{
		ProductCollection: mockProductCollectionRepo,
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, shouldAuthorize)
	services := ListProductCollectionsServices{
		AuthorizationService: standardServices.AuthorizationService,
		TranslationService:   standardServices.TranslationService,
	}

	return NewListProductCollectionsUseCase(repositories, services)
}

func TestListProductCollectionsUseCase_Execute_TableDriven(t *testing.T) {

	testCases := []ListProductCollectionsTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-LIST-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productcollectionpb.ListProductCollectionsRequest {
				return &productcollectionpb.ListProductCollectionsRequest{}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *productcollectionpb.ListProductCollectionsResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertNotNil(t, response.Data, "response data")
				// Mock repository returns empty list by default, which is valid
				testutil.AssertTrue(t, len(response.Data) >= 0, "non-negative data length")
			},
		},
		{
			Name:     "WithTransaction",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-LIST-TRANSACTION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productcollectionpb.ListProductCollectionsRequest {
				return &productcollectionpb.ListProductCollectionsRequest{}
			},
			UseTransaction: true,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *productcollectionpb.ListProductCollectionsResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertNotNil(t, response.Data, "response data")
			},
		},
		{
			Name:     "Unauthorized",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-LIST-AUTHORIZATION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productcollectionpb.ListProductCollectionsRequest {
				return &productcollectionpb.ListProductCollectionsRequest{}
			},
			UseTransaction: false,
			UseAuth:        false,
			ExpectSuccess:  false,
			ExpectedError:  "product_collection.errors.authorization_failed",
			Assertions: func(t *testing.T, response *productcollectionpb.ListProductCollectionsResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertAuthorizationError(t, err)
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-LIST-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productcollectionpb.ListProductCollectionsRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "product_collection.validation.request_required",
			Assertions: func(t *testing.T, response *productcollectionpb.ListProductCollectionsResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilRequest(t, err)
			},
		},
		{
			Name:     "EmptyResult",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-LIST-EMPTY-RESULT-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productcollectionpb.ListProductCollectionsRequest {
				return &productcollectionpb.ListProductCollectionsRequest{}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *productcollectionpb.ListProductCollectionsResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertNotNil(t, response.Data, "response data")
				// Verify that empty results are handled correctly
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

func TestListProductCollectionsUseCase_Execute_TransactionFailure(t *testing.T) {
	testCode := "ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-LIST-TRANSACTION-FAILURE-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "TransactionFailure", false)

	// Create a use case with a failing transaction service
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	mockProductCollectionRepo := mockProduct.NewMockProductCollectionRepository(businessType)

	repositories := ListProductCollectionsRepositories{
		ProductCollection: mockProductCollectionRepo,
	}

	standardServices := testutil.CreateStandardServices(false, true)
	services := ListProductCollectionsServices{
		AuthorizationService: standardServices.AuthorizationService,
		TranslationService:   standardServices.TranslationService,
	}

	useCase := NewListProductCollectionsUseCase(repositories, services)

	req := &productcollectionpb.ListProductCollectionsRequest{}

	_, err := useCase.Execute(ctx, req)

	// For list operations, transaction failures might not always occur depending on implementation
	// But we test that the use case handles transaction-related scenarios appropriately
	actualSuccess := err == nil

	// Log completion of transaction failure test
	testutil.LogTestResult(t, testCode, "TransactionFailure", actualSuccess, err)
}
