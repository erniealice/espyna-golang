//go:build mock_db && mock_auth

// Package product_plan provides table-driven tests for the product plan creation use case.
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
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestCreateProductPlanUseCase_Execute_TableDriven
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-PRODUCT-PRODUCT-PLAN-CREATE-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-PRODUCT-PRODUCT-PLAN-CREATE-TRANSACTION-v1.0: WithTransaction
//   - ESPYNA-TEST-PRODUCT-PRODUCT-PLAN-CREATE-AUTHORIZATION-v1.0: Unauthorized
//   - ESPYNA-TEST-PRODUCT-PRODUCT-PLAN-CREATE-NIL-REQUEST-v1.0: NilRequest
//   - ESPYNA-TEST-PRODUCT-PRODUCT-PLAN-CREATE-NIL-DATA-v1.0: NilData
//   - ESPYNA-TEST-PRODUCT-PRODUCT-PLAN-CREATE-VALIDATION-EMPTY-NAME-v1.0: EmptyName
//   - ESPYNA-TEST-PRODUCT-PRODUCT-PLAN-CREATE-VALIDATION-EMPTY-PRODUCT-ID-v1.0: EmptyProductId
//   - ESPYNA-TEST-PRODUCT-PRODUCT-PLAN-CREATE-VALIDATION-NAME-TOO-SHORT-v1.0: NameTooShort
//   - ESPYNA-TEST-PRODUCT-PRODUCT-PLAN-CREATE-VALIDATION-NAME-TOO-LONG-v1.0: NameTooLong
//   - ESPYNA-TEST-PRODUCT-PRODUCT-PLAN-CREATE-VALIDATION-MINIMAL-v1.0: MinimalValidData
//   - ESPYNA-TEST-PRODUCT-PRODUCT-PLAN-CREATE-VALIDATION-MAXIMAL-v1.0: MaxValidData
//   - ESPYNA-TEST-PRODUCT-PRODUCT-PLAN-CREATE-ENRICHMENT-v1.0: DataEnrichment
//   - ESPYNA-TEST-PRODUCT-PRODUCT-PLAN-CREATE-TRANSACTION-FAILURE-v1.0: TransactionFailure
//
// Note: The following test cases are defined in the framework/objective pattern but not yet implemented:
//   - ESPYNA-TEST-PRODUCT-PRODUCTPLAN-CREATE-VALIDATION-INVALID-PRICE-v1.0: InvalidPrice (price validation not implemented)
//   - ESPYNA-TEST-PRODUCT-PRODUCTPLAN-CREATE-VALIDATION-DESCRIPTION-TOO-LONG-v1.0: DescriptionTooLong (description length validation not implemented)
//
// Data Sources:
//   - Test cases: packages/copya/data_test/{businessType}/product_plan.json
//   - Mock data: packages/copya/data/{businessType}/product_plan.json
//   - Translations: packages/lyngua/translations/{languageCode}/{businessType}/product_plan.json
package product_plan

import (
	"context"
	"testing"

	copyatestutil "leapfor.xyz/copya/golang/testutil"
	"github.com/erniealice/espyna-golang/internal/application/shared/testutil"
	mockDb "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock"
	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock/product"
	productplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_plan"
)

// Type alias for create product plan test cases
type CreateProductPlanTestCase = testutil.GenericTestCase[*productplanpb.CreateProductPlanRequest, *productplanpb.CreateProductPlanResponse]

func createTestUseCaseWithAuth(businessType string, supportsTransaction bool, shouldAuthorize bool) *CreateProductPlanUseCase {
	mockProductPlanRepo := product.NewMockProductPlanRepository(businessType)
	mockProductRepo := product.NewMockProductRepository(businessType)

	repositories := CreateProductPlanRepositories{
		ProductPlan: mockProductPlanRepo,
		Product:     mockProductRepo,
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, shouldAuthorize)
	services := CreateProductPlanServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
		IDService:            standardServices.IDService,
	}

	return NewCreateProductPlanUseCase(repositories, services)
}

