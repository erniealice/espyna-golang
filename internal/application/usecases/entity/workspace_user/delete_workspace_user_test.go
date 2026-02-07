//go:build mock_db && mock_auth

// Package workspace_user provides test cases for workspace user delete use case.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user")
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education")
//
// Usage: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Methods:
//   - TestDeleteWorkspaceUserUseCase_Execute_Success: ESPYNA-TEST-ENTITY-WORKSPACEUSER-SUCCESS-v1.0 Validates successful workspace user relationship deletion with cross-operation verification
//   - TestDeleteWorkspaceUserUseCase_Execute_NotFound: ESPYNA-TEST-ENTITY-WORKSPACEUSER-VALIDATION-v1.0 Tests error handling for non-existent workspace user IDs
//   - TestDeleteWorkspaceUserUseCase_Execute_EmptyId: ESPYNA-TEST-ENTITY-WORKSPACEUSER-VALIDATION-v1.0 Validates input validation for empty ID field

package workspace_user

import (
	"testing"

	"leapfor.xyz/espyna/internal/application/shared/testutil"
	mockAuth "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/auth/mock"
	"leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock/entity"
	workspaceuserpb "leapfor.xyz/esqyma/golang/v1/domain/entity/workspace_user"
)

// createTestDeleteWorkspaceUserUseCase is a helper function to create the use case with mock dependencies
func createTestDeleteWorkspaceUserUseCase(businessType string) *DeleteWorkspaceUserUseCase {
	repositories := DeleteWorkspaceUserRepositories{
		WorkspaceUser: entity.NewMockWorkspaceUserRepository(businessType),
	}
	standardServices := testutil.CreateStandardServices(false, true)
	services := DeleteWorkspaceUserServices{
		AuthorizationService: mockAuth.NewAllowAllAuth().SetUserWorkspaces("test-user", "workspace-elementary", "workspace-middle", "workspace-high"),
		TranslationService:   standardServices.TranslationService,
	}
	return NewDeleteWorkspaceUserUseCase(repositories, services)
}

func TestDeleteWorkspaceUserUseCase_Execute_Success(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	// Create shared mock repository
	sharedWorkspaceUserRepo := entity.NewMockWorkspaceUserRepository(businessType)

	// Create delete use case with shared repository
	deleteRepositories := DeleteWorkspaceUserRepositories{
		WorkspaceUser: sharedWorkspaceUserRepo,
	}
	standardServices := testutil.CreateStandardServices(false, true)
	deleteServices := DeleteWorkspaceUserServices{
		AuthorizationService: mockAuth.NewAllowAllAuth().SetUserWorkspaces("test-user", "workspace-elementary", "workspace-middle", "workspace-high"),
		TranslationService:   standardServices.TranslationService,
	}
	deleteUseCase := NewDeleteWorkspaceUserUseCase(deleteRepositories, deleteServices)

	// Create read use case with same shared repository
	readRepositories := ReadWorkspaceUserRepositories{
		WorkspaceUser: sharedWorkspaceUserRepo,
	}
	readServices := ReadWorkspaceUserServices{
		TranslationService: standardServices.TranslationService,
	}
	readUseCase := NewReadWorkspaceUserUseCase(readRepositories, readServices)

	// This ID will be "deleted" from the mock repository
	existingID := "workspace-user-001"

	req := &workspaceuserpb.DeleteWorkspaceUserRequest{
		Data: &workspaceuserpb.WorkspaceUser{Id: existingID},
	}

	res, err := deleteUseCase.Execute(ctx, req)
	testutil.AssertNoError(t, err)

	testutil.AssertNotNil(t, res, "response")

	testutil.AssertTrue(t, res.Success, "success")

	// Verify the relationship is actually deleted using shared repository
	readReq := &workspaceuserpb.ReadWorkspaceUserRequest{Data: &workspaceuserpb.WorkspaceUser{Id: existingID}}
	_, err = readUseCase.Execute(ctx, readReq)
	testutil.AssertError(t, err)
}

func TestDeleteWorkspaceUserUseCase_Execute_NotFound(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestDeleteWorkspaceUserUseCase(businessType)

	nonExistentID := "workspace-user-999"
	req := &workspaceuserpb.DeleteWorkspaceUserRequest{
		Data: &workspaceuserpb.WorkspaceUser{Id: nonExistentID},
	}

	_, err := useCase.Execute(ctx, req)
	testutil.AssertError(t, err)
	testutil.AssertTranslatedError(t, err, "workspace_user.errors.authorization_failed", useCase.services.TranslationService, ctx)
}

func TestDeleteWorkspaceUserUseCase_Execute_EmptyId(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestDeleteWorkspaceUserUseCase(businessType)

	req := &workspaceuserpb.DeleteWorkspaceUserRequest{
		Data: &workspaceuserpb.WorkspaceUser{Id: ""},
	}
	_, err := useCase.Execute(ctx, req)
	testutil.AssertError(t, err)
	testutil.AssertTranslatedError(t, err, "workspace_user.validation.id_required", useCase.services.TranslationService, ctx)
}
