//go:build mock_db && mock_auth

// Package resource provides table-driven tests for the resource reading use case.
//
// The tests cover various scenarios, including success, validation errors,
// business logic verification, and domain-specific functionality.
// Each test case is defined in a table with a specific test code, request setup,
// and assertions.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user").
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education").
//
// Usage:
//   - Run all tests: go test -tags="mock_db,mock_auth" ./...
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestReadResourceUseCase_Execute_TableDriven
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-PRODUCT-RESOURCE-READ-SUCCESS-CLASSROOM-v1.0: ClassroomResource
//   - ESPYNA-TEST-PRODUCT-RESOURCE-READ-SUCCESS-SCIENCE-LAB-v1.0: ScienceLab
//   - ESPYNA-TEST-PRODUCT-RESOURCE-READ-SUCCESS-AUDITORIUM-v1.0: Auditorium
//   - ESPYNA-TEST-PRODUCT-RESOURCE-READ-SUCCESS-COMPUTER-LAB-v1.0: ComputerLab
//   - ESPYNA-TEST-PRODUCT-RESOURCE-READ-NOT-FOUND-v1.0: ResourceNotFound
//   - ESPYNA-TEST-PRODUCT-RESOURCE-READ-NIL-REQUEST-v1.0: NilRequest
//   - ESPYNA-TEST-PRODUCT-RESOURCE-READ-NIL-DATA-v1.0: NilData
//   - ESPYNA-TEST-PRODUCT-RESOURCE-READ-EMPTY-ID-v1.0: EmptyId
//   - ESPYNA-TEST-PRODUCT-RESOURCE-READ-DATA-INTEGRITY-v1.0: ValidateDataIntegrity
//   - ESPYNA-TEST-PRODUCT-RESOURCE-READ-EDUCATION-CONTEXT-v1.0: EducationBusinessContext
//   - ESPYNA-TEST-PRODUCT-RESOURCE-READ-AUTHORIZATION-v1.0: Unauthorized
//
// Data Sources:
//   - Test cases: packages/copya/data_test/{businessType}/resource.json
//   - Mock data: packages/copya/data/{businessType}/resource.json
//   - Translations: packages/lyngua/translations/{languageCode}/{businessType}/resource.json

package resource

import (
	"context"
	"testing"

	"leapfor.xyz/espyna/internal/application/shared/testutil"
	"leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock/product"
	resourcepb "leapfor.xyz/esqyma/golang/v1/domain/product/resource"
)

// Type alias for read resource test cases
type ReadResourceTestCase = testutil.GenericTestCase[*resourcepb.ReadResourceRequest, *resourcepb.ReadResourceResponse]

func createReadTestUseCaseWithAuth(businessType string, shouldAuthorize bool) *ReadResourceUseCase {
	mockResourceRepo := product.NewMockResourceRepository(businessType)

	repositories := ReadResourceRepositories{
		Resource: mockResourceRepo,
	}

	standardServices := testutil.CreateStandardServices(false, shouldAuthorize)
	services := ReadResourceServices{
		AuthorizationService: standardServices.AuthorizationService,
		TranslationService:   standardServices.TranslationService,
	}

	return NewReadResourceUseCase(repositories, services)
}

