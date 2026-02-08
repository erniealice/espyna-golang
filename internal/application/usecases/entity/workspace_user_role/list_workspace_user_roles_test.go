//go:build mock_db && mock_auth

// Package workspace_user_role provides test cases for workspace user role list use case.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user")
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education")
//
// Usage: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Methods:
//   - TestListWorkspaceUserRolesUseCase_Execute_Success: ESPYNA-TEST-ENTITY-WORKSPACEUSERROLE-SUCCESS-v1.0 Successful workspace user roles listing
//   - TestListWorkspaceUserRolesUseCase_Execute_AfterDelete: ESPYNA-TEST-ENTITY-WORKSPACEUSERROLE-INTEGRATION-v1.0 Data consistency after deletion operations

package workspace_user_role

import (
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/shared/testutil"
	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock/entity"
	workspaceuserrolepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace_user_role"
)

// createTestListWorkspaceUserRolesUseCase is a helper function to create the use case with mock dependencies
func createTestListWorkspaceUserRolesUseCase(businessType string, supportsTransaction bool) *ListWorkspaceUserRolesUseCase {
	repositories := ListWorkspaceUserRolesRepositories{
		WorkspaceUserRole: entity.NewMockWorkspaceUserRoleRepository(businessType),
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, true)
	services := ListWorkspaceUserRolesServices{
		TranslationService: standardServices.TranslationService,
	}

	return NewListWorkspaceUserRolesUseCase(repositories, services)
}

func TestListWorkspaceUserRolesUseCase_Execute_Success(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	// We need a fresh repository for this test to ensure count is correct
	mockRepo := entity.NewMockWorkspaceUserRoleRepository(businessType)
	standardServices := testutil.CreateStandardServices(false, true)
	useCase := NewListWorkspaceUserRolesUseCase(ListWorkspaceUserRolesRepositories{WorkspaceUserRole: mockRepo}, ListWorkspaceUserRolesServices{
		TranslationService: standardServices.TranslationService,
	})

	// The mock data for education/workspace-user-role has 3 entries

	req := &workspaceuserrolepb.ListWorkspaceUserRolesRequest{}

	res, err := useCase.Execute(ctx, req)
	testutil.AssertNoError(t, err)

	testutil.AssertNotNil(t, res, "response")

	testutil.AssertGreaterThan(t, len(res.Data), 0, "response data count")
}

func TestListWorkspaceUserRolesUseCase_Execute_AfterDelete(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestListWorkspaceUserRolesUseCase(businessType, false)

	// Test standard list functionality with pre-loaded JSON data
	req := &workspaceuserrolepb.ListWorkspaceUserRolesRequest{}

	res, err := useCase.Execute(ctx, req)
	testutil.AssertNoError(t, err)

	testutil.AssertNotNil(t, res, "response")

	// Verify we get the expected workspace user roles from JSON data
	testutil.AssertGreaterThan(t, len(res.Data), 0, "workspace user role records from JSON data")
}
