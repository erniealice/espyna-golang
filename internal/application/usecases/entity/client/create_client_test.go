//go:build mock_db && mock_auth

// Package client provides comprehensive tests for the client creation use case.
//
// The tests cover various scenarios, including success, transaction handling,
// authorization, nil requests, validation errors, data enrichment, and boundary conditions.
// Each test function has a specific test code for tracking and logging.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user").
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education").
//
// Usage:
//   - Run all tests: go test -tags="mock_db,mock_auth" ./...
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestCreateClientUseCase_Execute_Success
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-ENTITY-CLIENT-CREATE-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-ENTITY-CLIENT-CREATE-VALIDATION-ERRORS-v1.0: ValidationErrors
//   - ESPYNA-TEST-ENTITY-CLIENT-CREATE-ENRICHMENT-v1.0: DataEnrichment
//
// Data Sources:
//   - Test cases: packages/copya/data_test/{businessType}/client.json
//   - Mock data: packages/copya/data/{businessType}/client.json
//   - Translations: packages/lyngua/translations/{languageCode}/{businessType}/client.json
package client

import (
	"testing"
	"time"

	copyatestutil "leapfor.xyz/copya/golang/testutil"
	"github.com/erniealice/espyna-golang/internal/application/shared/testutil"
	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock/entity"

	clientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client"
	userpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/user"
)

// createTestCreateClientUseCase is a helper function to create the use case with mock dependencies
func createTestCreateClientUseCase(businessType string) *CreateClientUseCase {
	repositories := CreateClientRepositories{
		Client: entity.NewMockClientRepository(businessType),
	}

	services := testutil.CreateStandardServices(false, true)
	createClientServices := CreateClientServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	return NewCreateClientUseCase(repositories, createClientServices)
}

func TestCreateClientUseCase_Execute_Success(t *testing.T) {
	testCode := "ESPYNA-TEST-ENTITY-CLIENT-CREATE-SUCCESS-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "Success", true)

	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestCreateClientUseCase(businessType)

	// Load test data resolvers
	createSuccessResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "client", "CreateClient_Success")
	testutil.AssertTestCaseLoad(t, err, "CreateClient_Success")

	req := &clientpb.CreateClientRequest{
		Data: &clientpb.Client{
			User: &userpb.User{
				FirstName:    createSuccessResolver.MustGetString("newClientFirstName"),
				LastName:     createSuccessResolver.MustGetString("newClientLastName"),
				EmailAddress: createSuccessResolver.MustGetString("newClientEmailAddress"),
			},
		},
	}

	res, err := useCase.Execute(ctx, req)
	testutil.AssertNoError(t, err)

	testutil.AssertNotNil(t, res, "response")

	testutil.AssertEqual(t, 1, len(res.Data), "number of clients in response data")

	createdClient := res.Data[0]
	testutil.AssertNonEmptyString(t, createdClient.Id, "client ID")
	testutil.AssertNonEmptyString(t, createdClient.InternalId, "client InternalId")
	testutil.AssertTrue(t, createdClient.Active, "client active status")

	// Log test completion with result
	testutil.LogTestResult(t, testCode, "Success", true, nil)
}

