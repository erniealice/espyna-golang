//go:build mock_db && mock_auth

// Package permission provides table-driven tests for the permission creation use case.
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
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestCreatePermissionUseCase_Execute_TableDriven
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-ENTITY-PERMISSION-CREATE-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-ENTITY-PERMISSION-CREATE-TRANSACTION-v1.0: WithTransaction
//   - ESPYNA-TEST-ENTITY-PERMISSION-CREATE-NIL-REQUEST-v1.0: NilRequest
//   - ESPYNA-TEST-ENTITY-PERMISSION-CREATE-NIL-DATA-v1.0: NilData
//   - ESPYNA-TEST-ENTITY-PERMISSION-CREATE-VALIDATION-EMPTY-WORKSPACE-ID-v1.0: EmptyWorkspaceId
//   - ESPYNA-TEST-ENTITY-PERMISSION-CREATE-VALIDATION-EMPTY-USER-ID-v1.0: EmptyUserId
//   - ESPYNA-TEST-ENTITY-PERMISSION-CREATE-VALIDATION-EMPTY-PERMISSION-CODE-v1.0: EmptyPermissionCode
//   - ESPYNA-TEST-ENTITY-PERMISSION-CREATE-VALIDATION-SELF-GRANT-v1.0: SelfGrantNotAllowed
//   - ESPYNA-TEST-ENTITY-PERMISSION-CREATE-VALIDATION-PERMISSION-CODE-TOO-SHORT-v1.0: PermissionCodeTooShort
//   - ESPYNA-TEST-ENTITY-PERMISSION-CREATE-ENRICHMENT-v1.0: DataEnrichment
//
// Data Sources:
//   - Test cases: packages/copya/data_test/{businessType}/permission.json
//   - Mock data: packages/copya/data/{businessType}/permission.json
//   - Translations: packages/lyngua/translations/{languageCode}/{businessType}/permission.json
package permission

import (
	"context"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/shared/testutil"
	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock/entity"
	permissionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/permission"
)

// Type alias for create permission test cases
type CreatePermissionTestCase = testutil.GenericTestCase[*permissionpb.CreatePermissionRequest, *permissionpb.CreatePermissionResponse]

func createTestUseCaseWithAuth(businessType string, supportsTransaction bool, shouldAuthorize bool) *CreatePermissionUseCase {
	repositories := CreatePermissionRepositories{
		Permission: entity.NewMockPermissionRepository(businessType),
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, shouldAuthorize)
	services := CreatePermissionServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
		IDService:            standardServices.IDService,
	}

	return NewCreatePermissionUseCase(repositories, services)
}

