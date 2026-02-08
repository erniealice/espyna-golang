//go:build mock_db && mock_auth

// Package collection_plan provides table-driven tests for the collection plan creation use case.
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
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestCreateCollectionPlanUseCase_Execute_TableDriven
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-PRODUCT-COLLECTION-PLAN-CREATE-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-PRODUCT-COLLECTION-PLAN-CREATE-TRANSACTION-v1.0: WithTransaction
//   - ESPYNA-TEST-PRODUCT-COLLECTION-PLAN-CREATE-AUTHORIZATION-v1.0: Unauthorized
//   - ESPYNA-TEST-PRODUCT-COLLECTION-PLAN-CREATE-NIL-REQUEST-v1.0: NilRequest
//   - ESPYNA-TEST-PRODUCT-COLLECTION-PLAN-CREATE-NIL-DATA-v1.0: NilData
//   - ESPYNA-TEST-PRODUCT-COLLECTION-PLAN-CREATE-VALIDATION-EMPTY-COLLECTION-ID-v1.0: EmptyCollectionId
//   - ESPYNA-TEST-PRODUCT-COLLECTION-PLAN-CREATE-VALIDATION-EMPTY-PLAN-ID-v1.0: EmptyPlanId
//   - ESPYNA-TEST-PRODUCT-COLLECTION-PLAN-CREATE-VALIDATION-INVALID-COLLECTION-ID-v1.0: InvalidCollectionId
//   - ESPYNA-TEST-PRODUCT-COLLECTION-PLAN-CREATE-VALIDATION-INVALID-PLAN-ID-v1.0: InvalidPlanId
//   - ESPYNA-TEST-PRODUCT-COLLECTION-PLAN-CREATE-VALIDATION-MINIMAL-v1.0: MinimalValidData
//   - ESPYNA-TEST-PRODUCT-COLLECTION-PLAN-CREATE-VALIDATION-MAXIMAL-v1.0: MaxValidData
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

// Type alias for create collection plan test cases
type CreateCollectionPlanTestCase = testutil.GenericTestCase[*collectionplanpb.CreateCollectionPlanRequest, *collectionplanpb.CreateCollectionPlanResponse]

func createTestUseCaseWithAuth(businessType string, supportsTransaction bool, shouldAuthorize bool) *CreateCollectionPlanUseCase {
	mockCollectionPlanRepo := product.NewMockCollectionPlanRepository(businessType)
	mockCollectionRepo := product.NewMockCollectionRepository(businessType)
	mockPlanRepo := subscription.NewMockPlanRepository(businessType)

	repositories := CreateCollectionPlanRepositories{
		CollectionPlan: mockCollectionPlanRepo,
		Collection:     mockCollectionRepo,
		Plan:           mockPlanRepo,
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, shouldAuthorize)
	services := CreateCollectionPlanServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
		IDService:            standardServices.IDService,
	}

	return NewCreateCollectionPlanUseCase(repositories, services)
}

