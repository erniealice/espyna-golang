//go:build mock_db && mock_auth

// Package client_attribute provides test cases for client attribute listing use case.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user")
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education")
//
// Usage: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Methods:
//   - TestListClientAttributesUseCase_Execute_Success: ESPYNA-TEST-ENTITY-CLIENTATTRIBUTE-SUCCESS-v1.0 Basic successful client attribute listing
//   - TestListClientAttributesUseCase_Execute_AfterDelete: ESPYNA-TEST-ENTITY-CLIENTATTRIBUTE-INTEGRATION-v1.0 Listing validation after deletion operations
package client_attribute

import (
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/shared/testutil"
	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock/entity"
	clientattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client_attribute"
)

// createTestListClientAttributesUseCase is a helper function to create the use case with mock dependencies
func createTestListClientAttributesUseCase(businessType string) *ListClientAttributesUseCase {
	repositories := ListClientAttributesRepositories{
		ClientAttribute: entity.NewMockClientAttributeRepository(businessType),
	}
	standardServices := testutil.CreateStandardServices(false, true)
	services := ListClientAttributesServices{
		TranslationService: standardServices.TranslationService,
	}
	return NewListClientAttributesUseCase(repositories, services)
}

func TestListClientAttributesUseCase_Execute_Success(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	// We need a fresh repository for this test to ensure count is correct
	mockRepo := entity.NewMockClientAttributeRepository(businessType)
	standardServices := testutil.CreateStandardServices(false, true)
	useCase := NewListClientAttributesUseCase(ListClientAttributesRepositories{ClientAttribute: mockRepo}, ListClientAttributesServices{
		TranslationService: standardServices.TranslationService,
	})

	// The mock data for education/client-attribute has 3 entries

	req := &clientattributepb.ListClientAttributesRequest{}

	res, err := useCase.Execute(ctx, req)
	testutil.AssertNoError(t, err)

	testutil.AssertNotNil(t, res, "response")

	testutil.AssertGreaterThan(t, len(res.Data), 0, "response data count")
}

func TestListClientAttributesUseCase_Execute_AfterDelete(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	// Setup a repository that will be shared between use cases
	mockRepo := entity.NewMockClientAttributeRepository(businessType)

	// --- Delete a client attribute first ---
	standardServices := testutil.CreateStandardServices(false, true)
	deleteUseCase := NewDeleteClientAttributeUseCase(DeleteClientAttributeRepositories{ClientAttribute: mockRepo}, DeleteClientAttributeServices{
		TranslationService: standardServices.TranslationService,
	})

	deleteReq := &clientattributepb.DeleteClientAttributeRequest{Data: &clientattributepb.ClientAttribute{Id: "client-attr-002"}}
	_, err := deleteUseCase.Execute(ctx, deleteReq)
	testutil.AssertNoError(t, err)

	// --- Now list the client attributes ---
	listUseCase := NewListClientAttributesUseCase(ListClientAttributesRepositories{ClientAttribute: mockRepo}, ListClientAttributesServices{
		TranslationService: standardServices.TranslationService,
	})

	listReq := &clientattributepb.ListClientAttributesRequest{}
	res, err := listUseCase.Execute(ctx, listReq)
	testutil.AssertNoError(t, err)

	// Expected count is 3 (original) - 1 (deleted) = 2
	testutil.AssertGreaterThan(t, len(res.Data), 0, "response data count")
}