func TestReadResourceUseCase_Execute_TableDriven(t *testing.T) {

	testCases := []ReadResourceTestCase{
		{
			Name:     "ClassroomResource",
			TestCode: "ESPYNA-TEST-PRODUCT-RESOURCE-READ-SUCCESS-CLASSROOM-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *resourcepb.ReadResourceRequest {
				return &resourcepb.ReadResourceRequest{
					Data: &resourcepb.Resource{
						Id: "resource-classroom-101",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *resourcepb.ReadResourceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertEqual(t, 1, len(response.Data), "response data length")

				resource := response.Data[0]
				testutil.AssertStringEqual(t, "resource-classroom-101", resource.Id, "resource ID")
				testutil.AssertNonEmptyString(t, resource.Name, "resource name")
				testutil.AssertNonEmptyString(t, resource.ProductId, "resource product ID")
				testutil.AssertTrue(t, resource.Active, "resource active status")
			},
		},
		{
			Name:     "ScienceLab",
			TestCode: "ESPYNA-TEST-PRODUCT-RESOURCE-READ-SUCCESS-SCIENCE-LAB-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *resourcepb.ReadResourceRequest {
				return &resourcepb.ReadResourceRequest{
					Data: &resourcepb.Resource{
						Id: "resource-lab-science",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *resourcepb.ReadResourceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertEqual(t, 1, len(response.Data), "response data length")

				resource := response.Data[0]
				testutil.AssertStringEqual(t, "resource-lab-science", resource.Id, "resource ID")
				testutil.AssertNonEmptyString(t, resource.Name, "resource name")
				testutil.AssertNonEmptyString(t, resource.ProductId, "resource product ID")
			},
		},
		{
			Name:     "Auditorium",
			TestCode: "ESPYNA-TEST-PRODUCT-RESOURCE-READ-SUCCESS-AUDITORIUM-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *resourcepb.ReadResourceRequest {
				return &resourcepb.ReadResourceRequest{
					Data: &resourcepb.Resource{
						Id: "resource-auditorium",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *resourcepb.ReadResourceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				resource := response.Data[0]
				testutil.AssertStringEqual(t, "resource-auditorium", resource.Id, "resource ID")
				testutil.AssertNonEmptyString(t, resource.Name, "resource name")
				testutil.AssertNonEmptyString(t, resource.ProductId, "resource product ID")
			},
		},
		{
			Name:     "ComputerLab",
			TestCode: "ESPYNA-TEST-PRODUCT-RESOURCE-READ-SUCCESS-COMPUTER-LAB-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *resourcepb.ReadResourceRequest {
				return &resourcepb.ReadResourceRequest{
					Data: &resourcepb.Resource{
						Id: "resource-computer-lab",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *resourcepb.ReadResourceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				resource := response.Data[0]
				testutil.AssertStringEqual(t, "resource-computer-lab", resource.Id, "resource ID")
				testutil.AssertNonEmptyString(t, resource.Name, "resource name")
				testutil.AssertNonEmptyString(t, resource.ProductId, "resource product ID")

				if resource.Description != nil {
					testutil.AssertTrue(t, len(*resource.Description) > 10, "resource has meaningful description")
				}
			},
		},
		{
			Name:     "ResourceNotFound",
			TestCode: "ESPYNA-TEST-PRODUCT-RESOURCE-READ-NOT-FOUND-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *resourcepb.ReadResourceRequest {
				return &resourcepb.ReadResourceRequest{
					Data: &resourcepb.Resource{
						Id: "non-existent-resource",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "resource.errors.read_failed",
			Assertions: func(t *testing.T, response *resourcepb.ReadResourceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertError(t, err)
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-PRODUCT-RESOURCE-READ-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *resourcepb.ReadResourceRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "resource.validation.request_required",
			Assertions: func(t *testing.T, response *resourcepb.ReadResourceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilRequest(t, err)
			},
		},
		{
			Name:     "NilData",
			TestCode: "ESPYNA-TEST-PRODUCT-RESOURCE-READ-NIL-DATA-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *resourcepb.ReadResourceRequest {
				return &resourcepb.ReadResourceRequest{Data: nil}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "resource.validation.data_required",
			Assertions: func(t *testing.T, response *resourcepb.ReadResourceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilData(t, err)
			},
		},
		{
			Name:     "EmptyId",
			TestCode: "ESPYNA-TEST-PRODUCT-RESOURCE-READ-EMPTY-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *resourcepb.ReadResourceRequest {
				return &resourcepb.ReadResourceRequest{
					Data: &resourcepb.Resource{
						Id: "",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "resource.validation.id_required",
			Assertions: func(t *testing.T, response *resourcepb.ReadResourceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty ID")
			},
		},
		{
			Name:     "ValidateDataIntegrity",
			TestCode: "ESPYNA-TEST-PRODUCT-RESOURCE-READ-DATA-INTEGRITY-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *resourcepb.ReadResourceRequest {
				return &resourcepb.ReadResourceRequest{
					Data: &resourcepb.Resource{
						Id: "resource-classroom-101", // Use a hardcoded ID that exists in mock data
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *resourcepb.ReadResourceResponse, err error, useCase interface{}, ctx context.Context) {
				resource := response.Data[0]

				// Validate all audit fields are present
				testutil.AssertFieldSet(t, resource.DateCreated, "DateCreated")
				testutil.AssertFieldSet(t, resource.DateCreatedString, "DateCreatedString")
				testutil.AssertFieldSet(t, resource.DateModified, "DateModified")
				testutil.AssertFieldSet(t, resource.DateModifiedString, "DateModifiedString")

				// Validate foreign key integrity
				testutil.AssertNonEmptyString(t, resource.ProductId, "ProductId")

				// Validate resource has appropriate education domain characteristics
				validProductIds := map[string]bool{
					"subject-math":               true,
					"subject-science":            true,
					"subject-english":            true,
					"subject-physical-education": true,
					"subject-arts":               true,
					"subject-health":             true,
				}

				testutil.AssertTrue(t, validProductIds[resource.ProductId], "valid education product ID")
			},
		},
		{
			Name:     "EducationBusinessContext",
			TestCode: "ESPYNA-TEST-PRODUCT-RESOURCE-READ-EDUCATION-CONTEXT-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *resourcepb.ReadResourceRequest {
				return &resourcepb.ReadResourceRequest{
					Data: &resourcepb.Resource{
						Id: "resource-lab-science",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *resourcepb.ReadResourceResponse, err error, useCase interface{}, ctx context.Context) {
				resource := response.Data[0]

				// Validate education-specific attributes
				testutil.AssertNonEmptyString(t, resource.Name, "education-specific name")

				if resource.Description != nil {
					testutil.AssertTrue(t, len(*resource.Description) >= 10, "comprehensive description for educational resource")
				}

				// Validate that this is an education product reference
				validEducationProductIds := map[string]bool{
					"subject-math":               true,
					"subject-science":            true,
					"subject-english":            true,
					"subject-physical-education": true,
					"subject-arts":               true,
					"subject-health":             true,
				}

				testutil.AssertTrue(t, validEducationProductIds[resource.ProductId], "education product reference")
			},
		},
		{
			Name:     "Unauthorized",
			TestCode: "ESPYNA-TEST-PRODUCT-RESOURCE-READ-AUTHORIZATION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *resourcepb.ReadResourceRequest {
				return &resourcepb.ReadResourceRequest{
					Data: &resourcepb.Resource{
						Id: "resource-classroom-101", // Use a hardcoded ID that exists in mock data
					},
				}
			},
			UseTransaction: false,
			UseAuth:        false,
			ExpectSuccess:  false,
			ExpectedError:  "resource.errors.authorization_failed",
			Assertions: func(t *testing.T, response *resourcepb.ReadResourceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertAuthorizationError(t, err)
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
			useCase := createReadTestUseCaseWithAuth(businessType, tc.UseAuth)

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