func TestCreateCollectionPlanUseCase_Execute_TableDriven(t *testing.T) {
	businessType := testutil.GetTestBusinessType()

	// Load test data resolvers before defining test cases
	createSuccessResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "collection_plan", "CreateCollectionPlan_Success")
	testutil.AssertTestCaseLoad(t, err, "CreateCollectionPlan_Success")

	authorizationUnauthorizedResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "collection_plan", "Authorization_Unauthorized")
	testutil.AssertTestCaseLoad(t, err, "Authorization_Unauthorized")

	validationErrorEmptyCollectionIdResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "collection_plan", "ValidationError_EmptyCollectionId")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_EmptyCollectionId")

	validationErrorEmptyPlanIdResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "collection_plan", "ValidationError_EmptyPlanId")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_EmptyPlanId")

	validationErrorInvalidCollectionIdResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "collection_plan", "ValidationError_InvalidCollectionId")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_InvalidCollectionId")

	validationErrorInvalidPlanIdResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "collection_plan", "ValidationError_InvalidPlanId")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_InvalidPlanId")

	boundaryMinimalResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "collection_plan", "BoundaryTest_MinimalValid")
	testutil.AssertTestCaseLoad(t, err, "BoundaryTest_MinimalValid")

	boundaryMaximalResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "collection_plan", "BoundaryTest_MaximalValid")
	testutil.AssertTestCaseLoad(t, err, "BoundaryTest_MaximalValid")

	testCases := []CreateCollectionPlanTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-PRODUCT-COLLECTION-PLAN-CREATE-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *collectionplanpb.CreateCollectionPlanRequest {
				return &collectionplanpb.CreateCollectionPlanRequest{
					Data: &collectionplanpb.CollectionPlan{
						CollectionId: createSuccessResolver.MustGetString("validCollectionId"),
						PlanId:       createSuccessResolver.MustGetString("validPlanId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *collectionplanpb.CreateCollectionPlanResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertEqual(t, 1, len(response.Data), "response data length")
				createdCollectionPlan := response.Data[0]
				testutil.AssertStringEqual(t, createSuccessResolver.MustGetString("validCollectionId"), createdCollectionPlan.CollectionId, "collection ID")
				testutil.AssertStringEqual(t, createSuccessResolver.MustGetString("validPlanId"), createdCollectionPlan.PlanId, "plan ID")
				testutil.AssertNonEmptyString(t, createdCollectionPlan.Id, "collection plan ID")
				testutil.AssertTrue(t, createdCollectionPlan.Active, "collection plan active status")
				testutil.AssertFieldSet(t, createdCollectionPlan.DateCreated, "DateCreated")
				testutil.AssertFieldSet(t, createdCollectionPlan.DateCreatedString, "DateCreatedString")
			},
		},
		{
			Name:     "WithTransaction",
			TestCode: "ESPYNA-TEST-PRODUCT-COLLECTION-PLAN-CREATE-TRANSACTION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *collectionplanpb.CreateCollectionPlanRequest {
				return &collectionplanpb.CreateCollectionPlanRequest{
					Data: &collectionplanpb.CollectionPlan{
						CollectionId: createSuccessResolver.MustGetString("validCollectionId"),
						PlanId:       createSuccessResolver.MustGetString("validPlanId"),
					},
				}
			},
			UseTransaction: true,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *collectionplanpb.CreateCollectionPlanResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				createdCollectionPlan := response.Data[0]
				testutil.AssertStringEqual(t, createSuccessResolver.MustGetString("validCollectionId"), createdCollectionPlan.CollectionId, "collection ID")
				testutil.AssertStringEqual(t, createSuccessResolver.MustGetString("validPlanId"), createdCollectionPlan.PlanId, "plan ID")
			},
		},
		{
			Name:     "Unauthorized",
			TestCode: "ESPYNA-TEST-PRODUCT-COLLECTION-PLAN-CREATE-AUTHORIZATION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *collectionplanpb.CreateCollectionPlanRequest {
				return &collectionplanpb.CreateCollectionPlanRequest{
					Data: &collectionplanpb.CollectionPlan{
						CollectionId: authorizationUnauthorizedResolver.MustGetString("unauthorizedCollectionId"),
						PlanId:       authorizationUnauthorizedResolver.MustGetString("unauthorizedPlanId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        false,
			ExpectSuccess:  false,
			ExpectedError:  "collection_plan.errors.authorization_failed",
			Assertions: func(t *testing.T, response *collectionplanpb.CreateCollectionPlanResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertAuthorizationError(t, err)
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-PRODUCT-COLLECTION-PLAN-CREATE-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *collectionplanpb.CreateCollectionPlanRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "collection_plan.validation.request_required",
			Assertions: func(t *testing.T, response *collectionplanpb.CreateCollectionPlanResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilRequest(t, err)
			},
		},
		{
			Name:     "NilData",
			TestCode: "ESPYNA-TEST-PRODUCT-COLLECTION-PLAN-CREATE-NIL-DATA-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *collectionplanpb.CreateCollectionPlanRequest {
				return &collectionplanpb.CreateCollectionPlanRequest{Data: nil}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "collection_plan.validation.data_required",
			Assertions: func(t *testing.T, response *collectionplanpb.CreateCollectionPlanResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "nil data")
			},
		},
		{
			Name:     "EmptyCollectionId",
			TestCode: "ESPYNA-TEST-PRODUCT-COLLECTION-PLAN-CREATE-VALIDATION-EMPTY-COLLECTION-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *collectionplanpb.CreateCollectionPlanRequest {
				return &collectionplanpb.CreateCollectionPlanRequest{
					Data: &collectionplanpb.CollectionPlan{
						CollectionId: validationErrorEmptyCollectionIdResolver.MustGetString("emptyCollectionId"),
						PlanId:       validationErrorEmptyCollectionIdResolver.MustGetString("validPlanId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "collection_plan.validation.collection_id_required",
			Assertions: func(t *testing.T, response *collectionplanpb.CreateCollectionPlanResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty collection ID")
			},
		},
		{
			Name:     "EmptyPlanId",
			TestCode: "ESPYNA-TEST-PRODUCT-COLLECTION-PLAN-CREATE-VALIDATION-EMPTY-PLAN-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *collectionplanpb.CreateCollectionPlanRequest {
				return &collectionplanpb.CreateCollectionPlanRequest{
					Data: &collectionplanpb.CollectionPlan{
						CollectionId: validationErrorEmptyPlanIdResolver.MustGetString("validCollectionId"),
						PlanId:       validationErrorEmptyPlanIdResolver.MustGetString("emptyPlanId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "collection_plan.validation.plan_id_required",
			Assertions: func(t *testing.T, response *collectionplanpb.CreateCollectionPlanResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty plan ID")
			},
		},
		{
			Name:     "InvalidCollectionId",
			TestCode: "ESPYNA-TEST-PRODUCT-COLLECTION-PLAN-CREATE-VALIDATION-INVALID-COLLECTION-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *collectionplanpb.CreateCollectionPlanRequest {
				return &collectionplanpb.CreateCollectionPlanRequest{
					Data: &collectionplanpb.CollectionPlan{
						CollectionId: validationErrorInvalidCollectionIdResolver.MustGetString("invalidCollectionId"),
						PlanId:       validationErrorInvalidCollectionIdResolver.MustGetString("validPlanId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "collection_plan.validation.collection_not_found",
			Assertions: func(t *testing.T, response *collectionplanpb.CreateCollectionPlanResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "invalid collection ID")
			},
		},
		{
			Name:     "InvalidPlanId",
			TestCode: "ESPYNA-TEST-PRODUCT-COLLECTION-PLAN-CREATE-VALIDATION-INVALID-PLAN-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *collectionplanpb.CreateCollectionPlanRequest {
				return &collectionplanpb.CreateCollectionPlanRequest{
					Data: &collectionplanpb.CollectionPlan{
						CollectionId: validationErrorInvalidPlanIdResolver.MustGetString("validCollectionId"),
						PlanId:       validationErrorInvalidPlanIdResolver.MustGetString("invalidPlanId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "collection_plan.validation.plan_not_found",
			Assertions: func(t *testing.T, response *collectionplanpb.CreateCollectionPlanResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "invalid plan ID")
			},
		},
		{
			Name:     "MinimalValidData",
			TestCode: "ESPYNA-TEST-PRODUCT-COLLECTION-PLAN-CREATE-VALIDATION-MINIMAL-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *collectionplanpb.CreateCollectionPlanRequest {
				return &collectionplanpb.CreateCollectionPlanRequest{
					Data: &collectionplanpb.CollectionPlan{
						CollectionId: boundaryMinimalResolver.MustGetString("minValidCollectionId"),
						PlanId:       boundaryMinimalResolver.MustGetString("minValidPlanId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *collectionplanpb.CreateCollectionPlanResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				createdCollectionPlan := response.Data[0]
				testutil.AssertStringEqual(t, boundaryMinimalResolver.MustGetString("minValidCollectionId"), createdCollectionPlan.CollectionId, "collection ID")
				testutil.AssertStringEqual(t, boundaryMinimalResolver.MustGetString("minValidPlanId"), createdCollectionPlan.PlanId, "plan ID")
				testutil.AssertNonEmptyString(t, createdCollectionPlan.Id, "collection plan ID")
			},
		},
		{
			Name:     "MaxValidData",
			TestCode: "ESPYNA-TEST-PRODUCT-COLLECTION-PLAN-CREATE-VALIDATION-MAXIMAL-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *collectionplanpb.CreateCollectionPlanRequest {
				return &collectionplanpb.CreateCollectionPlanRequest{
					Data: &collectionplanpb.CollectionPlan{
						CollectionId: boundaryMaximalResolver.MustGetString("maxValidCollectionId"),
						PlanId:       boundaryMaximalResolver.MustGetString("maxValidPlanId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *collectionplanpb.CreateCollectionPlanResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				createdCollectionPlan := response.Data[0]
				testutil.AssertStringEqual(t, boundaryMaximalResolver.MustGetString("maxValidCollectionId"), createdCollectionPlan.CollectionId, "collection ID")
				testutil.AssertStringEqual(t, boundaryMaximalResolver.MustGetString("maxValidPlanId"), createdCollectionPlan.PlanId, "plan ID")
				testutil.AssertNonEmptyString(t, createdCollectionPlan.Id, "collection plan ID")
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
