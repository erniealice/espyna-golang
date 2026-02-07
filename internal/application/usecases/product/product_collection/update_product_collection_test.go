//go:build mock_db && mock_auth

// Package product_collection provides table-driven tests for the product collection update use case.
//
// The tests cover various scenarios, including success, transaction handling,
// authorization, nil requests, validation errors, not found, business rule validation, and boundary conditions.
// Each test case is defined in a table with a specific test code, request setup,
// and assertions.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user").
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education").
//
// Usage:
//   - Run all tests: go test -tags="mock_db,mock_auth" ./...
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestUpdateProductCollectionUseCase_Execute_TableDriven
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-UPDATE-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-UPDATE-TRANSACTION-v1.0: WithTransaction
//   - ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-UPDATE-AUTHORIZATION-v1.0: Unauthorized
//   - ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-UPDATE-NIL-REQUEST-v1.0: NilRequest
//   - ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-UPDATE-NIL-DATA-v1.0: NilData
//   - ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-UPDATE-VALIDATION-EMPTY-ID-v1.0: EmptyId
//   - ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-UPDATE-VALIDATION-EMPTY-PRODUCT-ID-v1.0: EmptyProductId
//   - ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-UPDATE-VALIDATION-EMPTY-COLLECTION-ID-v1.0: EmptyCollectionId
//   - ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-UPDATE-VALIDATION-PRODUCT-ID-TOO-SHORT-v1.0: ProductIdTooShort
//   - ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-UPDATE-VALIDATION-COLLECTION-ID-TOO-SHORT-v1.0: CollectionIdTooShort
//   - ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-UPDATE-NOT-FOUND-v1.0: NotFound
//   - ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-UPDATE-TRANSACTION-FAILURE-v1.0: TransactionFailure
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

// Type alias for update product collection test cases
type UpdateProductCollectionTestCase = testutil.GenericTestCase[*productcollectionpb.UpdateProductCollectionRequest, *productcollectionpb.UpdateProductCollectionResponse]

func createUpdateTestUseCaseWithAuth(businessType string, supportsTransaction bool, shouldAuthorize bool) *UpdateProductCollectionUseCase {
	mockProductCollectionRepo := mockProduct.NewMockProductCollectionRepository(businessType)
	mockProductRepo := mockProduct.NewMockProductRepository(businessType)
	mockCollectionRepo := mockProduct.NewMockCollectionRepository(businessType)

	repositories := UpdateProductCollectionRepositories{
		ProductCollection: mockProductCollectionRepo,
		Product:           mockProductRepo,
		Collection:        mockCollectionRepo,
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, shouldAuthorize)
	services := UpdateProductCollectionServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
	}

	return NewUpdateProductCollectionUseCase(repositories, services)
}

