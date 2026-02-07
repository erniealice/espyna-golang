//go:build mock_db && mock_auth

// Package role_permission provides table-driven tests for the role permission creation use case.
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
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestCreateRolePermissionUseCase_Execute_TableDriven
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-ENTITY-ROLE-PERMISSION-CREATE-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-ENTITY-ROLE-PERMISSION-CREATE-TRANSACTION-v1.0: WithTransaction
//   - ESPYNA-TEST-ENTITY-ROLE-PERMISSION-CREATE-NIL-REQUEST-v1.0: NilRequest
//   - ESPYNA-TEST-ENTITY-ROLE-PERMISSION-CREATE-NIL-DATA-v1.0: NilData
//   - ESPYNA-TEST-ENTITY-ROLE-PERMISSION-CREATE-VALIDATION-EMPTY-ROLE-ID-v1.0: EmptyRoleId
//   - ESPYNA-TEST-ENTITY-ROLE-PERMISSION-CREATE-VALIDATION-EMPTY-PERMISSION-ID-v1.0: EmptyPermissionId
//   - ESPYNA-TEST-ENTITY-ROLE-PERMISSION-CREATE-VALIDATION-INVALID-ROLE-ID-v1.0: InvalidRoleId
//   - ESPYNA-TEST-ENTITY-ROLE-PERMISSION-CREATE-VALIDATION-INVALID-PERMISSION-ID-v1.0: InvalidPermissionId
//   - ESPYNA-TEST-ENTITY-ROLE-PERMISSION-CREATE-ENRICHMENT-v1.0: DataEnrichment
//
// Data Sources:
//   - Test cases: packages/copya/data_test/{businessType}/role_permission.json
//   - Mock data: packages/copya/data/{businessType}/role_permission.json
//   - Translations: packages/lyngua/translations/{languageCode}/{businessType}/role_permission.json
package role_permission

import (
	"context"
	"testing"

	"leapfor.xyz/espyna/internal/application/shared/testutil"
	"leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock/entity"
	rolepermissionpb "leapfor.xyz/esqyma/golang/v1/domain/entity/role_permission"
)

// Type alias for create role permission test cases
type CreateRolePermissionTestCase = testutil.GenericTestCase[*rolepermissionpb.CreateRolePermissionRequest, *rolepermissionpb.CreateRolePermissionResponse]

func createTestCreateRolePermissionUseCase(businessType string) *CreateRolePermissionUseCase {
	repositories := CreateRolePermissionRepositories{
		RolePermission: entity.NewMockRolePermissionRepository(businessType),
		Role:           entity.NewMockRoleRepository(businessType),
		Permission:     entity.NewMockPermissionRepository(businessType),
	}

	standardServices := testutil.CreateStandardServices(false, true)
	services := CreateRolePermissionServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
		IDService:            standardServices.IDService,
	}

	return NewCreateRolePermissionUseCase(repositories, services)
}

func createTestUseCaseWithAuth(businessType string, supportsTransaction bool, shouldAuthorize bool) *CreateRolePermissionUseCase {
	repositories := CreateRolePermissionRepositories{
		RolePermission: entity.NewMockRolePermissionRepository(businessType),
		Role:           entity.NewMockRoleRepository(businessType),
		Permission:     entity.NewMockPermissionRepository(businessType),
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, shouldAuthorize)
	services := CreateRolePermissionServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
		IDService:            standardServices.IDService,
	}

	return NewCreateRolePermissionUseCase(repositories, services)
}

