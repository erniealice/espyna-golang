//go:build mock_db && mock_auth

// Package role_permission provides test cases for role permission reading use case.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user")
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education")
//
// Usage: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Methods:
//   - TestReadRolePermissionUseCase_Execute_Success: ESPYNA-TEST-ENTITY-ROLEPERMISSION-SUCCESS-v1.0 Tests successful reading of an existing role permission by ID
//   - TestReadRolePermissionUseCase_Execute_NotFound: ESPYNA-TEST-ENTITY-ROLEPERMISSION-NIL-v1.0 Tests error handling for non-existent role permission
//   - TestReadRolePermissionUseCase_Execute_EmptyId: ESPYNA-TEST-ENTITY-ROLEPERMISSION-VALIDATION-v1.0 Tests validation error for empty ID
package role_permission

import (
	"strings"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/shared/testutil"
	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock/entity"
	rolepermissionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/role_permission"
)

// createTestReadRolePermissionUseCase is a helper function to create the use case with mock dependencies
func createTestReadRolePermissionUseCase(businessType string) *ReadRolePermissionUseCase {
	repositories := ReadRolePermissionRepositories{
		RolePermission: entity.NewMockRolePermissionRepository(businessType),
	}
	standardServices := testutil.CreateStandardServices(false, true)
	services := ReadRolePermissionServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
	}
	return NewReadRolePermissionUseCase(repositories, services)
}

func TestReadRolePermissionUseCase_Execute_Success(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestReadRolePermissionUseCase(businessType)

	// ID from packages/copya/data/education/role-permission.json
	existingID := "role-permission-001"

	req := &rolepermissionpb.ReadRolePermissionRequest{
		Data: &rolepermissionpb.RolePermission{Id: existingID},
	}

	res, err := useCase.Execute(ctx, req)
	testutil.AssertNoError(t, err)

	testutil.AssertNotNil(t, res, "response")

	testutil.AssertEqual(t, 1, len(res.Data), "response data length")

	readRel := res.Data[0]
	testutil.AssertStringEqual(t, existingID, readRel.Id, "role-permission ID")
	testutil.AssertStringEqual(t, "role-001", readRel.RoleId, "RoleId")
}

func TestReadRolePermissionUseCase_Execute_NotFound(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestReadRolePermissionUseCase(businessType)

	nonExistentID := "role-permission-999"

	req := &rolepermissionpb.ReadRolePermissionRequest{
		Data: &rolepermissionpb.RolePermission{Id: nonExistentID},
	}

	_, err := useCase.Execute(ctx, req)
	testutil.AssertError(t, err)
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Expected error message to contain 'not found', but got '%s'", err.Error())
	}
}

func TestReadRolePermissionUseCase_Execute_EmptyId(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestReadRolePermissionUseCase(businessType)

	req := &rolepermissionpb.ReadRolePermissionRequest{
		Data: &rolepermissionpb.RolePermission{Id: ""},
	}
	_, err := useCase.Execute(ctx, req)
	testutil.AssertTranslatedError(t, err, "role_permission.validation.id_required_with_prefix", useCase.services.TranslationService, ctx)
}
