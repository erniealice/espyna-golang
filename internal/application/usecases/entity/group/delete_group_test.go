//go:build mock_db && mock_auth

// Package group provides test cases for group deletion use case.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user")
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education")
//
// Usage: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Methods:
//   - TestDeleteGroupUseCase_Execute_Success: ESPYNA-TEST-ENTITY-GROUP-SUCCESS-v1.0 Basic successful group deletion
//   - TestDeleteGroupUseCase_Execute_NotFound: ESPYNA-TEST-ENTITY-GROUP-NIL-v1.0 Non-existent group deletion error handling
//   - TestDeleteGroupUseCase_Execute_EmptyId: ESPYNA-TEST-ENTITY-GROUP-VALIDATION-v1.0 Empty ID validation error handling
package group

import (
	"strings"
	"testing"

	"leapfor.xyz/espyna/internal/application/shared/testutil"
	"leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock/entity"
	grouppb "leapfor.xyz/esqyma/golang/v1/domain/entity/group"
)

// createTestDeleteGroupUseCase is a helper function to create the use case with mock dependencies
func createTestDeleteGroupUseCase(businessType string, supportsTransaction bool) *DeleteGroupUseCase {
	repositories := DeleteGroupRepositories{
		Group: entity.NewMockGroupRepository(businessType),
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, true)
	services := DeleteGroupServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
	}

	return NewDeleteGroupUseCase(repositories, services)
}

func TestDeleteGroupUseCase_Execute_Success(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	// Create shared repository instance
	groupRepo := entity.NewMockGroupRepository(businessType)

	// Create delete use case with shared repository
	deleteRepositories := DeleteGroupRepositories{
		Group: groupRepo,
	}
	standardServices := testutil.CreateStandardServices(false, true)
	deleteServices := DeleteGroupServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
	}
	useCase := NewDeleteGroupUseCase(deleteRepositories, deleteServices)

	// This ID will be "deleted" from the mock repository
	existingID := "group-science-fair"

	req := &grouppb.DeleteGroupRequest{
		Data: &grouppb.Group{Id: existingID},
	}

	res, err := useCase.Execute(ctx, req)
	testutil.AssertNoError(t, err)

	testutil.AssertNotNil(t, res, "response")

	testutil.AssertTrue(t, res.Success, "success")

	// Verify the group is actually deleted using the same repository instance
	readRepositories := ReadGroupRepositories{
		Group: groupRepo, // Use same repository instance
	}
	readStandardServices := testutil.CreateStandardServices(false, true)
	readServices := ReadGroupServices{
		TranslationService: readStandardServices.TranslationService,
	}
	readUseCase := NewReadGroupUseCase(readRepositories, readServices)

	readReq := &grouppb.ReadGroupRequest{Data: &grouppb.Group{Id: existingID}}
	readRes, err := readUseCase.Execute(ctx, readReq)
	testutil.AssertNoError(t, err)
	testutil.AssertEqual(t, 0, len(readRes.Data), "deleted group data count")
}

func TestDeleteGroupUseCase_Execute_NotFound(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestDeleteGroupUseCase(businessType, false)

	nonExistentID := "group-999"
	req := &grouppb.DeleteGroupRequest{
		Data: &grouppb.Group{Id: nonExistentID},
	}

	_, err := useCase.Execute(ctx, req)
	testutil.AssertError(t, err)
	if !strings.Contains(err.Error(), "Group deletion failed") {
		t.Errorf("Expected error to contain 'Group deletion failed', but got '%s'", err.Error())
	}
}

func TestDeleteGroupUseCase_Execute_EmptyId(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestDeleteGroupUseCase(businessType, false)

	req := &grouppb.DeleteGroupRequest{
		Data: &grouppb.Group{Id: ""},
	}
	_, err := useCase.Execute(ctx, req)
	testutil.AssertError(t, err)
	if !strings.Contains(err.Error(), "Group ID is required") {
		t.Errorf("Expected error to contain 'Group ID is required', but got '%s'", err.Error())
	}
}
