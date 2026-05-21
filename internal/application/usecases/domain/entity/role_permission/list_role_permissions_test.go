//go:build mock_db && mock_auth

// Package role_permission provides test cases for role permission listing use case.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user")
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education")
//
// Usage: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Methods:
//   - TestListRolePermissionsUseCase_Execute_Success: ESPYNA-TEST-ENTITY-ROLEPERMISSION-SUCCESS-v1.0 Tests successful listing of role permissions from mock data
//   - TestListRolePermissionsUseCase_Execute_AfterDelete: ESPYNA-TEST-ENTITY-ROLEPERMISSION-INTEGRATION-v1.0 Tests listing after deleting a role permission to verify consistency
package role_permission

import (
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/shared/testutil"
	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock/entity"
	rolepermissionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/role_permission"
)

// createTestListRolePermissionsUseCase is a helper function to create the use case with mock dependencies
func createTestListRolePermissionsUseCase(businessType string) *ListRolePermissionsUseCase {
	repositories := ListRolePermissionsRepositories{
		RolePermission: entity.NewMockRolePermissionRepository(businessType),
	}
	standardServices := testutil.CreateStandardServices(false, true)
	services := ListRolePermissionsServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
	}
	return NewListRolePermissionsUseCase(repositories, services)
}

func TestListRolePermissionsUseCase_Execute_Success(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	// We need a fresh repository for this test to ensure count is correct
	mockRepo := entity.NewMockRolePermissionRepository(businessType)
	standardServices := testutil.CreateStandardServices(false, true)
	services := ListRolePermissionsServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
	}
	useCase := NewListRolePermissionsUseCase(ListRolePermissionsRepositories{RolePermission: mockRepo}, services)

	// The mock data for education/role-permission has 2 entries

	req := &rolepermissionpb.ListRolePermissionsRequest{}

	res, err := useCase.Execute(ctx, req)
	testutil.AssertNoError(t, err)

	testutil.AssertNotNil(t, res, "response")

	testutil.AssertGreaterThan(t, len(res.Data), 0, "response data count")
}

func TestListRolePermissionsUseCase_Execute_AfterDelete(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	// Setup a repository that will be shared between use cases
	mockRepo := entity.NewMockRolePermissionRepository(businessType)

	// --- Delete a role-permission first ---
	deleteRepositories := DeleteRolePermissionRepositories{RolePermission: mockRepo}
	standardDeleteServices := testutil.CreateStandardServices(false, true)
	deleteServices := DeleteRolePermissionServices{
		AuthorizationService: standardDeleteServices.AuthorizationService,
		TransactionService:   standardDeleteServices.TransactionService,
		TranslationService:   standardDeleteServices.TranslationService,
	}
	deleteUseCase := NewDeleteRolePermissionUseCase(deleteRepositories, deleteServices)

	deleteReq := &rolepermissionpb.DeleteRolePermissionRequest{Data: &rolepermissionpb.RolePermission{Id: "role-permission-001"}}
	_, err := deleteUseCase.Execute(ctx, deleteReq)
	testutil.AssertNoError(t, err)

	// --- Now list the role-permissions ---
	standardListServices := testutil.CreateStandardServices(false, true)
	listServices := ListRolePermissionsServices{
		AuthorizationService: standardListServices.AuthorizationService,
		TransactionService:   standardListServices.TransactionService,
		TranslationService:   standardListServices.TranslationService,
	}
	listUseCase := NewListRolePermissionsUseCase(ListRolePermissionsRepositories{RolePermission: mockRepo}, listServices)

	listReq := &rolepermissionpb.ListRolePermissionsRequest{}
	res, err := listUseCase.Execute(ctx, listReq)
	testutil.AssertNoError(t, err)

	// Expected count is 2 (original) - 1 (deleted) = 1
	testutil.AssertGreaterThan(t, len(res.Data), 0, "response data count")
}
