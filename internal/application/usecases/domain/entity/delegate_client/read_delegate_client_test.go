//go:build mock_db && mock_auth

// Package delegate_client provides test cases for delegate client reading use case.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user")
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education")
//
// Usage: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Methods:
//   - TestReadDelegateClientUseCase_Execute_Success: ESPYNA-TEST-ENTITY-DELEGATECLIENT-SUCCESS-v1.0 Basic successful delegate client retrieval
//   - TestReadDelegateClientUseCase_Execute_NotFound: ESPYNA-TEST-ENTITY-DELEGATECLIENT-NIL-v1.0 Read attempt with non-existent delegate client
//   - TestReadDelegateClientUseCase_Execute_EmptyId: ESPYNA-TEST-ENTITY-DELEGATECLIENT-VALIDATION-v1.0 Empty ID validation
package delegate_client

import (
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/shared/testutil"
	mockAuth "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/auth/mock"
	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock/entity"

	delegateclientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/delegate_client"
)

// createTestReadDelegateClientUseCase is a helper function to create the use case with mock dependencies
func createTestReadDelegateClientUseCase(businessType string) *ReadDelegateClientUseCase {
	repositories := ReadDelegateClientRepositories{
		DelegateClient: entity.NewMockDelegateClientRepository(businessType),
		Delegate:       entity.NewMockDelegateRepository(businessType),
		Client:         entity.NewMockClientRepository(businessType),
	}

	standardServices := testutil.CreateStandardServices(false, true)
	services := ReadDelegateClientServices{
		AuthorizationService: mockAuth.NewDisabledAuth(), // Use disabled auth to match other modules
		TranslationService:   standardServices.TranslationService,
	}
	return NewReadDelegateClientUseCase(repositories, services)
}

func TestReadDelegateClientUseCase_Execute_Success(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestReadDelegateClientUseCase(businessType)

	// ID from packages/copya/data/education/delegate-client.json
	existingID := "delegate-client-001"

	req := &delegateclientpb.ReadDelegateClientRequest{
		Data: &delegateclientpb.DelegateClient{Id: existingID},
	}

	res, err := useCase.Execute(ctx, req)
	testutil.AssertNoError(t, err)

	testutil.AssertNotNil(t, res, "response")

	testutil.AssertEqual(t, 1, len(res.Data), "response data length")

	readRel := res.Data[0]
	testutil.AssertStringEqual(t, existingID, readRel.Id, "delegate-client ID")
	testutil.AssertStringEqual(t, "parent-001", readRel.DelegateId, "DelegateId")
}

func TestReadDelegateClientUseCase_Execute_NotFound(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestReadDelegateClientUseCase(businessType)

	nonExistentID := "delegate-client-999"

	req := &delegateclientpb.ReadDelegateClientRequest{
		Data: &delegateclientpb.DelegateClient{Id: nonExistentID},
	}

	res, err := useCase.Execute(ctx, req)
	testutil.AssertNoError(t, err)

	testutil.AssertNotNil(t, res, "response")

	testutil.AssertEqual(t, 0, len(res.Data), "response data length for non-existent delegate-client")
}

func TestReadDelegateClientUseCase_Execute_EmptyId(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestReadDelegateClientUseCase(businessType)

	req := &delegateclientpb.ReadDelegateClientRequest{
		Data: &delegateclientpb.DelegateClient{Id: ""},
	}
	_, err := useCase.Execute(ctx, req)
	testutil.AssertError(t, err)
	testutil.AssertTranslatedError(t, err, "delegate_client.validation.id_required", useCase.services.TranslationService, ctx)
}
