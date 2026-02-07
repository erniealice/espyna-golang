//go:build mock_db && mock_auth

// Package workspace_user provides test cases for workspace user read use case.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user")
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education")
//
// Usage: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Methods:
//   - TestReadWorkspaceUserUseCase_Execute_Success: ESPYNA-TEST-ENTITY-WORKSPACEUSER-SUCCESS-v1.0 Validates successful workspace user relationship retrieval by ID
//   - TestReadWorkspaceUserUseCase_Execute_NotFound: ESPYNA-TEST-ENTITY-WORKSPACEUSER-VALIDATION-v1.0 Tests error handling for non-existent workspace user IDs
//   - TestReadWorkspaceUserUseCase_Execute_EmptyId: ESPYNA-TEST-ENTITY-WORKSPACEUSER-VALIDATION-v1.0 Validates input validation for empty ID field

package workspace_user

import (
	"testing"

	"leapfor.xyz/espyna/internal/application/shared/testutil"
	"leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock/entity"
	workspaceuserpb "leapfor.xyz/esqyma/golang/v1/domain/entity/workspace_user"
)

// createTestReadWorkspaceUserUseCase is a helper function to create the use case with mock dependencies
func createTestReadWorkspaceUserUseCase(businessType string) *ReadWorkspaceUserUseCase {
	repositories := ReadWorkspaceUserRepositories{
		WorkspaceUser: entity.NewMockWorkspaceUserRepository(businessType),
	}
	standardServices := testutil.CreateStandardServices(false, true)
	services := ReadWorkspaceUserServices{
		TranslationService: standardServices.TranslationService,
	}
	return NewReadWorkspaceUserUseCase(repositories, services)
}

func TestReadWorkspaceUserUseCase_Execute_Success(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestReadWorkspaceUserUseCase(businessType)

	// ID from packages/copya/data/education/workspace-user.json
	existingID := "workspace-user-001"

	req := &workspaceuserpb.ReadWorkspaceUserRequest{
		Data: &workspaceuserpb.WorkspaceUser{Id: existingID},
	}

	res, err := useCase.Execute(ctx, req)
	if err != nil {
		t.Fatalf("Expected no error, but got: %v", err)
	}

	if res == nil {
		t.Fatal("Expected a response, but got nil")
	}

	if len(res.Data) != 1 {
		t.Fatalf("Expected 1 workspace-user in response data, but got %d", len(res.Data))
	}

	readRel := res.Data[0]
	if readRel.Id != existingID {
		t.Errorf("Expected workspace-user ID to be '%s', but got '%s'", existingID, readRel.Id)
	}
	if readRel.UserId != "user-admin-001" {
		t.Errorf("Expected UserId to be 'user-admin-001', but got '%s'", readRel.UserId)
	}
}

func TestReadWorkspaceUserUseCase_Execute_NotFound(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestReadWorkspaceUserUseCase(businessType)

	nonExistentID := "workspace-user-999"

	req := &workspaceuserpb.ReadWorkspaceUserRequest{
		Data: &workspaceuserpb.WorkspaceUser{Id: nonExistentID},
	}

	_, err := useCase.Execute(ctx, req)
	if err == nil {
		t.Fatal("Expected an error for a non-existent workspace-user, but got none")
	}
	testutil.AssertTranslatedErrorWithContext(t, err, "workspace_user.errors.not_found", "{\"workspaceUserId\": \"workspace-user-999\"}", useCase.services.TranslationService, ctx)
}

func TestReadWorkspaceUserUseCase_Execute_EmptyId(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestReadWorkspaceUserUseCase(businessType)

	req := &workspaceuserpb.ReadWorkspaceUserRequest{
		Data: &workspaceuserpb.WorkspaceUser{Id: ""},
	}
	_, err := useCase.Execute(ctx, req)
	if err == nil {
		t.Fatal("Expected an error for an empty ID, but got none")
	}
	testutil.AssertTranslatedError(t, err, "workspace_user.validation.id_required", useCase.services.TranslationService, ctx)
}