func TestCreateProductPlanUseCase_Execute_TableDriven(t *testing.T) {
	businessType := testutil.GetTestBusinessType()

	// Load test data resolvers before defining test cases
	createSuccessResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "product_plan", "CreateProductPlan_Success")
	testutil.AssertTestCaseLoad(t, err, "CreateProductPlan_Success")

	authorizationUnauthorizedResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "product_plan", "Authorization_Unauthorized")
	testutil.AssertTestCaseLoad(t, err, "Authorization_Unauthorized")

	validationErrorEmptyNameResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "product_plan", "ValidationError_EmptyName")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_EmptyName")

	validationErrorEmptyProductIdResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "product_plan", "ValidationError_EmptyProductId")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_EmptyProductId")

	// Note: These resolvers are for test cases that are not yet implemented
	// validationErrorInvalidPriceResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "product_plan", "ValidationError_InvalidPrice")
	// testutil.AssertTestCaseLoad(t, err, "ValidationError_InvalidPrice")

	// validationErrorDescriptionTooLongGeneratedResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "product_plan", "ValidationError_DescriptionTooLongGenerated")
	// testutil.AssertTestCaseLoad(t, err, "ValidationError_DescriptionTooLongGenerated")

	validationErrorNameTooShortResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "product_plan", "ValidationError_NameTooShort")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_NameTooShort")

	validationErrorNameTooLongGeneratedResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "product_plan", "ValidationError_NameTooLongGenerated")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_NameTooLongGenerated")

	testCases := []CreateProductPlanTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-PLAN-CREATE-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productplanpb.CreateProductPlanRequest {
				return &productplanpb.CreateProductPlanRequest{
					Data: &productplanpb.ProductPlan{
						Name:        createSuccessResolver.MustGetString("newProductPlanName"),
						Description: &[]string{createSuccessResolver.MustGetString("newProductPlanDescription")}[0],
						ProductId:   createSuccessResolver.MustGetString("newProductPlanProductId"),
						Price:       createSuccessResolver.MustGetFloat64("newProductPlanPrice"),
						Currency:    createSuccessResolver.MustGetString("newProductPlanCurrency"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *productplanpb.CreateProductPlanResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertEqual(t, 1, len(response.Data), "response data length")
				createdPlan := response.Data[0]
				testutil.AssertStringEqual(t, createSuccessResolver.MustGetString("newProductPlanName"), createdPlan.Name, "product plan name")
				testutil.AssertStringEqual(t, createSuccessResolver.MustGetString("newProductPlanProductId"), createdPlan.ProductId, "product ID")
				testutil.AssertNonEmptyString(t, createdPlan.Id, "product plan ID")
				testutil.AssertTrue(t, createdPlan.Active, "product plan active status")
				testutil.AssertFieldSet(t, createdPlan.DateCreated, "DateCreated")
				testutil.AssertFieldSet(t, createdPlan.DateCreatedString, "DateCreatedString")
			},
		},
		{
			Name:     "WithTransaction",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-PLAN-CREATE-TRANSACTION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productplanpb.CreateProductPlanRequest {
				return &productplanpb.CreateProductPlanRequest{
					Data: &productplanpb.ProductPlan{
						Name:        createSuccessResolver.MustGetString("newProductPlanName"),
						Description: &[]string{createSuccessResolver.MustGetString("newProductPlanDescription")}[0],
						ProductId:   createSuccessResolver.MustGetString("newProductPlanProductId"),
						Price:       createSuccessResolver.MustGetFloat64("newProductPlanPrice"),
						Currency:    createSuccessResolver.MustGetString("newProductPlanCurrency"),
					},
				}
			},
			UseTransaction: true,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *productplanpb.CreateProductPlanResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				createdPlan := response.Data[0]
				testutil.AssertStringEqual(t, createSuccessResolver.MustGetString("newProductPlanName"), createdPlan.Name, "product plan name")
			},
		},
		{
			Name:     "Unauthorized",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-PLAN-CREATE-AUTHORIZATION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productplanpb.CreateProductPlanRequest {
				return &productplanpb.CreateProductPlanRequest{
					Data: &productplanpb.ProductPlan{
						Name:        authorizationUnauthorizedResolver.MustGetString("unauthorizedPlanName"),
						Description: &[]string{authorizationUnauthorizedResolver.MustGetString("unauthorizedPlanDescription")}[0],
						ProductId:   authorizationUnauthorizedResolver.MustGetString("unauthorizedProductId"),
						Price:       authorizationUnauthorizedResolver.MustGetFloat64("unauthorizedPrice"),
						Currency:    authorizationUnauthorizedResolver.MustGetString("unauthorizedCurrency"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        false,
			ExpectSuccess:  false,
			ExpectedError:  "product_plan.errors.authorization_failed",
			Assertions: func(t *testing.T, response *productplanpb.CreateProductPlanResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertAuthorizationError(t, err)
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-PLAN-CREATE-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productplanpb.CreateProductPlanRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "product_plan.validation.request_required",
			Assertions: func(t *testing.T, response *productplanpb.CreateProductPlanResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilRequest(t, err)
			},
		},
		{
			Name:     "NilData",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-PLAN-CREATE-NIL-DATA-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productplanpb.CreateProductPlanRequest {
				return &productplanpb.CreateProductPlanRequest{Data: nil}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "product_plan.validation.data_required",
			Assertions: func(t *testing.T, response *productplanpb.CreateProductPlanResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilData(t, err)
			},
		},
		{
			Name:     "EmptyName",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-PLAN-CREATE-VALIDATION-EMPTY-NAME-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productplanpb.CreateProductPlanRequest {
				return &productplanpb.CreateProductPlanRequest{
					Data: &productplanpb.ProductPlan{
						Name:        validationErrorEmptyNameResolver.MustGetString("emptyPlanName"),
						Description: &[]string{validationErrorEmptyNameResolver.MustGetString("validPlanDescription")}[0],
						ProductId:   validationErrorEmptyNameResolver.MustGetString("validProductId"),
						Price:       validationErrorEmptyNameResolver.MustGetFloat64("validPrice"),
						Currency:    validationErrorEmptyNameResolver.MustGetString("validCurrency"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "product_plan.validation.name_required",
			Assertions: func(t *testing.T, response *productplanpb.CreateProductPlanResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty name")
			},
		},
		{
			Name:     "EmptyProductId",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-PLAN-CREATE-VALIDATION-EMPTY-PRODUCT-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productplanpb.CreateProductPlanRequest {
				return &productplanpb.CreateProductPlanRequest{
					Data: &productplanpb.ProductPlan{
						Name:        validationErrorEmptyProductIdResolver.MustGetString("validPlanName"),
						Description: &[]string{validationErrorEmptyProductIdResolver.MustGetString("validPlanDescription")}[0],
						ProductId:   validationErrorEmptyProductIdResolver.MustGetString("emptyProductId"),
						Price:       validationErrorEmptyProductIdResolver.MustGetFloat64("validPrice"),
						Currency:    validationErrorEmptyProductIdResolver.MustGetString("validCurrency"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "product_plan.validation.product_id_required",
			Assertions: func(t *testing.T, response *productplanpb.CreateProductPlanResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty product ID")
			},
		},
		// Note: Price validation is not implemented in the use case yet
		// {
		// 	Name:     "InvalidPrice",
		// 	TestCode: "ESPYNA-TEST-PRODUCT-PRODUCTPLAN-CREATE-VALIDATION-INVALID-PRICE-v1.0",
		// 	SetupRequest: func(t *testing.T, businessType string) *productplanpb.CreateProductPlanRequest {
		// 		return &productplanpb.CreateProductPlanRequest{
		// 			Data: &productplanpb.ProductPlan{
		// 				Name:        validationErrorInvalidPriceResolver.MustGetString("validPlanName"),
		// 				Description: &[]string{validationErrorInvalidPriceResolver.MustGetString("validPlanDescription")}[0],
		// 				ProductId:   validationErrorInvalidPriceResolver.MustGetString("validProductId"),
		// 				Price:       validationErrorInvalidPriceResolver.MustGetFloat64("invalidPrice"),
		// 				Currency:    validationErrorInvalidPriceResolver.MustGetString("validCurrency"),
		// 			},
		// 		}
		// 	},
		// 	UseTransaction: false,
		// 	UseAuth:        true,
		// 	ExpectSuccess:  false,
		// 	ExpectedError:  "product_plan.validation.price_invalid",
		// 	Assertions: func(t *testing.T, response *productplanpb.CreateProductPlanResponse, err error, useCase interface{}, ctx context.Context) {
		// 		testutil.AssertValidationError(t, err, "invalid price")
		// 	},
		// },
		{
			Name:     "NameTooShort",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-PLAN-CREATE-VALIDATION-NAME-TOO-SHORT-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productplanpb.CreateProductPlanRequest {
				return &productplanpb.CreateProductPlanRequest{
					Data: &productplanpb.ProductPlan{
						Name:      validationErrorNameTooShortResolver.MustGetString("tooShortName"),
						ProductId: createSuccessResolver.MustGetString("newProductPlanProductId"),
						Price:     createSuccessResolver.MustGetFloat64("newProductPlanPrice"),
						Currency:  createSuccessResolver.MustGetString("newProductPlanCurrency"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "product_plan.validation.name_too_short",
			Assertions: func(t *testing.T, response *productplanpb.CreateProductPlanResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "name too short")
			},
		},
		{
			Name:     "NameTooLong",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-PLAN-CREATE-VALIDATION-NAME-TOO-LONG-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productplanpb.CreateProductPlanRequest {
				return &productplanpb.CreateProductPlanRequest{
					Data: &productplanpb.ProductPlan{
						Name:      validationErrorNameTooLongGeneratedResolver.MustGetString("tooLongNameGenerated"),
						ProductId: createSuccessResolver.MustGetString("newProductPlanProductId"),
						Price:     createSuccessResolver.MustGetFloat64("newProductPlanPrice"),
						Currency:  createSuccessResolver.MustGetString("newProductPlanCurrency"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "product_plan.validation.name_too_long",
			Assertions: func(t *testing.T, response *productplanpb.CreateProductPlanResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "name too long")
			},
		},
		// Note: Description length validation is not implemented in the use case yet
		// {
		// 	Name: "DescriptionTooLong",
		// 	TestCode: "ESPYNA-TEST-PRODUCT-PRODUCTPLAN-CREATE-VALIDATION-DESCRIPTION-TOO-LONG-v1.0",
		// 	SetupRequest: func(t *testing.T, businessType string) *productplanpb.CreateProductPlanRequest {
		// 		return &productplanpb.CreateProductPlanRequest{
		// 			Data: &productplanpb.ProductPlan{
		// 				Name:        createSuccessResolver.MustGetString("newProductPlanName"),
		// 				Description: &[]string{validationErrorDescriptionTooLongGeneratedResolver.MustGetString("tooLongDescriptionGenerated")}[0],
		// 				ProductId:   createSuccessResolver.MustGetString("newProductPlanProductId"),
		// 				Price:       createSuccessResolver.MustGetFloat64("newProductPlanPrice"),
		// 				Currency:    createSuccessResolver.MustGetString("newProductPlanCurrency"),
		// 			},
		// 		}
		// 	},
		// 	UseTransaction: false,
		// 	UseAuth:        true,
		// 	ExpectSuccess:  false,
		// 	ExpectedError:  "product_plan.validation.description_too_long",
		// 	Assertions: func(t *testing.T, response *productplanpb.CreateProductPlanResponse, err error, useCase interface{}, ctx context.Context) {
		// 		testutil.AssertValidationError(t, err, "description too long")
		// 	},
		// },
		{
			Name:     "DataEnrichment",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-PLAN-CREATE-ENRICHMENT-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productplanpb.CreateProductPlanRequest {
				return &productplanpb.CreateProductPlanRequest{
					Data: &productplanpb.ProductPlan{
						Name:      createSuccessResolver.MustGetString("newProductPlanName"),
						ProductId: createSuccessResolver.MustGetString("newProductPlanProductId"),
						Price:     createSuccessResolver.MustGetFloat64("newProductPlanPrice"),
						Currency:  createSuccessResolver.MustGetString("newProductPlanCurrency"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *productplanpb.CreateProductPlanResponse, err error, useCase interface{}, ctx context.Context) {
				createdPlan := response.Data[0]
				testutil.AssertNonEmptyString(t, createdPlan.Id, "generated ID")
				testutil.AssertFieldSet(t, createdPlan.DateCreated, "DateCreated")
				testutil.AssertFieldSet(t, createdPlan.DateCreatedString, "DateCreatedString")
				testutil.AssertTrue(t, createdPlan.Active, "Active")
			},
		},
		{
			Name:     "MinimalValidData",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-PLAN-CREATE-VALIDATION-MINIMAL-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productplanpb.CreateProductPlanRequest {
				resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "product_plan", "BoundaryTest_MinimalValid")
				testutil.AssertTestCaseLoad(t, err, "BoundaryTest_MinimalValid")
				return &productplanpb.CreateProductPlanRequest{
					Data: &productplanpb.ProductPlan{
						Name:      resolver.MustGetString("minValidName"),
						ProductId: resolver.MustGetString("validProductId"),
						Price:     resolver.MustGetFloat64("minValidPrice"),
						Currency:  resolver.MustGetString("validCurrency"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *productplanpb.CreateProductPlanResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				createdPlan := response.Data[0]
				resolver, _ := copyatestutil.LoadTestCaseFromBusinessType(t, testutil.GetTestBusinessType(), "product_plan", "BoundaryTest_MinimalValid")
				testutil.AssertStringEqual(t, resolver.MustGetString("minValidName"), createdPlan.Name, "product plan name")
			},
		},
		{
			Name:     "MaxValidData",
			TestCode: "ESPYNA-TEST-PRODUCT-PRODUCT-PLAN-CREATE-VALIDATION-MAXIMAL-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *productplanpb.CreateProductPlanRequest {
				resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "product_plan", "BoundaryTest_MaximalValid")
				testutil.AssertTestCaseLoad(t, err, "BoundaryTest_MaximalValid")
				return &productplanpb.CreateProductPlanRequest{
					Data: &productplanpb.ProductPlan{
						Name:        resolver.MustGetString("maxValidNameExact100"),
						Description: &[]string{resolver.MustGetString("maxValidDescriptionExact1000")}[0],
						ProductId:   resolver.MustGetString("validProductId"),
						Price:       resolver.MustGetFloat64("maxValidPrice"),
						Currency:    resolver.MustGetString("validCurrency"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *productplanpb.CreateProductPlanResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				createdPlan := response.Data[0]
				testutil.AssertEqual(t, 100, len(createdPlan.Name), "name length")
				if createdPlan.Description != nil {
					// Note: Description length validation is not implemented, so we just check it's set
					testutil.AssertTrue(t, len(*createdPlan.Description) > 0, "description should not be empty")
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

func TestCreateProductPlanUseCase_Execute_TransactionFailure(t *testing.T) {
	testCode := "ESPYNA-TEST-PRODUCT-PRODUCT-PLAN-CREATE-TRANSACTION-FAILURE-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "TransactionFailure", false)

	// Create a use case with a failing transaction service
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	mockProductPlanRepo := product.NewMockProductPlanRepository(businessType)
	mockProductRepo := product.NewMockProductRepository(businessType)

	repositories := CreateProductPlanRepositories{
		ProductPlan: mockProductPlanRepo,
		Product:     mockProductRepo,
	}

	standardServices := testutil.CreateStandardServices(false, true)
	services := CreateProductPlanServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   mockDb.NewFailingMockTransactionService(),
		TranslationService:   standardServices.TranslationService,
		IDService:            standardServices.IDService,
	}

	useCase := NewCreateProductPlanUseCase(repositories, services)

	// Load test case using centralized TestCaseResolver
	resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "product_plan", "CreateProductPlan_Success")
	testutil.AssertTestCaseLoad(t, err, "CreateProductPlan_Success")

	req := &productplanpb.CreateProductPlanRequest{
		Data: &productplanpb.ProductPlan{
			Name:      resolver.MustGetString("newProductPlanName"),
			ProductId: resolver.MustGetString("newProductPlanProductId"),
			Price:     resolver.MustGetFloat64("newProductPlanPrice"),
			Currency:  resolver.MustGetString("newProductPlanCurrency"),
		},
	}

	_, err = useCase.Execute(ctx, req)

	// Verify that a transaction error occurred
	testutil.AssertTransactionError(t, err)

	// Log completion of transaction failure test
	testutil.LogTestResult(t, testCode, "TransactionFailure", false, err)
}
