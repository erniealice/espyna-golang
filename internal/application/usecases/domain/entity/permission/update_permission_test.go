//go:build mock_db && mock_auth

// Package permission provides test cases for permission updating use case.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user")
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education")
//
// Usage: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Methods:
//   - TestUpdatePermissionUseCase_Execute_Success: ESPYNA-TEST-ENTITY-PERMISSION-SUCCESS-v1.0 Tests successful update of permission properties like type
//   - TestUpdatePermissionUseCase_Execute_NotFound: ESPYNA-TEST-ENTITY-PERMISSION-NIL-v1.0 Tests updating a non-existent permission returns error
//   - TestUpdatePermissionUseCase_Execute_ValidationErrors: ESPYNA-TEST-ENTITY-PERMISSION-VALIDATION-v1.0 Tests input validation scenarios (empty ID)
//   - TestUpdatePermissionUseCase_Execute_SelfProvenanceSucceeds: ESPYNA-TEST-ENTITY-PERMISSION-SELF-PROVENANCE-v1.0 A self-provenanced definition (UserId == GrantedByUserId) passes validation
package permission

import (
	"strings"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/shared/testutil"
	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock/entity"
	permissionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/permission"
)

// createTestUpdatePermissionUseCase is a helper function to create the use case with mock dependencies
func createTestUpdatePermissionUseCase(businessType string) *UpdatePermissionUseCase {
	repositories := UpdatePermissionRepositories{
		Permission: entity.NewMockPermissionRepository(businessType),
	}

	standardServices := testutil.CreateStandardServices(false, true)
	services := UpdatePermissionServices{
		Authorizer: standardServices.Authorizer,
		Transactor: standardServices.Transactor,
		Translator: standardServices.Translator,
	}

	return NewUpdatePermissionUseCase(repositories, services)
}

func TestUpdatePermissionUseCase_Execute_Success(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestUpdatePermissionUseCase(businessType)

	existingID := "client.update"
	originalTime := int64(1725148800000)

	// In this entity, an "update" might be to change the type or active status.
	// We'll test changing the type.
	req := &permissionpb.UpdatePermissionRequest{
		Data: &permissionpb.Permission{
			Id:              existingID,
			WorkspaceId:     "workspace-001",
			UserId:          "user-student-001",
			GrantedByUserId: "user-admin-001",
			PermissionCode:  "read:student_record",
			PermissionType:  permissionpb.PermissionType_PERMISSION_TYPE_DENY, // Changing type
		},
	}

	res, err := useCase.Execute(ctx, req)
	testutil.AssertNoError(t, err)

	testutil.AssertNotNil(t, res, "response")

	updatedPermission := res.Data[0]
	testutil.AssertEqual(t, permissionpb.PermissionType_PERMISSION_TYPE_DENY, updatedPermission.PermissionType, "permission type")

	testutil.AssertFieldSet(t, updatedPermission.DateModified, "DateModified")
	testutil.AssertGreaterThan(t, int(*updatedPermission.DateModified), int(originalTime), "DateModified timestamp")
}

func TestUpdatePermissionUseCase_Execute_NotFound(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestUpdatePermissionUseCase(businessType)

	nonExistentID := "nonexistent-permission"
	req := &permissionpb.UpdatePermissionRequest{
		Data: &permissionpb.Permission{
			Id:              nonExistentID,
			WorkspaceId:     "workspace-001",
			UserId:          "user-student-001",
			GrantedByUserId: "user-admin-001",
			PermissionCode:  "read:student_record",
			PermissionType:  permissionpb.PermissionType_PERMISSION_TYPE_ALLOW,
		},
	}

	_, err := useCase.Execute(ctx, req)
	testutil.AssertError(t, err)
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Expected error message to contain 'not found', but got '%s'", err.Error())
	}
}

func TestUpdatePermissionUseCase_Execute_ValidationErrors(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestUpdatePermissionUseCase(businessType)

	testCases := []struct {
		name          string
		permission    *permissionpb.Permission
		expectedError string
	}{
		{
			name: "Empty ID",
			permission: &permissionpb.Permission{
				WorkspaceId: "ws-1", UserId: "user-1", GrantedByUserId: "user-2", PermissionCode: "code", PermissionType: permissionpb.PermissionType_PERMISSION_TYPE_ALLOW,
			},
			expectedError: "Permission ID is required",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := &permissionpb.UpdatePermissionRequest{Data: tc.permission}
			_, err := useCase.Execute(ctx, req)
			testutil.AssertError(t, err)
			if !strings.Contains(err.Error(), tc.expectedError) {
				t.Errorf("Expected error message to contain '%s', but got '%s'", tc.expectedError, err.Error())
			}
		})
	}
}

// TestUpdatePermissionUseCase_Execute_SelfProvenanceSucceeds verifies that a
// self-provenanced definition (UserId == GrantedByUserId) passes validation. The
// former self-grant rejection was removed: this entity is the permission-code
// DEFINITION row, not a grant edge — UserId/GrantedByUserId are provenance
// bookkeeping only (copya seeds are self-provenanced to superadmin-001, and an
// admin editing a definition through a single session principal ALWAYS has
// UserId == GrantedByUserId). role_permission is the grant vehicle and the Layer-2
// permission:update gate authorizes the route.
func TestUpdatePermissionUseCase_Execute_SelfProvenanceSucceeds(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestUpdatePermissionUseCase(businessType)

	existingID := "client.update"
	req := &permissionpb.UpdatePermissionRequest{
		Data: &permissionpb.Permission{
			Id:              existingID,
			WorkspaceId:     "workspace-001",
			UserId:          "user-1",
			GrantedByUserId: "user-1",
			PermissionCode:  "read:student_record",
			PermissionType:  permissionpb.PermissionType_PERMISSION_TYPE_ALLOW,
		},
	}

	res, err := useCase.Execute(ctx, req)
	testutil.AssertNoError(t, err)
	testutil.AssertNotNil(t, res, "response")

	updatedPermission := res.Data[0]
	testutil.AssertStringEqual(t, "user-1", updatedPermission.UserId, "user ID")
	testutil.AssertStringEqual(t, "user-1", updatedPermission.GrantedByUserId, "granted-by user ID")
}
