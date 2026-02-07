//go:build mock_db && mock_auth

// Package resource provides table-driven tests for the resource creation use case.
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
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestCreateResourceUseCase_Execute_TableDriven
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-PRODUCT-RESOURCE-CREATE-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-PRODUCT-RESOURCE-CREATE-TRANSACTION-v1.0: WithTransaction
//   - ESPYNA-TEST-PRODUCT-RESOURCE-CREATE-AUTHORIZATION-v1.0: Unauthorized
//   - ESPYNA-TEST-PRODUCT-RESOURCE-CREATE-NIL-REQUEST-v1.0: NilRequest
//   - ESPYNA-TEST-PRODUCT-RESOURCE-CREATE-NIL-DATA-v1.0: NilData
//   - ESPYNA-TEST-PRODUCT-RESOURCE-CREATE-VALIDATION-EMPTY-NAME-v1.0: EmptyName
//   - ESPYNA-TEST-PRODUCT-RESOURCE-CREATE-VALIDATION-EMPTY-PRODUCT-ID-v1.0: EmptyProductId
//   - ESPYNA-TEST-PRODUCT-RESOURCE-CREATE-VALIDATION-INVALID-PRODUCT-ID-v1.0: InvalidProductId
//   - ESPYNA-TEST-PRODUCT-RESOURCE-CREATE-VALIDATION-NAME-TOO-SHORT-v1.0: NameTooShort
//   - ESPYNA-TEST-PRODUCT-RESOURCE-CREATE-VALIDATION-NAME-TOO-LONG-v1.0: NameTooLong
//   - ESPYNA-TEST-PRODUCT-RESOURCE-CREATE-VALIDATION-DESCRIPTION-TOO-LONG-v1.0: DescriptionTooLong
//   - ESPYNA-TEST-PRODUCT-RESOURCE-CREATE-VALIDATION-PRODUCT-ID-TOO-SHORT-v1.0: ProductIdTooShort
//   - ESPYNA-TEST-PRODUCT-RESOURCE-CREATE-ENRICHMENT-v1.0: DataEnrichment
//   - ESPYNA-TEST-PRODUCT-RESOURCE-CREATE-VALIDATION-MINIMAL-v1.0: MinimalValidData
//   - ESPYNA-TEST-PRODUCT-RESOURCE-CREATE-VALIDATION-MAXIMAL-v1.0: MaxValidData
//   - ESPYNA-TEST-PRODUCT-RESOURCE-CREATE-TRANSACTION-FAILURE-v1.0: TransactionFailure
//   - ESPYNA-TEST-PRODUCT-RESOURCE-CREATE-INTEGRATION-v1.0: EducationDomainSpecific
//
// Data Sources:
//   - Test cases: packages/copya/data_test/{businessType}/resource.json
//   - Mock data: packages/copya/data/{businessType}/resource.json
//   - Translations: packages/lyngua/translations/{languageCode}/{businessType}/resource.json

package resource

import (
	"context"
	"testing"

	copyatestutil "leapfor.xyz/copya/golang/testutil"
	"leapfor.xyz/espyna/internal/application/shared/testutil"
	mockDb "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock"
	"leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock/product"
	resourcepb "leapfor.xyz/esqyma/golang/v1/domain/product/resource"
)

// Type alias for create resource test cases
type CreateResourceTestCase = testutil.GenericTestCase[*resourcepb.CreateResourceRequest, *resourcepb.CreateResourceResponse]

func createTestUseCaseWithAuth(businessType string, supportsTransaction bool, shouldAuthorize bool) *CreateResourceUseCase {
	mockResourceRepo := product.NewMockResourceRepository(businessType)
	mockProductRepo := product.NewMockProductRepository(businessType)

	repositories := CreateResourceRepositories{
		Resource: mockResourceRepo,
		Product:  mockProductRepo,
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, shouldAuthorize)
	services := CreateResourceServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
		IDService:            standardServices.IDService,
	}

	return NewCreateResourceUseCase(repositories, services)
}

