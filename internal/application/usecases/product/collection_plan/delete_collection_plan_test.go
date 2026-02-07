//go:build mock_db && mock_auth

// Package collection_plan provides table-driven tests for the collection plan deletion use case.
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
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestDeleteCollectionPlanUseCase_Execute_TableDriven
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-PRODUCT-COLLECTION-PLAN-DELETE-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-PRODUCT-COLLECTION-PLAN-DELETE-TRANSACTION-v1.0: WithTransaction
//   - ESPYNA-TEST-PRODUCT-COLLECTION-PLAN-DELETE-AUTHORIZATION-v1.0: Unauthorized
//   - ESPYNA-TEST-PRODUCT-COLLECTION-PLAN-DELETE-NOT-FOUND-v1.0: NotFound
//   - ESPYNA-TEST-PRODUCT-COLLECTION-PLAN-DELETE-NIL-REQUEST-v1.0: NilRequest
//   - ESPYNA-TEST-PRODUCT-COLLECTION-PLAN-DELETE-NIL-DATA-v1.0: NilData
//   - ESPYNA-TEST-PRODUCT-COLLECTION-PLAN-DELETE-VALIDATION-EMPTY-ID-v1.0: EmptyId
//
// Data Sources:
//   - Test cases: packages/copya/data_test/{businessType}/collection_plan.json
//   - Mock data: packages/copya/data/{businessType}/collection_plan.json
//   - Translations: packages/lyngua/translations/{languageCode}/{businessType}/collection_plan.json
package collection_plan

import (
	"context"
	"testing"

	copyatestutil "leapfor.xyz/copya/golang/testutil"
	"leapfor.xyz/espyna/internal/application/shared/testutil"
	"leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock/product"
	collectionplanpb "leapfor.xyz/esqyma/golang/v1/domain/product/collection_plan"
)

// Type alias for delete collection plan test cases
type DeleteCollectionPlanTestCase = testutil.GenericTestCase[*collectionplanpb.DeleteCollectionPlanRequest, *collectionplanpb.DeleteCollectionPlanResponse]

func deleteTestUseCaseWithAuth(businessType string, supportsTransaction bool, shouldAuthorize bool) *DeleteCollectionPlanUseCase {
	mockCollectionPlanRepo := product.NewMockCollectionPlanRepository(businessType)

	repositories := DeleteCollectionPlanRepositories{
		CollectionPlan: mockCollectionPlanRepo,
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, shouldAuthorize)
	services := DeleteCollectionPlanServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
	}

	return NewDeleteCollectionPlanUseCase(repositories, services)
}

func TestDeleteCollectionPlanUseCase_Execute_TableDriven(t *testing.T) {
	businessType := testutil.GetTestBusinessType()

	// Load test data resolvers before defining test cases
	commonDataResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "collection_plan", "CollectionPlan_CommonData")
	testutil.AssertTestCaseLoad(t, err, "CollectionPlan_CommonData")

	authorizationUnauthorizedResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "collection_plan", "Authorization_Unauthorized")
	testutil.AssertTestCaseLoad(t, err, "Authorization_Unauthorized")

	testCases := []DeleteCollectionPlanTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-PRODUCT-COLLECTION-PLAN-DELETE-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *collectionplanpb.DeleteCollectionPlanRequest {
				return &collectionplanpb.DeleteCollectionPlanRequest{
					Data: &collectionplanpb.CollectionPlan{
						Id: commonDataResolver.MustGetString("primaryCollectionPlanId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *collectionplanpb.DeleteCollectionPlanResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
			},
		},
		{
			Name:     "WithTransaction",
			TestCode: "ESPYNA-TEST-PRODUCT-COLLECTION-PLAN-DELETE-TRANSACTION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *collectionplanpb.DeleteCollectionPlanRequest {
				return &collectionplanpb.DeleteCollectionPlanRequest{
					Data: &collectionplanpb.CollectionPlan{
						Id: commonDataResolver.MustGetString("secondaryCollectionPlanId"),
					},
				}
			},
			UseTransaction: true,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *collectionplanpb.DeleteCollectionPlanResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
			},
		},
		{
			Name:     "Unauthorized",
			TestCode: "ESPYNA-TEST-PRODUCT-COLLECTION-PLAN-DELETE-AUTHORIZATION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *collectionplanpb.DeleteCollectionPlanRequest {
				return &collectionplanpb.DeleteCollectionPlanRequest{
					Data: &collectionplanpb.CollectionPlan{
						Id: authorizationUnauthorizedResolver.MustGetString("targetCollectionPlanId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        false,
			ExpectSuccess:  false,
			ExpectedError:  "collection_plan.errors.authorization_failed",
			Assertions: func(t *testing.T, response *collectionplanpb.DeleteCollectionPlanResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertAuthorizationError(t, err)
			},
		},
		{
			Name:     "NotFound",
			TestCode: "ESPYNA-TEST-PRODUCT-COLLECTION-PLAN-DELETE-NOT-FOUND-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *collectionplanpb.DeleteCollectionPlanRequest {
				return &collectionplanpb.DeleteCollectionPlanRequest{
					Data: &collectionplanpb.CollectionPlan{
						Id: commonDataResolver.MustGetString("nonExistentId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "collection_plan.errors.delete_failed_not_found",
			Assertions: func(t *testing.T, response *collectionplanpb.DeleteCollectionPlanResponse, err error, useCase interface{}, ctx context.Context) {
				uc := useCase.(*DeleteCollectionPlanUseCase)
				testutil.AssertTranslatedError(t, err, "collection_plan.errors.delete_failed_not_found", uc.services.TranslationService, ctx)
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-PRODUCT-COLLECTION-PLAN-DELETE-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *collectionplanpb.DeleteCollectionPlanRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "collection_plan.validation.request_required",
			Assertions: func(t *testing.T, response *collectionplanpb.DeleteCollectionPlanResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilRequest(t, err)
			},
		},
		{
			Name:     "NilData",
			TestCode: "ESPYNA-TEST-PRODUCT-COLLECTION-PLAN-DELETE-NIL-DATA-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *collectionplanpb.DeleteCollectionPlanRequest {
				return &collectionplanpb.DeleteCollectionPlanRequest{Data: nil}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "collection_plan.validation.data_required",
			Assertions: func(t *testing.T, response *collectionplanpb.DeleteCollectionPlanResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "nil data")
			},
		},
		{
			Name:     "EmptyId",
			TestCode: "ESPYNA-TEST-PRODUCT-COLLECTION-PLAN-DELETE-VALIDATION-EMPTY-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *collectionplanpb.DeleteCollectionPlanRequest {
				return &collectionplanpb.DeleteCollectionPlanRequest{
					Data: &collectionplanpb.CollectionPlan{
						Id: "",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "collection_plan.validation.id_required",
			Assertions: func(t *testing.T, response *collectionplanpb.DeleteCollectionPlanResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty ID")
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
			useCase := deleteTestUseCaseWithAuth(businessType, tc.UseTransaction, tc.UseAuth)

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
