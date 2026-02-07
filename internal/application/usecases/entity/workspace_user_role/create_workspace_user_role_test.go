//go:build mock_db && mock_auth

// Package workspace_user_role provides table-driven tests for the workspace user role creation use case.
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
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestCreateWorkspaceUserRoleUseCase_Execute_TableDriven
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-ENTITY-WORKSPACE-USER-ROLE-CREATE-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-ENTITY-WORKSPACE-USER-ROLE-CREATE-TRANSACTION-v1.0: WithTransaction
//   - ESPYNA-TEST-ENTITY-WORKSPACE-USER-ROLE-CREATE-NIL-REQUEST-v1.0: NilRequest
//   - ESPYNA-TEST-ENTITY-WORKSPACE-USER-ROLE-CREATE-NIL-DATA-v1.0: NilData
//   - ESPYNA-TEST-ENTITY-WORKSPACE-USER-ROLE-CREATE-VALIDATION-EMPTY-WORKSPACE-USER-ID-v1.0: EmptyWorkspaceUserId
//   - ESPYNA-TEST-ENTITY-WORKSPACE-USER-ROLE-CREATE-VALIDATION-EMPTY-ROLE-ID-v1.0: EmptyRoleId
//   - ESPYNA-TEST-ENTITY-WORKSPACE-USER-ROLE-CREATE-VALIDATION-INVALID-WORKSPACE-USER-ID-v1.0: InvalidWorkspaceUserId
//   - ESPYNA-TEST-ENTITY-WORKSPACE-USER-ROLE-CREATE-VALIDATION-INVALID-ROLE-ID-v1.0: InvalidRoleId
//   - ESPYNA-TEST-ENTITY-WORKSPACE-USER-ROLE-CREATE-ENRICHMENT-v1.0: DataEnrichment
//
// Data Sources:
//   - Test cases: packages/copya/data_test/{businessType}/workspace_user_role.json
//   - Mock data: packages/copya/data/{businessType}/workspace_user_role.json
//   - Translations: packages/lyngua/translations/{languageCode}/{businessType}/workspace_user_role.json

package workspace_user_role

import (
	"context"
	"testing"

	"leapfor.xyz/espyna/internal/application/shared/testutil"
	mockAuth "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/auth/mock"
	"leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock/entity"
	workspaceuserrolepb "leapfor.xyz/esqyma/golang/v1/domain/entity/workspace_user_role"
)

// Type alias for create workspace user role test cases
type CreateWorkspaceUserRoleTestCase = testutil.GenericTestCase[*workspaceuserrolepb.CreateWorkspaceUserRoleRequest, *workspaceuserrolepb.CreateWorkspaceUserRoleResponse]

func createTestUseCaseWithAuth(businessType string, supportsTransaction bool, shouldAuthorize bool) *CreateWorkspaceUserRoleUseCase {
	repositories := CreateWorkspaceUserRoleRepositories{
		WorkspaceUserRole: entity.NewMockWorkspaceUserRoleRepository(businessType),
		WorkspaceUser:     entity.NewMockWorkspaceUserRepository(businessType),
		Role:              entity.NewMockRoleRepository(businessType),
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, shouldAuthorize)
	services := CreateWorkspaceUserRoleServices{
		AuthorizationService: mockAuth.NewAllowAllAuth(),
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
		IDService:            standardServices.IDService,
	}

	return NewCreateWorkspaceUserRoleUseCase(repositories, services)
}

