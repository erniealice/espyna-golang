//go:build mock_db && mock_auth

// Package delegate_client provides test cases for delegate client updating use case.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user")
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education")
//
// Usage: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Methods:
//   - TestUpdateDelegateClientUseCase_Execute_Success: ESPYNA-TEST-ENTITY-DELEGATECLIENT-SUCCESS-v1.0 Basic successful delegate client update
//   - TestUpdateDelegateClientUseCase_Execute_NotFound: ESPYNA-TEST-ENTITY-DELEGATECLIENT-NIL-v1.0 Update attempt with non-existent delegate client
//   - TestUpdateDelegateClientUseCase_Execute_InvalidReference: ESPYNA-TEST-ENTITY-DELEGATECLIENT-VALIDATION-v1.0 Invalid entity reference validation
package delegate_client

import (
	"testing"

	"leapfor.xyz/espyna/internal/application/shared/testutil"
	mockAuth "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/auth/mock"
	"leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock/entity"

	delegateclientpb "leapfor.xyz/esqyma/golang/v1/domain/entity/delegate_client"
)

// createTestUpdateDelegateClientUseCase is a helper function to create the use case with mock dependencies
func createTestUpdateDelegateClientUseCase(businessType string, supportsTransaction bool) *UpdateDelegateClientUseCase {
	repositories := UpdateDelegateClientRepositories{
		DelegateClient: entity.NewMockDelegateClientRepository(businessType),
		Delegate:       entity.NewMockDelegateRepository(businessType),
		Client:         entity.NewMockClientRepository(businessType),
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, true)
	services := UpdateDelegateClientServices{
		AuthorizationService: mockAuth.NewDisabledAuth(), // Use disabled auth to match other modules
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
	}

	return NewUpdateDelegateClientUseCase(repositories, services)
}

func TestUpdateDelegateClientUseCase_Execute_Success(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestUpdateDelegateClientUseCase(businessType, false)

	existingID := "delegate-client-001"
	originalTime := int64(1725148800000)

	// In this entity, an "update" is mostly just touching the DateModified timestamp
	// as the core fields (DelegateId, ClientId) are foreign keys that shouldn't change.
	// We'll "update" by providing the same data.
	req := &delegateclientpb.UpdateDelegateClientRequest{
		Data: &delegateclientpb.DelegateClient{
			Id:         existingID,
			DelegateId: "parent-001",
			ClientId:   "student-001",
		},
	}

	res, err := useCase.Execute(ctx, req)
	testutil.AssertNoError(t, err)

	testutil.AssertNotNil(t, res, "response")

	updatedRel := res.Data[0]
	testutil.AssertStringEqual(t, existingID, updatedRel.Id, "ID")

	testutil.AssertFieldSet(t, updatedRel.DateModified, "DateModified")
	testutil.AssertGreaterThan(t, int(*updatedRel.DateModified), int(originalTime), "DateModified")
}

func TestUpdateDelegateClientUseCase_Execute_NotFound(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestUpdateDelegateClientUseCase(businessType, false)

	nonExistentID := "delegate-client-999"
	req := &delegateclientpb.UpdateDelegateClientRequest{
		Data: &delegateclientpb.DelegateClient{
			Id:         nonExistentID,
			DelegateId: "parent-001",
			ClientId:   "student-001",
		},
	}

	_, err := useCase.Execute(ctx, req)
	testutil.AssertError(t, err)
	// The actual error will be from the repository layer for non-existent entities
	testutil.AssertTranslatedError(t, err, "delegate_client.errors.update_failed", useCase.services.TranslationService, ctx)
}

func TestUpdateDelegateClientUseCase_Execute_InvalidReference(t *testing.T) {
	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestUpdateDelegateClientUseCase(businessType, false)

	req := &delegateclientpb.UpdateDelegateClientRequest{
		Data: &delegateclientpb.DelegateClient{
			Id:         "delegate-client-001",
			DelegateId: "parent-999", // Non-existent delegate
			ClientId:   "student-001",
		},
	}

	_, err := useCase.Execute(ctx, req)
	testutil.AssertError(t, err)
	testutil.AssertTranslatedErrorWithContext(t, err, "delegate_client.errors.delegate_not_found", "{\"delegateId\": \"parent-999\"}", useCase.services.TranslationService, ctx)
}
