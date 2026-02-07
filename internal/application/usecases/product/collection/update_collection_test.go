//go:build mock_db && mock_auth

// Package collection provides table-driven tests for the collection update use case.
//
// The tests cover various scenarios, including success, transaction handling,
// nil requests, validation errors, not found, and boundary conditions.
// Each test case is defined in a table with a specific test code, request setup,
// and assertions.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user").
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education").
//
// Usage:
//   - Run all tests: go test -tags="mock_db,mock_auth" ./...
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestUpdateCollectionUseCase_Execute_TableDriven
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-PRODUCT-COLLECTION-UPDATE-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-PRODUCT-COLLECTION-UPDATE-TRANSACTION-v1.0: WithTransaction
//   - ESPYNA-TEST-PRODUCT-COLLECTION-UPDATE-UNAUTHORIZED-v1.0: Unauthorized
//   - ESPYNA-TEST-PRODUCT-COLLECTION-UPDATE-NIL-REQUEST-v1.0: NilRequest
//   - ESPYNA-TEST-PRODUCT-COLLECTION-UPDATE-NIL-DATA-v1.0: NilData
//   - ESPYNA-TEST-PRODUCT-COLLECTION-UPDATE-VALIDATION-EMPTY-ID-v1.0: EmptyId
//   - ESPYNA-TEST-PRODUCT-COLLECTION-UPDATE-VALIDATION-EMPTY-NAME-v1.0: EmptyName
//   - ESPYNA-TEST-PRODUCT-COLLECTION-UPDATE-NOT-FOUND-v1.0: NotFound
//   - ESPYNA-TEST-PRODUCT-COLLECTION-UPDATE-VALIDATION-MINIMAL-v1.0: MinimalValidData
//   - ESPYNA-TEST-PRODUCT-COLLECTION-UPDATE-VALIDATION-MAXIMAL-v1.0: MaxValidData
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
	"leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock/product"
	collectionpb "leapfor.xyz/esqyma/golang/v1/domain/product/collection"
)

// Type alias for update collection test cases
type UpdateCollectionTestCase = testutil.GenericTestCase[*collectionpb.UpdateCollectionRequest, *collectionpb.UpdateCollectionResponse]

func createTestUpdateCollectionUseCaseWithAuth(businessType string, supportsTransaction bool, shouldAuthorize bool) *UpdateCollectionUseCase {
	mockCollectionRepo := product.NewMockCollectionRepository(businessType)

	repositories := UpdateCollectionRepositories{
		Collection: mockCollectionRepo,
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, shouldAuthorize)
	services := UpdateCollectionServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
	}

	return NewUpdateCollectionUseCase(repositories, services)
}

