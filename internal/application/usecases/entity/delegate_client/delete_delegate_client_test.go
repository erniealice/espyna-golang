//go:build mock_db && mock_auth

// Package delegate_client provides test cases for delegate client deletion use case.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user")
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education")
//
// Usage: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Methods:
//   - TestDeleteDelegateClientUseCase_Execute_Success: ESPYNA-TEST-ENTITY-DELEGATECLIENT-SUCCESS-v1.0 Basic successful delegate client deletion
//   - TestDeleteDelegateClientUseCase_Execute_NotFound: ESPYNA-TEST-ENTITY-DELEGATECLIENT-NIL-v1.0 Deletion attempt with non-existent delegate client
//   - TestDeleteDelegateClientUseCase_Execute_EmptyId: ESPYNA-TEST-ENTITY-DELEGATECLIENT-VALIDATION-v1.0 Empty ID validation
package delegate_client

import (
	"testing"

	"leapfor.xyz/espyna/internal/application/shared/testutil"
	mockAuth "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/auth/mock"
	"leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock/entity"

	delegateclientpb "leapfor.xyz/esqyma/golang/v1/domain/entity/delegate_client"
)

// createTestDeleteDelegateClientUseCase is a helper function to create the use case with mock dependencies
func createTestDeleteDelegateClientUseCase(businessType string, supportsTransaction bool) *DeleteDelegateClientUseCase {
	repositories := DeleteDelegateClientRepositories{
		DelegateClient: entity.NewMockDelegateClientRepository(businessType),
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, true)
	services := DeleteDelegateClientServices{
		AuthorizationService: mockAuth.NewDisabledAuth(), // Use disabled auth to match other modules
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
	}
	return NewDeleteDelegateClientUseCase(repositories, services)
}

func TestDeleteDelegateClientUseCase_Execute_Success(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestDeleteDelegateClientUseCase(businessType, false)

	// Test deleting an existing delegate-client from JSON data
	existingID := "delegate-client-002" // Exists in delegate-client.json

	req := &delegateclientpb.DeleteDelegateClientRequest{
		Data: &delegateclientpb.DelegateClient{Id: existingID},
	}

	res, err := useCase.Execute(ctx, req)
	testutil.AssertNoError(t, err)

	testutil.AssertNotNil(t, res, "response")

	testutil.AssertTrue(t, res.Success, "success")

	// Verify deletion by attempting to read the deleted relationship
	readResp, err := useCase.repositories.DelegateClient.ReadDelegateClient(ctx, &delegateclientpb.ReadDelegateClientRequest{
		Data: &delegateclientpb.DelegateClient{Id: existingID},
	})
	if err == nil && readResp != nil && readResp.Success && len(readResp.Data) > 0 {
		t.Error("Expected delegate-client to be deleted, but it still exists")
	}
}

func TestDeleteDelegateClientUseCase_Execute_NotFound(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestDeleteDelegateClientUseCase(businessType, false)

	nonExistentID := "delegate-client-999"
	req := &delegateclientpb.DeleteDelegateClientRequest{
		Data: &delegateclientpb.DelegateClient{Id: nonExistentID},
	}

	_, err := useCase.Execute(ctx, req)
	testutil.AssertError(t, err)
	testutil.AssertTranslatedError(t, err, "delegate_client.errors.deletion_failed", useCase.services.TranslationService, ctx)
}

func TestDeleteDelegateClientUseCase_Execute_EmptyId(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestDeleteDelegateClientUseCase(businessType, false)

	req := &delegateclientpb.DeleteDelegateClientRequest{
		Data: &delegateclientpb.DelegateClient{Id: ""},
	}
	_, err := useCase.Execute(ctx, req)
	testutil.AssertError(t, err)
	testutil.AssertTranslatedError(t, err, "delegate_client.validation.id_required", useCase.services.TranslationService, ctx)
}
