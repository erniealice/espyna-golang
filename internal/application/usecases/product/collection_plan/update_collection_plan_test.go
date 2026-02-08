//go:build mock_db && mock_auth

// Package collection_plan provides table-driven tests for the collection plan update use case.
//
// The tests cover various scenarios, including success, transaction handling,
// authorization, nil requests, validation errors, not found errors, and boundary conditions.
// Each test case is defined in a table with a specific test code, request setup,
// and assertions.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user").
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education").
//
// Usage:
//   - Run all tests: go test -tags="mock_db,mock_auth" ./...
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestUpdateCollectionPlanUseCase_Execute_TableDriven
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-PRODUCT-COLLECTION-PLAN-UPDATE-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-PRODUCT-COLLECTION-PLAN-UPDATE-TRANSACTION-v1.0: WithTransaction
//   - ESPYNA-TEST-PRODUCT-COLLECTION-PLAN-UPDATE-AUTHORIZATION-v1.0: Unauthorized
//   - ESPYNA-TEST-PRODUCT-COLLECTION-PLAN-UPDATE-NOT-FOUND-v1.0: NotFound
//   - ESPYNA-TEST-PRODUCT-COLLECTION-PLAN-UPDATE-NIL-REQUEST-v1.0: NilRequest
//   - ESPYNA-TEST-PRODUCT-COLLECTION-PLAN-UPDATE-NIL-DATA-v1.0: NilData
//   - ESPYNA-TEST-PRODUCT-COLLECTION-PLAN-UPDATE-VALIDATION-EMPTY-ID-v1.0: EmptyId
//   - ESPYNA-TEST-PRODUCT-COLLECTION-PLAN-UPDATE-VALIDATION-INVALID-COLLECTION-ID-v1.0: InvalidCollectionId
//   - ESPYNA-TEST-PRODUCT-COLLECTION-PLAN-UPDATE-VALIDATION-INVALID-PLAN-ID-v1.0: InvalidPlanId
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
	"github.com/erniealice/espyna-golang/internal/application/shared/testutil"
	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock/product"
	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock/subscription"
	collectionplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/collection_plan"
)

// Type alias for update collection plan test cases
type UpdateCollectionPlanTestCase = testutil.GenericTestCase[*collectionplanpb.UpdateCollectionPlanRequest, *collectionplanpb.UpdateCollectionPlanResponse]

func updateTestUseCaseWithAuth(businessType string, supportsTransaction bool, shouldAuthorize bool) *UpdateCollectionPlanUseCase {
	mockCollectionPlanRepo := product.NewMockCollectionPlanRepository(businessType)
	mockCollectionRepo := product.NewMockCollectionRepository(businessType)
	mockPlanRepo := subscription.NewMockPlanRepository(businessType)

	repositories := UpdateCollectionPlanRepositories{
		CollectionPlan: mockCollectionPlanRepo,
		Collection:     mockCollectionRepo,
		Plan:           mockPlanRepo,
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, shouldAuthorize)
	services := UpdateCollectionPlanServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
	}

	return NewUpdateCollectionPlanUseCase(repositories, services)
}

