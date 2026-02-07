//go:build mock_db && mock_auth

// Package workspace_user_role provides test cases for workspace user role delete use case.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user")
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education")
//
// Usage: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Methods:
//   - TestDeleteWorkspaceUserRoleUseCase_Execute_Success: ESPYNA-TEST-ENTITY-WORKSPACEUSERROLE-SUCCESS-v1.0 Successful workspace user role deletion with verification
//   - TestDeleteWorkspaceUserRoleUseCase_Execute_NotFound: ESPYNA-TEST-ENTITY-WORKSPACEUSERROLE-VALIDATION-v1.0 Non-existent ID error handling
//   - TestDeleteWorkspaceUserRoleUseCase_Execute_EmptyId: ESPYNA-TEST-ENTITY-WORKSPACEUSERROLE-VALIDATION-v1.0 Empty ID validation testing

package workspace_user_role

import (
	"testing"

	"leapfor.xyz/espyna/internal/application/shared/testutil"
	mockAuth "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/auth/mock"
	"leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock/entity"
	workspaceuserrolepb "leapfor.xyz/esqyma/golang/v1/domain/entity/workspace_user_role"
)

// createTestDeleteWorkspaceUserRoleUseCase is a helper function to create the use case with mock dependencies
func createTestDeleteWorkspaceUserRoleUseCase(businessType string, supportsTransaction bool) *DeleteWorkspaceUserRoleUseCase {
	repositories := DeleteWorkspaceUserRoleRepositories{
		WorkspaceUserRole: entity.NewMockWorkspaceUserRoleRepository(businessType),
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, true)
	services := DeleteWorkspaceUserRoleServices{
		AuthorizationService: mockAuth.NewAllowAllAuth(),
		TranslationService:   standardServices.TranslationService,
	}

	return NewDeleteWorkspaceUserRoleUseCase(repositories, services)
}

func TestDeleteWorkspaceUserRoleUseCase_Execute_Success(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestDeleteWorkspaceUserRoleUseCase(businessType, false)

	// This ID will be "deleted" from the mock repository
	existingID := "workspace-user-role-001"

	req := &workspaceuserrolepb.DeleteWorkspaceUserRoleRequest{
		Data: &workspaceuserrolepb.WorkspaceUserRole{Id: existingID},
	}

	res, err := useCase.Execute(ctx, req)
	testutil.AssertNoError(t, err)

	testutil.AssertNotNil(t, res, "response")

	testutil.AssertTrue(t, res.Success, "success")

	// Verify the relationship is actually deleted - use same repository instance
	readUseCase := NewReadWorkspaceUserRoleUseCase(
		ReadWorkspaceUserRoleRepositories{WorkspaceUserRole: useCase.repositories.WorkspaceUserRole},
		ReadWorkspaceUserRoleServices{
			AuthorizationService: mockAuth.NewAllowAllAuth(),
			TranslationService:   testutil.CreateStandardServices(false, true).TranslationService,
		},
	)
	readReq := &workspaceuserrolepb.ReadWorkspaceUserRoleRequest{Data: &workspaceuserrolepb.WorkspaceUserRole{Id: existingID}}
	_, err = readUseCase.Execute(ctx, readReq)
	testutil.AssertError(t, err)
}

func TestDeleteWorkspaceUserRoleUseCase_Execute_NotFound(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestDeleteWorkspaceUserRoleUseCase(businessType, false)

	nonExistentID := "workspace-user-role-999"
	req := &workspaceuserrolepb.DeleteWorkspaceUserRoleRequest{
		Data: &workspaceuserrolepb.WorkspaceUserRole{Id: nonExistentID},
	}

	_, err := useCase.Execute(ctx, req)
	testutil.AssertError(t, err)

	testutil.AssertTranslatedError(t, err, "workspace_user_role.errors.deletion_failed", useCase.services.TranslationService, ctx)
}

func TestDeleteWorkspaceUserRoleUseCase_Execute_EmptyId(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestDeleteWorkspaceUserRoleUseCase(businessType, false)

	req := &workspaceuserrolepb.DeleteWorkspaceUserRoleRequest{
		Data: &workspaceuserrolepb.WorkspaceUserRole{Id: ""},
	}
	_, err := useCase.Execute(ctx, req)
	testutil.AssertError(t, err)

	testutil.AssertTranslatedError(t, err, "workspace_user_role.validation.id_required", useCase.services.TranslationService, ctx)
}
