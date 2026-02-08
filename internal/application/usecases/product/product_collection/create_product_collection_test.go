//go:build mock_db && mock_auth

// Package product_collection provides table-driven tests for the product collection creation use case.
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
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestCreateProductCollectionUseCase_Execute_TableDriven
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-CREATE-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-CREATE-TRANSACTION-v1.0: WithTransaction
//   - ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-CREATE-AUTHORIZATION-v1.0: Unauthorized
//   - ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-CREATE-NIL-REQUEST-v1.0: NilRequest
//   - ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-CREATE-NIL-DATA-v1.0: NilData
//   - ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-CREATE-VALIDATION-EMPTY-PRODUCT-ID-v1.0: EmptyProductId
//   - ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-CREATE-VALIDATION-EMPTY-COLLECTION-ID-v1.0: EmptyCollectionId
//   - ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-CREATE-ENRICHMENT-v1.0: DataEnrichment
//   - ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-CREATE-TRANSACTION-FAILURE-v1.0: TransactionFailure
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
	"github.com/erniealice/espyna-golang/internal/application/shared/testutil"
	mockDb "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock"
	mockProduct "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock/product"
	productcollectionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_collection"
)

// Type alias for create product collection test cases
type CreateProductCollectionTestCase = testutil.GenericTestCase[*productcollectionpb.CreateProductCollectionRequest, *productcollectionpb.CreateProductCollectionResponse]

func createTestUseCaseWithAuth(businessType string, supportsTransaction bool, shouldAuthorize bool) *CreateProductCollectionUseCase {
	mockProductCollectionRepo := mockProduct.NewMockProductCollectionRepository(businessType)
	mockProductRepo := mockProduct.NewMockProductRepository(businessType)
	mockCollectionRepo := mockProduct.NewMockCollectionRepository(businessType)

	repositories := CreateProductCollectionRepositories{
		ProductCollection: mockProductCollectionRepo,
		Product:           mockProductRepo,
		Collection:        mockCollectionRepo,
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, shouldAuthorize)
	services := CreateProductCollectionServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
		IDService:            standardServices.IDService,
	}

	return NewCreateProductCollectionUseCase(repositories, services)
}