func TestUpdateCollectionUseCase_Execute_TableDriven(t *testing.T) {
	businessType := testutil.GetTestBusinessType()

	// Load test data resolvers before defining test cases
	updateSuccessResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "collection", "UpdateCollection_Success")
	testutil.AssertTestCaseLoad(t, err, "UpdateCollection_Success")

	commonDataResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "collection", "Collection_CommonData")
	testutil.AssertTestCaseLoad(t, err, "Collection_CommonData")

	testCases := []UpdateCollectionTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-PRODUCT-COLLECTION-UPDATE-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *collectionpb.UpdateCollectionRequest {
				return &collectionpb.UpdateCollectionRequest{
					Data: &collectionpb.Collection{
						Id:          commonDataResolver.MustGetString("primaryCollectionId"),
						Name:        updateSuccessResolver.MustGetString("enhancedCollectionName"),
						Description: updateSuccessResolver.MustGetString("enhancedCollectionDescription"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *collectionpb.UpdateCollectionResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertEqual(t, 1, len(response.Data), "response data length")
				updatedCollection := response.Data[0]
				testutil.AssertStringEqual(t, updateSuccessResolver.MustGetString("enhancedCollectionName"), updatedCollection.Name, "updated name")
				testutil.AssertStringEqual(t, updateSuccessResolver.MustGetString("enhancedCollectionDescription"), updatedCollection.Description, "updated description")
				testutil.AssertFieldSet(t, updatedCollection.DateModified, "DateModified")
				testutil.AssertFieldSet(t, updatedCollection.DateModifiedString, "DateModifiedString")
			},
		},
		{
			Name:     "WithTransaction",
			TestCode: "ESPYNA-TEST-PRODUCT-COLLECTION-UPDATE-TRANSACTION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *collectionpb.UpdateCollectionRequest {
				return &collectionpb.UpdateCollectionRequest{
					Data: &collectionpb.Collection{
						Id:          commonDataResolver.MustGetString("secondaryCollectionId"),
						Name:        updateSuccessResolver.MustGetString("enhancedCollectionNameAlt"),
						Description: updateSuccessResolver.MustGetString("enhancedCollectionDescriptionAlt"),
					},
				}
			},
			UseTransaction: true,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *collectionpb.UpdateCollectionResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				updatedCollection := response.Data[0]
				testutil.AssertStringEqual(t, updateSuccessResolver.MustGetString("enhancedCollectionNameAlt"), updatedCollection.Name, "updated name")
			},
		},
		{
			Name:     "Unauthorized",
			TestCode: "ESPYNA-TEST-PRODUCT-COLLECTION-UPDATE-UNAUTHORIZED-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *collectionpb.UpdateCollectionRequest {
				return &collectionpb.UpdateCollectionRequest{
					Data: &collectionpb.Collection{
						Id:   commonDataResolver.MustGetString("primaryCollectionId"),
						Name: "Valid Collection Name",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        false,
			ExpectSuccess:  false,
			ExpectedError:  "authorization.errors.insufficient_permissions",
			Assertions: func(t *testing.T, response *collectionpb.UpdateCollectionResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertAuthorizationError(t, err)
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-PRODUCT-COLLECTION-UPDATE-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *collectionpb.UpdateCollectionRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "collection.validation.request_required",
			Assertions: func(t *testing.T, response *collectionpb.UpdateCollectionResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilRequest(t, err)
			},
		},
		{
			Name:     "NilData",
			TestCode: "ESPYNA-TEST-PRODUCT-COLLECTION-UPDATE-NIL-DATA-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *collectionpb.UpdateCollectionRequest {
				return &collectionpb.UpdateCollectionRequest{Data: nil}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "collection.validation.data_required",
			Assertions: func(t *testing.T, response *collectionpb.UpdateCollectionResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilData(t, err)
			},
		},
		{
			Name:     "EmptyId",
			TestCode: "ESPYNA-TEST-PRODUCT-COLLECTION-UPDATE-VALIDATION-EMPTY-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *collectionpb.UpdateCollectionRequest {
				return &collectionpb.UpdateCollectionRequest{
					Data: &collectionpb.Collection{
						Id:   "",
						Name: "Valid Collection Name",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "collection.validation.id_required",
			Assertions: func(t *testing.T, response *collectionpb.UpdateCollectionResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty ID")
			},
		},
		{
			Name:     "EmptyName",
			TestCode: "ESPYNA-TEST-PRODUCT-COLLECTION-UPDATE-VALIDATION-EMPTY-NAME-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *collectionpb.UpdateCollectionRequest {
				return &collectionpb.UpdateCollectionRequest{
					Data: &collectionpb.Collection{
						Id:   commonDataResolver.MustGetString("primaryCollectionId"),
						Name: "",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "collection.validation.name_required",
			Assertions: func(t *testing.T, response *collectionpb.UpdateCollectionResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty name")
			},
		},
		{
			Name:     "NotFound",
			TestCode: "ESPYNA-TEST-PRODUCT-COLLECTION-UPDATE-NOT-FOUND-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *collectionpb.UpdateCollectionRequest {
				return &collectionpb.UpdateCollectionRequest{
					Data: &collectionpb.Collection{
						Id:   "collection-non-existent",
						Name: "Valid Collection Name",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "Collection with ID 'collection-non-existent' not found",
			ExactError:     true,
			Assertions: func(t *testing.T, response *collectionpb.UpdateCollectionResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertError(t, err)
			},
		},
		{
			Name:     "MinimalValidData",
			TestCode: "ESPYNA-TEST-PRODUCT-COLLECTION-UPDATE-VALIDATION-MINIMAL-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *collectionpb.UpdateCollectionRequest {
				boundaryResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "product_collection", "BoundaryTest_MinimalValid")
				testutil.AssertTestCaseLoad(t, err, "BoundaryTest_MinimalValid")
				return &collectionpb.UpdateCollectionRequest{
					Data: &collectionpb.Collection{
						Id:   commonDataResolver.MustGetString("primaryCollectionId"),
						Name: boundaryResolver.MustGetString("minValidName"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *collectionpb.UpdateCollectionResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				updatedCollection := response.Data[0]
				boundaryResolver, _ := copyatestutil.LoadTestCaseFromBusinessType(t, testutil.GetTestBusinessType(), "product_collection", "BoundaryTest_MinimalValid")
				testutil.AssertStringEqual(t, boundaryResolver.MustGetString("minValidName"), updatedCollection.Name, "collection name")
			},
		},
		{
			Name:     "MaxValidData",
			TestCode: "ESPYNA-TEST-PRODUCT-COLLECTION-UPDATE-VALIDATION-MAXIMAL-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *collectionpb.UpdateCollectionRequest {
				boundaryResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "product_collection", "BoundaryTest_MaximalValid")
				testutil.AssertTestCaseLoad(t, err, "BoundaryTest_MaximalValid")
				return &collectionpb.UpdateCollectionRequest{
					Data: &collectionpb.Collection{
						Id:          commonDataResolver.MustGetString("primaryCollectionId"),
						Name:        boundaryResolver.MustGetString("maxValidNameExact100"),
						Description: boundaryResolver.MustGetString("maxValidDescriptionExact500"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *collectionpb.UpdateCollectionResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				updatedCollection := response.Data[0]
				testutil.AssertEqual(t, 100, len(updatedCollection.Name), "name length")
				testutil.AssertEqual(t, 500, len(updatedCollection.Description), "description length")
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
			useCase := createTestUpdateCollectionUseCaseWithAuth(businessType, tc.UseTransaction, tc.UseAuth)

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
					if tc.ExactError {
						testutil.AssertStringEqual(t, tc.ExpectedError, err.Error(), "error message")
					} else if tc.ErrorTags != nil {
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
