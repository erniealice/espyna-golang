//go:build mock_db && mock_auth

// Package role_permission provides test cases for role permission deletion use case.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user")
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education")
//
// Usage: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Methods:
//   - TestDeleteRolePermissionUseCase_Execute_Success: ESPYNA-TEST-ENTITY-ROLEPERMISSION-SUCCESS-v1.0 Tests successful deletion of an existing role permission
//   - TestDeleteRolePermissionUseCase_Execute_NotFound: ESPYNA-TEST-ENTITY-ROLEPERMISSION-NIL-v1.0 Tests error handling for non-existent role permission
//   - TestDeleteRolePermissionUseCase_Execute_EmptyId: ESPYNA-TEST-ENTITY-ROLEPERMISSION-VALIDATION-v1.0 Tests validation error for empty ID
package role_permission

import (
	"strings"
	"testing"

	"leapfor.xyz/espyna/internal/application/shared/testutil"
	"leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock/entity"
	rolepermissionpb "leapfor.xyz/esqyma/golang/v1/domain/entity/role_permission"
)

// createTestDeleteRolePermissionUseCase is a helper function to create the use case with mock dependencies
func createTestDeleteRolePermissionUseCase(businessType string) *DeleteRolePermissionUseCase {
	repositories := DeleteRolePermissionRepositories{
		RolePermission: entity.NewMockRolePermissionRepository(businessType),
	}
	standardServices := testutil.CreateStandardServices(false, true)
	services := DeleteRolePermissionServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
	}
	return NewDeleteRolePermissionUseCase(repositories, services)
}

func TestDeleteRolePermissionUseCase_Execute_Success(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestDeleteRolePermissionUseCase(businessType)

	// Test deleting an existing role-permission from JSON data
	existingID := "role-permission-admin-002" // Exists in role-permission.json

	req := &rolepermissionpb.DeleteRolePermissionRequest{
		Data: &rolepermissionpb.RolePermission{Id: existingID},
	}

	res, err := useCase.Execute(ctx, req)
	testutil.AssertNoError(t, err)

	testutil.AssertNotNil(t, res, "response")

	testutil.AssertTrue(t, res.Success, "success")

	// Verify deletion by attempting to read the deleted role-permission
	readResp, err := useCase.repositories.RolePermission.ReadRolePermission(ctx, &rolepermissionpb.ReadRolePermissionRequest{
		Data: &rolepermissionpb.RolePermission{Id: existingID},
	})
	if err == nil && readResp != nil && readResp.Success && len(readResp.Data) > 0 {
		t.Error("Expected role-permission to be deleted, but it still exists")
	}
}

func TestDeleteRolePermissionUseCase_Execute_NotFound(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestDeleteRolePermissionUseCase(businessType)

	nonExistentID := "role-permission-999"
	req := &rolepermissionpb.DeleteRolePermissionRequest{
		Data: &rolepermissionpb.RolePermission{Id: nonExistentID},
	}

	_, err := useCase.Execute(ctx, req)
	testutil.AssertError(t, err)
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Expected error message to contain 'not found', but got '%s'", err.Error())
	}
}

func TestDeleteRolePermissionUseCase_Execute_EmptyId(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestDeleteRolePermissionUseCase(businessType)

	req := &rolepermissionpb.DeleteRolePermissionRequest{
		Data: &rolepermissionpb.RolePermission{Id: ""},
	}
	_, err := useCase.Execute(ctx, req)
	testutil.AssertTranslatedError(t, err, "role_permission.validation.id_required_with_prefix", useCase.services.TranslationService, ctx)
}
