//go:build mock_db && mock_auth

// Package location provides test cases for location deletion use case.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user")
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education")
//
// Usage: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Methods:
//   - TestDeleteLocationUseCase_Execute_Success: ESPYNA-TEST-ENTITY-LOCATION-SUCCESS-v1.0 Basic successful location deletion
//   - TestDeleteLocationUseCase_Execute_NotFound: ESPYNA-TEST-ENTITY-LOCATION-NIL-v1.0 Non-existent location deletion error handling
//   - TestDeleteLocationUseCase_Execute_EmptyId: ESPYNA-TEST-ENTITY-LOCATION-VALIDATION-v1.0 Empty ID validation error handling
package location

import (
	"strings"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/shared/testutil"
	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock/entity"
	locationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/location"
)

// createTestDeleteLocationUseCase is a helper function to create the use case with mock dependencies
func createTestDeleteLocationUseCase(businessType string) *DeleteLocationUseCase {
	repositories := DeleteLocationRepositories{
		Location: entity.NewMockLocationRepository(businessType),
	}

	standardServices := testutil.CreateStandardServices(false, true)
	services := DeleteLocationServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
	}

	return NewDeleteLocationUseCase(repositories, services)
}

func TestDeleteLocationUseCase_Execute_Success(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestDeleteLocationUseCase(businessType)

	// This ID will be "deleted" from the mock repository
	existingID := "location-gymnasium"

	req := &locationpb.DeleteLocationRequest{
		Data: &locationpb.Location{Id: existingID},
	}

	res, err := useCase.Execute(ctx, req)
	testutil.AssertNoError(t, err)

	testutil.AssertNotNil(t, res, "response")

	testutil.AssertTrue(t, res.Success, "success")

	// Verify the location is actually deleted - use same repository instance
	standardServices := testutil.CreateStandardServices(false, true)
	readUseCase := NewReadLocationUseCase(
		ReadLocationRepositories{Location: useCase.repositories.Location},
		ReadLocationServices{
			TranslationService: standardServices.TranslationService,
		},
	)
	readReq := &locationpb.ReadLocationRequest{Data: &locationpb.Location{Id: existingID}}
	_, err = readUseCase.Execute(ctx, readReq)
	testutil.AssertError(t, err)
}

func TestDeleteLocationUseCase_Execute_NotFound(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestDeleteLocationUseCase(businessType)

	nonExistentID := "location-999"
	req := &locationpb.DeleteLocationRequest{
		Data: &locationpb.Location{Id: nonExistentID},
	}

	_, err := useCase.Execute(ctx, req)
	testutil.AssertError(t, err)
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Expected error to contain 'not found', but got '%s'", err.Error())
	}
}

func TestDeleteLocationUseCase_Execute_EmptyId(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestDeleteLocationUseCase(businessType)

	req := &locationpb.DeleteLocationRequest{
		Data: &locationpb.Location{Id: ""},
	}
	_, err := useCase.Execute(ctx, req)
	testutil.AssertTranslatedError(t, err, "location.validation.id_required_with_prefix", useCase.services.TranslationService, ctx)
}