func TestCreateClientUseCase_Execute_ValidationErrors(t *testing.T) {
	testCode := "ESPYNA-TEST-ENTITY-CLIENT-CREATE-VALIDATION-ERRORS-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "ValidationErrors", false)

	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestCreateClientUseCase(businessType)

	// Load test data resolvers
	missingFirstNameResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "client", "ValidationError_MissingFirstName")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_MissingFirstName")

	invalidEmailResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "client", "ValidationError_InvalidEmail")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_InvalidEmail")

	fullNameTooShortResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "client", "ValidationError_FullNameTooShort")
	testutil.AssertTestCaseLoad(t, err, "ValidationError_FullNameTooShort")

	testCases := []struct {
		name             string
		client           *clientpb.Client
		expectedErrorKey string
	}{
		{
			name:             "Nil Data",
			client:           nil,
			expectedErrorKey: "client.validation.data_required",
		},
		{
			name:             "Nil User",
			client:           &clientpb.Client{},
			expectedErrorKey: "client.validation.user_data_required",
		},
		{
			name: "Missing First Name",
			client: &clientpb.Client{
				User: &userpb.User{LastName: missingFirstNameResolver.MustGetString("validLastName"), EmailAddress: missingFirstNameResolver.MustGetString("validEmailAddress")},
			},
			expectedErrorKey: "client.validation.first_name_required",
		},
		{
			name: "Invalid Email",
			client: &clientpb.Client{
				User: &userpb.User{FirstName: invalidEmailResolver.MustGetString("validFirstName"), LastName: invalidEmailResolver.MustGetString("validLastName"), EmailAddress: invalidEmailResolver.MustGetString("invalidEmailAddress")},
			},
			expectedErrorKey: "client.validation.email_invalid",
		},
		{
			name: "Full name too short",
			client: &clientpb.Client{
				User: &userpb.User{FirstName: fullNameTooShortResolver.MustGetString("shortFirstName"), LastName: fullNameTooShortResolver.MustGetString("shortLastName"), EmailAddress: fullNameTooShortResolver.MustGetString("validEmailAddress")},
			},
			expectedErrorKey: "client.validation.full_name_too_short",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := &clientpb.CreateClientRequest{Data: tc.client}
			_, err := useCase.Execute(ctx, req)
			testutil.AssertError(t, err)

			testutil.AssertTranslatedError(t, err, tc.expectedErrorKey, useCase.services.TranslationService, ctx)
		})
	}

	// Log test completion with result
	testutil.LogTestResult(t, testCode, "ValidationErrors", true, nil)
}

func TestCreateClientUseCase_DataEnrichment(t *testing.T) {
	testCode := "ESPYNA-TEST-ENTITY-CLIENT-CREATE-ENRICHMENT-v1.0"
	testutil.SetTestCode(t, testCode)
	testutil.LogTestExecution(t, testCode, "DataEnrichment", true)

	ctx := testutil.CreateTestContext()
	businessType := testutil.GetTestBusinessType()
	useCase := createTestCreateClientUseCase(businessType)

	// Load test data resolvers
	dataEnrichmentResolver, err := copyatestutil.LoadTestCaseFromBusinessType(t, businessType, "client", "DataEnrichment_TestClient")
	testutil.AssertTestCaseLoad(t, err, "DataEnrichment_TestClient")

	req := &clientpb.CreateClientRequest{
		Data: &clientpb.Client{
			User: &userpb.User{
				FirstName:    dataEnrichmentResolver.MustGetString("enrichFirstName"),
				LastName:     dataEnrichmentResolver.MustGetString("enrichLastName"),
				EmailAddress: dataEnrichmentResolver.MustGetString("enrichEmailAddress"),
			},
		},
	}

	res, err := useCase.Execute(ctx, req)
	testutil.AssertNoError(t, err)

	createdClient := res.Data[0]

	// Test core enrichment fields that should always be present
	testutil.AssertNonEmptyString(t, createdClient.Id, "ID")
	testutil.AssertNonEmptyString(t, createdClient.InternalId, "InternalId")
	testutil.AssertTrue(t, createdClient.Active, "Active status")
	testutil.AssertNotNil(t, createdClient.DateCreated, "DateCreated")
	testutil.AssertNotNil(t, createdClient.DateCreatedString, "DateCreatedString")
	testutil.AssertNotNil(t, createdClient.DateModified, "DateModified")
	testutil.AssertNotNil(t, createdClient.DateModifiedString, "DateModifiedString")

	// Verify timestamps are reasonable (within last 5 seconds)
	now := time.Now().UnixMilli()
	testutil.AssertTrue(t, *createdClient.DateCreated >= now-5000 && *createdClient.DateCreated <= now+5000, "DateCreated should be recent")

	// Log test completion with result
	testutil.LogTestResult(t, testCode, "DataEnrichment", true, nil)
}
