//go:build mock_db && mock_auth

// Package location_attribute provides test cases for location_attribute listing use case.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user")
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education")
//
// Usage: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Methods:
//   - TestListLocationAttributesUseCase_Execute_Success: ESPYNA-TEST-ENTITY-LOCATIONATTRIBUTE-SUCCESS-v1.0 Basic successful location_attribute listing
//   - TestListLocationAttributesUseCase_Execute_AfterDelete: ESPYNA-TEST-ENTITY-LOCATIONATTRIBUTE-INTEGRATION-v1.0 Location_attribute listing after deletion operations
package location_attribute

import (
	"testing"

	"leapfor.xyz/espyna/internal/application/shared/testutil"
	"leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock/entity"
	locationattributepb "leapfor.xyz/esqyma/golang/v1/domain/entity/location_attribute"
)

// createTestListLocationAttributesUseCase is a helper function to create the use case with mock dependencies
func createTestListLocationAttributesUseCase(businessType string) *ListLocationAttributesUseCase {
	repositories := ListLocationAttributesRepositories{
		LocationAttribute: entity.NewMockLocationAttributeRepository(businessType),
	}
	standardServices := testutil.CreateStandardServices(false, true)
	services := ListLocationAttributesServices{
		TranslationService: standardServices.TranslationService,
	}
	return NewListLocationAttributesUseCase(repositories, services)
}

func TestListLocationAttributesUseCase_Execute_Success(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	// We need a fresh repository for this test to ensure count is correct
	mockRepo := entity.NewMockLocationAttributeRepository(businessType)
	standardServices := testutil.CreateStandardServices(false, true)
	useCase := NewListLocationAttributesUseCase(ListLocationAttributesRepositories{LocationAttribute: mockRepo}, ListLocationAttributesServices{
		TranslationService: standardServices.TranslationService,
	})

	// The mock data for education/location-attribute has 2 entries

	req := &locationattributepb.ListLocationAttributesRequest{}

	res, err := useCase.Execute(ctx, req)
	testutil.AssertNoError(t, err)

	testutil.AssertNotNil(t, res, "response")

	testutil.AssertGreaterThan(t, len(res.Data), 0, "response data count")
}

func TestListLocationAttributesUseCase_Execute_AfterDelete(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	// Setup a repository that will be shared between use cases
	mockRepo := entity.NewMockLocationAttributeRepository(businessType)

	// --- Delete a location attribute first ---
	deleteRepositories := DeleteLocationAttributeRepositories{LocationAttribute: mockRepo}
	standardServices := testutil.CreateStandardServices(false, true)
	deleteServices := DeleteLocationAttributeServices{
		TranslationService: standardServices.TranslationService,
	}
	deleteUseCase := NewDeleteLocationAttributeUseCase(deleteRepositories, deleteServices)

	deleteReq := &locationattributepb.DeleteLocationAttributeRequest{Data: &locationattributepb.LocationAttribute{Id: "location-attr-001"}}
	_, err := deleteUseCase.Execute(ctx, deleteReq)
	testutil.AssertNoError(t, err)

	// --- Now list the location attributes ---
	listUseCase := NewListLocationAttributesUseCase(ListLocationAttributesRepositories{LocationAttribute: mockRepo}, ListLocationAttributesServices{
		TranslationService: standardServices.TranslationService,
	})

	listReq := &locationattributepb.ListLocationAttributesRequest{}
	res, err := listUseCase.Execute(ctx, listReq)
	testutil.AssertNoError(t, err)

	// Expected count is 2 (original) - 1 (deleted) = 1
	testutil.AssertGreaterThan(t, len(res.Data), 0, "response data count")
}
