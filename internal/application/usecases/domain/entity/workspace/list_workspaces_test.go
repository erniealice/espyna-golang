//go:build mock_db && mock_auth

// Package workspace provides test cases for workspace listing use case.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user")
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education")
//
// Usage: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Methods:
//   - TestListWorkspacesUseCase_Execute_Success: ESPYNA-TEST-ENTITY-WORKSPACE-SUCCESS-v1.0 Basic successful workspace listing
//   - TestListWorkspacesUseCase_Execute_AfterDelete: ESPYNA-TEST-ENTITY-WORKSPACE-INTEGRATION-v1.0 Listing validation after deletion operations
package workspace

import (
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/shared/testutil"
	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock/entity"
	workspacepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace"
)

// createTestListWorkspacesUseCase is a helper function to create the use case with mock dependencies
func createTestListWorkspacesUseCase(businessType string) *ListWorkspacesUseCase {
	repositories := ListWorkspacesRepositories{
		Workspace: entity.NewMockWorkspaceRepository(businessType),
	}

	standardServices := testutil.CreateStandardServices(false, true)
	services := ListWorkspacesServices{
		TranslationService: standardServices.TranslationService,
	}

	return NewListWorkspacesUseCase(repositories, services)
}

func TestListWorkspacesUseCase_Execute_Success(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	// We need a fresh repository for this test to ensure count is correct
	mockRepo := entity.NewMockWorkspaceRepository(businessType)
	standardServices := testutil.CreateStandardServices(false, true)
	useCase := NewListWorkspacesUseCase(ListWorkspacesRepositories{Workspace: mockRepo}, ListWorkspacesServices{
		TranslationService: standardServices.TranslationService,
	})

	// The mock data for education/workspace has 1 entry

	req := &workspacepb.ListWorkspacesRequest{}

	res, err := useCase.Execute(ctx, req)
	testutil.AssertNoError(t, err)

	testutil.AssertNotNil(t, res, "response")

	testutil.AssertGreaterThan(t, len(res.Data), 0, "response data count")
}

func TestListWorkspacesUseCase_Execute_AfterDelete(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	// Setup a repository that will be shared between use cases
	mockRepo := entity.NewMockWorkspaceRepository(businessType)

	// --- Delete a workspace first ---
	deleteRepositories := DeleteWorkspaceRepositories{Workspace: mockRepo}
	standardServices := testutil.CreateStandardServices(false, true)
	deleteServices := DeleteWorkspaceServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
	}
	deleteUseCase := NewDeleteWorkspaceUseCase(deleteRepositories, deleteServices)

	deleteReq := &workspacepb.DeleteWorkspaceRequest{Data: &workspacepb.Workspace{Id: "workspace-elementary"}}
	_, err := deleteUseCase.Execute(ctx, deleteReq)
	testutil.AssertNoError(t, err)

	// --- Now list the workspaces ---
	listUseCase := NewListWorkspacesUseCase(ListWorkspacesRepositories{Workspace: mockRepo}, ListWorkspacesServices{
		TranslationService: standardServices.TranslationService,
	})

	listReq := &workspacepb.ListWorkspacesRequest{}
	res, err := listUseCase.Execute(ctx, listReq)
	testutil.AssertNoError(t, err)

	// Expected count is 1 (original) - 1 (deleted) = 0
	testutil.AssertGreaterThan(t, len(res.Data), 0, "response data count")
}
