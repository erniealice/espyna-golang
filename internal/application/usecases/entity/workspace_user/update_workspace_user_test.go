//go:build mock_db && mock_auth

// Package workspace_user provides test cases for workspace user update use case.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user")
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education")
//
// Usage: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Methods:
//   - TestUpdateWorkspaceUserUseCase_Execute_Success: ESPYNA-TEST-ENTITY-WORKSPACEUSER-SUCCESS-v1.0 Validates successful workspace user relationship modification with role updates and timestamp tracking
//   - TestUpdateWorkspaceUserUseCase_Execute_NotFound: ESPYNA-TEST-ENTITY-WORKSPACEUSER-VALIDATION-v1.0 Tests error handling for non-existent workspace user IDs
//   - TestUpdateWorkspaceUserUseCase_Execute_InvalidReference: ESPYNA-TEST-ENTITY-WORKSPACEUSER-VALIDATION-v1.0 Validates foreign key constraints for workspace references

package workspace_user

import (
	"testing"

	"leapfor.xyz/espyna/internal/application/shared/testutil"
	mockAuth "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/auth/mock"
	"leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock/entity"
	rolepb "leapfor.xyz/esqyma/golang/v1/domain/entity/role"
	workspaceuserpb "leapfor.xyz/esqyma/golang/v1/domain/entity/workspace_user"
	workspaceuserrolepb "leapfor.xyz/esqyma/golang/v1/domain/entity/workspace_user_role"
)

// createTestUpdateWorkspaceUserUseCase is a helper function to create the use case with mock dependencies
func createTestUpdateWorkspaceUserUseCase(businessType string) *UpdateWorkspaceUserUseCase {
	repositories := UpdateWorkspaceUserRepositories{
		WorkspaceUser: entity.NewMockWorkspaceUserRepository(businessType),
		Workspace:     entity.NewMockWorkspaceRepository(businessType),
		User:          entity.NewMockUserRepository(businessType),
	}

	standardServices := testutil.CreateStandardServices(false, true)
	services := UpdateWorkspaceUserServices{
		AuthorizationService: mockAuth.NewAllowAllAuth().SetUserWorkspaces("test-user", "workspace-elementary", "workspace-middle", "workspace-high"),
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
	}

	return NewUpdateWorkspaceUserUseCase(repositories, services)
}

func TestUpdateWorkspaceUserUseCase_Execute_Success(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestUpdateWorkspaceUserUseCase(businessType)

	existingID := "workspace-user-001"
	originalTime := int64(1725148800000)

	// Update the roles for this workspace-user relationship
	req := &workspaceuserpb.UpdateWorkspaceUserRequest{
		Data: &workspaceuserpb.WorkspaceUser{
			Id:          existingID,
			WorkspaceId: "workspace-elementary",
			UserId:      "user-admin-001",
			WorkspaceUserRoles: []*workspaceuserrolepb.WorkspaceUserRole{
				{
					Id:     "wsur-003",
					RoleId: "role-super-admin",
					Role:   &rolepb.Role{Id: "role-super-admin", Name: "super-admin"},
					Active: true,
				},
				{
					Id:     "wsur-004",
					RoleId: "role-auditor",
					Role:   &rolepb.Role{Id: "role-auditor", Name: "auditor"},
					Active: true,
				},
			},
		},
	}

	res, err := useCase.Execute(ctx, req)
	testutil.AssertNoError(t, err)

	testutil.AssertNotNil(t, res, "response")

	updatedRel := res.Data[0]
	testutil.AssertEqual(t, 2, len(updatedRel.WorkspaceUserRoles), "workspace user roles count")
	testutil.AssertStringEqual(t, "super-admin", updatedRel.WorkspaceUserRoles[0].Role.Name, "first role name")

	testutil.AssertFieldSet(t, updatedRel.DateModified, "DateModified")
	testutil.AssertGreaterThan(t, int(*updatedRel.DateModified), int(originalTime), "DateModified")
}

func TestUpdateWorkspaceUserUseCase_Execute_NotFound(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestUpdateWorkspaceUserUseCase(businessType)

	nonExistentID := "workspace-user-999"
	req := &workspaceuserpb.UpdateWorkspaceUserRequest{
		Data: &workspaceuserpb.WorkspaceUser{
			Id:          nonExistentID,
			WorkspaceId: "workspace-elementary",
			UserId:      "user-admin-001",
		},
	}

	_, err := useCase.Execute(ctx, req)
	testutil.AssertError(t, err)

	expectedError := "workspace user with ID '" + nonExistentID + "' not found"
	testutil.AssertStringEqual(t, expectedError, err.Error(), "error message")
}

func TestUpdateWorkspaceUserUseCase_Execute_InvalidReference(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestUpdateWorkspaceUserUseCase(businessType)

	req := &workspaceuserpb.UpdateWorkspaceUserRequest{
		Data: &workspaceuserpb.WorkspaceUser{
			Id:          "workspace-user-001",
			WorkspaceId: "workspace-999", // Non-existent workspace
			UserId:      "user-admin-001",
		},
	}

	_, err := useCase.Execute(ctx, req)
	testutil.AssertError(t, err)
	testutil.AssertTranslatedError(t, err, "workspace_user.errors.workspace_access_denied", useCase.services.TranslationService, ctx)
}
