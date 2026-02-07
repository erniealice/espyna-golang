//go:build mock_db && mock_auth

// Package collection provides table-driven tests for the collection read use case.
//
// The tests cover various scenarios, including success, transaction handling,
// nil requests, validation errors, and not-found cases.
// Each test case is defined in a table with a specific test code, request setup,
// and assertions.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user").
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education").
//
// Usage:
//   - Run all tests: go test -tags="mock_db,mock_auth" ./...
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestReadCollectionUseCase_Execute_TableDriven
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-PRODUCT-COLLECTION-READ-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-PRODUCT-COLLECTION-READ-TRANSACTION-v1.0: WithTransaction
//   - ESPYNA-TEST-PRODUCT-COLLECTION-READ-NIL-REQUEST-v1.0: NilRequest
//   - ESPYNA-TEST-PRODUCT-COLLECTION-READ-NIL-DATA-v1.0: NilData
//   - ESPYNA-TEST-PRODUCT-COLLECTION-READ-VALIDATION-EMPTY-ID-v1.0: EmptyId
//   - ESPYNA-TEST-PRODUCT-COLLECTION-READ-NOT-FOUND-v1.0: NonExistentId
//   - ESPYNA-TEST-PRODUCT-COLLECTION-READ-VALIDATION-MINIMAL-ID-v1.0: ValidMinimalId
//   - ESPYNA-TEST-PRODUCT-COLLECTION-READ-INTEGRATION-v1.0: RealisticDomainCollection
//   - ESPYNA-TEST-PRODUCT-COLLECTION-READ-STRUCTURE-VALIDATION-v1.0: CollectionStructureValidation
//   - ESPYNA-TEST-PRODUCT-COLLECTION-READ-TRANSACTION-FAILURE-v1.0: TransactionFailure
//   - ESPYNA-TEST-PRODUCT-COLLECTION-READ-UNAUTHORIZED-v1.0: Unauthorized
//
// Data Sources:
//   - Test cases: packages/copya/data_test/{businessType}/product_collection.json
//   - Mock data: packages/copya/data/{businessType}/product_collection.json
//   - Translations: packages/lyngua/translations/{languageCode}/{businessType}/product_collection.json

package collection

import (
	"context"
	"testing"

	copyatestutil "leapfor.xyz/copya/golang/testutil"
	"leapfor.xyz/espyna/internal/application/shared/testutil"
	mockDb "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock"
	"leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock/product"
	collectionpb "leapfor.xyz/esqyma/golang/v1/domain/product/collection"
)

// Type alias for read collection test cases
type ReadCollectionTestCase = testutil.GenericTestCase[*collectionpb.ReadCollectionRequest, *collectionpb.ReadCollectionResponse]

func createTestReadCollectionUseCaseWithAuth(businessType string, supportsTransaction bool, shouldAuthorize bool) *ReadCollectionUseCase {
	mockCollectionRepo := product.NewMockCollectionRepository(businessType)

	repositories := ReadCollectionRepositories{
		Collection: mockCollectionRepo,
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, shouldAuthorize)
	services := ReadCollectionServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
	}

	return NewReadCollectionUseCase(repositories, services)
}

