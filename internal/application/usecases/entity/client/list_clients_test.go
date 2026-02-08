//go:build mock_db && mock_auth

// Package client provides comprehensive tests for the client listing use case.
//
// The tests cover various scenarios, including basic listing, integration with deletion,
// authorization, and boundary conditions. Each test function has a specific test code
// for tracking and logging.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user").
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education").
//
// Usage:
//   - Run all tests: go test -tags="mock_db,mock_auth" ./...
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestListClientsUseCase_Execute_Success
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-ENTITY-CLIENT-LIST-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-ENTITY-CLIENT-LIST-INTEGRATION-v1.0: Integration
//
// Data Sources:
//   - Test cases: packages/copya/data_test/{businessType}/client.json
//   - Mock data: packages/copya/data/{businessType}/client.json
//   - Translations: packages/lyngua/translations/{languageCode}/{businessType}/client.json
package client

import (
	"testing"

	copyatestutil "leapfor.xyz/copya/golang/testutil"
	"github.com/erniealice/espyna-golang/internal/application/shared/testutil"
	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock/entity"

	clientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client"
)

func createTestListClientsUseCase(businessType string, supportsTransaction bool) *ListClientsUseCase {
	repositories := ListClientsRepositories{
		Client: entity.NewMockClientRepository(businessType),
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, true)
	services := ListClientsServices{
		AuthorizationService: standardServices.AuthorizationService,
		TranslationService:   standardServices.TranslationService,
	}

	return NewListClientsUseCase(repositories, services)
}

func TestListClientsUseCase_Execute_Success(t *testing.T) {
	testCode := "ESPYNA-TEST-ENTITY-CLIENT-LIST-SUCCESS-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "Success", true)

	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestListClientsUseCase(businessType, false)

	// Load test data resolvers
	commonDataResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "client", "Client_CommonData")
	testutil.AssertTestCaseLoad(t, err, "Client_CommonData")

	req := &clientpb.ListClientsRequest{}

	res, err := useCase.Execute(ctx, req)
	testutil.AssertNoError(t, err)

	testutil.AssertNotNil(t, res, "response")

	testutil.AssertTrue(t, len(res.Data) > 0, "should have at least some clients")

	// Verify we can find expected client IDs from test data
	primaryClientId := commonDataResolver.MustGetString("primaryClientId")
	found := false
	for _, client := range res.Data {
		if client.Id == primaryClientId {
			found = true
			break
		}
	}
	testutil.AssertTrue(t, found, "should find primary client in list")

	// Log test completion with result
	testutil.LogTestResult(t, testCode, "Success", true, nil)
}

func TestListClientsUseCase_Execute_AfterDelete(t *testing.T) {
	testCode := "ESPYNA-TEST-ENTITY-CLIENT-LIST-INTEGRATION-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "Integration", true)

	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()

	// Load test data resolvers
	commonDataResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "client", "Client_CommonData")
	testutil.AssertTestCaseLoad(t, err, "Client_CommonData")

	// Setup a repository that will be shared between use cases
	mockRepo := entity.NewMockClientRepository(businessType)

	// --- Delete a client first ---
	standardServices := testutil.CreateStandardServices(false, true)
	deleteUseCase := NewDeleteClientUseCase(DeleteClientRepositories{Client: mockRepo}, DeleteClientServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
	})

	clientToDelete := commonDataResolver.MustGetString("secondaryClientId")
	deleteReq := &clientpb.DeleteClientRequest{Data: &clientpb.Client{Id: clientToDelete}}
	_, err = deleteUseCase.Execute(ctx, deleteReq)
	testutil.AssertNoError(t, err)

	// --- Now list the clients ---
	listUseCase := NewListClientsUseCase(ListClientsRepositories{Client: mockRepo}, ListClientsServices{
		AuthorizationService: standardServices.AuthorizationService,
		TranslationService:   standardServices.TranslationService,
	})

	listReq := &clientpb.ListClientsRequest{}
	res, err := listUseCase.Execute(ctx, listReq)
	testutil.AssertNoError(t, err)

	testutil.AssertNotNil(t, res, "response")
	testutil.AssertTrue(t, len(res.Data) > 0, "should still have remaining clients after deletion")

	// Verify the deleted client is not in the list
	for _, client := range res.Data {
		testutil.AssertNotEqual(t, clientToDelete, client.Id, "deleted client should not appear in list")
	}

	// Log test completion with result
	testutil.LogTestResult(t, testCode, "Integration", true, nil)
}
