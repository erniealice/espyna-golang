//go:build mock_db && mock_auth

// Package collection_plan provides table-driven tests for the collection plan listing use case.
//
// The tests cover various scenarios, including success, authorization, empty lists,
// detail verification, and nil requests. Each test case is defined in a table
// with a specific test code, request setup, and assertions.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user").
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education").
//
// Usage:
//   - Run all tests: go test -tags="mock_db,mock_auth" ./...
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestListCollectionPlansUseCase_Execute_TableDriven
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-PRODUCT-COLLECTION-PLAN-LIST-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-PRODUCT-COLLECTION-PLAN-LIST-AUTHORIZATION-v1.0: Unauthorized
//   - ESPYNA-TEST-PRODUCT-COLLECTION-PLAN-LIST-EMPTY-v1.0: EmptyList
//   - ESPYNA-TEST-PRODUCT-COLLECTION-PLAN-LIST-VERIFY-DETAILS-v1.0: VerifyDetails
//   - ESPYNA-TEST-PRODUCT-COLLECTION-PLAN-LIST-NIL-REQUEST-v1.0: NilRequest
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
	collectionplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/collection_plan"
)

// Type alias for list collection plans test cases
type ListCollectionPlansTestCase = testutil.GenericTestCase[*collectionplanpb.ListCollectionPlansRequest, *collectionplanpb.ListCollectionPlansResponse]

func listTestUseCaseWithAuth(businessType string, shouldAuthorize bool, repoOptions ...product.CollectionPlanRepositoryOption) *ListCollectionPlansUseCase {
	mockCollectionPlanRepo := product.NewMockCollectionPlanRepository(businessType, repoOptions...)

	repositories := ListCollectionPlansRepositories{
		CollectionPlan: mockCollectionPlanRepo,
	}

	standardServices := testutil.CreateStandardServices(false, shouldAuthorize)
	services := ListCollectionPlansServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
	}

	return NewListCollectionPlansUseCase(repositories, services)
}

func TestListCollectionPlansUseCase_Execute_TableDriven(t *testing.T) {
	businessType := testutil.GetTestBusinessType()

	// Load test data resolvers before defining test cases
	listSuccessResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "collection_plan", "ListCollectionPlans_Success")
	testutil.AssertTestCaseLoad(t, err, "ListCollectionPlans_Success")

	listVerifyDetailsResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "collection_plan", "ListCollectionPlans_VerifyDetails")
	testutil.AssertTestCaseLoad(t, err, "ListCollectionPlans_VerifyDetails")

	testCases := []ListCollectionPlansTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-PRODUCT-COLLECTION-PLAN-LIST-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *collectionplanpb.ListCollectionPlansRequest {
				return &collectionplanpb.ListCollectionPlansRequest{}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *collectionplanpb.ListCollectionPlansResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				expectedCount := listSuccessResolver.MustGetInt("expectedCollectionPlanCount")
				testutil.AssertEqual(t, expectedCount, len(response.Data), "collection plan count")
				if len(response.Data) > 0 {
					// Verify first collection plan has required fields
					firstCollectionPlan := response.Data[0]
					testutil.AssertNonEmptyString(t, firstCollectionPlan.Id, "collection plan ID")
					testutil.AssertNonEmptyString(t, firstCollectionPlan.CollectionId, "collection ID")
					testutil.AssertNonEmptyString(t, firstCollectionPlan.PlanId, "plan ID")
					testutil.AssertFieldSet(t, firstCollectionPlan.DateCreated, "DateCreated")
					testutil.AssertFieldSet(t, firstCollectionPlan.DateCreatedString, "DateCreatedString")
				}
			},
		},
		{
			Name:     "Unauthorized",
			TestCode: "ESPYNA-TEST-PRODUCT-COLLECTION-PLAN-LIST-AUTHORIZATION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *collectionplanpb.ListCollectionPlansRequest {
				return &collectionplanpb.ListCollectionPlansRequest{}
			},
			UseTransaction: false,
			UseAuth:        false,
			ExpectSuccess:  false,
			ExpectedError:  "collection_plan.errors.authorization_failed",
			Assertions: func(t *testing.T, response *collectionplanpb.ListCollectionPlansResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertAuthorizationError(t, err)
			},
		},
		{
			Name:     "EmptyList",
			TestCode: "ESPYNA-TEST-PRODUCT-COLLECTION-PLAN-LIST-EMPTY-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *collectionplanpb.ListCollectionPlansRequest {
				return &collectionplanpb.ListCollectionPlansRequest{}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *collectionplanpb.ListCollectionPlansResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertEqual(t, 0, len(response.Data), "empty collection plan list")
			},
		},
		{
			Name:     "VerifyDetails",
			TestCode: "ESPYNA-TEST-PRODUCT-COLLECTION-PLAN-LIST-VERIFY-DETAILS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *collectionplanpb.ListCollectionPlansRequest {
				return &collectionplanpb.ListCollectionPlansRequest{}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *collectionplanpb.ListCollectionPlansResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertTrue(t, len(response.Data) > 0, "collection plans exist")

				// Get verification targets from test data
				// Access the verificationTargets as a raw interface{} and cast it
				verificationTargetsRaw := listVerifyDetailsResolver.GetTestCase().DataReferences["verificationTargets"]
				verificationTargets, ok := verificationTargetsRaw.([]interface{})
				if !ok {
					t.Fatalf("verificationTargets is not a slice")
				}

				for _, targetInterface := range verificationTargets {
					target, ok := targetInterface.(map[string]interface{})
					if !ok {
						t.Fatalf("verification target is not a map")
					}
					targetId := target["id"].(string)
					expectedCollectionId := target["expectedCollectionId"].(string)
					expectedPlanId := target["expectedPlanId"].(string)
					expectedActive := target["expectedActive"].(bool)

					// Find the collection plan in response
					var foundCollectionPlan *collectionplanpb.CollectionPlan
					for _, cp := range response.Data {
						if cp.Id == targetId {
							foundCollectionPlan = cp
							break
						}
					}

					if foundCollectionPlan != nil {
						testutil.AssertStringEqual(t, expectedCollectionId, foundCollectionPlan.CollectionId, "collection ID for "+targetId)
						testutil.AssertStringEqual(t, expectedPlanId, foundCollectionPlan.PlanId, "plan ID for "+targetId)
						testutil.AssertEqual(t, expectedActive, foundCollectionPlan.Active, "active status for "+targetId)
					}
				}
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-PRODUCT-COLLECTION-PLAN-LIST-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *collectionplanpb.ListCollectionPlansRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "collection_plan.validation.request_required",
			Assertions: func(t *testing.T, response *collectionplanpb.ListCollectionPlansResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilRequest(t, err)
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

			// Setup repository options for empty data test case
			var repoOptions []product.CollectionPlanRepositoryOption
			if tc.Name == "EmptyList" {
				repoOptions = append(repoOptions, product.WithoutCollectionPlanInitialData())
			}

			useCase := listTestUseCaseWithAuth(businessType, tc.UseAuth, repoOptions...)

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