func TestUpdateCollectionPlanUseCase_Execute_TableDriven(t *testing.T) {
	businessType := testutil.GetTestBusinessType()

	// Load test data resolvers before defining test cases
	updateSuccessResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "collection_plan", "UpdateCollectionPlan_Success")
	testutil.AssertTestCaseLoad(t, err, "UpdateCollectionPlan_Success")

	authorizationUnauthorizedResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "collection_plan", "Authorization_Unauthorized")
	testutil.AssertTestCaseLoad(t, err, "Authorization_Unauthorized")

	validationErrorInvalidCollectionIdResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "collection_plan", "ValidationError_InvalidCollectionId")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_InvalidCollectionId")

	validationErrorInvalidPlanIdResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "collection_plan", "ValidationError_InvalidPlanId")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_InvalidPlanId")

	collectionPlanCommonDataResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "collection_plan", "CollectionPlan_CommonData")
	testutil.AssertTestCaseLoad(t, err, "CollectionPlan_CommonData")

	testCases := []UpdateCollectionPlanTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-PRODUCT-COLLECTION-PLAN-UPDATE-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *collectionplanpb.UpdateCollectionPlanRequest {
				return &collectionplanpb.UpdateCollectionPlanRequest{
					Data: &collectionplanpb.CollectionPlan{
						Id:           collectionPlanCommonDataResolver.MustGetString("primaryCollectionPlanId"),
						CollectionId: updateSuccessResolver.MustGetString("updatedCollectionId"),
						PlanId:       updateSuccessResolver.MustGetString("updatedPlanId"),
						Active:       true,
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *collectionplanpb.UpdateCollectionPlanResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertEqual(t, 1, len(response.Data), "response data length")
				updatedCollectionPlan := response.Data[0]
				testutil.AssertStringEqual(t, collectionPlanCommonDataResolver.MustGetString("primaryCollectionPlanId"), updatedCollectionPlan.Id, "collection plan ID")
				testutil.AssertStringEqual(t, updateSuccessResolver.MustGetString("updatedCollectionId"), updatedCollectionPlan.CollectionId, "collection ID")
				testutil.AssertStringEqual(t, updateSuccessResolver.MustGetString("updatedPlanId"), updatedCollectionPlan.PlanId, "plan ID")
				testutil.AssertTrue(t, updatedCollectionPlan.Active, "collection plan active status")
				testutil.AssertFieldSet(t, updatedCollectionPlan.DateModified, "DateModified")
				testutil.AssertFieldSet(t, updatedCollectionPlan.DateModifiedString, "DateModifiedString")
			},
		},
		{
			Name:     "WithTransaction",
			TestCode: "ESPYNA-TEST-PRODUCT-COLLECTION-PLAN-UPDATE-TRANSACTION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *collectionplanpb.UpdateCollectionPlanRequest {
				return &collectionplanpb.UpdateCollectionPlanRequest{
					Data: &collectionplanpb.CollectionPlan{
						Id:           collectionPlanCommonDataResolver.MustGetString("secondaryCollectionPlanId"),
						CollectionId: updateSuccessResolver.MustGetString("enhancedCollectionId"),
						PlanId:       updateSuccessResolver.MustGetString("enhancedPlanId"),
						Active:       true,
					},
				}
			},
			UseTransaction: true,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *collectionplanpb.UpdateCollectionPlanResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				updatedCollectionPlan := response.Data[0]
				testutil.AssertStringEqual(t, collectionPlanCommonDataResolver.MustGetString("secondaryCollectionPlanId"), updatedCollectionPlan.Id, "collection plan ID")
				testutil.AssertStringEqual(t, updateSuccessResolver.MustGetString("enhancedCollectionId"), updatedCollectionPlan.CollectionId, "collection ID")
				testutil.AssertStringEqual(t, updateSuccessResolver.MustGetString("enhancedPlanId"), updatedCollectionPlan.PlanId, "plan ID")
			},
		},
		{
			Name:     "Unauthorized",
			TestCode: "ESPYNA-TEST-PRODUCT-COLLECTION-PLAN-UPDATE-AUTHORIZATION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *collectionplanpb.UpdateCollectionPlanRequest {
				return &collectionplanpb.UpdateCollectionPlanRequest{
					Data: &collectionplanpb.CollectionPlan{
						Id:           authorizationUnauthorizedResolver.MustGetString("targetCollectionPlanId"),
						CollectionId: authorizationUnauthorizedResolver.MustGetString("unauthorizedCollectionId"),
						PlanId:       authorizationUnauthorizedResolver.MustGetString("unauthorizedPlanId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        false,
			ExpectSuccess:  false,
			ExpectedError:  "collection_plan.errors.authorization_failed",
			Assertions: func(t *testing.T, response *collectionplanpb.UpdateCollectionPlanResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertAuthorizationError(t, err)
			},
		},
		{
			Name:     "NotFound",
			TestCode: "ESPYNA-TEST-PRODUCT-COLLECTION-PLAN-UPDATE-NOT-FOUND-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *collectionplanpb.UpdateCollectionPlanRequest {
				return &collectionplanpb.UpdateCollectionPlanRequest{
					Data: &collectionplanpb.CollectionPlan{
						Id:           collectionPlanCommonDataResolver.MustGetString("nonExistentId"),
						CollectionId: updateSuccessResolver.MustGetString("validCollectionId"),
						PlanId:       updateSuccessResolver.MustGetString("validPlanId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "collection_plan.errors.update_failed",
			Assertions: func(t *testing.T, response *collectionplanpb.UpdateCollectionPlanResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "not found")
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-PRODUCT-COLLECTION-PLAN-UPDATE-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *collectionplanpb.UpdateCollectionPlanRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "collection_plan.validation.request_required",
			Assertions: func(t *testing.T, response *collectionplanpb.UpdateCollectionPlanResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilRequest(t, err)
			},
		},
		{
			Name:     "NilData",
			TestCode: "ESPYNA-TEST-PRODUCT-COLLECTION-PLAN-UPDATE-NIL-DATA-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *collectionplanpb.UpdateCollectionPlanRequest {
				return &collectionplanpb.UpdateCollectionPlanRequest{Data: nil}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "collection_plan.validation.data_required",
			Assertions: func(t *testing.T, response *collectionplanpb.UpdateCollectionPlanResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "nil data")
			},
		},
		{
			Name:     "EmptyId",
			TestCode: "ESPYNA-TEST-PRODUCT-COLLECTION-PLAN-UPDATE-VALIDATION-EMPTY-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *collectionplanpb.UpdateCollectionPlanRequest {
				return &collectionplanpb.UpdateCollectionPlanRequest{
					Data: &collectionplanpb.CollectionPlan{
						Id:           "",
						CollectionId: updateSuccessResolver.MustGetString("validCollectionId"),
						PlanId:       updateSuccessResolver.MustGetString("validPlanId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "collection_plan.validation.id_required",
			Assertions: func(t *testing.T, response *collectionplanpb.UpdateCollectionPlanResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty ID")
			},
		},
		{
			Name:     "InvalidCollectionId",
			TestCode: "ESPYNA-TEST-PRODUCT-COLLECTION-PLAN-UPDATE-VALIDATION-INVALID-COLLECTION-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *collectionplanpb.UpdateCollectionPlanRequest {
				return &collectionplanpb.UpdateCollectionPlanRequest{
					Data: &collectionplanpb.CollectionPlan{
						Id:           collectionPlanCommonDataResolver.MustGetString("primaryCollectionPlanId"),
						CollectionId: validationErrorInvalidCollectionIdResolver.MustGetString("invalidCollectionId"),
						PlanId:       validationErrorInvalidCollectionIdResolver.MustGetString("validPlanId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "collection_plan.validation.collection_not_found",
			Assertions: func(t *testing.T, response *collectionplanpb.UpdateCollectionPlanResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "invalid collection ID")
			},
		},
		{
			Name:     "InvalidPlanId",
			TestCode: "ESPYNA-TEST-PRODUCT-COLLECTION-PLAN-UPDATE-VALIDATION-INVALID-PLAN-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *collectionplanpb.UpdateCollectionPlanRequest {
				return &collectionplanpb.UpdateCollectionPlanRequest{
					Data: &collectionplanpb.CollectionPlan{
						Id:           collectionPlanCommonDataResolver.MustGetString("primaryCollectionPlanId"),
						CollectionId: validationErrorInvalidPlanIdResolver.MustGetString("validCollectionId"),
						PlanId:       validationErrorInvalidPlanIdResolver.MustGetString("invalidPlanId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "collection_plan.validation.plan_not_found",
			Assertions: func(t *testing.T, response *collectionplanpb.UpdateCollectionPlanResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "invalid plan ID")
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
			useCase := updateTestUseCaseWithAuth(businessType, tc.UseTransaction, tc.UseAuth)

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
