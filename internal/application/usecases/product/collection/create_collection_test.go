//go:build mock_db && mock_auth

// Package collection provides table-driven tests for the collection creation use case.
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
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestCreateCollectionUseCase_Execute_TableDriven
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-PRODUCT-COLLECTION-CREATE-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-PRODUCT-COLLECTION-CREATE-TRANSACTION-v1.0: WithTransaction
//   - ESPYNA-TEST-PRODUCT-COLLECTION-CREATE-AUTHORIZATION-v1.0: Unauthorized
//   - ESPYNA-TEST-PRODUCT-COLLECTION-CREATE-NIL-REQUEST-v1.0: NilRequest
//   - ESPYNA-TEST-PRODUCT-COLLECTION-CREATE-NIL-DATA-v1.0: NilData
//   - ESPYNA-TEST-PRODUCT-COLLECTION-CREATE-VALIDATION-EMPTY-NAME-v1.0: EmptyName
//   - ESPYNA-TEST-PRODUCT-COLLECTION-CREATE-VALIDATION-NAME-TOO-SHORT-v1.0: NameTooShort
//   - ESPYNA-TEST-PRODUCT-COLLECTION-CREATE-VALIDATION-NAME-TOO-LONG-v1.0: NameTooLong
//   - ESPYNA-TEST-PRODUCT-COLLECTION-CREATE-VALIDATION-DESCRIPTION-TOO-LONG-v1.0: DescriptionTooLong
//   - ESPYNA-TEST-PRODUCT-COLLECTION-CREATE-VALIDATION-MINIMAL-v1.0: MinimalValidData
//   - ESPYNA-TEST-PRODUCT-COLLECTION-CREATE-VALIDATION-MAXIMAL-v1.0: MaxValidData
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
	"leapfor.xyz/espyna/internal/application/shared/testutil"
	"leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock/product"
	collectionpb "leapfor.xyz/esqyma/golang/v1/domain/product/collection"
)

// Type alias for create collection test cases
type CreateCollectionTestCase = testutil.GenericTestCase[*collectionpb.CreateCollectionRequest, *collectionpb.CreateCollectionResponse]

func createTestUseCaseWithAuth(businessType string, supportsTransaction bool, shouldAuthorize bool) *CreateCollectionUseCase {
	mockRepo := product.NewMockCollectionRepository(businessType)

	repositories := CreateCollectionRepositories{
		Collection: mockRepo,
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, shouldAuthorize)
	services := CreateCollectionServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
		IDService:            standardServices.IDService,
	}

	return NewCreateCollectionUseCase(repositories, services)
}