func TestCreatePermissionUseCase_Execute_TableDriven(t *testing.T) {
	testCases := []CreatePermissionTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-ENTITY-PERMISSION-CREATE-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *permissionpb.CreatePermissionRequest {
				return &permissionpb.CreatePermissionRequest{
					Data: &permissionpb.Permission{
						WorkspaceId:     "workspace-001",
						UserId:          "user-student-001",
						GrantedByUserId: "user-admin-001",
						PermissionCode:  "read:student_record",
						PermissionType:  permissionpb.PermissionType_PERMISSION_TYPE_ALLOW,
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *permissionpb.CreatePermissionResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertEqual(t, 1, len(response.Data), "response data length")
				createdPermission := response.Data[0]
				testutil.AssertStringEqual(t, "read:student_record", createdPermission.PermissionCode, "permission code")
				testutil.AssertStringEqual(t, "workspace-001", createdPermission.WorkspaceId, "workspace ID")
				testutil.AssertStringEqual(t, "user-student-001", createdPermission.UserId, "user ID")
				testutil.AssertNonEmptyString(t, createdPermission.Id, "permission ID")
				testutil.AssertTrue(t, createdPermission.Active, "permission active status")
				testutil.AssertFieldSet(t, createdPermission.DateCreated, "DateCreated")
			},
		},
		{
			Name:     "WithTransaction",
			TestCode: "ESPYNA-TEST-ENTITY-PERMISSION-CREATE-TRANSACTION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *permissionpb.CreatePermissionRequest {
				return &permissionpb.CreatePermissionRequest{
					Data: &permissionpb.Permission{
						WorkspaceId:     "workspace-002",
						UserId:          "user-student-002",
						GrantedByUserId: "user-admin-002",
						PermissionCode:  "write:grades",
						PermissionType:  permissionpb.PermissionType_PERMISSION_TYPE_ALLOW,
					},
				}
			},
			UseTransaction: true,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *permissionpb.CreatePermissionResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				createdPermission := response.Data[0]
				testutil.AssertStringEqual(t, "write:grades", createdPermission.PermissionCode, "permission code")
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-ENTITY-PERMISSION-CREATE-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *permissionpb.CreatePermissionRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "permission.validation.request_required",
			Assertions: func(t *testing.T, response *permissionpb.CreatePermissionResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilRequest(t, err)
			},
		},
		{
			Name:     "NilData",
			TestCode: "ESPYNA-TEST-ENTITY-PERMISSION-CREATE-NIL-DATA-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *permissionpb.CreatePermissionRequest {
				return &permissionpb.CreatePermissionRequest{Data: nil}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "permission.validation.data_required",
			Assertions: func(t *testing.T, response *permissionpb.CreatePermissionResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilData(t, err)
			},
		},
		{
			Name:     "EmptyWorkspaceId",
			TestCode: "ESPYNA-TEST-ENTITY-PERMISSION-CREATE-VALIDATION-EMPTY-WORKSPACE-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *permissionpb.CreatePermissionRequest {
				return &permissionpb.CreatePermissionRequest{
					Data: &permissionpb.Permission{
						WorkspaceId:     "",
						UserId:          "user-1",
						GrantedByUserId: "user-2",
						PermissionCode:  "read:data",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "permission.validation.workspace_id_required",
			Assertions: func(t *testing.T, response *permissionpb.CreatePermissionResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty workspace ID")
			},
		},
		{
			Name:     "EmptyUserId",
			TestCode: "ESPYNA-TEST-ENTITY-PERMISSION-CREATE-VALIDATION-EMPTY-USER-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *permissionpb.CreatePermissionRequest {
				return &permissionpb.CreatePermissionRequest{
					Data: &permissionpb.Permission{
						WorkspaceId:     "workspace-1",
						UserId:          "",
						GrantedByUserId: "user-2",
						PermissionCode:  "read:data",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "permission.validation.user_id_required",
			Assertions: func(t *testing.T, response *permissionpb.CreatePermissionResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty user ID")
			},
		},
		{
			Name:     "EmptyPermissionCode",
			TestCode: "ESPYNA-TEST-ENTITY-PERMISSION-CREATE-VALIDATION-EMPTY-PERMISSION-CODE-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *permissionpb.CreatePermissionRequest {
				return &permissionpb.CreatePermissionRequest{
					Data: &permissionpb.Permission{
						WorkspaceId:     "workspace-1",
						UserId:          "user-1",
						GrantedByUserId: "user-2",
						PermissionCode:  "",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "permission.validation.permission_code_required",
			Assertions: func(t *testing.T, response *permissionpb.CreatePermissionResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty permission code")
			},
		},
		{
			Name:     "SelfGrantNotAllowed",
			TestCode: "ESPYNA-TEST-ENTITY-PERMISSION-CREATE-VALIDATION-SELF-GRANT-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *permissionpb.CreatePermissionRequest {
				return &permissionpb.CreatePermissionRequest{
					Data: &permissionpb.Permission{
						WorkspaceId:     "workspace-1",
						UserId:          "user-1",
						GrantedByUserId: "user-1",
						PermissionCode:  "read:data",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "permission.validation.self_grant_not_allowed",
			Assertions: func(t *testing.T, response *permissionpb.CreatePermissionResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "self grant not allowed")
			},
		},
		{
			Name:     "PermissionCodeTooShort",
			TestCode: "ESPYNA-TEST-ENTITY-PERMISSION-CREATE-VALIDATION-PERMISSION-CODE-TOO-SHORT-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *permissionpb.CreatePermissionRequest {
				return &permissionpb.CreatePermissionRequest{
					Data: &permissionpb.Permission{
						WorkspaceId:     "workspace-1",
						UserId:          "user-1",
						GrantedByUserId: "user-2",
						PermissionCode:  "a",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "permission.validation.permission_code_too_short",
			Assertions: func(t *testing.T, response *permissionpb.CreatePermissionResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "permission code too short")
			},
		},
		{
			Name:     "DataEnrichment",
			TestCode: "ESPYNA-TEST-ENTITY-PERMISSION-CREATE-ENRICHMENT-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *permissionpb.CreatePermissionRequest {
				return &permissionpb.CreatePermissionRequest{
					Data: &permissionpb.Permission{
						WorkspaceId:     "workspace-enrichment",
						UserId:          "user-enrichment",
						GrantedByUserId: "user-granter",
						PermissionCode:  "write:grades",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *permissionpb.CreatePermissionResponse, err error, useCase interface{}, ctx context.Context) {
				createdPermission := response.Data[0]
				testutil.AssertNonEmptyString(t, createdPermission.Id, "generated ID")
				testutil.AssertFieldSet(t, createdPermission.DateCreated, "DateCreated")
				testutil.AssertFieldSet(t, createdPermission.DateCreatedString, "DateCreatedString")
				testutil.AssertTrue(t, createdPermission.Active, "Active")
				testutil.AssertEqual(t, permissionpb.PermissionType_PERMISSION_TYPE_ALLOW, createdPermission.PermissionType, "default permission type")
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
