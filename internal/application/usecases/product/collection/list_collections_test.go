//go:build mock_db && mock_auth

// Package collection provides table-driven tests for the collection listing use case.
//
// The tests cover various scenarios, including success, transaction handling,
// authorization, nil requests, and collection details verification.
// Each test case is defined in a table with a specific test code, request setup,
// and assertions.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user").
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education").
//
// Usage:
//   - Run all tests: go test -tags="mock_db,mock_auth" ./...
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestListCollectionsUseCase_Execute_TableDriven
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-PRODUCT-COLLECTION-LIST-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-PRODUCT-COLLECTION-LIST-TRANSACTION-SUCCESS-v1.0: WithTransactionSuccess
//   - ESPYNA-TEST-PRODUCT-COLLECTION-LIST-AUTHORIZATION-DENIED-v1.0: AuthorizationDenied
//   - ESPYNA-TEST-PRODUCT-COLLECTION-LIST-NIL-REQUEST-v1.0: NilRequest
//   - ESPYNA-TEST-PRODUCT-COLLECTION-LIST-VERIFY-DETAILS-v1.0: VerifyCollectionDetails
//   - ESPYNA-TEST-PRODUCT-COLLECTION-LIST-EMPTY-COLLECTION-v1.0: EmptyCollection
//   - ESPYNA-TEST-PRODUCT-COLLECTION-LIST-BUSINESS-LOGIC-VALIDATION-v1.0: BusinessLogicValidation
//
// Data Sources:
//   - Test cases: packages/copya/data_test/{businessType}/collection.json
//   - Mock data: packages/copya/data/{businessType}/collection.json
//   - Translations: packages/lyngua/translations/{languageCode}/{businessType}/collection.json
package collection

import (
	"context"
	"testing"

	copyatestutil "leapfor.xyz/copya/golang/testutil"
	"github.com/erniealice/espyna-golang/internal/application/shared/testutil"
	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock/product"
	collectionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/collection"
)

// Type alias for list collections test cases
type ListCollectionsTestCase = testutil.GenericTestCase[*collectionpb.ListCollectionsRequest, *collectionpb.ListCollectionsResponse]

func createListTestUseCaseWithAuth(businessType string, supportsTransaction bool, shouldAuthorize bool, repoOptions ...product.CollectionRepositoryOption) *ListCollectionsUseCase {
	mockRepo := product.NewMockCollectionRepository(businessType, repoOptions...)

	repositories := ListCollectionsRepositories{
		Collection: mockRepo,
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, shouldAuthorize)
	services := ListCollectionsServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
	}

	return NewListCollectionsUseCase(repositories, services)
}

