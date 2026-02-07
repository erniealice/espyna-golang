//go:build mock_db && mock_auth

// Package product_collection provides table-driven tests for the product collection read use case.
//
// The tests cover various scenarios, including success, transaction handling,
// authorization, nil requests, validation errors, not found, and boundary conditions.
// Each test case is defined in a table with a specific test code, request setup,
// and assertions.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user").
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education").
//
// Usage:
//   - Run all tests: go test -tags="mock_db,mock_auth" ./...
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestReadProductCollectionUseCase_Execute_TableDriven
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-READ-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-READ-TRANSACTION-v1.0: WithTransaction
//   - ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-READ-AUTHORIZATION-v1.0: Unauthorized
//   - ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-READ-NIL-REQUEST-v1.0: NilRequest
//   - ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-READ-NIL-DATA-v1.0: NilData
//   - ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-READ-VALIDATION-EMPTY-ID-v1.0: EmptyId
//   - ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-READ-NOT-FOUND-v1.0: NotFound
//   - ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-READ-TRANSACTION-FAILURE-v1.0: TransactionFailure
//
// Data Sources:
//   - Test cases: packages/copya/data_test/{businessType}/product_collection.json
//   - Mock data: packages/copya/data/{businessType}/product_collection.json
//   - Translations: packages/lyngua/translations/{languageCode}/{businessType}/product_collection.json
package product_collection

import (
	"context"
	"testing"

	copyatestutil "leapfor.xyz/copya/golang/testutil"
	"leapfor.xyz/espyna/internal/application/shared/testutil"
	mockDb "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock"
	mockProduct "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock/product"
	productcollectionpb "leapfor.xyz/esqyma/golang/v1/domain/product/product_collection"
)

// Type alias for read product collection test cases
type ReadProductCollectionTestCase = testutil.GenericTestCase[*productcollectionpb.ReadProductCollectionRequest, *productcollectionpb.ReadProductCollectionResponse]

func createReadTestUseCaseWithAuth(businessType string, supportsTransaction bool, shouldAuthorize bool) *ReadProductCollectionUseCase {
	mockProductCollectionRepo := mockProduct.NewMockProductCollectionRepository(businessType)

	repositories := ReadProductCollectionRepositories{
		ProductCollection: mockProductCollectionRepo,
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, shouldAuthorize)
	services := ReadProductCollectionServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
	}

	return NewReadProductCollectionUseCase(repositories, services)
}

func TestReadProductCollectionUseCase_Execute_TableDriven(t *testing.T) {
	businessType := testutil.GetTestBusinessType()

	// Load test data resolvers before defining test cases
	readNotFoundResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "product_collection", "ReadProductCollection_NotFound")
	testutil.AssertTestCaseLoad(t, err, "ReadProductCollection_NotFound")

	testCases := []ReadProductCollectionTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-READ-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productcollectionpb.ReadProductCollectionRequest {
				return &productcollectionpb.ReadProductCollectionRequest{
					Data: &productcollectionpb.ProductCollection{
						Id: "product-collection-001", // Pre-loaded in mock data
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *productcollectionpb.ReadProductCollectionResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertEqual(t, 1, len(response.Data), "response data length")
				readCollection := response.Data[0]
				testutil.AssertStringEqual(t, "product-collection-001", readCollection.Id, "product collection ID")
				testutil.AssertStringEqual(t, "subject-math", readCollection.ProductId, "product ID")
				testutil.AssertStringEqual(t, "collection-g1-seahorse", readCollection.CollectionId, "collection ID")
			},
		},
		{
			Name:     "WithTransaction",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-READ-TRANSACTION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productcollectionpb.ReadProductCollectionRequest {
				return &productcollectionpb.ReadProductCollectionRequest{
					Data: &productcollectionpb.ProductCollection{
						Id: "product-collection-001",
					},
				}
			},
			UseTransaction: true,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *productcollectionpb.ReadProductCollectionResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				readCollection := response.Data[0]
				testutil.AssertStringEqual(t, "product-collection-001", readCollection.Id, "product collection ID")
			},
		},
		{
			Name:     "Unauthorized",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-READ-AUTHORIZATION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productcollectionpb.ReadProductCollectionRequest {
				return &productcollectionpb.ReadProductCollectionRequest{
					Data: &productcollectionpb.ProductCollection{
						Id: "product-collection-001",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        false,
			ExpectSuccess:  false,
			ExpectedError:  "product_collection.errors.authorization_failed",
			Assertions: func(t *testing.T, response *productcollectionpb.ReadProductCollectionResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertAuthorizationError(t, err)
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-READ-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productcollectionpb.ReadProductCollectionRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "product_collection.validation.request_required",
			Assertions: func(t *testing.T, response *productcollectionpb.ReadProductCollectionResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilRequest(t, err)
			},
		},
		{
			Name:     "NilData",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-READ-NIL-DATA-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productcollectionpb.ReadProductCollectionRequest {
				return &productcollectionpb.ReadProductCollectionRequest{Data: nil}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "product_collection.validation.data_required",
			Assertions: func(t *testing.T, response *productcollectionpb.ReadProductCollectionResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilData(t, err)
			},
		},
		{
			Name:     "EmptyId",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-READ-VALIDATION-EMPTY-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productcollectionpb.ReadProductCollectionRequest {
				return &productcollectionpb.ReadProductCollectionRequest{
					Data: &productcollectionpb.ProductCollection{
						Id: "",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "product_collection.validation.id_required",
			Assertions: func(t *testing.T, response *productcollectionpb.ReadProductCollectionResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty ID")
			},
		},
		{
			Name:     "NotFound",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-READ-NOT-FOUND-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productcollectionpb.ReadProductCollectionRequest {
				return &productcollectionpb.ReadProductCollectionRequest{
					Data: &productcollectionpb.ProductCollection{
						Id: readNotFoundResolver.MustGetString("nonExistentProductCollectionId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "product_collection.errors.not_found",
			ErrorTags:      map[string]any{"productCollectionId": readNotFoundResolver.MustGetString("nonExistentProductCollectionId")},
			Assertions: func(t *testing.T, response *productcollectionpb.ReadProductCollectionResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "not found")
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
			useCase := createReadTestUseCaseWithAuth(businessType, tc.UseTransaction, tc.UseAuth)

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

func TestReadProductCollectionUseCase_Execute_TransactionFailure(t *testing.T) {
	testCode := "ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-READ-TRANSACTION-FAILURE-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "TransactionFailure", false)

	// Create a use case with a failing transaction service
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	mockProductCollectionRepo := mockProduct.NewMockProductCollectionRepository(businessType)

	repositories := ReadProductCollectionRepositories{
		ProductCollection: mockProductCollectionRepo,
	}

	standardServices := testutil.CreateStandardServices(false, true)
	services := ReadProductCollectionServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   mockDb.NewFailingMockTransactionService(),
		TranslationService:   standardServices.TranslationService,
	}

	useCase := NewReadProductCollectionUseCase(repositories, services)

	req := &productcollectionpb.ReadProductCollectionRequest{
		Data: &productcollectionpb.ProductCollection{
			Id: "product-collection-001",
		},
	}

	_, err := useCase.Execute(ctx, req)

	// Verify that a transaction error occurred
	testutil.AssertTransactionError(t, err)

	// Log completion of transaction failure test
	testutil.LogTestResult(t, testCode, "TransactionFailure", false, err)
}
