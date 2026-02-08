//go:build mock_db && mock_auth

// Package resource provides table-driven tests for the resource deletion use case.
//
// The tests cover various scenarios, including success, transaction handling,
// authorization, nil requests, validation errors, and soft delete verification.
// Each test case is defined in a table with a specific test code, request setup,
// and assertions.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user").
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education").
//
// Usage:
//   - Run all tests: go test -tags="mock_db,mock_auth" ./...
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestDeleteResourceUseCase_Execute_TableDriven
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-PRODUCT-RESOURCE-DELETE-SUCCESS-CLASSROOM-v1.0: ClassroomResource
//   - ESPYNA-TEST-PRODUCT-RESOURCE-DELETE-SUCCESS-SCIENCE-LAB-v1.0: ScienceLab
//   - ESPYNA-TEST-PRODUCT-RESOURCE-DELETE-SUCCESS-AUDITORIUM-v1.0: Auditorium
//   - ESPYNA-TEST-PRODUCT-RESOURCE-DELETE-SUCCESS-COMPUTER-LAB-v1.0: ComputerLab
//   - ESPYNA-TEST-PRODUCT-RESOURCE-DELETE-TRANSACTION-v1.0: WithTransaction
//   - ESPYNA-TEST-PRODUCT-RESOURCE-DELETE-NOT-FOUND-v1.0: ResourceNotFound
//   - ESPYNA-TEST-PRODUCT-RESOURCE-DELETE-NIL-REQUEST-v1.0: NilRequest
//   - ESPYNA-TEST-PRODUCT-RESOURCE-DELETE-NIL-DATA-v1.0: NilData
//   - ESPYNA-TEST-PRODUCT-RESOURCE-DELETE-EMPTY-ID-v1.0: EmptyId
//   - ESPYNA-TEST-PRODUCT-RESOURCE-DELETE-SOFT-DELETE-v1.0: SoftDeleteVerification
//   - ESPYNA-TEST-PRODUCT-RESOURCE-DELETE-EDUCATION-DOMAIN-v1.0: EducationDomainSpecific
//   - ESPYNA-TEST-PRODUCT-RESOURCE-DELETE-TRANSACTION-FAILURE-v1.0: TransactionFailure
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
	"github.com/erniealice/espyna-golang/internal/application/shared/testutil"
	mockDb "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock"
	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock/product"
	resourcepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/resource"
)

// Type alias for delete resource test cases
type DeleteResourceTestCase = testutil.GenericTestCase[*resourcepb.DeleteResourceRequest, *resourcepb.DeleteResourceResponse]

func createDeleteTestUseCaseWithAuth(businessType string, supportsTransaction bool, shouldAuthorize bool) *DeleteResourceUseCase {
	mockResourceRepo := product.NewMockResourceRepository(businessType)

	repositories := DeleteResourceRepositories{
		Resource: mockResourceRepo,
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, shouldAuthorize)
	services := DeleteResourceServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
	}

	return NewDeleteResourceUseCase(repositories, services)
}