func TestUpdateProductCollectionUseCase_Execute_TableDriven(t *testing.T) {
	businessType := testutil.GetTestBusinessType()

	// Load test data resolvers before defining test cases
	updateSuccessResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "product_collection", "UpdateProductCollection_Success")
	testutil.AssertTestCaseLoad(t, err, "UpdateProductCollection_Success")

	authorizationUnauthorizedResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "product_collection", "Authorization_Unauthorized")
	testutil.AssertTestCaseLoad(t, err, "Authorization_Unauthorized")

	updateNotFoundResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "product_collection", "UpdateProductCollection_NotFound")
	testutil.AssertTestCaseLoad(t, err, "UpdateProductCollection_NotFound")

	testCases := []UpdateProductCollectionTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-UPDATE-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productcollectionpb.UpdateProductCollectionRequest {
				return &productcollectionpb.UpdateProductCollectionRequest{
					Data: &productcollectionpb.ProductCollection{
						Id:           updateSuccessResolver.MustGetString("validProductCollectionId"),
						ProductId:    updateSuccessResolver.MustGetString("validProductId"),
						CollectionId: updateSuccessResolver.MustGetString("validCollectionId"),
						SortOrder:    int32(updateSuccessResolver.MustGetInt("updatedSortOrder")),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *productcollectionpb.UpdateProductCollectionResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertEqual(t, 1, len(response.Data), "response data length")
				updatedCollection := response.Data[0]
				testutil.AssertStringEqual(t, updateSuccessResolver.MustGetString("validProductId"), updatedCollection.ProductId, "product ID")
				testutil.AssertStringEqual(t, updateSuccessResolver.MustGetString("validCollectionId"), updatedCollection.CollectionId, "collection ID")
				testutil.AssertFieldSet(t, updatedCollection.DateModified, "DateModified")
				testutil.AssertFieldSet(t, updatedCollection.DateModifiedString, "DateModifiedString")
			},
		},
		{
			Name:     "WithTransaction",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-UPDATE-TRANSACTION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productcollectionpb.UpdateProductCollectionRequest {
				return &productcollectionpb.UpdateProductCollectionRequest{
					Data: &productcollectionpb.ProductCollection{
						Id:           updateSuccessResolver.MustGetString("validProductCollectionId"),
						ProductId:    updateSuccessResolver.MustGetString("validProductId"),
						CollectionId: updateSuccessResolver.MustGetString("validCollectionId"),
						SortOrder:    int32(updateSuccessResolver.MustGetInt("updatedSortOrder")),
					},
				}
			},
			UseTransaction: true,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *productcollectionpb.UpdateProductCollectionResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				updatedCollection := response.Data[0]
				testutil.AssertStringEqual(t, updateSuccessResolver.MustGetString("validProductId"), updatedCollection.ProductId, "product ID")
			},
		},
		{
			Name:     "Unauthorized",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-UPDATE-AUTHORIZATION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productcollectionpb.UpdateProductCollectionRequest {
				return &productcollectionpb.UpdateProductCollectionRequest{
					Data: &productcollectionpb.ProductCollection{
						Id:           authorizationUnauthorizedResolver.MustGetString("unauthorizedProductCollectionId"),
						ProductId:    authorizationUnauthorizedResolver.MustGetString("unauthorizedProductId"),
						CollectionId: authorizationUnauthorizedResolver.MustGetString("unauthorizedCollectionId"),
						SortOrder:    int32(authorizationUnauthorizedResolver.MustGetInt("unauthorizedSortOrder")),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        false,
			ExpectSuccess:  false,
			ExpectedError:  "product_collection.errors.authorization_failed",
			Assertions: func(t *testing.T, response *productcollectionpb.UpdateProductCollectionResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertAuthorizationError(t, err)
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-UPDATE-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productcollectionpb.UpdateProductCollectionRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "product_collection.validation.request_required",
			Assertions: func(t *testing.T, response *productcollectionpb.UpdateProductCollectionResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilRequest(t, err)
			},
		},
		{
			Name:     "NilData",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-UPDATE-NIL-DATA-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productcollectionpb.UpdateProductCollectionRequest {
				return &productcollectionpb.UpdateProductCollectionRequest{Data: nil}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "product_collection.validation.data_required",
			Assertions: func(t *testing.T, response *productcollectionpb.UpdateProductCollectionResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilData(t, err)
			},
		},
		{
			Name:     "EmptyId",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-UPDATE-VALIDATION-EMPTY-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productcollectionpb.UpdateProductCollectionRequest {
				return &productcollectionpb.UpdateProductCollectionRequest{
					Data: &productcollectionpb.ProductCollection{
						Id:           "",
						ProductId:    updateSuccessResolver.MustGetString("validProductId"),
						CollectionId: updateSuccessResolver.MustGetString("validCollectionId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "product_collection.validation.id_required",
			Assertions: func(t *testing.T, response *productcollectionpb.UpdateProductCollectionResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty ID")
			},
		},
		{
			Name:     "EmptyProductId",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-UPDATE-VALIDATION-EMPTY-PRODUCT-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productcollectionpb.UpdateProductCollectionRequest {
				return &productcollectionpb.UpdateProductCollectionRequest{
					Data: &productcollectionpb.ProductCollection{
						Id:           "test-id",
						ProductId:    "",
						CollectionId: "collection-g1-seahorse",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "product_collection.validation.product_id_required",
			Assertions: func(t *testing.T, response *productcollectionpb.UpdateProductCollectionResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty product ID")
			},
		},
		{
			Name:     "EmptyCollectionId",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-UPDATE-VALIDATION-EMPTY-COLLECTION-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productcollectionpb.UpdateProductCollectionRequest {
				return &productcollectionpb.UpdateProductCollectionRequest{
					Data: &productcollectionpb.ProductCollection{
						Id:           "test-id",
						ProductId:    "subject-math",
						CollectionId: "",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "product_collection.validation.collection_id_required",
			Assertions: func(t *testing.T, response *productcollectionpb.UpdateProductCollectionResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty collection ID")
			},
		},
		{
			Name:     "ProductIdTooShort",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-UPDATE-VALIDATION-PRODUCT-ID-TOO-SHORT-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productcollectionpb.UpdateProductCollectionRequest {
				return &productcollectionpb.UpdateProductCollectionRequest{
					Data: &productcollectionpb.ProductCollection{
						Id:           "product-collection-003",
						ProductId:    "abc", // Less than 5 characters
						CollectionId: "collection-g1-seahorse",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "product_collection.validation.product_id_min_length",
			Assertions: func(t *testing.T, response *productcollectionpb.UpdateProductCollectionResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "product ID too short")
			},
		},
		{
			Name:     "CollectionIdTooShort",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-UPDATE-VALIDATION-COLLECTION-ID-TOO-SHORT-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productcollectionpb.UpdateProductCollectionRequest {
				return &productcollectionpb.UpdateProductCollectionRequest{
					Data: &productcollectionpb.ProductCollection{
						Id:           updateSuccessResolver.MustGetString("validProductCollectionId"),
						ProductId:    updateSuccessResolver.MustGetString("validProductId"),
						CollectionId: "a", // Less than 2 characters
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "product_collection.validation.collection_id_min_length",
			Assertions: func(t *testing.T, response *productcollectionpb.UpdateProductCollectionResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "collection ID too short")
			},
		},
		{
			Name:     "NotFound",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-UPDATE-NOT-FOUND-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productcollectionpb.UpdateProductCollectionRequest {
				return &productcollectionpb.UpdateProductCollectionRequest{
					Data: &productcollectionpb.ProductCollection{
						Id:           updateNotFoundResolver.MustGetString("invalidProductCollectionId"),
						ProductId:    updateNotFoundResolver.MustGetString("validProductId"),
						CollectionId: updateNotFoundResolver.MustGetString("validCollectionId"),
						SortOrder:    int32(updateNotFoundResolver.MustGetInt("updatedSortOrder")),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "product_collection.errors.not_found",
			ErrorTags:      map[string]any{"productCollectionId": updateNotFoundResolver.MustGetString("invalidProductCollectionId")},
			Assertions: func(t *testing.T, response *productcollectionpb.UpdateProductCollectionResponse, err error, useCase interface{}, ctx context.Context) {
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

func TestUpdateProductCollectionUseCase_Execute_TransactionFailure(t *testing.T) {
	testCode := "ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-UPDATE-TRANSACTION-FAILURE-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "TransactionFailure", false)

	// Create a use case with a failing transaction service
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	mockProductCollectionRepo := mockProduct.NewMockProductCollectionRepository(businessType)
	mockProductRepo := mockProduct.NewMockProductRepository(businessType)
	mockCollectionRepo := mockProduct.NewMockCollectionRepository(businessType)

	repositories := UpdateProductCollectionRepositories{
		ProductCollection: mockProductCollectionRepo,
		Product:           mockProductRepo,
		Collection:        mockCollectionRepo,
	}

	standardServices := testutil.CreateStandardServices(false, true)
	services := UpdateProductCollectionServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   mockDb.NewFailingMockTransactionService(),
		TranslationService:   standardServices.TranslationService,
	}

	useCase := NewUpdateProductCollectionUseCase(repositories, services)

	// Load test case using centralized TestCaseResolver
	resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "product_collection", "UpdateProductCollection_Success")
	testutil.AssertTestCaseLoad(t, err, "UpdateProductCollection_Success")

	req := &productcollectionpb.UpdateProductCollectionRequest{
		Data: &productcollectionpb.ProductCollection{
			Id:           resolver.MustGetString("validProductCollectionId"),
			ProductId:    resolver.MustGetString("validProductId"),
			CollectionId: resolver.MustGetString("validCollectionId"),
			SortOrder:    int32(resolver.MustGetInt("updatedSortOrder")),
		},
	}

	_, err = useCase.Execute(ctx, req)

	// Verify that a transaction error occurred
	testutil.AssertTransactionError(t, err)

	// Log completion of transaction failure test
	testutil.LogTestResult(t, testCode, "TransactionFailure", false, err)
}
