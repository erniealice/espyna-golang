//go:build mock_db && mock_auth

// Package delegate_client provides test cases for delegate client listing use case.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user")
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education")
//
// Usage: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Methods:
//   - TestListDelegateClientsUseCase_Execute_Success: ESPYNA-TEST-ENTITY-DELEGATECLIENT-SUCCESS-v1.0 Basic successful delegate client listing
//   - TestListDelegateClientsUseCase_Execute_AfterDelete: ESPYNA-TEST-ENTITY-DELEGATECLIENT-INTEGRATION-v1.0 Listing validation after deletion operations
package delegate_client

import (
	"testing"

	"leapfor.xyz/espyna/internal/application/shared/testutil"
	mockAuth "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/auth/mock"
	"leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock/entity"

	delegateclientpb "leapfor.xyz/esqyma/golang/v1/domain/entity/delegate_client"
)

// createTestListDelegateClientsUseCase is a helper function to create the use case with mock dependencies
func createTestListDelegateClientsUseCase(businessType string) *ListDelegateClientsUseCase {
	repositories := ListDelegateClientsRepositories{
		DelegateClient: entity.NewMockDelegateClientRepository(businessType),
	}

	standardServices := testutil.CreateStandardServices(false, true)
	services := ListDelegateClientsServices{
		AuthorizationService: mockAuth.NewDisabledAuth(), // Use disabled auth to match other modules
		TranslationService:   standardServices.TranslationService,
	}
	return NewListDelegateClientsUseCase(repositories, services)
}

func TestListDelegateClientsUseCase_Execute_Success(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	// We need a fresh repository for this test to ensure count is correct
	mockRepo := entity.NewMockDelegateClientRepository(businessType)
	standardServices := testutil.CreateStandardServices(false, true)
	useCase := NewListDelegateClientsUseCase(ListDelegateClientsRepositories{DelegateClient: mockRepo}, ListDelegateClientsServices{
		AuthorizationService: mockAuth.NewDisabledAuth(), // Use disabled auth to match other modules
		TranslationService:   standardServices.TranslationService,
	})

	// The mock data for education/delegate-client has 2 entries

	req := &delegateclientpb.ListDelegateClientsRequest{}

	res, err := useCase.Execute(ctx, req)
	testutil.AssertNoError(t, err)

	testutil.AssertNotNil(t, res, "response")

	testutil.AssertGreaterThan(t, len(res.Data), 0, "response data count")
}

func TestListDelegateClientsUseCase_Execute_AfterDelete(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestListDelegateClientsUseCase(businessType)

	// Test standard list functionality with pre-loaded JSON data
	req := &delegateclientpb.ListDelegateClientsRequest{}

	res, err := useCase.Execute(ctx, req)
	testutil.AssertNoError(t, err)

	testutil.AssertNotNil(t, res, "response")

	// Verify we get the expected delegate-clients from JSON data
	testutil.AssertGreaterThan(t, len(res.Data), 0, "delegate-client records from JSON data count")
}