func TestListCollectionsUseCase_Execute_TableDriven(t *testing.T) {
	businessType := testutil.GetTestBusinessType()

	// Load test data resolvers before defining test cases
	listSuccessResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "collection", "ListCollections_Success")
	testutil.AssertTestCaseLoad(t, err, "ListCollections_Success")

	testCases := []ListCollectionsTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-PRODUCT-COLLECTION-LIST-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *collectionpb.ListCollectionsRequest {
				return &collectionpb.ListCollectionsRequest{}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *collectionpb.ListCollectionsResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertNotNil(t, response.Data, "response data")
				expectedCount := listSuccessResolver.MustGetInt("expectedCollectionCount")
				testutil.AssertEqual(t, expectedCount, len(response.Data), "collection count")
				expectedCollectionIds := listSuccessResolver.MustGetStringArray("expectedCollectionIds")
				collectionIds := make(map[string]bool)
				for _, col := range response.Data {
					collectionIds[col.Id] = true
				}
				for _, expectedId := range expectedCollectionIds {
					testutil.AssertTrue(t, collectionIds[expectedId], "expected collection '"+expectedId+"' found")
				}
			},
		},
		{
			Name:     "WithTransactionSuccess",
			TestCode: "ESPYNA-TEST-PRODUCT-COLLECTION-LIST-TRANSACTION-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *collectionpb.ListCollectionsRequest {
				return &collectionpb.ListCollectionsRequest{}
			},
			UseTransaction: true,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *collectionpb.ListCollectionsResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				expectedCount := listSuccessResolver.MustGetInt("expectedCollectionCount")
				testutil.AssertEqual(t, expectedCount, len(response.Data), "collection count with transaction")
			},
		},
		{
			Name:     "AuthorizationDenied",
			TestCode: "ESPYNA-TEST-PRODUCT-COLLECTION-LIST-AUTHORIZATION-DENIED-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *collectionpb.ListCollectionsRequest {
				return &collectionpb.ListCollectionsRequest{}
			},
			UseTransaction: false,
			UseAuth:        false,
			ExpectSuccess:  false,
			ExpectedError:  "collection.errors.authorization_failed",
			Assertions: func(t *testing.T, response *collectionpb.ListCollectionsResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertAuthorizationError(t, err)
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-PRODUCT-COLLECTION-LIST-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *collectionpb.ListCollectionsRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "collection.validation.request_required",
			Assertions: func(t *testing.T, response *collectionpb.ListCollectionsResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilRequest(t, err)
			},
		},
		{
			Name:     "VerifyCollectionDetails",
			TestCode: "ESPYNA-TEST-PRODUCT-COLLECTION-LIST-VERIFY-DETAILS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *collectionpb.ListCollectionsRequest {
				return &collectionpb.ListCollectionsRequest{}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *collectionpb.ListCollectionsResponse, err error, useCase interface{}, ctx context.Context) {
				resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, testutil.GetTestBusinessType(), "collection", "ListCollections_VerifyDetails")
				testutil.AssertTestCaseLoad(t, err, "ListCollections_VerifyDetails")
				verificationTargetsRaw := resolver.GetTestCase().DataReferences["verificationTargets"]
				verificationTargets := testutil.AssertArray(t, verificationTargetsRaw, "verificationTargets")

				for _, targetInterface := range verificationTargets {
					target := targetInterface.(map[string]interface{})
					targetId := target["id"].(string)
					expectedName := target["expectedName"].(string)
					expectedActive := target["expectedActive"].(bool)

					// Find the collection in the response
					var foundCol *collectionpb.Collection
					for _, col := range response.Data {
						if col.Id == targetId {
							foundCol = col
							break
						}
					}

					testutil.AssertNotNil(t, foundCol, targetId+" collection")
					if foundCol != nil {
						testutil.AssertStringEqual(t, expectedName, foundCol.Name, targetId+" collection name")
						testutil.AssertTrue(t, foundCol.Active == expectedActive, targetId+" collection active")
					}
				}
			},
		},
		{
			Name:     "EmptyCollection",
			TestCode: "ESPYNA-TEST-PRODUCT-COLLECTION-LIST-EMPTY-COLLECTION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *collectionpb.ListCollectionsRequest {
				return &collectionpb.ListCollectionsRequest{}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *collectionpb.ListCollectionsResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertNotNil(t, response.Data, "response data")
				testutil.AssertEqual(t, 0, len(response.Data), "empty collection count")
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

			// Handle special case for EmptyCollection test
			var repoOptions []product.CollectionRepositoryOption
			if tc.Name == "EmptyCollection" {
				repoOptions = []product.CollectionRepositoryOption{product.WithoutInitialData()}
			}
			useCase := createListTestUseCaseWithAuth(businessType, tc.UseTransaction, tc.UseAuth, repoOptions...)

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

func TestListCollectionsUseCase_Execute_VerifyCollectionDetails(t *testing.T) {
	testCode := "ESPYNA-TEST-PRODUCT-COLLECTION-LIST-VERIFY-DETAILS-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "VerifyCollectionDetails", true)

	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createListTestUseCaseWithAuth(businessType, false, true)

	// Load test case using centralized TestCaseResolver
	resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "collection", "ListCollections_VerifyDetails")
	testutil.AssertTestCaseLoad(t, err, "ListCollections_VerifyDetails")

	req := &collectionpb.ListCollectionsRequest{}

	response, err := useCase.Execute(ctx, req)

	testutil.AssertNoError(t, err)

	testutil.AssertNotNil(t, response, "response")
	testutil.AssertGreaterThan(t, len(response.Data), 0, "collections count")

	// Find and verify specific collections using test data
	verificationTargetsRaw := resolver.GetTestCase().DataReferences["verificationTargets"]
	verificationTargets := testutil.AssertArray(t, verificationTargetsRaw, "verificationTargets")

	for _, targetInterface := range verificationTargets {
		target := targetInterface.(map[string]interface{})
		targetId := target["id"].(string)
		expectedName := target["expectedName"].(string)
		expectedActive := target["expectedActive"].(bool)

		// Find the collection in the response
		var foundCol *collectionpb.Collection
		for _, col := range response.Data {
			if col.Id == targetId {
				foundCol = col
				break
			}
		}

		testutil.AssertNotNil(t, foundCol, targetId+" collection")
		if foundCol != nil {
			testutil.AssertStringEqual(t, expectedName, foundCol.Name, targetId+" collection name")
			testutil.AssertTrue(t, foundCol.Active == expectedActive, targetId+" collection active")
		}
	}

	// Log test completion with result
	testutil.LogTestResult(t, testCode, "VerifyCollectionDetails", true, nil)
}

func TestListCollectionsUseCase_Execute_EmptyCollectionTest(t *testing.T) {
	testCode := "ESPYNA-TEST-PRODUCT-COLLECTION-LIST-EMPTY-COLLECTION-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "EmptyCollectionTest", true)

	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createListTestUseCaseWithAuth(businessType, false, true, product.WithoutInitialData())

	req := &collectionpb.ListCollectionsRequest{}

	response, err := useCase.Execute(ctx, req)

	testutil.AssertNoError(t, err)

	testutil.AssertNotNil(t, response, "response")
	testutil.AssertTrue(t, response.Success, "response success")
	testutil.AssertEqual(t, 0, len(response.Data), "empty collections count")

	// Log test completion with result
	testutil.LogTestResult(t, testCode, "EmptyCollectionTest", true, nil)
}

func TestListCollectionsUseCase_Execute_BusinessLogicValidation(t *testing.T) {
	testCode := "ESPYNA-TEST-PRODUCT-COLLECTION-LIST-BUSINESS-LOGIC-VALIDATION-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "BusinessLogicValidation", true)

	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()

	testCases := []struct {
		name           string
		request        *collectionpb.ListCollectionsRequest
		expectError    bool
		minCollections int
	}{
		{
			name:           "ValidRequest",
			request:        &collectionpb.ListCollectionsRequest{},
			expectError:    false,
			minCollections: 0,
		},
		{
			name:           "NilRequest",
			request:        nil,
			expectError:    true,
			minCollections: 0,
		},
	}

	useCase := createListTestUseCaseWithAuth(businessType, false, true)

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			response, err := useCase.Execute(ctx, testCase.request)

			if testCase.expectError {
				testutil.AssertError(t, err)
				testutil.AssertNil(t, response, "response for "+testCase.name)
			} else {
				testutil.AssertNoError(t, err)
				testutil.AssertNotNil(t, response, "response for "+testCase.name)
				if response != nil {
					testutil.AssertGreaterThanOrEqual(t, len(response.Data), testCase.minCollections, "collection count for "+testCase.name)
				}
			}
		})
	}

	// Log test completion with result
	testutil.LogTestResult(t, testCode, "BusinessLogicValidation", true, nil)
}
