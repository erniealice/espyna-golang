//go:build mock_db && mock_auth

// Package workspace_user_role provides test cases for workspace user role read use case.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user")
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education")
//
// Usage: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Methods:
//   - TestReadWorkspaceUserRoleUseCase_Execute_Success: ESPYNA-TEST-ENTITY-WORKSPACEUSERROLE-SUCCESS-v1.0 Successful workspace user role retrieval
//   - TestReadWorkspaceUserRoleUseCase_Execute_NotFound: ESPYNA-TEST-ENTITY-WORKSPACEUSERROLE-SUCCESS-v1.0 Non-existent ID graceful handling
//   - TestReadWorkspaceUserRoleUseCase_Execute_EmptyId: ESPYNA-TEST-ENTITY-WORKSPACEUSERROLE-VALIDATION-v1.0 Empty ID validation testing

package workspace_user_role

import (
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/shared/testutil"
	mockAuth "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/auth/mock"
	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock/entity"
	workspaceuserrolepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace_user_role"
)

// createTestReadWorkspaceUserRoleUseCase is a helper function to create the use case with mock dependencies
func createTestReadWorkspaceUserRoleUseCase(businessType string, supportsTransaction bool) *ReadWorkspaceUserRoleUseCase {
	repositories := ReadWorkspaceUserRoleRepositories{
		WorkspaceUserRole: entity.NewMockWorkspaceUserRoleRepository(businessType),
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, true)
	services := ReadWorkspaceUserRoleServices{
		AuthorizationService: mockAuth.NewAllowAllAuth(),
		TranslationService:   standardServices.TranslationService,
	}

	return NewReadWorkspaceUserRoleUseCase(repositories, services)
}

func TestReadWorkspaceUserRoleUseCase_Execute_Success(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestReadWorkspaceUserRoleUseCase(businessType, false)

	// ID from packages/copya/data/education/workspace-user-role.json
	existingID := "workspace-user-role-001"

	req := &workspaceuserrolepb.ReadWorkspaceUserRoleRequest{
		Data: &workspaceuserrolepb.WorkspaceUserRole{Id: existingID},
	}

	res, err := useCase.Execute(ctx, req)
	testutil.AssertNoError(t, err)

	testutil.AssertNotNil(t, res, "response")

	testutil.AssertEqual(t, 1, len(res.Data), "response data length")

	readRel := res.Data[0]
	testutil.AssertStringEqual(t, existingID, readRel.Id, "workspace-user-role ID")
	testutil.AssertStringEqual(t, "role-admin", readRel.RoleId, "RoleId")
}

func TestReadWorkspaceUserRoleUseCase_Execute_NotFound(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestReadWorkspaceUserRoleUseCase(businessType, false)

	nonExistentID := "workspace-user-role-999"

	req := &workspaceuserrolepb.ReadWorkspaceUserRoleRequest{
		Data: &workspaceuserrolepb.WorkspaceUserRole{Id: nonExistentID},
	}

	_, err := useCase.Execute(ctx, req)
	testutil.AssertError(t, err)
}

func TestReadWorkspaceUserRoleUseCase_Execute_EmptyId(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestReadWorkspaceUserRoleUseCase(businessType, false)

	req := &workspaceuserrolepb.ReadWorkspaceUserRoleRequest{
		Data: &workspaceuserrolepb.WorkspaceUserRole{Id: ""},
	}
	_, err := useCase.Execute(ctx, req)
	testutil.AssertError(t, err)

	testutil.AssertTranslatedError(t, err, "workspace_user_role.validation.id_required", useCase.services.TranslationService, ctx)
}
