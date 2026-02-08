//go:build mock_db && mock_auth

// Package workspace provides test cases for workspace update use case.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user")
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education")
//
// Usage: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Methods:
//   - TestUpdateWorkspaceUseCase_Execute_Success: ESPYNA-TEST-ENTITY-WORKSPACE-SUCCESS-v1.0 Successful workspace update with timestamp verification
//   - TestUpdateWorkspaceUseCase_Execute_NotFound: ESPYNA-TEST-ENTITY-WORKSPACE-VALIDATION-v1.0 Error handling for non-existent workspace IDs
//   - TestUpdateWorkspaceUseCase_Execute_ValidationErrors: ESPYNA-TEST-ENTITY-WORKSPACE-VALIDATION-v1.0 Input validation and constraint checking

package workspace

import (
	"strings"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/shared/testutil"
	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock/entity"
	workspacepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace"
)

// createTestUpdateWorkspaceUseCase is a helper function to create the use case with mock dependencies
func createTestUpdateWorkspaceUseCase(businessType string) *UpdateWorkspaceUseCase {
	repositories := UpdateWorkspaceRepositories{
		Workspace: entity.NewMockWorkspaceRepository(businessType),
	}

	standardServices := testutil.CreateStandardServices(false, true)
	services := UpdateWorkspaceServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
	}

	return NewUpdateWorkspaceUseCase(repositories, services)
}

func TestUpdateWorkspaceUseCase_Execute_Success(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestUpdateWorkspaceUseCase(businessType)

	existingID := "workspace-elementary"
	updatedName := "Leapfor University Campus"
	originalTime := int64(1725148800000)

	req := &workspacepb.UpdateWorkspaceRequest{
		Data: &workspacepb.Workspace{
			Id:   existingID,
			Name: updatedName,
		},
	}

	res, err := useCase.Execute(ctx, req)
	testutil.AssertNoError(t, err)

	testutil.AssertNotNil(t, res, "response")

	updatedWorkspace := res.Data[0]
	testutil.AssertStringEqual(t, updatedName, updatedWorkspace.Name, "name")

	testutil.AssertFieldSet(t, updatedWorkspace.DateModified, "DateModified")
	testutil.AssertGreaterThan(t, int(*updatedWorkspace.DateModified), int(originalTime), "DateModified timestamp")
}

func TestUpdateWorkspaceUseCase_Execute_NotFound(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestUpdateWorkspaceUseCase(businessType)

	nonExistentID := "workspace-999"
	req := &workspacepb.UpdateWorkspaceRequest{
		Data: &workspacepb.Workspace{Id: nonExistentID, Name: "Ghost Workspace"},
	}

	_, err := useCase.Execute(ctx, req)
	testutil.AssertError(t, err)

	// Check for the actual error message format (from mock repository)
	expectedContent := "workspace with ID '" + nonExistentID + "' not found"
	if !strings.Contains(err.Error(), expectedContent) {
		t.Errorf("Expected error to contain '%s', got '%s'", expectedContent, err.Error())
	}
}

func TestUpdateWorkspaceUseCase_Execute_ValidationErrors(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestUpdateWorkspaceUseCase(businessType)
	existingID := "workspace-elementary"

	testCases := []struct {
		name             string
		workspace        *workspacepb.Workspace
		expectedErrorKey string
	}{
		{
			name:             "Empty ID",
			workspace:        &workspacepb.Workspace{Name: "test"},
			expectedErrorKey: "workspace.validation.id_required",
		},
		{
			name:             "Name too short",
			workspace:        &workspacepb.Workspace{Id: existingID, Name: "A"},
			expectedErrorKey: "workspace.validation.name_too_short",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := &workspacepb.UpdateWorkspaceRequest{Data: tc.workspace}
			_, err := useCase.Execute(ctx, req)
			testutil.AssertError(t, err)
			testutil.AssertTranslatedError(t, err, tc.expectedErrorKey, useCase.services.TranslationService, ctx)
		})
	}
}
