//go:build mock_db && mock_auth

// Package workspace provides test cases for workspace delete use case.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user")
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education")
//
// Usage: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Methods:
//   - TestDeleteWorkspaceUseCase_Execute_Success: ESPYNA-TEST-ENTITY-WORKSPACE-INTEGRATION-v1.0 Successful workspace deletion with cross-operation verification
//   - TestDeleteWorkspaceUseCase_Execute_NotFound: ESPYNA-TEST-ENTITY-WORKSPACE-VALIDATION-v1.0 Error handling for non-existent workspace IDs
//   - TestDeleteWorkspaceUseCase_Execute_EmptyId: ESPYNA-TEST-ENTITY-WORKSPACE-NIL-v1.0 Empty ID validation and error handling

package workspace

import (
	"strings"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/shared/testutil"
	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock/entity"
	workspacepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace"
)

// createTestDeleteWorkspaceUseCase is a helper function to create the use case with mock dependencies
func createTestDeleteWorkspaceUseCase(businessType string) *DeleteWorkspaceUseCase {
	repositories := DeleteWorkspaceRepositories{
		Workspace: entity.NewMockWorkspaceRepository(businessType),
	}

	standardServices := testutil.CreateStandardServices(false, true)
	services := DeleteWorkspaceServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
	}

	return NewDeleteWorkspaceUseCase(repositories, services)
}

func TestDeleteWorkspaceUseCase_Execute_Success(t *testing.T) {
	businessType := testutil.GetTestBusinessType()
	// Create shared repository to maintain state between operations
	sharedWorkspaceRepo := entity.NewMockWorkspaceRepository(businessType)

	// Create delete use case with shared repo
	deleteRepositories := DeleteWorkspaceRepositories{
		Workspace: sharedWorkspaceRepo,
	}
	standardServices := testutil.CreateStandardServices(false, true)
	deleteServices := DeleteWorkspaceServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
	}
	deleteUseCase := NewDeleteWorkspaceUseCase(deleteRepositories, deleteServices)

	// Create read use case with same shared repo
	readRepositories := ReadWorkspaceRepositories{
		Workspace: sharedWorkspaceRepo,
	}
	readServices := ReadWorkspaceServices{
		TranslationService: standardServices.TranslationService,
	}
	readUseCase := NewReadWorkspaceUseCase(readRepositories, readServices)

	ctx := testutil.CreateTestContext()

	// This ID will be "deleted" from the mock repository
	existingID := "workspace-elementary"

	req := &workspacepb.DeleteWorkspaceRequest{
		Data: &workspacepb.Workspace{Id: existingID},
	}

	res, err := deleteUseCase.Execute(ctx, req)
	testutil.AssertNoError(t, err)

	testutil.AssertNotNil(t, res, "response")

	testutil.AssertTrue(t, res.Success, "success")

	// Verify the workspace is actually deleted using the same shared repo
	readReq := &workspacepb.ReadWorkspaceRequest{Data: &workspacepb.Workspace{Id: existingID}}
	_, err = readUseCase.Execute(ctx, readReq)
	testutil.AssertError(t, err)
}

func TestDeleteWorkspaceUseCase_Execute_NotFound(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestDeleteWorkspaceUseCase(businessType)

	nonExistentID := "workspace-999"
	req := &workspacepb.DeleteWorkspaceRequest{
		Data: &workspacepb.Workspace{Id: nonExistentID},
	}

	_, err := useCase.Execute(ctx, req)
	testutil.AssertError(t, err)

	// Check for the actual error message format (from mock repository)
	expectedContent := "workspace with ID '" + nonExistentID + "' not found"
	if !strings.Contains(err.Error(), expectedContent) {
		t.Errorf("Expected error to contain '%s', got '%s'", expectedContent, err.Error())
	}
}

func TestDeleteWorkspaceUseCase_Execute_EmptyId(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestDeleteWorkspaceUseCase(businessType)

	req := &workspacepb.DeleteWorkspaceRequest{
		Data: &workspacepb.Workspace{Id: ""},
	}
	_, err := useCase.Execute(ctx, req)
	testutil.AssertError(t, err)
	testutil.AssertTranslatedError(t, err, "workspace.validation.id_required", useCase.services.TranslationService, ctx)
}
