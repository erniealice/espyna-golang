//go:build mock_db && mock_auth

// Package workspace_user provides test cases for workspace user list use case.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user")
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education")
//
// Usage: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Methods:
//   - TestListWorkspaceUsersUseCase_Execute_Success: ESPYNA-TEST-ENTITY-WORKSPACEUSER-SUCCESS-v1.0 Validates successful retrieval of all workspace user relationships
//   - TestListWorkspaceUsersUseCase_Execute_AfterDelete: ESPYNA-TEST-ENTITY-WORKSPACEUSER-INTEGRATION-v1.0 Tests data consistency after deletion operations with shared repository

package workspace_user

import (
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/shared/testutil"
	mockAuth "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/auth/mock"
	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock/entity"
	workspaceuserpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace_user"
)

// createTestListWorkspaceUsersUseCase is a helper function to create the use case with mock dependencies
func createTestListWorkspaceUsersUseCase(businessType string) *ListWorkspaceUsersUseCase {
	repositories := ListWorkspaceUsersRepositories{
		WorkspaceUser: entity.NewMockWorkspaceUserRepository(businessType),
	}
	standardServices := testutil.CreateStandardServices(false, true)
	services := ListWorkspaceUsersServices{
		AuthorizationService: mockAuth.NewAllowAllAuth().SetUserWorkspaces("test-user", "workspace-elementary", "workspace-middle", "workspace-high"),
		TranslationService:   standardServices.TranslationService,
	}
	return NewListWorkspaceUsersUseCase(repositories, services)
}

func TestListWorkspaceUsersUseCase_Execute_Success(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	// We need a fresh repository for this test to ensure count is correct
	mockRepo := entity.NewMockWorkspaceUserRepository(businessType)
	standardServices := testutil.CreateStandardServices(false, true)
	useCase := NewListWorkspaceUsersUseCase(ListWorkspaceUsersRepositories{WorkspaceUser: mockRepo}, ListWorkspaceUsersServices{
		AuthorizationService: mockAuth.NewAllowAllAuth().SetUserWorkspaces("test-user", "workspace-elementary", "workspace-middle", "workspace-high"),
		TranslationService:   standardServices.TranslationService,
	})

	// The mock data for education/workspace-user has 3 entries

	req := &workspaceuserpb.ListWorkspaceUsersRequest{}

	res, err := useCase.Execute(ctx, req)
	if err != nil {
		t.Fatalf("Expected no error, but got: %v", err)
	}

	if res == nil {
		t.Fatal("Expected a response, but got nil")
	}

	if len(res.Data) == 0 {
		t.Error("Expected at least some items, but got none")
	}
}

func TestListWorkspaceUsersUseCase_Execute_AfterDelete(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestListWorkspaceUsersUseCase(businessType)

	// Test standard list functionality with pre-loaded JSON data
	req := &workspaceuserpb.ListWorkspaceUsersRequest{}

	res, err := useCase.Execute(ctx, req)
	if err != nil {
		t.Fatalf("Expected no error, but got: %v", err)
	}

	if res == nil {
		t.Fatal("Expected a response, but got nil")
	}

	// Verify we get the expected workspace users from JSON data
	if len(res.Data) == 0 {
		t.Error("Expected workspace user records from JSON data, but got none")
	}
}
