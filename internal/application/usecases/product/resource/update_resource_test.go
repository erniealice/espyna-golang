//go:build mock_db && mock_auth

// Package resource provides table-driven tests for the resource update use case.
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
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestUpdateResourceUseCase_Execute_TableDriven
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-PRODUCT-RESOURCE-UPDATE-SUCCESS-v1.0: UpdateName
//   - ESPYNA-TEST-PRODUCT-RESOURCE-UPDATE-INTEGRATION-v1.0: UpdateProductReference
//   - ESPYNA-TEST-PRODUCT-RESOURCE-UPDATE-TRANSACTION-v1.0: WithTransaction
//   - ESPYNA-TEST-PRODUCT-RESOURCE-UPDATE-VALIDATION-INVALID-PRODUCT-v1.0: InvalidProductReference
//   - ESPYNA-TEST-PRODUCT-RESOURCE-UPDATE-VALIDATION-NOT-FOUND-v1.0: ResourceNotFound
//   - ESPYNA-TEST-PRODUCT-RESOURCE-UPDATE-NIL-REQUEST-v1.0: NilRequest
//   - ESPYNA-TEST-PRODUCT-RESOURCE-UPDATE-NIL-DATA-v1.0: NilData
//   - ESPYNA-TEST-PRODUCT-RESOURCE-UPDATE-VALIDATION-EMPTY-ID-v1.0: EmptyId
//   - ESPYNA-TEST-PRODUCT-RESOURCE-UPDATE-VALIDATION-EMPTY-NAME-v1.0: EmptyName
//   - ESPYNA-TEST-PRODUCT-RESOURCE-UPDATE-VALIDATION-EMPTY-PRODUCT-ID-v1.0: EmptyProductId
//   - ESPYNA-TEST-PRODUCT-RESOURCE-UPDATE-VALIDATION-NAME-TOO-SHORT-v1.0: NameTooShort
//   - ESPYNA-TEST-PRODUCT-RESOURCE-UPDATE-VALIDATION-NAME-TOO-LONG-v1.0: NameTooLong
//   - ESPYNA-TEST-PRODUCT-RESOURCE-UPDATE-VALIDATION-DESCRIPTION-TOO-LONG-v1.0: DescriptionTooLong
//   - ESPYNA-TEST-PRODUCT-RESOURCE-UPDATE-VALIDATION-PRODUCT-ID-TOO-SHORT-v1.0: ProductIdTooShort
//   - ESPYNA-TEST-PRODUCT-RESOURCE-UPDATE-ENRICHMENT-v1.0: DataEnrichment
//   - ESPYNA-TEST-PRODUCT-RESOURCE-UPDATE-VALIDATION-MINIMAL-v1.0: MinimalValidData
//   - ESPYNA-TEST-PRODUCT-RESOURCE-UPDATE-INTEGRATION-EDUCATION-v1.0: EducationDomainSpecific
//   - ESPYNA-TEST-PRODUCT-RESOURCE-UPDATE-TRANSACTION-FAILURE-v1.0: TransactionFailure
//
// Data Sources:
//   - Test cases: packages/copya/data_test/{businessType}/resource.json
//   - Mock data: packages/copya/data/{businessType}/resource.json
//   - Translations: packages/lyngua/translations/{languageCode}/{businessType}/resource.json

package resource

import (
	"context"
	"testing"
	"time"

	copyatestutil "leapfor.xyz/copya/golang/testutil"
	"github.com/erniealice/espyna-golang/internal/application/shared/testutil"
	mockDb "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock"
	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock/product"
	resourcepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/resource"
)

// Type alias for update resource test cases
type UpdateResourceTestCase = testutil.GenericTestCase[*resourcepb.UpdateResourceRequest, *resourcepb.UpdateResourceResponse]

func createUpdateTestUseCaseWithAuth(businessType string, supportsTransaction bool, shouldAuthorize bool) *UpdateResourceUseCase {
	mockResourceRepo := product.NewMockResourceRepository(businessType)
	mockProductRepo := product.NewMockProductRepository(businessType)

	repositories := UpdateResourceRepositories{
		Resource: mockResourceRepo,
		Product:  mockProductRepo,
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, shouldAuthorize)
	services := UpdateResourceServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
	}

	return NewUpdateResourceUseCase(repositories, services)
}