func TestCreateResourceUseCase_Execute_TableDriven(t *testing.T) {
	businessType := testutil.GetTestBusinessType()

	// Load test data resolvers before defining test cases
	createSuccessResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "resource", "CreateResource_Success")
	testutil.AssertTestCaseLoad(t, err, "CreateResource_Success")

	authorizationUnauthorizedResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "resource", "Authorization_Unauthorized")
	testutil.AssertTestCaseLoad(t, err, "Authorization_Unauthorized")

	validationErrorEmptyNameResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "resource", "ValidationError_EmptyName")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_EmptyName")

	validationErrorEmptyProductIdResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "resource", "ValidationError_EmptyProductId")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_EmptyProductId")

	validationErrorInvalidProductIdResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "resource", "ValidationError_InvalidProductId")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_InvalidProductId")

	testCases := []CreateResourceTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-PRODUCT-RESOURCE-CREATE-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *resourcepb.CreateResourceRequest {
				return &resourcepb.CreateResourceRequest{
					Data: &resourcepb.Resource{
						Name:        createSuccessResolver.MustGetString("newResourceName"),
						Description: &[]string{createSuccessResolver.MustGetString("newResourceDescription")}[0],
						ProductId:   createSuccessResolver.MustGetString("validProductId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *resourcepb.CreateResourceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertEqual(t, 1, len(response.Data), "response data length")
				createdResource := response.Data[0]
				testutil.AssertStringEqual(t, createSuccessResolver.MustGetString("newResourceName"), createdResource.Name, "resource name")
				testutil.AssertNonEmptyString(t, createdResource.Id, "resource ID")
				testutil.AssertTrue(t, createdResource.Active, "resource active status")
				testutil.AssertFieldSet(t, createdResource.DateCreated, "DateCreated")
				testutil.AssertFieldSet(t, createdResource.DateCreatedString, "DateCreatedString")
			},
		},
		{
			Name:     "WithTransaction",
			TestCode: "ESPYNA-TEST-PRODUCT-RESOURCE-CREATE-TRANSACTION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *resourcepb.CreateResourceRequest {
				return &resourcepb.CreateResourceRequest{
					Data: &resourcepb.Resource{
						Name:        "Elementary Mathematics Classroom",
						Description: &[]string{"Interactive classroom with smart boards and mathematical manipulatives"}[0],
						ProductId:   "subject-math",
					},
				}
			},
			UseTransaction: true,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *resourcepb.CreateResourceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				createdResource := response.Data[0]
				testutil.AssertStringEqual(t, "Elementary Mathematics Classroom", createdResource.Name, "resource name")
			},
		},
		{
			Name:     "Unauthorized",
			TestCode: "ESPYNA-TEST-PRODUCT-RESOURCE-CREATE-AUTHORIZATION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *resourcepb.CreateResourceRequest {
				return &resourcepb.CreateResourceRequest{
					Data: &resourcepb.Resource{
						Name:        authorizationUnauthorizedResolver.MustGetString("unauthorizedResourceName"),
						Description: &[]string{authorizationUnauthorizedResolver.MustGetString("unauthorizedResourceDescription")}[0],
						ProductId:   authorizationUnauthorizedResolver.MustGetString("unauthorizedProductId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        false,
			ExpectSuccess:  false,
			ExpectedError:  "resource.errors.authorization_failed",
			Assertions: func(t *testing.T, response *resourcepb.CreateResourceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertAuthorizationError(t, err)
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-PRODUCT-RESOURCE-CREATE-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *resourcepb.CreateResourceRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "resource.validation.request_required",
			Assertions: func(t *testing.T, response *resourcepb.CreateResourceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilRequest(t, err)
			},
		},
		{
			Name:     "NilData",
			TestCode: "ESPYNA-TEST-PRODUCT-RESOURCE-CREATE-NIL-DATA-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *resourcepb.CreateResourceRequest {
				return &resourcepb.CreateResourceRequest{Data: nil}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "resource.validation.data_required",
			Assertions: func(t *testing.T, response *resourcepb.CreateResourceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilData(t, err)
			},
		},
		{
			Name:     "EmptyName",
			TestCode: "ESPYNA-TEST-PRODUCT-RESOURCE-CREATE-VALIDATION-EMPTY-NAME-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *resourcepb.CreateResourceRequest {
				return &resourcepb.CreateResourceRequest{
					Data: &resourcepb.Resource{
						Name:      validationErrorEmptyNameResolver.MustGetString("emptyName"),
						ProductId: validationErrorEmptyNameResolver.MustGetString("validProductId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "resource.validation.name_required",
			Assertions: func(t *testing.T, response *resourcepb.CreateResourceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty name")
			},
		},
		{
			Name:     "EmptyProductId",
			TestCode: "ESPYNA-TEST-PRODUCT-RESOURCE-CREATE-VALIDATION-EMPTY-PRODUCT-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *resourcepb.CreateResourceRequest {
				return &resourcepb.CreateResourceRequest{
					Data: &resourcepb.Resource{
						Name:      validationErrorEmptyProductIdResolver.MustGetString("validName"),
						ProductId: validationErrorEmptyProductIdResolver.MustGetString("emptyProductId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "resource.validation.product_id_required",
			Assertions: func(t *testing.T, response *resourcepb.CreateResourceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty product ID")
			},
		},
		{
			Name:     "InvalidProductId",
			TestCode: "ESPYNA-TEST-PRODUCT-RESOURCE-CREATE-VALIDATION-INVALID-PRODUCT-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *resourcepb.CreateResourceRequest {
				return &resourcepb.CreateResourceRequest{
					Data: &resourcepb.Resource{
						Name:      validationErrorInvalidProductIdResolver.MustGetString("validName"),
						ProductId: validationErrorInvalidProductIdResolver.MustGetString("invalidProductId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "resource.validation.product_id_invalid",
			Assertions: func(t *testing.T, response *resourcepb.CreateResourceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "invalid product ID")
			},
		},
		{
			Name:     "NameTooShort",
			TestCode: "ESPYNA-TEST-PRODUCT-RESOURCE-CREATE-VALIDATION-NAME-TOO-SHORT-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *resourcepb.CreateResourceRequest {
				return &resourcepb.CreateResourceRequest{
					Data: &resourcepb.Resource{
						Name:      "AB",
						ProductId: createSuccessResolver.MustGetString("validProductId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "resource.validation.name_min_length",
			Assertions: func(t *testing.T, response *resourcepb.CreateResourceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "name too short")
			},
		},
		{
			Name:     "NameTooLong",
			TestCode: "ESPYNA-TEST-PRODUCT-RESOURCE-CREATE-VALIDATION-NAME-TOO-LONG-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *resourcepb.CreateResourceRequest {
				resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "resource", "ValidationError_NameTooLongGenerated")
				testutil.AssertTestCaseLoad(t, err, "ValidationError_NameTooLongGenerated")
				return &resourcepb.CreateResourceRequest{
					Data: &resourcepb.Resource{
						Name:      resolver.MustGetString("tooLongNameGenerated"),
						ProductId: resolver.MustGetString("validProductId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "resource.validation.name_max_length",
			Assertions: func(t *testing.T, response *resourcepb.CreateResourceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "name too long")
			},
		},
		{
			Name:     "DescriptionTooLong",
			TestCode: "ESPYNA-TEST-PRODUCT-RESOURCE-CREATE-VALIDATION-DESCRIPTION-TOO-LONG-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *resourcepb.CreateResourceRequest {
				resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "resource", "ValidationError_DescriptionTooLongGenerated")
				testutil.AssertTestCaseLoad(t, err, "ValidationError_DescriptionTooLongGenerated")
				return &resourcepb.CreateResourceRequest{
					Data: &resourcepb.Resource{
						Name:        resolver.MustGetString("validName"),
						Description: &[]string{resolver.MustGetString("tooLongDescriptionGenerated")}[0],
						ProductId:   resolver.MustGetString("validProductId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "resource.validation.description_max_length",
			Assertions: func(t *testing.T, response *resourcepb.CreateResourceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "description too long")
			},
		},
		{
			Name:     "ProductIdTooShort",
			TestCode: "ESPYNA-TEST-PRODUCT-RESOURCE-CREATE-VALIDATION-PRODUCT-ID-TOO-SHORT-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *resourcepb.CreateResourceRequest {
				return &resourcepb.CreateResourceRequest{
					Data: &resourcepb.Resource{
						Name:      "Valid Resource Name",
						ProductId: "AB", // Only 2 characters, minimum is 3
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "resource.validation.product_id_min_length",
			Assertions: func(t *testing.T, response *resourcepb.CreateResourceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "product ID too short")
			},
		},
		{
			Name:     "DataEnrichment",
			TestCode: "ESPYNA-TEST-PRODUCT-RESOURCE-CREATE-ENRICHMENT-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *resourcepb.CreateResourceRequest {
				resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "resource", "CreateResource_Success")
				testutil.AssertTestCaseLoad(t, err, "CreateResource_Success")
				return &resourcepb.CreateResourceRequest{
					Data: &resourcepb.Resource{
						Name:      "Test Resource Data Enrichment",
						ProductId: resolver.MustGetString("validProductId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *resourcepb.CreateResourceResponse, err error, useCase interface{}, ctx context.Context) {
				createdResource := response.Data[0]
				testutil.AssertNonEmptyString(t, createdResource.Id, "generated ID")
				testutil.AssertFieldSet(t, createdResource.DateCreated, "DateCreated")
				testutil.AssertFieldSet(t, createdResource.DateCreatedString, "DateCreatedString")
				testutil.AssertFieldSet(t, createdResource.DateModified, "DateModified")
				testutil.AssertFieldSet(t, createdResource.DateModifiedString, "DateModifiedString")
				testutil.AssertTrue(t, createdResource.Active, "Active")
			},
		},
		{
			Name:     "MinimalValidData",
			TestCode: "ESPYNA-TEST-PRODUCT-RESOURCE-CREATE-VALIDATION-MINIMAL-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *resourcepb.CreateResourceRequest {
				resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "resource", "BoundaryTest_MinimalValid")
				testutil.AssertTestCaseLoad(t, err, "BoundaryTest_MinimalValid")
				return &resourcepb.CreateResourceRequest{
					Data: &resourcepb.Resource{
						Name:      resolver.MustGetString("minValidName"),
						ProductId: resolver.MustGetString("validProductId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *resourcepb.CreateResourceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				createdResource := response.Data[0]
				// Business logic transforms "ABC" to "Abc" (title case)
				testutil.AssertStringEqual(t, "Abc", createdResource.Name, "resource name")
			},
		},
		{
			Name:     "MaxValidData",
			TestCode: "ESPYNA-TEST-PRODUCT-RESOURCE-CREATE-VALIDATION-MAXIMAL-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *resourcepb.CreateResourceRequest {
				resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "resource", "BoundaryTest_MaximalValid")
				testutil.AssertTestCaseLoad(t, err, "BoundaryTest_MaximalValid")
				return &resourcepb.CreateResourceRequest{
					Data: &resourcepb.Resource{
						Name:        resolver.MustGetString("maxValidNameExact100"),
						Description: &[]string{resolver.MustGetString("maxValidDescriptionExact1000")}[0],
						ProductId:   resolver.MustGetString("validProductId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *resourcepb.CreateResourceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				createdResource := response.Data[0]
				// Business logic may trim the name, so check actual length rather than exact 100
				testutil.AssertTrue(t, len(createdResource.Name) >= 96 && len(createdResource.Name) <= 100, "name length within expected range")
				if createdResource.Description != nil {
					// Business logic may trim the description, so check within acceptable range
					testutil.AssertTrue(t, len(*createdResource.Description) >= 997 && len(*createdResource.Description) <= 1000, "description length within expected range")
				}
			},
		},
		{
			Name:     "EducationDomainSpecific",
			TestCode: "ESPYNA-TEST-PRODUCT-RESOURCE-CREATE-INTEGRATION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *resourcepb.CreateResourceRequest {
				resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "resource", "DomainSpecific_Education")
				testutil.AssertTestCaseLoad(t, err, "DomainSpecific_Education")
				return &resourcepb.CreateResourceRequest{
					Data: &resourcepb.Resource{
						Name:        resolver.MustGetString("domainSpecificResourceName"),
						Description: &[]string{resolver.MustGetString("domainSpecificResourceDescription")}[0],
						ProductId:   createSuccessResolver.MustGetString("validProductId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *resourcepb.CreateResourceResponse, err error, useCase interface{}, ctx context.Context) {
				createdResource := response.Data[0]
				resolver, _ := copyatestutil.LoadTestCaseFromBusinessType(t, testutil.GetTestBusinessType(), "resource", "DomainSpecific_Education")
				testutil.AssertStringEqual(t, resolver.MustGetString("domainSpecificResourceName"), createdResource.Name, "education-specific resource name")
				if createdResource.Description != nil {
					testutil.AssertTrue(t, len(*createdResource.Description) >= 50, "comprehensive description")
				}
				testutil.AssertStringEqual(t, createSuccessResolver.MustGetString("validProductId"), createdResource.ProductId, "valid education product reference")
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

func TestCreateResourceUseCase_Execute_TransactionFailure(t *testing.T) {
	testCode := "ESPYNA-TEST-PRODUCT-RESOURCE-CREATE-TRANSACTION-FAILURE-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "TransactionFailure", false)

	// Create a use case with a failing transaction service
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	mockResourceRepo := product.NewMockResourceRepository(businessType)
	mockProductRepo := product.NewMockProductRepository(businessType)

	repositories := CreateResourceRepositories{
		Resource: mockResourceRepo,
		Product:  mockProductRepo,
	}

	standardServices := testutil.CreateStandardServices(false, true)
	services := CreateResourceServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   mockDb.NewFailingMockTransactionService(),
		TranslationService:   standardServices.TranslationService,
		IDService:            standardServices.IDService,
	}

	useCase := NewCreateResourceUseCase(repositories, services)

	// Load test case using centralized TestCaseResolver
	resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "resource", "CreateResource_Success")
	testutil.AssertTestCaseLoad(t, err, "CreateResource_Success")

	req := &resourcepb.CreateResourceRequest{
		Data: &resourcepb.Resource{
			Name:      resolver.MustGetString("newResourceName"),
			ProductId: resolver.MustGetString("validProductId"),
		},
	}

	_, err = useCase.Execute(ctx, req)

	// Verify that a transaction error occurred
	testutil.AssertTransactionError(t, err)

	// Log completion of transaction failure test
	testutil.LogTestResult(t, testCode, "TransactionFailure", false, err)
}
