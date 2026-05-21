//go:build mock_db && mock_auth

// Package workspace_user_role provides test cases for workspace user role update use case.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user")
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education")
//
// Usage: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Methods:
//   - TestUpdateWorkspaceUserRoleUseCase_Execute_Success: ESPYNA-TEST-ENTITY-WORKSPACEUSERROLE-SUCCESS-v1.0 Successful workspace user role update with timestamp tracking
//   - TestUpdateWorkspaceUserRoleUseCase_Execute_NotFound: ESPYNA-TEST-ENTITY-WORKSPACEUSERROLE-VALIDATION-v1.0 Non-existent ID error handling
//   - TestUpdateWorkspaceUserRoleUseCase_Execute_InvalidReference: ESPYNA-TEST-ENTITY-WORKSPACEUSERROLE-VALIDATION-v1.0 Foreign key reference validation testing

package workspace_user_role

import (
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/shared/testutil"
	mockAuth "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/auth/mock"
	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock/entity"
	workspaceuserrolepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace_user_role"
)

// createTestUpdateWorkspaceUserRoleUseCase is a helper function to create the use case with mock dependencies
func createTestUpdateWorkspaceUserRoleUseCase(businessType string, supportsTransaction bool) *UpdateWorkspaceUserRoleUseCase {
	repositories := UpdateWorkspaceUserRoleRepositories{
		WorkspaceUserRole: entity.NewMockWorkspaceUserRoleRepository(businessType),
		WorkspaceUser:     entity.NewMockWorkspaceUserRepository(businessType),
		Role:              entity.NewMockRoleRepository(businessType),
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, true)
	services := UpdateWorkspaceUserRoleServices{
		AuthorizationService: mockAuth.NewAllowAllAuth(),
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
	}

	return NewUpdateWorkspaceUserRoleUseCase(repositories, services)
}

func TestUpdateWorkspaceUserRoleUseCase_Execute_Success(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestUpdateWorkspaceUserRoleUseCase(businessType, false)

	existingID := "workspace-user-role-001"
	originalTime := int64(1725148800000)

	// An update to this entity is mostly just touching the DateModified timestamp
	req := &workspaceuserrolepb.UpdateWorkspaceUserRoleRequest{
		Data: &workspaceuserrolepb.WorkspaceUserRole{
			Id:              existingID,
			WorkspaceUserId: "workspace-user-001",
			RoleId:          "role-admin",
		},
	}

	res, err := useCase.Execute(ctx, req)
	testutil.AssertNoError(t, err)

	testutil.AssertNotNil(t, res, "response")

	updatedRel := res.Data[0]
	testutil.AssertStringEqual(t, existingID, updatedRel.Id, "ID")

	testutil.AssertFieldSet(t, updatedRel.DateModified, "DateModified")
	testutil.AssertGreaterThan(t, int(*updatedRel.DateModified), int(originalTime), "DateModified")
}

func TestUpdateWorkspaceUserRoleUseCase_Execute_NotFound(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestUpdateWorkspaceUserRoleUseCase(businessType, false)

	nonExistentID := "workspace-user-role-999"
	req := &workspaceuserrolepb.UpdateWorkspaceUserRoleRequest{
		Data: &workspaceuserrolepb.WorkspaceUserRole{
			Id:              nonExistentID,
			WorkspaceUserId: "workspace-user-001",
			RoleId:          "role-admin",
		},
	}

	_, err := useCase.Execute(ctx, req)
	testutil.AssertError(t, err)
	testutil.AssertTranslatedError(t, err, "workspace_user_role.errors.update_failed", useCase.services.TranslationService, ctx)
}

func TestUpdateWorkspaceUserRoleUseCase_Execute_InvalidReference(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestUpdateWorkspaceUserRoleUseCase(businessType, false)

	req := &workspaceuserrolepb.UpdateWorkspaceUserRoleRequest{
		Data: &workspaceuserrolepb.WorkspaceUserRole{
			Id:              "workspace-user-role-001",
			WorkspaceUserId: "wu-999", // Non-existent workspace user
			RoleId:          "role-admin",
		},
	}

	_, err := useCase.Execute(ctx, req)
	testutil.AssertError(t, err)
	testutil.AssertTranslatedErrorWithContext(t, err, "workspace_user_role.errors.workspace_user_not_found", "{\"workspaceUserId\": \"wu-999\"}", useCase.services.TranslationService, ctx)
}