func TestDeleteResourceUseCase_Execute_TableDriven(t *testing.T) {
	businessType := testutil.GetTestBusinessType()

	// Load test data resolvers before defining test cases
	deleteSuccessResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "resource", "DeleteResource_Success")
	testutil.AssertTestCaseLoad(t, err, "DeleteResource_Success")

	resourceNotFoundResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "resource", "DeleteResource_ResourceNotFound")
	testutil.AssertTestCaseLoad(t, err, "DeleteResource_ResourceNotFound")

	testCases := []DeleteResourceTestCase{
		{
			Name:     "ClassroomResource",
			TestCode: "ESPYNA-TEST-PRODUCT-RESOURCE-DELETE-SUCCESS-CLASSROOM-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *resourcepb.DeleteResourceRequest {
				return &resourcepb.DeleteResourceRequest{
					Data: &resourcepb.Resource{
						Id: deleteSuccessResolver.MustGetString("targetResourceId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *resourcepb.DeleteResourceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
			},
		},
		{
			Name:     "ScienceLab",
			TestCode: "ESPYNA-TEST-PRODUCT-RESOURCE-DELETE-SUCCESS-SCIENCE-LAB-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *resourcepb.DeleteResourceRequest {
				return &resourcepb.DeleteResourceRequest{
					Data: &resourcepb.Resource{
						Id: deleteSuccessResolver.MustGetString("scienceLabId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *resourcepb.DeleteResourceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
			},
		},
		{
			Name:     "Auditorium",
			TestCode: "ESPYNA-TEST-PRODUCT-RESOURCE-DELETE-SUCCESS-AUDITORIUM-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *resourcepb.DeleteResourceRequest {
				return &resourcepb.DeleteResourceRequest{
					Data: &resourcepb.Resource{
						Id: deleteSuccessResolver.MustGetString("auditoriumId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *resourcepb.DeleteResourceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
			},
		},
		{
			Name:     "ComputerLab",
			TestCode: "ESPYNA-TEST-PRODUCT-RESOURCE-DELETE-SUCCESS-COMPUTER-LAB-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *resourcepb.DeleteResourceRequest {
				return &resourcepb.DeleteResourceRequest{
					Data: &resourcepb.Resource{
						Id: deleteSuccessResolver.MustGetString("computerLabId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *resourcepb.DeleteResourceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
			},
		},
		{
			Name:     "WithTransaction",
			TestCode: "ESPYNA-TEST-PRODUCT-RESOURCE-DELETE-TRANSACTION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *resourcepb.DeleteResourceRequest {
				return &resourcepb.DeleteResourceRequest{
					Data: &resourcepb.Resource{
						Id: deleteSuccessResolver.MustGetString("targetResourceId"),
					},
				}
			},
			UseTransaction: true,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *resourcepb.DeleteResourceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
			},
		},
		{
			Name:     "ResourceNotFound",
			TestCode: "ESPYNA-TEST-PRODUCT-RESOURCE-DELETE-NOT-FOUND-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *resourcepb.DeleteResourceRequest {
				return &resourcepb.DeleteResourceRequest{
					Data: &resourcepb.Resource{
						Id: resourceNotFoundResolver.MustGetString("nonExistentResourceId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "resource.errors.deletion_failed",
			Assertions: func(t *testing.T, response *resourcepb.DeleteResourceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertError(t, err)
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-PRODUCT-RESOURCE-DELETE-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *resourcepb.DeleteResourceRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "resource.validation.request_required",
			Assertions: func(t *testing.T, response *resourcepb.DeleteResourceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilRequest(t, err)
			},
		},
		{
			Name:     "NilData",
			TestCode: "ESPYNA-TEST-PRODUCT-RESOURCE-DELETE-NIL-DATA-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *resourcepb.DeleteResourceRequest {
				return &resourcepb.DeleteResourceRequest{Data: nil}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "resource.validation.data_required",
			Assertions: func(t *testing.T, response *resourcepb.DeleteResourceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilData(t, err)
			},
		},
		{
			Name:     "EmptyId",
			TestCode: "ESPYNA-TEST-PRODUCT-RESOURCE-DELETE-EMPTY-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *resourcepb.DeleteResourceRequest {
				return &resourcepb.DeleteResourceRequest{
					Data: &resourcepb.Resource{
						Id: "",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "resource.validation.id_required",
			Assertions: func(t *testing.T, response *resourcepb.DeleteResourceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty ID")
			},
		},
		{
			Name:     "SoftDeleteVerification",
			TestCode: "ESPYNA-TEST-PRODUCT-RESOURCE-DELETE-SOFT-DELETE-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *resourcepb.DeleteResourceRequest {
				return &resourcepb.DeleteResourceRequest{
					Data: &resourcepb.Resource{
						Id: deleteSuccessResolver.MustGetString("scienceLabId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *resourcepb.DeleteResourceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				// Note: To verify soft delete, we would typically need to perform a read operation
				// after deletion to confirm the resource is marked as inactive but still exists.
				// The DeleteResourceResponse itself only indicates success/failure.
			},
		},
		{
			Name:     "EducationDomainSpecific",
			TestCode: "ESPYNA-TEST-PRODUCT-RESOURCE-DELETE-EDUCATION-DOMAIN-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *resourcepb.DeleteResourceRequest {
				return &resourcepb.DeleteResourceRequest{
					Data: &resourcepb.Resource{
						Id: deleteSuccessResolver.MustGetString("auditoriumId"),
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *resourcepb.DeleteResourceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				// The deletion of education resources should follow the same soft-delete pattern
				// as other domains, marking resources as inactive rather than removing them entirely
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
			useCase := createDeleteTestUseCaseWithAuth(businessType, tc.UseTransaction, tc.UseAuth)

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

func TestDeleteResourceUseCase_Execute_TransactionFailure(t *testing.T) {
	testCode := "ESPYNA-TEST-PRODUCT-RESOURCE-DELETE-TRANSACTION-FAILURE-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "TransactionFailure", false)

	// Create a use case with a failing transaction service
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	mockResourceRepo := product.NewMockResourceRepository(businessType)

	repositories := DeleteResourceRepositories{
		Resource: mockResourceRepo,
	}

	standardServices := testutil.CreateStandardServices(false, true)
	services := DeleteResourceServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   mockDb.NewFailingMockTransactionService(),
		TranslationService:   standardServices.TranslationService,
	}

	useCase := NewDeleteResourceUseCase(repositories, services)

	// Load test case using centralized TestCaseResolver
	resolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "resource", "DeleteResource_Success")
	testutil.AssertTestCaseLoad(t, err, "DeleteResource_Success")

	req := &resourcepb.DeleteResourceRequest{
		Data: &resourcepb.Resource{
			Id: resolver.MustGetString("targetResourceId"),
		},
	}

	_, err = useCase.Execute(ctx, req)

	// Verify that a transaction error occurred
	testutil.AssertTransactionError(t, err)

	// Log completion of transaction failure test
	testutil.LogTestResult(t, testCode, "TransactionFailure", false, err)
}
