//go:build mock_db && mock_auth

// Package permission provides test cases for permission reading use case.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user")
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education")
//
// Usage: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Methods:
//   - TestReadPermissionUseCase_Execute_Success: ESPYNA-TEST-ENTITY-PERMISSION-SUCCESS-v1.0 Tests successful retrieval of an existing permission by ID
//   - TestReadPermissionUseCase_Execute_NotFound: ESPYNA-TEST-ENTITY-PERMISSION-NIL-v1.0 Tests reading a non-existent permission returns error
//   - TestReadPermissionUseCase_Execute_EmptyId: ESPYNA-TEST-ENTITY-PERMISSION-VALIDATION-v1.0 Tests reading with empty ID returns validation error
package permission

import (
	"strings"
	"testing"

	"leapfor.xyz/espyna/internal/application/shared/testutil"
	"leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock/entity"
	permissionpb "leapfor.xyz/esqyma/golang/v1/domain/entity/permission"
)

// createTestReadPermissionUseCase is a helper function to create the use case with mock dependencies
func createTestReadPermissionUseCase(businessType string) *ReadPermissionUseCase {
	repositories := ReadPermissionRepositories{
		Permission: entity.NewMockPermissionRepository(businessType),
	}
	standardServices := testutil.CreateStandardServices(false, true)
	services := ReadPermissionServices{
		TranslationService: standardServices.TranslationService,
	}
	return NewReadPermissionUseCase(repositories, services)
}

func TestReadPermissionUseCase_Execute_Success(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestReadPermissionUseCase(businessType)

	// ID from packages/copya/data/education/permission.json
	existingID := "client.list"

	req := &permissionpb.ReadPermissionRequest{
		Data: &permissionpb.Permission{Id: existingID},
	}

	res, err := useCase.Execute(ctx, req)
	if err != nil {
		t.Fatalf("Expected no error, but got: %v", err)
	}

	if res == nil {
		t.Fatal("Expected a response, but got nil")
	}

	if len(res.Data) != 1 {
		t.Fatalf("Expected 1 permission in response data, but got %d", len(res.Data))
	}

	readPermission := res.Data[0]
	if readPermission.Id != existingID {
		t.Errorf("Expected permission ID to be '%s', but got '%s'", existingID, readPermission.Id)
	}
	// Skip permission code validation as it depends on mock repository implementation
}

func TestReadPermissionUseCase_Execute_NotFound(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestReadPermissionUseCase(businessType)

	nonExistentID := "permission-999"

	req := &permissionpb.ReadPermissionRequest{
		Data: &permissionpb.Permission{Id: nonExistentID},
	}

	_, err := useCase.Execute(ctx, req)
	if err == nil {
		t.Fatal("Expected an error for non-existent permission, but got none")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Expected error message to contain 'not found', but got '%s'", err.Error())
	}
}

func TestReadPermissionUseCase_Execute_EmptyId(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestReadPermissionUseCase(businessType)

	req := &permissionpb.ReadPermissionRequest{
		Data: &permissionpb.Permission{Id: ""},
	}
	_, err := useCase.Execute(ctx, req)
	if err == nil {
		t.Fatal("Expected an error for an empty ID, but got none")
	}
	testutil.AssertTranslatedError(t, err, "permission.validation.id_required", useCase.services.TranslationService, ctx)
}
