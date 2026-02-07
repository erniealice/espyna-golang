//go:build mock_db && mock_auth

// Package role_permission provides test cases for role permission updating use case.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user")
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education")
//
// Usage: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Methods:
//   - TestUpdateRolePermissionUseCase_Execute_Success: ESPYNA-TEST-ENTITY-ROLEPERMISSION-SUCCESS-v1.0 Tests successful update of permission type and modification timestamp
//   - TestUpdateRolePermissionUseCase_Execute_NotFound: ESPYNA-TEST-ENTITY-ROLEPERMISSION-NIL-v1.0 Tests error handling for non-existent role permission
//   - TestUpdateRolePermissionUseCase_Execute_InvalidReference: ESPYNA-TEST-ENTITY-ROLEPERMISSION-VALIDATION-v1.0 Tests validation error for invalid role reference
package role_permission

import (
	"strings"
	"testing"

	"leapfor.xyz/espyna/internal/application/shared/testutil"
	"leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock/entity"
	permissionpb "leapfor.xyz/esqyma/golang/v1/domain/entity/permission"
	rolepermissionpb "leapfor.xyz/esqyma/golang/v1/domain/entity/role_permission"
)

// createTestUpdateRolePermissionUseCase is a helper function to create the use case with mock dependencies
func createTestUpdateRolePermissionUseCase(businessType string) *UpdateRolePermissionUseCase {
	repositories := UpdateRolePermissionRepositories{
		RolePermission: entity.NewMockRolePermissionRepository(businessType),
		Role:           entity.NewMockRoleRepository(businessType),
		Permission:     entity.NewMockPermissionRepository(businessType),
	}

	standardServices := testutil.CreateStandardServices(false, true)
	services := UpdateRolePermissionServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
	}

	return NewUpdateRolePermissionUseCase(repositories, services)
}

func TestUpdateRolePermissionUseCase_Execute_Success(t *testing.T) {
	businessType := testutil.GetTestBusinessType()
	// Create shared repositories with existing role and permission data
	repositories := UpdateRolePermissionRepositories{
		RolePermission: entity.NewMockRolePermissionRepository(businessType),
		Role:           entity.NewMockRoleRepository(businessType),
		Permission:     entity.NewMockPermissionRepository(businessType),
	}

	standardServices := testutil.CreateStandardServices(false, true)
	services := UpdateRolePermissionServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
	}

	useCase := NewUpdateRolePermissionUseCase(repositories, services)
	ctx := testutil.CreateTestContext()

	existingID := "role-permission-001"
	originalTime := int64(1725148800000)

	// Update the permission type for this role-permission relationship
	// Use existing role and permission IDs from the mock data
	req := &rolepermissionpb.UpdateRolePermissionRequest{
		Data: &rolepermissionpb.RolePermission{
			Id:             existingID,
			RoleId:         "role-admin",  // Use existing role from mock data
			PermissionId:   "client.list", // Use existing permission from mock data
			PermissionType: permissionpb.PermissionType_PERMISSION_TYPE_DENY,
		},
	}

	res, err := useCase.Execute(ctx, req)
	testutil.AssertNoError(t, err)

	testutil.AssertNotNil(t, res, "response")

	updatedRel := res.Data[0]
	testutil.AssertEqual(t, permissionpb.PermissionType_PERMISSION_TYPE_DENY, updatedRel.PermissionType, "permission type")

	testutil.AssertFieldSet(t, updatedRel.DateModified, "DateModified")
	testutil.AssertGreaterThan(t, int(*updatedRel.DateModified), int(originalTime), "DateModified timestamp")
}

func TestUpdateRolePermissionUseCase_Execute_NotFound(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestUpdateRolePermissionUseCase(businessType)

	nonExistentID := "role-permission-999"
	req := &rolepermissionpb.UpdateRolePermissionRequest{
		Data: &rolepermissionpb.RolePermission{
			Id:             nonExistentID,
			RoleId:         "role-admin",  // Use valid role
			PermissionId:   "client.list", // Use valid permission
			PermissionType: permissionpb.PermissionType_PERMISSION_TYPE_ALLOW,
		},
	}

	_, err := useCase.Execute(ctx, req)
	testutil.AssertError(t, err)
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Expected error message to contain 'not found', but got '%s'", err.Error())
	}
}

func TestUpdateRolePermissionUseCase_Execute_InvalidReference(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestUpdateRolePermissionUseCase(businessType)

	req := &rolepermissionpb.UpdateRolePermissionRequest{
		Data: &rolepermissionpb.RolePermission{
			Id:             "role-permission-001",
			RoleId:         "role-999",                                       // Non-existent role
			PermissionId:   "client.list",                                    // Valid permission
			PermissionType: permissionpb.PermissionType_PERMISSION_TYPE_DENY, // Add required permission type
		},
	}

	_, err := useCase.Execute(ctx, req)
	testutil.AssertError(t, err)
	if !strings.Contains(err.Error(), "does not exist") {
		t.Errorf("Expected error message to contain 'does not exist', but got '%s'", err.Error())
	}
}