func TestCreateProductCollectionUseCase_Execute_TableDriven(t *testing.T) {
	businessType := testutil.GetTestBusinessType()

	// Load test data resolvers before defining test cases
	createSuccessResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "product_collection", "CreateProductCollection_Success")
	testutil.AssertTestCaseLoad(t, err, "CreateProductCollection_Success")

	authorizationUnauthorizedResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "product_collection", "Authorization_Unauthorized")
	testutil.AssertTestCaseLoad(t, err, "Authorization_Unauthorized")

	validationErrorEmptyProductIdResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "product_collection", "ValidationError_EmptyProductId")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_EmptyProductId")

	validationErrorEmptyCollectionIdResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "product_collection", "ValidationError_EmptyCollectionId")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_EmptyCollectionId")

	testCases := []CreateProductCollectionTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-CREATE-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productcollectionpb.CreateProductCollectionRequest {
				return &productcollectionpb.CreateProductCollectionRequest{
					Data: &productcollectionpb.ProductCollection{
						ProductId:    createSuccessResolver.MustGetString("newProductCollectionProductId"),
						CollectionId: createSuccessResolver.MustGetString("newProductCollectionCollectionId"),
						SortOrder:    int32(createSuccessResolver.MustGetInt("newProductCollectionSortOrder")),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *productcollectionpb.CreateProductCollectionResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertEqual(t, 1, len(response.Data), "response data length")
				createdCollection := response.Data[0]
				testutil.AssertStringEqual(t, createSuccessResolver.MustGetString("newProductCollectionProductId"), createdCollection.ProductId, "product ID")
				testutil.AssertStringEqual(t, createSuccessResolver.MustGetString("newProductCollectionCollectionId"), createdCollection.CollectionId, "collection ID")
				testutil.AssertNonEmptyString(t, createdCollection.Id, "product collection ID")
				testutil.AssertTrue(t, createdCollection.Active, "product collection active status")
				testutil.AssertFieldSet(t, createdCollection.DateCreated, "DateCreated")
				testutil.AssertFieldSet(t, createdCollection.DateCreatedString, "DateCreatedString")
			},
		},
		{
			Name:     "WithTransaction",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-CREATE-TRANSACTION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productcollectionpb.CreateProductCollectionRequest {
				return &productcollectionpb.CreateProductCollectionRequest{
					Data: &productcollectionpb.ProductCollection{
						ProductId:    createSuccessResolver.MustGetString("newProductCollectionProductId"),
						CollectionId: createSuccessResolver.MustGetString("newProductCollectionCollectionId"),
						SortOrder:    int32(createSuccessResolver.MustGetInt("newProductCollectionSortOrder")),
					},
				}
			},
			UseTransaction: true,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *productcollectionpb.CreateProductCollectionResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				createdCollection := response.Data[0]
				testutil.AssertStringEqual(t, createSuccessResolver.MustGetString("newProductCollectionProductId"), createdCollection.ProductId, "product ID")
				testutil.AssertStringEqual(t, createSuccessResolver.MustGetString("newProductCollectionCollectionId"), createdCollection.CollectionId, "collection ID")
			},
		},
		{
			Name:     "Unauthorized",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-CREATE-AUTHORIZATION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productcollectionpb.CreateProductCollectionRequest {
				return &productcollectionpb.CreateProductCollectionRequest{
					Data: &productcollectionpb.ProductCollection{
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
			Assertions: func(t *testing.T, response *productcollectionpb.CreateProductCollectionResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertAuthorizationError(t, err)
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-CREATE-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productcollectionpb.CreateProductCollectionRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "product_collection.validation.request_required",
			Assertions: func(t *testing.T, response *productcollectionpb.CreateProductCollectionResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilRequest(t, err)
			},
		},
		{
			Name:     "NilData",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-CREATE-NIL-DATA-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productcollectionpb.CreateProductCollectionRequest {
				return &productcollectionpb.CreateProductCollectionRequest{Data: nil}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "product_collection.validation.data_required",
			Assertions: func(t *testing.T, response *productcollectionpb.CreateProductCollectionResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilData(t, err)
			},
		},
		{
			Name:     "EmptyProductId",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-CREATE-VALIDATION-EMPTY-PRODUCT-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productcollectionpb.CreateProductCollectionRequest {
				return &productcollectionpb.CreateProductCollectionRequest{
					Data: &productcollectionpb.ProductCollection{
						ProductId:    validationErrorEmptyProductIdResolver.MustGetString("emptyProductId"),
						CollectionId: validationErrorEmptyProductIdResolver.MustGetString("validCollectionId"),
						SortOrder:    int32(validationErrorEmptyProductIdResolver.MustGetInt("validSortOrder")),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "product_collection.validation.product_id_required",
			Assertions: func(t *testing.T, response *productcollectionpb.CreateProductCollectionResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty product ID")
			},
		},
		{
			Name:     "EmptyCollectionId",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-CREATE-VALIDATION-EMPTY-COLLECTION-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productcollectionpb.CreateProductCollectionRequest {
				return &productcollectionpb.CreateProductCollectionRequest{
					Data: &productcollectionpb.ProductCollection{
						ProductId:    validationErrorEmptyCollectionIdResolver.MustGetString("validProductId"),
						CollectionId: validationErrorEmptyCollectionIdResolver.MustGetString("emptyCollectionId"),
						SortOrder:    int32(validationErrorEmptyCollectionIdResolver.MustGetInt("validSortOrder")),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "product_collection.validation.collection_id_required",
			Assertions: func(t *testing.T, response *productcollectionpb.CreateProductCollectionResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty collection ID")
			},
		},
		{
			Name:     "DataEnrichment",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-CREATE-ENRICHMENT-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productcollectionpb.CreateProductCollectionRequest {
				return &productcollectionpb.CreateProductCollectionRequest{
					Data: &productcollectionpb.ProductCollection{
						ProductId:    createSuccessResolver.MustGetString("newProductCollectionProductId"),
						CollectionId: createSuccessResolver.MustGetString("newProductCollectionCollectionId"),
						SortOrder:    int32(createSuccessResolver.MustGetInt("newProductCollectionSortOrder")),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *productcollectionpb.CreateProductCollectionResponse, err error, useCase interface{}, ctx context.Context) {
				createdCollection := response.Data[0]
				testutil.AssertNonEmptyString(t, createdCollection.Id, "generated ID")
				testutil.AssertFieldSet(t, createdCollection.DateCreated, "DateCreated")
				testutil.AssertFieldSet(t, createdCollection.DateCreatedString, "DateCreatedString")
				testutil.AssertTrue(t, createdCollection.Active, "Active")
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

func TestCreateProductCollectionUseCase_Execute_TransactionFailure(t *testing.T) {
	testCode := "ESPYNA-TEST-PRODUCT-PRODUCT-COLLECTION-CREATE-TRANSACTION-FAILURE-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "TransactionFailure", false)

	// Create a use case with a failing transaction service
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	mockProductCollectionRepo := mockProduct.NewMockProductCollectionRepository(businessType)
	mockProductRepo := mockProduct.NewMockProductRepository(businessType)
	mockCollectionRepo := mockProduct.NewMockCollectionRepository(businessType)

	repositories := CreateProductCollectionRepositories{
		ProductCollection: mockProductCollectionRepo,
		Product:           mockProductRepo,
		Collection:        mockCollectionRepo,
	}

	standardServices := testutil.CreateStandardServices(false, true)
	services := CreateProductCollectionServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   mockDb.NewFailingMockTransactionService(),
		TranslationService:   standardServices.TranslationService,
		IDService:            standardServices.IDService,
	}

	useCase := NewCreateProductCollectionUseCase(repositories, services)

	// Load test case using centralized TestCaseResolver
	resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "product_collection", "CreateProductCollection_Success")
	testutil.AssertTestCaseLoad(t, err, "CreateProductCollection_Success")

	req := &productcollectionpb.CreateProductCollectionRequest{
		Data: &productcollectionpb.ProductCollection{
			ProductId:    resolver.MustGetString("newProductCollectionProductId"),
			CollectionId: resolver.MustGetString("newProductCollectionCollectionId"),
			SortOrder:    int32(resolver.MustGetInt("newProductCollectionSortOrder")),
		},
	}

	_, err = useCase.Execute(ctx, req)

	// Verify that a transaction error occurred
	testutil.AssertTransactionError(t, err)

	// Log completion of transaction failure test
	testutil.LogTestResult(t, testCode, "TransactionFailure", false, err)
}
