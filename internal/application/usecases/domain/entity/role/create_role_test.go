//go:build mock_db && mock_auth

// Package role provides table-driven tests for the role creation use case.
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
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestCreateRoleUseCase_Execute_TableDriven
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-ENTITY-ROLE-CREATE-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-ENTITY-ROLE-CREATE-TRANSACTION-v1.0: WithTransaction
//   - ESPYNA-TEST-ENTITY-ROLE-CREATE-NIL-REQUEST-v1.0: NilRequest
//   - ESPYNA-TEST-ENTITY-ROLE-CREATE-NIL-DATA-v1.0: NilData
//   - ESPYNA-TEST-ENTITY-ROLE-CREATE-VALIDATION-EMPTY-NAME-v1.0: EmptyName
//   - ESPYNA-TEST-ENTITY-ROLE-CREATE-VALIDATION-NAME-TOO-SHORT-v1.0: NameTooShort
//   - ESPYNA-TEST-ENTITY-ROLE-CREATE-VALIDATION-INVALID-COLOR-v1.0: InvalidColor
//   - ESPYNA-TEST-ENTITY-ROLE-CREATE-ENRICHMENT-v1.0: DataEnrichment
//   - ESPYNA-TEST-ENTITY-ROLE-CREATE-VALIDATION-MINIMAL-v1.0: MinimalValidData
//
// Data Sources:
//   - Test cases: packages/copya/data_test/{businessType}/role.json
//   - Mock data: packages/copya/data/{businessType}/role.json
//   - Translations: packages/lyngua/translations/{languageCode}/{businessType}/role.json
package role

import (
	"context"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/shared/testutil"
	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock/entity"
	rolepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/role"
)

// Type alias for create role test cases
type CreateRoleTestCase = testutil.GenericTestCase[*rolepb.CreateRoleRequest, *rolepb.CreateRoleResponse]

func createTestUseCaseWithAuth(businessType string, supportsTransaction bool, shouldAuthorize bool) *CreateRoleUseCase {
	repositories := CreateRoleRepositories{
		Role: entity.NewMockRoleRepository(businessType),
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, shouldAuthorize)
	services := CreateRoleServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
		IDService:            standardServices.IDService,
	}

	return NewCreateRoleUseCase(repositories, services)
}

func TestCreateRoleUseCase_Execute_TableDriven(t *testing.T) {
	testCases := []CreateRoleTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-ENTITY-ROLE-CREATE-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *rolepb.CreateRoleRequest {
				return &rolepb.CreateRoleRequest{
					Data: &rolepb.Role{
						Name: "Student",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *rolepb.CreateRoleResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertEqual(t, 1, len(response.Data), "response data length")
				createdRole := response.Data[0]
				testutil.AssertStringEqual(t, "Student", createdRole.Name, "role name")
				testutil.AssertNonEmptyString(t, createdRole.Id, "role ID")
				testutil.AssertTrue(t, createdRole.Active, "role active status")
				testutil.AssertFieldSet(t, createdRole.DateCreated, "DateCreated")
				testutil.AssertFieldSet(t, createdRole.DateCreatedString, "DateCreatedString")
			},
		},
		{
			Name:     "WithTransaction",
			TestCode: "ESPYNA-TEST-ENTITY-ROLE-CREATE-TRANSACTION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *rolepb.CreateRoleRequest {
				return &rolepb.CreateRoleRequest{
					Data: &rolepb.Role{
						Name: "Teacher",
					},
				}
			},
			UseTransaction: true,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *rolepb.CreateRoleResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				createdRole := response.Data[0]
				testutil.AssertStringEqual(t, "Teacher", createdRole.Name, "role name")
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-ENTITY-ROLE-CREATE-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *rolepb.CreateRoleRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "role.validation.request_required",
			Assertions: func(t *testing.T, response *rolepb.CreateRoleResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilRequest(t, err)
			},
		},
		{
			Name:     "NilData",
			TestCode: "ESPYNA-TEST-ENTITY-ROLE-CREATE-NIL-DATA-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *rolepb.CreateRoleRequest {
				return &rolepb.CreateRoleRequest{Data: nil}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "role.validation.data_required",
			Assertions: func(t *testing.T, response *rolepb.CreateRoleResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilData(t, err)
			},
		},
		{
			Name:     "EmptyName",
			TestCode: "ESPYNA-TEST-ENTITY-ROLE-CREATE-VALIDATION-EMPTY-NAME-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *rolepb.CreateRoleRequest {
				return &rolepb.CreateRoleRequest{
					Data: &rolepb.Role{
						Name: "",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "role.validation.name_required",
			Assertions: func(t *testing.T, response *rolepb.CreateRoleResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty name")
			},
		},
		{
			Name:     "NameTooShort",
			TestCode: "ESPYNA-TEST-ENTITY-ROLE-CREATE-VALIDATION-NAME-TOO-SHORT-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *rolepb.CreateRoleRequest {
				return &rolepb.CreateRoleRequest{
					Data: &rolepb.Role{
						Name: "A",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "role.validation.name_too_short",
			Assertions: func(t *testing.T, response *rolepb.CreateRoleResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "name too short")
			},
		},
		{
			Name:     "InvalidColor",
			TestCode: "ESPYNA-TEST-ENTITY-ROLE-CREATE-VALIDATION-INVALID-COLOR-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *rolepb.CreateRoleRequest {
				return &rolepb.CreateRoleRequest{
					Data: &rolepb.Role{
						Name:  "Test Role",
						Color: "invalid-color",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "role.validation.color_invalid",
			Assertions: func(t *testing.T, response *rolepb.CreateRoleResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "invalid color")
			},
		},
		{
			Name:     "DataEnrichment",
			TestCode: "ESPYNA-TEST-ENTITY-ROLE-CREATE-ENRICHMENT-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *rolepb.CreateRoleRequest {
				return &rolepb.CreateRoleRequest{
					Data: &rolepb.Role{
						Name: "Enriched Role",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *rolepb.CreateRoleResponse, err error, useCase interface{}, ctx context.Context) {
				createdRole := response.Data[0]
				testutil.AssertNonEmptyString(t, createdRole.Id, "generated ID")
				testutil.AssertFieldSet(t, createdRole.DateCreated, "DateCreated")
				testutil.AssertFieldSet(t, createdRole.DateCreatedString, "DateCreatedString")
				testutil.AssertTrue(t, createdRole.Active, "Active")
				testutil.AssertStringEqual(t, "#3B82F6", createdRole.Color, "default color")
			},
		},
		{
			Name:     "MinimalValidData",
			TestCode: "ESPYNA-TEST-ENTITY-ROLE-CREATE-VALIDATION-MINIMAL-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *rolepb.CreateRoleRequest {
				return &rolepb.CreateRoleRequest{
					Data: &rolepb.Role{
						Name: "ABC",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *rolepb.CreateRoleResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				createdRole := response.Data[0]
				testutil.AssertStringEqual(t, "ABC", createdRole.Name, "role name")
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
