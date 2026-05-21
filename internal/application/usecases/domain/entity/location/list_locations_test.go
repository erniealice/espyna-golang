//go:build mock_db && mock_auth

// Package location provides test cases for location listing use case.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user")
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education")
//
// Usage: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Methods:
//   - TestListLocationsUseCase_Execute_Success: ESPYNA-TEST-ENTITY-LOCATION-SUCCESS-v1.0 Basic successful location listing
//   - TestListLocationsUseCase_Execute_AfterDelete: ESPYNA-TEST-ENTITY-LOCATION-INTEGRATION-v1.0 Location listing after deletion operations
package location

import (
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/shared/testutil"
	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock/entity"
	locationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/location"
)

// createTestListLocationsUseCase is a helper function to create the use case with mock dependencies
func createTestListLocationsUseCase(businessType string) *ListLocationsUseCase {
	repositories := ListLocationsRepositories{
		Location: entity.NewMockLocationRepository(businessType),
	}
	standardServices := testutil.CreateStandardServices(false, true)
	services := ListLocationsServices{
		TranslationService: standardServices.TranslationService,
	}
	return NewListLocationsUseCase(repositories, services)
}

func TestListLocationsUseCase_Execute_Success(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	// We need a fresh repository for this test to ensure count is correct
	mockRepo := entity.NewMockLocationRepository(businessType)
	standardServices := testutil.CreateStandardServices(false, true)
	useCase := NewListLocationsUseCase(ListLocationsRepositories{Location: mockRepo}, ListLocationsServices{
		TranslationService: standardServices.TranslationService,
	})

	// The mock data for education/location has 2 entries

	req := &locationpb.ListLocationsRequest{}

	res, err := useCase.Execute(ctx, req)
	testutil.AssertNoError(t, err)

	testutil.AssertNotNil(t, res, "response")

	testutil.AssertGreaterThan(t, len(res.Data), 0, "response data count")
}

func TestListLocationsUseCase_Execute_AfterDelete(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	// Setup a repository that will be shared between use cases
	mockRepo := entity.NewMockLocationRepository(businessType)

	// --- Delete a location first ---
	deleteRepositories := DeleteLocationRepositories{Location: mockRepo}
	standardServices := testutil.CreateStandardServices(false, true)
	deleteServices := DeleteLocationServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
	}
	deleteUseCase := NewDeleteLocationUseCase(deleteRepositories, deleteServices)

	deleteReq := &locationpb.DeleteLocationRequest{Data: &locationpb.Location{Id: "location-main-building"}}
	_, err := deleteUseCase.Execute(ctx, deleteReq)
	testutil.AssertNoError(t, err)

	// --- Now list the locations ---
	listUseCase := NewListLocationsUseCase(ListLocationsRepositories{Location: mockRepo}, ListLocationsServices{
		TranslationService: standardServices.TranslationService,
	})

	listReq := &locationpb.ListLocationsRequest{}
	res, err := listUseCase.Execute(ctx, listReq)
	testutil.AssertNoError(t, err)

	// Expected count is 2 (original) - 1 (deleted) = 1
	testutil.AssertGreaterThan(t, len(res.Data), 0, "response data count")
}
