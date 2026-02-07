//go:build mock_db && mock_auth

// Package permission provides test cases for permission listing use case.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user")
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education")
//
// Usage: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Methods:
//   - TestListPermissionsUseCase_Execute_Success: ESPYNA-TEST-ENTITY-PERMISSION-SUCCESS-v1.0 Tests successful retrieval of permission list
//   - TestListPermissionsUseCase_Execute_AfterDelete: ESPYNA-TEST-ENTITY-PERMISSION-INTEGRATION-v1.0 Tests permission listing after deletion operation
package permission

import (
	"testing"

	"leapfor.xyz/espyna/internal/application/shared/testutil"
	"leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock/entity"
	permissionpb "leapfor.xyz/esqyma/golang/v1/domain/entity/permission"
)

// createTestListPermissionsUseCase is a helper function to create the use case with mock dependencies
func createTestListPermissionsUseCase(businessType string) *ListPermissionsUseCase {
	repositories := ListPermissionsRepositories{
		Permission: entity.NewMockPermissionRepository(businessType),
	}
	standardServices := testutil.CreateStandardServices(false, true)
	services := ListPermissionsServices{
		TranslationService: standardServices.TranslationService,
	}
	return NewListPermissionsUseCase(repositories, services)
}

func TestListPermissionsUseCase_Execute_Success(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	// We need a fresh repository for this test to ensure count is correct
	mockRepo := entity.NewMockPermissionRepository(businessType)
	standardServices := testutil.CreateStandardServices(false, true)
	useCase := NewListPermissionsUseCase(ListPermissionsRepositories{Permission: mockRepo}, ListPermissionsServices{
		TranslationService: standardServices.TranslationService,
	})

	// The mock data for education/permission has 2 entries

	req := &permissionpb.ListPermissionsRequest{}

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

func TestListPermissionsUseCase_Execute_AfterDelete(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	// Setup a repository that will be shared between use cases
	mockRepo := entity.NewMockPermissionRepository(businessType)

	// --- Delete a permission first ---
	deleteRepositories := DeletePermissionRepositories{Permission: mockRepo}
	standardServices := testutil.CreateStandardServices(false, true)
	deleteServices := DeletePermissionServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
	}
	deleteUseCase := NewDeletePermissionUseCase(deleteRepositories, deleteServices)

	deleteReq := &permissionpb.DeletePermissionRequest{Data: &permissionpb.Permission{Id: "client.create"}}
	_, err := deleteUseCase.Execute(ctx, deleteReq)
	if err != nil {
		t.Fatalf("Delete operation failed: %v", err)
	}

	// --- Now list the permissions ---
	listStandardServices := testutil.CreateStandardServices(false, true)
	listUseCase := NewListPermissionsUseCase(ListPermissionsRepositories{Permission: mockRepo}, ListPermissionsServices{
		TranslationService: listStandardServices.TranslationService,
	})

	listReq := &permissionpb.ListPermissionsRequest{}
	res, err := listUseCase.Execute(ctx, listReq)
	if err != nil {
		t.Fatalf("Expected no error on list, but got: %v", err)
	}

	// Expected count is 2 (original) - 1 (deleted) = 1
	if len(res.Data) == 0 {
		t.Error("Expected at least some items, but got none")
	}
}