func TestCreateCollectionUseCase_Execute_TableDriven(t *testing.T) {
	businessType := testutil.GetTestBusinessType()

	// Load test data resolvers before defining test cases
	createSuccessResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "collection", "CreateCollection_Success")
	testutil.AssertTestCaseLoad(t, err, "CreateCollection_Success")

	authorizationUnauthorizedResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "collection", "Authorization_Unauthorized")
	testutil.AssertTestCaseLoad(t, err, "Authorization_Unauthorized")

	validationErrorEmptyNameResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "collection", "ValidationError_EmptyName")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_EmptyName")

	validationErrorNameTooShortResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "collection", "ValidationError_NameTooShort")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_NameTooShort")

	validationErrorNameTooLongResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "collection", "ValidationError_NameTooLongGenerated")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_NameTooLongGenerated")

	validationErrorDescriptionTooLongResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "collection", "ValidationError_DescriptionTooLongGenerated")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_DescriptionTooLongGenerated")

	boundaryMinimalResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "collection", "BoundaryTest_MinimalValid")
	testutil.AssertTestCaseLoad(t, err, "BoundaryTest_MinimalValid")

	boundaryMaximalResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "collection", "BoundaryTest_MaximalValid")
	testutil.AssertTestCaseLoad(t, err, "BoundaryTest_MaximalValid")

	testCases := []CreateCollectionTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-PRODUCT-COLLECTION-CREATE-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *collectionpb.CreateCollectionRequest {
				return &collectionpb.CreateCollectionRequest{
					Data: &collectionpb.Collection{
						Name:        createSuccessResolver.MustGetString("newCollectionName"),
						Description: createSuccessResolver.MustGetString("newCollectionDescription"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *collectionpb.CreateCollectionResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertEqual(t, 1, len(response.Data), "response data length")
				createdCollection := response.Data[0]
				testutil.AssertStringEqual(t, createSuccessResolver.MustGetString("newCollectionName"), createdCollection.Name, "collection name")
				testutil.AssertNonEmptyString(t, createdCollection.Id, "collection ID")
				testutil.AssertTrue(t, createdCollection.Active, "collection active status")
				testutil.AssertFieldSet(t, createdCollection.DateCreated, "DateCreated")
				testutil.AssertFieldSet(t, createdCollection.DateCreatedString, "DateCreatedString")
			},
		},
		{
			Name:     "WithTransaction",
			TestCode: "ESPYNA-TEST-PRODUCT-COLLECTION-CREATE-TRANSACTION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *collectionpb.CreateCollectionRequest {
				return &collectionpb.CreateCollectionRequest{
					Data: &collectionpb.Collection{
						Name:        createSuccessResolver.MustGetString("newCollectionName"),
						Description: createSuccessResolver.MustGetString("newCollectionDescription"),
					},
				}
			},
			UseTransaction: true,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *collectionpb.CreateCollectionResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				createdCollection := response.Data[0]
				testutil.AssertStringEqual(t, createSuccessResolver.MustGetString("newCollectionName"), createdCollection.Name, "collection name")
			},
		},
		{
			Name:     "Unauthorized",
			TestCode: "ESPYNA-TEST-PRODUCT-COLLECTION-CREATE-AUTHORIZATION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *collectionpb.CreateCollectionRequest {
				return &collectionpb.CreateCollectionRequest{
					Data: &collectionpb.Collection{
						Name:        authorizationUnauthorizedResolver.MustGetString("unauthorizedCollectionName"),
						Description: authorizationUnauthorizedResolver.MustGetString("unauthorizedCollectionDescription"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        false,
			ExpectSuccess:  false,
			ExpectedError:  "collection.errors.authorization_failed",
			Assertions: func(t *testing.T, response *collectionpb.CreateCollectionResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertAuthorizationError(t, err)
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-PRODUCT-COLLECTION-CREATE-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *collectionpb.CreateCollectionRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "collection.validation.request_required",
			Assertions: func(t *testing.T, response *collectionpb.CreateCollectionResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilRequest(t, err)
			},
		},
		{
			Name:     "NilData",
			TestCode: "ESPYNA-TEST-PRODUCT-COLLECTION-CREATE-NIL-DATA-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *collectionpb.CreateCollectionRequest {
				return &collectionpb.CreateCollectionRequest{Data: nil}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "collection.validation.data_required",
			Assertions: func(t *testing.T, response *collectionpb.CreateCollectionResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "nil data")
			},
		},
		{
			Name:     "EmptyName",
			TestCode: "ESPYNA-TEST-PRODUCT-COLLECTION-CREATE-VALIDATION-EMPTY-NAME-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *collectionpb.CreateCollectionRequest {
				return &collectionpb.CreateCollectionRequest{
					Data: &collectionpb.Collection{
						Name:        validationErrorEmptyNameResolver.MustGetString("emptyName"),
						Description: validationErrorEmptyNameResolver.MustGetString("validDescription"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "collection.validation.name_required",
			Assertions: func(t *testing.T, response *collectionpb.CreateCollectionResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty name")
			},
		},
		{
			Name:     "NameTooShort",
			TestCode: "ESPYNA-TEST-PRODUCT-COLLECTION-CREATE-VALIDATION-NAME-TOO-SHORT-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *collectionpb.CreateCollectionRequest {
				return &collectionpb.CreateCollectionRequest{
					Data: &collectionpb.Collection{
						Name:        validationErrorNameTooShortResolver.MustGetString("tooShortName"),
						Description: validationErrorNameTooShortResolver.MustGetString("validDescription"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "collection.validation.name_too_short",
			Assertions: func(t *testing.T, response *collectionpb.CreateCollectionResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "name too short")
			},
		},
		{
			Name:     "NameTooLong",
			TestCode: "ESPYNA-TEST-PRODUCT-COLLECTION-CREATE-VALIDATION-NAME-TOO-LONG-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *collectionpb.CreateCollectionRequest {
				return &collectionpb.CreateCollectionRequest{
					Data: &collectionpb.Collection{
						Name:        validationErrorNameTooLongResolver.MustGetString("tooLongNameGenerated"),
						Description: validationErrorNameTooLongResolver.MustGetString("validDescription"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "collection.validation.name_too_long",
			Assertions: func(t *testing.T, response *collectionpb.CreateCollectionResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "name too long")
			},
		},
		{
			Name:     "DescriptionTooLong",
			TestCode: "ESPYNA-TEST-PRODUCT-COLLECTION-CREATE-VALIDATION-DESCRIPTION-TOO-LONG-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *collectionpb.CreateCollectionRequest {
				return &collectionpb.CreateCollectionRequest{
					Data: &collectionpb.Collection{
						Name:        validationErrorDescriptionTooLongResolver.MustGetString("validName"),
						Description: validationErrorDescriptionTooLongResolver.MustGetString("tooLongDescriptionGenerated"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "collection.validation.description_too_long",
			Assertions: func(t *testing.T, response *collectionpb.CreateCollectionResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "description too long")
			},
		},
		{
			Name:     "MinimalValidData",
			TestCode: "ESPYNA-TEST-PRODUCT-COLLECTION-CREATE-VALIDATION-MINIMAL-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *collectionpb.CreateCollectionRequest {
				return &collectionpb.CreateCollectionRequest{
					Data: &collectionpb.Collection{
						Name:        boundaryMinimalResolver.MustGetString("minValidName"),
						Description: boundaryMinimalResolver.MustGetString("minValidDescription"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *collectionpb.CreateCollectionResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				createdCollection := response.Data[0]
				testutil.AssertNonEmptyString(t, createdCollection.Name, "collection name")
				testutil.AssertNonEmptyString(t, createdCollection.Id, "collection ID")
			},
		},
		{
			Name:     "MaxValidData",
			TestCode: "ESPYNA-TEST-PRODUCT-COLLECTION-CREATE-VALIDATION-MAXIMAL-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *collectionpb.CreateCollectionRequest {
				return &collectionpb.CreateCollectionRequest{
					Data: &collectionpb.Collection{
						Name:        boundaryMaximalResolver.MustGetString("maxValidName"),
						Description: boundaryMaximalResolver.MustGetString("maxValidDescription"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *collectionpb.CreateCollectionResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				createdCollection := response.Data[0]
				testutil.AssertNonEmptyString(t, createdCollection.Name, "collection name")
				testutil.AssertNonEmptyString(t, createdCollection.Id, "collection ID")
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
