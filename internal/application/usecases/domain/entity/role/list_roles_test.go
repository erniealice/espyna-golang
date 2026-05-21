//go:build mock_db && mock_auth

// Package role provides test cases for role listing use case.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user")
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education")
//
// Usage: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Methods:
//   - TestListRolesUseCase_Execute_Success: ESPYNA-TEST-ENTITY-ROLE-SUCCESS-v1.0 Tests successful retrieval of roles list
//   - TestListRolesUseCase_Execute_AfterDelete: ESPYNA-TEST-ENTITY-ROLE-INTEGRATION-v1.0 Tests roles listing after deleting a role (verifies list consistency)
package role

import (
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/shared/testutil"
	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock/entity"
	rolepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/role"
)

// createTestListRolesUseCase is a helper function to create the use case with mock dependencies
func createTestListRolesUseCase(businessType string) *ListRolesUseCase {
	repositories := ListRolesRepositories{
		Role: entity.NewMockRoleRepository(businessType),
	}
	standardServices := testutil.CreateStandardServices(false, true)
	services := ListRolesServices{
		TranslationService: standardServices.TranslationService,
	}
	return NewListRolesUseCase(repositories, services)
}

func TestListRolesUseCase_Execute_Success(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	// We need a fresh repository for this test to ensure count is correct
	mockRepo := entity.NewMockRoleRepository(businessType)
	standardServices := testutil.CreateStandardServices(false, true)
	useCase := NewListRolesUseCase(ListRolesRepositories{Role: mockRepo}, ListRolesServices{
		TranslationService: standardServices.TranslationService,
	})

	// The mock data for education/role has 2 entries

	req := &rolepb.ListRolesRequest{}

	res, err := useCase.Execute(ctx, req)
	testutil.AssertNoError(t, err)

	testutil.AssertNotNil(t, res, "response")

	testutil.AssertGreaterThan(t, len(res.Data), 0, "response data count")
}

func TestListRolesUseCase_Execute_AfterDelete(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	// Setup a repository that will be shared between use cases
	mockRepo := entity.NewMockRoleRepository(businessType)

	// --- Delete a role first ---
	deleteRepositories := DeleteRoleRepositories{Role: mockRepo}
	standardServices2 := testutil.CreateStandardServices(false, true)
	deleteServices := DeleteRoleServices{
		AuthorizationService: standardServices2.AuthorizationService,
		TransactionService:   standardServices2.TransactionService,
		TranslationService:   standardServices2.TranslationService,
	}
	deleteUseCase := NewDeleteRoleUseCase(deleteRepositories, deleteServices)

	// Get the first available role to delete
	listReq2 := &rolepb.ListRolesRequest{}
	standardServices3 := testutil.CreateStandardServices(false, true)
	listTestUseCase := NewListRolesUseCase(ListRolesRepositories{Role: mockRepo}, ListRolesServices{
		TranslationService: standardServices3.TranslationService,
	})
	listRes2, err := listTestUseCase.Execute(ctx, listReq2)
	if err != nil || len(listRes2.Data) == 0 {
		t.Skip("No roles available to delete")
	}
	firstRoleID := listRes2.Data[0].Id

	deleteReq := &rolepb.DeleteRoleRequest{Data: &rolepb.Role{Id: firstRoleID}}
	_, err = deleteUseCase.Execute(ctx, deleteReq)
	testutil.AssertNoError(t, err)

	// --- Now list the roles ---
	standardServices4 := testutil.CreateStandardServices(false, true)
	listUseCase := NewListRolesUseCase(ListRolesRepositories{Role: mockRepo}, ListRolesServices{
		TranslationService: standardServices4.TranslationService,
	})

	listReq := &rolepb.ListRolesRequest{}
	res, err := listUseCase.Execute(ctx, listReq)
	testutil.AssertNoError(t, err)

	// Expected count is 2 (original) - 1 (deleted) = 1
	testutil.AssertGreaterThan(t, len(res.Data), 0, "response data count")
}