func TestUpdateResourceUseCase_Execute_TableDriven(t *testing.T) {
	businessType := testutil.GetTestBusinessType()

	// Load test data resolvers before defining test cases
	updateSuccessResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "resource", "UpdateResource_Success")
	testutil.AssertTestCaseLoad(t, err, "UpdateResource_Success")

	validationErrorEmptyNameResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "resource", "ValidationError_EmptyName")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_EmptyName")

	validationErrorEmptyProductIdResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "resource", "ValidationError_EmptyProductId")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_EmptyProductId")

	validationErrorInvalidProductIdResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "resource", "ValidationError_InvalidProductId")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_InvalidProductId")

	testCases := []UpdateResourceTestCase{
		{
			Name:     "UpdateName",
			TestCode: "ESPYNA-TEST-PRODUCT-RESOURCE-UPDATE-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *resourcepb.UpdateResourceRequest {
				return &resourcepb.UpdateResourceRequest{
					Data: &resourcepb.Resource{
						Id:          "resource-classroom-101",
						Name:        updateSuccessResolver.MustGetString("enhancedLibraryName"),
						Description: &[]string{updateSuccessResolver.MustGetString("enhancedLibraryDescription")}[0],
						ProductId:   "subject-math",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *resourcepb.UpdateResourceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertEqual(t, 1, len(response.Data), "response data length")
				updatedResource := response.Data[0]
				testutil.AssertStringEqual(t, updateSuccessResolver.MustGetString("enhancedLibraryName"), updatedResource.Name, "resource name")
				testutil.AssertStringEqual(t, "resource-classroom-101", updatedResource.Id, "resource ID")
				testutil.AssertFieldSet(t, updatedResource.DateModified, "DateModified")
				testutil.AssertFieldSet(t, updatedResource.DateModifiedString, "DateModifiedString")
			},
		},
		{
			Name:     "UpdateProductReference",
			TestCode: "ESPYNA-TEST-PRODUCT-RESOURCE-UPDATE-INTEGRATION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *resourcepb.UpdateResourceRequest {
				return &resourcepb.UpdateResourceRequest{
					Data: &resourcepb.Resource{
						Id:          "resource-lab-science",
						Name:        "Interdisciplinary STEM Laboratory",
						Description: &[]string{"Multi-purpose lab supporting both science and mathematics instruction"}[0],
						ProductId:   "subject-math", // Change from science to math
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *resourcepb.UpdateResourceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				updatedResource := response.Data[0]
				testutil.AssertStringEqual(t, "subject-math", updatedResource.ProductId, "product ID")
			},
		},
		{
			Name:     "WithTransaction",
			TestCode: "ESPYNA-TEST-PRODUCT-RESOURCE-UPDATE-TRANSACTION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *resourcepb.UpdateResourceRequest {
				return &resourcepb.UpdateResourceRequest{
					Data: &resourcepb.Resource{
						Id:          "resource-auditorium",
						Name:        "Main Performance Auditorium",
						Description: &[]string{"Large auditorium with professional lighting and sound systems for school performances"}[0],
						ProductId:   "subject-english",
					},
				}
			},
			UseTransaction: true,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *resourcepb.UpdateResourceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				updatedResource := response.Data[0]
				testutil.AssertStringEqual(t, "Main Performance Auditorium", updatedResource.Name, "resource name")
			},
		},
		{
			Name:     "InvalidProductReference",
			TestCode: "ESPYNA-TEST-PRODUCT-RESOURCE-UPDATE-VALIDATION-INVALID-PRODUCT-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *resourcepb.UpdateResourceRequest {
				return &resourcepb.UpdateResourceRequest{
					Data: &resourcepb.Resource{
						Id:        "resource-classroom-101",
						Name:      "Classroom with Invalid Product Reference",
						ProductId: validationErrorInvalidProductIdResolver.MustGetString("invalidProductId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "resource.validation.product_id_invalid",
			Assertions: func(t *testing.T, response *resourcepb.UpdateResourceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "invalid product reference")
			},
		},
		{
			Name:     "ResourceNotFound",
			TestCode: "ESPYNA-TEST-PRODUCT-RESOURCE-UPDATE-VALIDATION-NOT-FOUND-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *resourcepb.UpdateResourceRequest {
				return &resourcepb.UpdateResourceRequest{
					Data: &resourcepb.Resource{
						Id:        "non-existent-resource",
						Name:      "This Resource Does Not Exist",
						ProductId: "subject-math",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			Assertions: func(t *testing.T, response *resourcepb.UpdateResourceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertError(t, err)
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-PRODUCT-RESOURCE-UPDATE-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *resourcepb.UpdateResourceRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "resource.validation.request_required",
			Assertions: func(t *testing.T, response *resourcepb.UpdateResourceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilRequest(t, err)
			},
		},
		{
			Name:     "NilData",
			TestCode: "ESPYNA-TEST-PRODUCT-RESOURCE-UPDATE-NIL-DATA-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *resourcepb.UpdateResourceRequest {
				return &resourcepb.UpdateResourceRequest{Data: nil}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "resource.validation.data_required",
			Assertions: func(t *testing.T, response *resourcepb.UpdateResourceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilData(t, err)
			},
		},
		{
			Name:     "EmptyId",
			TestCode: "ESPYNA-TEST-PRODUCT-RESOURCE-UPDATE-VALIDATION-EMPTY-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *resourcepb.UpdateResourceRequest {
				return &resourcepb.UpdateResourceRequest{
					Data: &resourcepb.Resource{
						Id:        "",
						Name:      "Test Resource",
						ProductId: "subject-math",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "resource.validation.id_required",
			Assertions: func(t *testing.T, response *resourcepb.UpdateResourceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty ID")
			},
		},
		{
			Name:     "EmptyName",
			TestCode: "ESPYNA-TEST-PRODUCT-RESOURCE-UPDATE-VALIDATION-EMPTY-NAME-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *resourcepb.UpdateResourceRequest {
				return &resourcepb.UpdateResourceRequest{
					Data: &resourcepb.Resource{
						Id:        "resource-classroom-101",
						Name:      validationErrorEmptyNameResolver.MustGetString("emptyName"),
						ProductId: "subject-math",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "resource.validation.name_required",
			Assertions: func(t *testing.T, response *resourcepb.UpdateResourceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty name")
			},
		},
		{
			Name:     "EmptyProductId",
			TestCode: "ESPYNA-TEST-PRODUCT-RESOURCE-UPDATE-VALIDATION-EMPTY-PRODUCT-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *resourcepb.UpdateResourceRequest {
				return &resourcepb.UpdateResourceRequest{
					Data: &resourcepb.Resource{
						Id:        "resource-classroom-101",
						Name:      "Test Resource",
						ProductId: validationErrorEmptyProductIdResolver.MustGetString("emptyProductId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "resource.validation.product_id_required",
			Assertions: func(t *testing.T, response *resourcepb.UpdateResourceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty product ID")
			},
		},
		{
			Name:     "NameTooShort",
			TestCode: "ESPYNA-TEST-PRODUCT-RESOURCE-UPDATE-VALIDATION-NAME-TOO-SHORT-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *resourcepb.UpdateResourceRequest {
				return &resourcepb.UpdateResourceRequest{
					Data: &resourcepb.Resource{
						Id:        "resource-classroom-101",
						Name:      "AB", // Only 2 characters, minimum is 3
						ProductId: "subject-math",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "resource.validation.name_min_length",
			Assertions: func(t *testing.T, response *resourcepb.UpdateResourceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "name too short")
			},
		},
		{
			Name:     "NameTooLong",
			TestCode: "ESPYNA-TEST-PRODUCT-RESOURCE-UPDATE-VALIDATION-NAME-TOO-LONG-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *resourcepb.UpdateResourceRequest {
				resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "resource", "ValidationError_NameTooLongGenerated")
				testutil.AssertTestCaseLoad(t, err, "ValidationError_NameTooLongGenerated")
				return &resourcepb.UpdateResourceRequest{
					Data: &resourcepb.Resource{
						Id:        "resource-classroom-101",
						Name:      resolver.MustGetString("tooLongNameGenerated"),
						ProductId: "subject-math",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "resource.validation.name_max_length",
			Assertions: func(t *testing.T, response *resourcepb.UpdateResourceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "name too long")
			},
		},
		{
			Name:     "DescriptionTooLong",
			TestCode: "ESPYNA-TEST-PRODUCT-RESOURCE-UPDATE-VALIDATION-DESCRIPTION-TOO-LONG-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *resourcepb.UpdateResourceRequest {
				resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "resource", "ValidationError_DescriptionTooLongGenerated")
				testutil.AssertTestCaseLoad(t, err, "ValidationError_DescriptionTooLongGenerated")
				return &resourcepb.UpdateResourceRequest{
					Data: &resourcepb.Resource{
						Id:          "resource-classroom-101",
						Name:        "Valid Resource Name",
						Description: &[]string{resolver.MustGetString("tooLongDescriptionGenerated")}[0],
						ProductId:   "subject-math",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "resource.validation.description_max_length",
			Assertions: func(t *testing.T, response *resourcepb.UpdateResourceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "description too long")
			},
		},
		{
			Name:     "ProductIdTooShort",
			TestCode: "ESPYNA-TEST-PRODUCT-RESOURCE-UPDATE-VALIDATION-PRODUCT-ID-TOO-SHORT-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *resourcepb.UpdateResourceRequest {
				return &resourcepb.UpdateResourceRequest{
					Data: &resourcepb.Resource{
						Id:        "resource-classroom-101",
						Name:      "Valid Resource Name",
						ProductId: "AB", // Only 2 characters, minimum is 3
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "resource.validation.product_id_min_length",
			Assertions: func(t *testing.T, response *resourcepb.UpdateResourceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "product ID too short")
			},
		},
		{
			Name:     "DataEnrichment",
			TestCode: "ESPYNA-TEST-PRODUCT-RESOURCE-UPDATE-ENRICHMENT-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *resourcepb.UpdateResourceRequest {
				return &resourcepb.UpdateResourceRequest{
					Data: &resourcepb.Resource{
						Id:        "resource-computer-lab",
						Name:      "Updated Computer Laboratory",
						ProductId: "subject-science",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *resourcepb.UpdateResourceResponse, err error, useCase interface{}, ctx context.Context) {
				updatedResource := response.Data[0]
				testutil.AssertFieldSet(t, updatedResource.DateModified, "DateModified")
				testutil.AssertFieldSet(t, updatedResource.DateModifiedString, "DateModifiedString")
				// Verify DateModified is recent
				now := time.Now().UnixMilli()
				testutil.AssertTrue(t, *updatedResource.DateModified >= now-5000 && *updatedResource.DateModified <= now+5000, "DateModified should be recent")
				testutil.AssertFieldSet(t, updatedResource.DateCreated, "DateCreated (preserved)")
			},
		},
		{
			Name:     "MinimalValidData",
			TestCode: "ESPYNA-TEST-PRODUCT-RESOURCE-UPDATE-VALIDATION-MINIMAL-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *resourcepb.UpdateResourceRequest {
				resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "resource", "BoundaryTest_MinimalValid")
				testutil.AssertTestCaseLoad(t, err, "BoundaryTest_MinimalValid")
				return &resourcepb.UpdateResourceRequest{
					Data: &resourcepb.Resource{
						Id:        "resource-classroom-101",
						Name:      resolver.MustGetString("minValidName"),
						ProductId: "subject-math",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *resourcepb.UpdateResourceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				updatedResource := response.Data[0]
				// Business logic converts name to title case, so "ABC" becomes "Abc"
				testutil.AssertStringEqual(t, "Abc", updatedResource.Name, "resource name")
			},
		},
		{
			Name:     "EducationDomainSpecific",
			TestCode: "ESPYNA-TEST-PRODUCT-RESOURCE-UPDATE-INTEGRATION-EDUCATION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *resourcepb.UpdateResourceRequest {
				resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "resource", "DomainSpecific_Education")
				testutil.AssertTestCaseLoad(t, err, "DomainSpecific_Education")
				return &resourcepb.UpdateResourceRequest{
					Data: &resourcepb.Resource{
						Id:          "resource-lab-science",
						Name:        resolver.MustGetString("domainSpecificResourceName"),
						Description: &[]string{resolver.MustGetString("domainSpecificResourceDescription")}[0],
						ProductId:   "subject-science",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *resourcepb.UpdateResourceResponse, err error, useCase interface{}, ctx context.Context) {
				updatedResource := response.Data[0]
				resolver, _ := copyatestutil.LoadTestCaseFromBusinessType(t, testutil.GetTestBusinessType(), "resource", "DomainSpecific_Education")
				testutil.AssertStringEqual(t, resolver.MustGetString("domainSpecificResourceName"), updatedResource.Name, "education-specific resource name")
				testutil.AssertFieldSet(t, updatedResource.Description, "description for educational resource")
				if updatedResource.Description != nil {
					testutil.AssertTrue(t, len(*updatedResource.Description) >= 50, "comprehensive description length for educational resource")
				}
				testutil.AssertStringEqual(t, "subject-science", updatedResource.ProductId, "valid education product reference")
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
			useCase := createUpdateTestUseCaseWithAuth(businessType, tc.UseTransaction, tc.UseAuth)

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

func TestUpdateResourceUseCase_Execute_TransactionFailure(t *testing.T) {
	testCode := "ESPYNA-TEST-PRODUCT-RESOURCE-UPDATE-TRANSACTION-FAILURE-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "TransactionFailure", false)

	// Create a use case with a failing transaction service
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	mockResourceRepo := product.NewMockResourceRepository(businessType)
	mockProductRepo := product.NewMockProductRepository(businessType)

	repositories := UpdateResourceRepositories{
		Resource: mockResourceRepo,
		Product:  mockProductRepo,
	}

	standardServices := testutil.CreateStandardServices(false, true)
	services := UpdateResourceServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   mockDb.NewFailingMockTransactionService(),
		TranslationService:   standardServices.TranslationService,
	}

	useCase := NewUpdateResourceUseCase(repositories, services)

	// Load test case using centralized TestCaseResolver
	resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "resource", "UpdateResource_Success")
	testutil.AssertTestCaseLoad(t, err, "UpdateResource_Success")

	req := &resourcepb.UpdateResourceRequest{
		Data: &resourcepb.Resource{
			Id:        "resource-classroom-101",
			Name:      resolver.MustGetString("validResourceName"),
			ProductId: "subject-math",
		},
	}

	_, err = useCase.Execute(ctx, req)

	// Verify that a transaction error occurred
	testutil.AssertTransactionError(t, err)

	// Log completion of transaction failure test
	testutil.LogTestResult(t, testCode, "TransactionFailure", false, err)
}