func TestCreateRolePermissionUseCase_Execute_TableDriven(t *testing.T) {
	// IDs from packages/copya/data/education/
	existingRoleID := "role-admin"
	existingPermissionID := "client.read"

	testCases := []CreateRolePermissionTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-ENTITY-ROLE-PERMISSION-CREATE-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *rolepermissionpb.CreateRolePermissionRequest {
				return &rolepermissionpb.CreateRolePermissionRequest{
					Data: &rolepermissionpb.RolePermission{
						RoleId:       existingRoleID,
						PermissionId: existingPermissionID,
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *rolepermissionpb.CreateRolePermissionResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertEqual(t, 1, len(response.Data), "response data length")
				createdRel := response.Data[0]
				testutil.AssertStringEqual(t, existingRoleID, createdRel.RoleId, "role ID")
				testutil.AssertStringEqual(t, existingPermissionID, createdRel.PermissionId, "permission ID")
				testutil.AssertNonEmptyString(t, createdRel.Id, "role permission ID")
				testutil.AssertFieldSet(t, createdRel.DateCreated, "DateCreated")
			},
		},
		{
			Name:     "WithTransaction",
			TestCode: "ESPYNA-TEST-ENTITY-ROLE-PERMISSION-CREATE-TRANSACTION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *rolepermissionpb.CreateRolePermissionRequest {
				return &rolepermissionpb.CreateRolePermissionRequest{
					Data: &rolepermissionpb.RolePermission{
						RoleId:       existingRoleID,
						PermissionId: existingPermissionID,
					},
				}
			},
			UseTransaction: true,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *rolepermissionpb.CreateRolePermissionResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				createdRel := response.Data[0]
				testutil.AssertStringEqual(t, existingRoleID, createdRel.RoleId, "role ID")
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-ENTITY-ROLE-PERMISSION-CREATE-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *rolepermissionpb.CreateRolePermissionRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "role_permission.validation.request_required",
			Assertions: func(t *testing.T, response *rolepermissionpb.CreateRolePermissionResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilRequest(t, err)
			},
		},
		{
			Name:     "NilData",
			TestCode: "ESPYNA-TEST-ENTITY-ROLE-PERMISSION-CREATE-NIL-DATA-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *rolepermissionpb.CreateRolePermissionRequest {
				return &rolepermissionpb.CreateRolePermissionRequest{Data: nil}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "role_permission.validation.data_required",
			Assertions: func(t *testing.T, response *rolepermissionpb.CreateRolePermissionResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilData(t, err)
			},
		},
		{
			Name:     "EmptyRoleId",
			TestCode: "ESPYNA-TEST-ENTITY-ROLE-PERMISSION-CREATE-VALIDATION-EMPTY-ROLE-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *rolepermissionpb.CreateRolePermissionRequest {
				return &rolepermissionpb.CreateRolePermissionRequest{
					Data: &rolepermissionpb.RolePermission{
						RoleId:       "",
						PermissionId: existingPermissionID,
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "role_permission.validation.role_id_required",
			Assertions: func(t *testing.T, response *rolepermissionpb.CreateRolePermissionResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty role ID")
			},
		},
		{
			Name:     "EmptyPermissionId",
			TestCode: "ESPYNA-TEST-ENTITY-ROLE-PERMISSION-CREATE-VALIDATION-EMPTY-PERMISSION-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *rolepermissionpb.CreateRolePermissionRequest {
				return &rolepermissionpb.CreateRolePermissionRequest{
					Data: &rolepermissionpb.RolePermission{
						RoleId:       existingRoleID,
						PermissionId: "",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "role_permission.validation.permission_id_required",
			Assertions: func(t *testing.T, response *rolepermissionpb.CreateRolePermissionResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty permission ID")
			},
		},
		{
			Name:     "InvalidRoleId",
			TestCode: "ESPYNA-TEST-ENTITY-ROLE-PERMISSION-CREATE-VALIDATION-INVALID-ROLE-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *rolepermissionpb.CreateRolePermissionRequest {
				return &rolepermissionpb.CreateRolePermissionRequest{
					Data: &rolepermissionpb.RolePermission{
						RoleId:       "role-999",
						PermissionId: existingPermissionID,
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "role_permission.errors.referenced_role_not_found",
			ErrorTags:      map[string]any{"roleId": "role-999"},
			Assertions: func(t *testing.T, response *rolepermissionpb.CreateRolePermissionResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "invalid role ID")
			},
		},
		{
			Name:     "InvalidPermissionId",
			TestCode: "ESPYNA-TEST-ENTITY-ROLE-PERMISSION-CREATE-VALIDATION-INVALID-PERMISSION-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *rolepermissionpb.CreateRolePermissionRequest {
				return &rolepermissionpb.CreateRolePermissionRequest{
					Data: &rolepermissionpb.RolePermission{
						RoleId:       existingRoleID,
						PermissionId: "permission-999",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "role_permission.errors.referenced_permission_not_found",
			ErrorTags:      map[string]any{"permissionId": "permission-999"},
			Assertions: func(t *testing.T, response *rolepermissionpb.CreateRolePermissionResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "invalid permission ID")
			},
		},
		{
			Name:     "DataEnrichment",
			TestCode: "ESPYNA-TEST-ENTITY-ROLE-PERMISSION-CREATE-ENRICHMENT-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *rolepermissionpb.CreateRolePermissionRequest {
				return &rolepermissionpb.CreateRolePermissionRequest{
					Data: &rolepermissionpb.RolePermission{
						RoleId:       existingRoleID,
						PermissionId: existingPermissionID,
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *rolepermissionpb.CreateRolePermissionResponse, err error, useCase interface{}, ctx context.Context) {
				createdRel := response.Data[0]
				testutil.AssertNonEmptyString(t, createdRel.Id, "generated ID")
				testutil.AssertFieldSet(t, createdRel.DateCreated, "DateCreated")
				testutil.AssertFieldSet(t, createdRel.DateCreatedString, "DateCreatedString")
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