func TestCreateWorkspaceUserRoleUseCase_Execute_TableDriven(t *testing.T) {
	// IDs from packages/copya/data/education/
	existingWorkspaceUserID := "workspace-user-001"
	existingRoleID := "role-admin"

	testCases := []CreateWorkspaceUserRoleTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-ENTITY-WORKSPACE-USER-ROLE-CREATE-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *workspaceuserrolepb.CreateWorkspaceUserRoleRequest {
				return &workspaceuserrolepb.CreateWorkspaceUserRoleRequest{
					Data: &workspaceuserrolepb.WorkspaceUserRole{
						WorkspaceUserId: existingWorkspaceUserID,
						RoleId:          existingRoleID,
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *workspaceuserrolepb.CreateWorkspaceUserRoleResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertEqual(t, 1, len(response.Data), "response data length")
				createdRel := response.Data[0]
				testutil.AssertStringEqual(t, existingRoleID, createdRel.RoleId, "role ID")
				testutil.AssertStringEqual(t, existingWorkspaceUserID, createdRel.WorkspaceUserId, "workspace user ID")
				testutil.AssertNonEmptyString(t, createdRel.Id, "workspace user role ID")
				testutil.AssertFieldSet(t, createdRel.DateCreated, "DateCreated")
			},
		},
		{
			Name:     "WithTransaction",
			TestCode: "ESPYNA-TEST-ENTITY-WORKSPACE-USER-ROLE-CREATE-TRANSACTION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *workspaceuserrolepb.CreateWorkspaceUserRoleRequest {
				return &workspaceuserrolepb.CreateWorkspaceUserRoleRequest{
					Data: &workspaceuserrolepb.WorkspaceUserRole{
						WorkspaceUserId: existingWorkspaceUserID,
						RoleId:          existingRoleID,
					},
				}
			},
			UseTransaction: true,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *workspaceuserrolepb.CreateWorkspaceUserRoleResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				createdRel := response.Data[0]
				testutil.AssertStringEqual(t, existingRoleID, createdRel.RoleId, "role ID")
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-ENTITY-WORKSPACE-USER-ROLE-CREATE-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *workspaceuserrolepb.CreateWorkspaceUserRoleRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "workspace_user_role.validation.request_required",
			Assertions: func(t *testing.T, response *workspaceuserrolepb.CreateWorkspaceUserRoleResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilRequest(t, err)
			},
		},
		{
			Name:     "NilData",
			TestCode: "ESPYNA-TEST-ENTITY-WORKSPACE-USER-ROLE-CREATE-NIL-DATA-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *workspaceuserrolepb.CreateWorkspaceUserRoleRequest {
				return &workspaceuserrolepb.CreateWorkspaceUserRoleRequest{Data: nil}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "workspace_user_role.validation.data_required",
			Assertions: func(t *testing.T, response *workspaceuserrolepb.CreateWorkspaceUserRoleResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilData(t, err)
			},
		},
		{
			Name:     "EmptyWorkspaceUserId",
			TestCode: "ESPYNA-TEST-ENTITY-WORKSPACE-USER-ROLE-CREATE-VALIDATION-EMPTY-WORKSPACE-USER-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *workspaceuserrolepb.CreateWorkspaceUserRoleRequest {
				return &workspaceuserrolepb.CreateWorkspaceUserRoleRequest{
					Data: &workspaceuserrolepb.WorkspaceUserRole{
						WorkspaceUserId: "",
						RoleId:          existingRoleID,
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "workspace_user_role.validation.workspace_user_id_required",
			Assertions: func(t *testing.T, response *workspaceuserrolepb.CreateWorkspaceUserRoleResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty workspace user ID")
			},
		},
		{
			Name:     "EmptyRoleId",
			TestCode: "ESPYNA-TEST-ENTITY-WORKSPACE-USER-ROLE-CREATE-VALIDATION-EMPTY-ROLE-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *workspaceuserrolepb.CreateWorkspaceUserRoleRequest {
				return &workspaceuserrolepb.CreateWorkspaceUserRoleRequest{
					Data: &workspaceuserrolepb.WorkspaceUserRole{
						WorkspaceUserId: existingWorkspaceUserID,
						RoleId:          "",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "workspace_user_role.validation.role_id_required",
			Assertions: func(t *testing.T, response *workspaceuserrolepb.CreateWorkspaceUserRoleResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty role ID")
			},
		},
		{
			Name:     "InvalidWorkspaceUserId",
			TestCode: "ESPYNA-TEST-ENTITY-WORKSPACE-USER-ROLE-CREATE-VALIDATION-INVALID-WORKSPACE-USER-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *workspaceuserrolepb.CreateWorkspaceUserRoleRequest {
				return &workspaceuserrolepb.CreateWorkspaceUserRoleRequest{
					Data: &workspaceuserrolepb.WorkspaceUserRole{
						WorkspaceUserId: "wu-999",
						RoleId:          existingRoleID,
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "workspace_user_role.errors.workspace_user_not_found",
			ErrorTags:      map[string]any{"workspaceUserId": "wu-999"},
			Assertions: func(t *testing.T, response *workspaceuserrolepb.CreateWorkspaceUserRoleResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "invalid workspace user ID")
			},
		},
		{
			Name:     "InvalidRoleId",
			TestCode: "ESPYNA-TEST-ENTITY-WORKSPACE-USER-ROLE-CREATE-VALIDATION-INVALID-ROLE-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *workspaceuserrolepb.CreateWorkspaceUserRoleRequest {
				return &workspaceuserrolepb.CreateWorkspaceUserRoleRequest{
					Data: &workspaceuserrolepb.WorkspaceUserRole{
						WorkspaceUserId: existingWorkspaceUserID,
						RoleId:          "role-999",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "workspace_user_role.errors.role_not_found",
			ErrorTags:      map[string]any{"roleId": "role-999"},
			Assertions: func(t *testing.T, response *workspaceuserrolepb.CreateWorkspaceUserRoleResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "invalid role ID")
			},
		},
		{
			Name:     "DataEnrichment",
			TestCode: "ESPYNA-TEST-ENTITY-WORKSPACE-USER-ROLE-CREATE-ENRICHMENT-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *workspaceuserrolepb.CreateWorkspaceUserRoleRequest {
				return &workspaceuserrolepb.CreateWorkspaceUserRoleRequest{
					Data: &workspaceuserrolepb.WorkspaceUserRole{
						WorkspaceUserId: existingWorkspaceUserID,
						RoleId:          existingRoleID,
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *workspaceuserrolepb.CreateWorkspaceUserRoleResponse, err error, useCase interface{}, ctx context.Context) {
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
