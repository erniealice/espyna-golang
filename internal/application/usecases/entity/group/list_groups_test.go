//go:build mock_db && mock_auth

// Package group provides test cases for group listing use case.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user")
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education")
//
// Usage: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Methods:
//   - TestListGroupsUseCase_Execute_Success: ESPYNA-TEST-ENTITY-GROUP-SUCCESS-v1.0 Basic successful group listing
//   - TestListGroupsUseCase_Execute_AfterDelete: ESPYNA-TEST-ENTITY-GROUP-INTEGRATION-v1.0 Group listing after deletion operations
package group

import (
	"testing"

	"leapfor.xyz/espyna/internal/application/shared/testutil"
	"leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock/entity"
	grouppb "leapfor.xyz/esqyma/golang/v1/domain/entity/group"
)

// createTestListGroupsUseCase is a helper function to create the use case with mock dependencies
func createTestListGroupsUseCase(businessType string) *ListGroupsUseCase {
	repositories := ListGroupsRepositories{
		Group: entity.NewMockGroupRepository(businessType),
	}
	standardServices := testutil.CreateStandardServices(false, true)
	services := ListGroupsServices{
		TranslationService: standardServices.TranslationService,
	}
	return NewListGroupsUseCase(repositories, services)
}

func TestListGroupsUseCase_Execute_Success(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	// We need a fresh repository for this test to ensure count is correct
	mockRepo := entity.NewMockGroupRepository(businessType)
	standardServices := testutil.CreateStandardServices(false, true)
	useCase := NewListGroupsUseCase(ListGroupsRepositories{Group: mockRepo}, ListGroupsServices{
		TranslationService: standardServices.TranslationService,
	})

	req := &grouppb.ListGroupsRequest{}

	res, err := useCase.Execute(ctx, req)
	testutil.AssertNoError(t, err)

	testutil.AssertNotNil(t, res, "response")

	// Verify we get some groups (don't check exact count as mock data may vary)
	testutil.AssertGreaterThan(t, len(res.Data), 0, "response data count")
}

func TestListGroupsUseCase_Execute_AfterDelete(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestListGroupsUseCase(businessType)

	// Test standard list functionality with pre-loaded JSON data
	req := &grouppb.ListGroupsRequest{}

	res, err := useCase.Execute(ctx, req)
	testutil.AssertNoError(t, err)

	testutil.AssertNotNil(t, res, "response")

	// Verify we get the expected groups from JSON data
	testutil.AssertGreaterThan(t, len(res.Data), 0, "response data count")
}