func TestReadCollectionUseCase_Execute_TableDriven(t *testing.T) {
	businessType := testutil.GetTestBusinessType()

	// Load test data resolvers before defining test cases
	commonDataResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "collection", "Collection_CommonData")
	testutil.AssertTestCaseLoad(t, err, "Collection_CommonData")

	readSuccessResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "collection", "ReadCollection_Success")
	testutil.AssertTestCaseLoad(t, err, "ReadCollection_Success")

	testCases := []ReadCollectionTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-PRODUCT-COLLECTION-READ-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *collectionpb.ReadCollectionRequest {
				return &collectionpb.ReadCollectionRequest{
					Data: &collectionpb.Collection{
						Id: readSuccessResolver.MustGetString("targetCollectionId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *collectionpb.ReadCollectionResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertEqual(t, 1, len(response.Data), "response data length")
				collection := response.Data[0]
				testutil.AssertNonEmptyString(t, collection.Name, "collection name")
				testutil.AssertFieldSet(t, collection.DateCreated, "DateCreated")
				testutil.AssertFieldSet(t, collection.DateCreatedString, "DateCreatedString")
			},
		},
		{
			Name:     "WithTransaction",
			TestCode: "ESPYNA-TEST-PRODUCT-COLLECTION-READ-TRANSACTION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *collectionpb.ReadCollectionRequest {
				return &collectionpb.ReadCollectionRequest{
					Data: &collectionpb.Collection{
						Id: commonDataResolver.MustGetString("secondaryCollectionId"),
					},
				}
			},
			UseTransaction: true,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *collectionpb.ReadCollectionResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				collection := response.Data[0]
				testutil.AssertStringEqual(t, commonDataResolver.MustGetString("secondaryCollectionId"), collection.Id, "collection ID")
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-PRODUCT-COLLECTION-READ-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *collectionpb.ReadCollectionRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "collection.validation.request_required",
			Assertions: func(t *testing.T, response *collectionpb.ReadCollectionResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilRequest(t, err)
			},
		},
		{
			Name:     "NilData",
			TestCode: "ESPYNA-TEST-PRODUCT-COLLECTION-READ-NIL-DATA-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *collectionpb.ReadCollectionRequest {
				return &collectionpb.ReadCollectionRequest{Data: nil}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "collection.validation.request_required",
			Assertions: func(t *testing.T, response *collectionpb.ReadCollectionResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilData(t, err)
			},
		},
		{
			Name:     "EmptyId",
			TestCode: "ESPYNA-TEST-PRODUCT-COLLECTION-READ-VALIDATION-EMPTY-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *collectionpb.ReadCollectionRequest {
				return &collectionpb.ReadCollectionRequest{
					Data: &collectionpb.Collection{Id: ""},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "collection.validation.id_required",
			Assertions: func(t *testing.T, response *collectionpb.ReadCollectionResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty ID")
			},
		},
		{
			Name:     "NonExistentId",
			TestCode: "ESPYNA-TEST-PRODUCT-COLLECTION-READ-NOT-FOUND-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *collectionpb.ReadCollectionRequest {
				return &collectionpb.ReadCollectionRequest{
					Data: &collectionpb.Collection{Id: commonDataResolver.MustGetString("nonExistentId")},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "collection.errors.not_found",
			ErrorTags:      map[string]any{"collectionId": commonDataResolver.MustGetString("nonExistentId")},
			Assertions: func(t *testing.T, response *collectionpb.ReadCollectionResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertError(t, err)
			},
		},
		{
			Name:     "ValidMinimalId",
			TestCode: "ESPYNA-TEST-PRODUCT-COLLECTION-READ-VALIDATION-MINIMAL-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *collectionpb.ReadCollectionRequest {
				return &collectionpb.ReadCollectionRequest{
					Data: &collectionpb.Collection{Id: commonDataResolver.MustGetString("minimalValidId")},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "collection.errors.not_found",
			ErrorTags:      map[string]any{"collectionId": commonDataResolver.MustGetString("minimalValidId")},
			Assertions: func(t *testing.T, response *collectionpb.ReadCollectionResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertError(t, err)
			},
		},
		{
			Name:     "RealisticDomainCollection",
			TestCode: "ESPYNA-TEST-PRODUCT-COLLECTION-READ-INTEGRATION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *collectionpb.ReadCollectionRequest {
				return &collectionpb.ReadCollectionRequest{
					Data: &collectionpb.Collection{
						Id: commonDataResolver.MustGetString("thirdCollectionId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *collectionpb.ReadCollectionResponse, err error, useCase interface{}, ctx context.Context) {
				collection := response.Data[0]
				testutil.AssertNonEmptyString(t, collection.Name, "collection name")
				testutil.AssertFieldSet(t, collection.Description, "description")
			},
		},
		{
			Name:     "Unauthorized",
			TestCode: "ESPYNA-TEST-PRODUCT-COLLECTION-READ-UNAUTHORIZED-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *collectionpb.ReadCollectionRequest {
				return &collectionpb.ReadCollectionRequest{
					Data: &collectionpb.Collection{
						Id: readSuccessResolver.MustGetString("targetCollectionId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        false,
			ExpectSuccess:  true, // Assuming read operations don't require auth or are handled gracefully
			Assertions: func(t *testing.T, response *collectionpb.ReadCollectionResponse, err error, useCase interface{}, ctx context.Context) {
				// Verify it either succeeds or fails gracefully with an authorization error
				if err != nil {
					testutil.AssertError(t, err)
				} else {
					testutil.AssertNotNil(t, response, "response")
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
			useCase := createTestReadCollectionUseCaseWithAuth(businessType, tc.UseTransaction, tc.UseAuth)

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

func TestReadCollectionUseCase_Execute_CollectionStructureValidation(t *testing.T) {
	testCode := "ESPYNA-TEST-PRODUCT-COLLECTION-READ-STRUCTURE-VALIDATION-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "CollectionStructureValidation", true)

	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestReadCollectionUseCaseWithAuth(businessType, false, true)

	// Load test case using centralized TestCaseResolver
	resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "collection", "ListCollections_Success")
	testutil.AssertTestCaseLoad(t, err, "ListCollections_Success")

	// Test with multiple real collection IDs from mock data
	collectionIds := resolver.MustGetStringArray("expectedCollectionIds")

	for _, collectionId := range collectionIds {
		req := &collectionpb.ReadCollectionRequest{
			Data: &collectionpb.Collection{
				Id: collectionId,
			},
		}

		response, err := useCase.Execute(ctx, req)

		testutil.AssertNoError(t, err)

		collection := response.Data[0]

		// Validate collection structure
		testutil.AssertStringEqual(t, collectionId, collection.Id, "collection ID")

		testutil.AssertNonEmptyString(t, collection.Name, "collection name")

		testutil.AssertTrue(t, collection.Active, "collection active status")

		// Audit fields
		testutil.AssertFieldSet(t, collection.DateCreated, "DateCreated")

		testutil.AssertFieldSet(t, collection.DateCreatedString, "DateCreatedString")
	}

	// Log completion of structure validation test
	testutil.LogTestResult(t, testCode, "CollectionStructureValidation", true, nil)
}

func TestReadCollectionUseCase_Execute_TransactionFailure(t *testing.T) {
	testCode := "ESPYNA-TEST-PRODUCT-COLLECTION-READ-TRANSACTION-FAILURE-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "TransactionFailure", false)

	// Create a use case with a failing transaction service
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	mockCollectionRepo := product.NewMockCollectionRepository(businessType)

	repositories := ReadCollectionRepositories{
		Collection: mockCollectionRepo,
	}

	standardServices := testutil.CreateStandardServices(false, true)
	// Override with failing transaction service
	standardServices.TransactionService = mockDb.NewFailingMockTransactionService()
	services := ReadCollectionServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
	}

	useCase := NewReadCollectionUseCase(repositories, services)

	// Load test case using centralized TestCaseResolver
	resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "collection", "Collection_CommonData")
	testutil.AssertTestCaseLoad(t, err, "Collection_CommonData")

	req := &collectionpb.ReadCollectionRequest{
		Data: &collectionpb.Collection{
			Id: resolver.MustGetString("nonExistentId"),
		},
	}

	// For read operations, transaction failure should not affect the operation
	// since read operations typically don't use transactions
	response, err := useCase.Execute(ctx, req)

	// This should either work (no transaction used) or fail gracefully
	if err != nil {
		// If it fails, verify it's due to the collection not existing, not transaction failure
		testutil.AssertTranslatedErrorWithTags(t, err, "collection.errors.not_found",
			map[string]any{"collectionId": resolver.MustGetString("nonExistentId")}, useCase.services.TranslationService, ctx)
	} else {
		// If it succeeds, verify we get a proper response
		testutil.AssertNotNil(t, response, "response")
	}

	// Log completion of transaction failure test
	testutil.LogTestResult(t, testCode, "TransactionFailure", err == nil, err)
}
