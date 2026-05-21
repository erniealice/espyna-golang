//go:build mock_db && mock_auth

// Package workspace provides test cases for workspace read use case.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user")
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education")
//
// Usage: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Methods:
//   - TestReadWorkspaceUseCase_Execute_Success: ESPYNA-TEST-ENTITY-WORKSPACE-SUCCESS-v1.0 Successful workspace retrieval by ID
//   - TestReadWorkspaceUseCase_Execute_NotFound: ESPYNA-TEST-ENTITY-WORKSPACE-VALIDATION-v1.0 Error handling for non-existent workspace IDs
//   - TestReadWorkspaceUseCase_Execute_EmptyId: ESPYNA-TEST-ENTITY-WORKSPACE-NIL-v1.0 Empty ID validation and error handling

package workspace

import (
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/shared/testutil"
	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock/entity"
	workspacepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace"
)

// createTestReadWorkspaceUseCase is a helper function to create the use case with mock dependencies
func createTestReadWorkspaceUseCase(businessType string) *ReadWorkspaceUseCase {
	repositories := ReadWorkspaceRepositories{
		Workspace: entity.NewMockWorkspaceRepository(businessType),
	}

	standardServices := testutil.CreateStandardServices(false, true)
	services := ReadWorkspaceServices{
		TranslationService: standardServices.TranslationService,
	}

	return NewReadWorkspaceUseCase(repositories, services)
}

func TestReadWorkspaceUseCase_Execute_Success(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestReadWorkspaceUseCase(businessType)

	// ID from packages/copya/data/education/workspace.json
	existingID := "workspace-elementary"

	req := &workspacepb.ReadWorkspaceRequest{
		Data: &workspacepb.Workspace{Id: existingID},
	}

	res, err := useCase.Execute(ctx, req)
	testutil.AssertNoError(t, err)

	testutil.AssertNotNil(t, res, "response")

	testutil.AssertEqual(t, 1, len(res.Data), "response data length")

	readWorkspace := res.Data[0]
	testutil.AssertStringEqual(t, existingID, readWorkspace.Id, "workspace ID")
	testutil.AssertStringEqual(t, "Elementary School Division", readWorkspace.Name, "workspace name")
}

func TestReadWorkspaceUseCase_Execute_NotFound(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestReadWorkspaceUseCase(businessType)

	nonExistentID := "workspace-999"

	req := &workspacepb.ReadWorkspaceRequest{
		Data: &workspacepb.Workspace{Id: nonExistentID},
	}

	_, err := useCase.Execute(ctx, req)
	testutil.AssertError(t, err)
}

func TestReadWorkspaceUseCase_Execute_EmptyId(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestReadWorkspaceUseCase(businessType)

	req := &workspacepb.ReadWorkspaceRequest{
		Data: &workspacepb.Workspace{Id: ""},
	}
	_, err := useCase.Execute(ctx, req)
	testutil.AssertError(t, err)
	testutil.AssertTranslatedError(t, err, "workspace.validation.id_required", useCase.services.TranslationService, ctx)
}
