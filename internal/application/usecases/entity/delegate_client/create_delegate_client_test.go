//go:build mock_db && mock_auth

// Package delegate_client provides test cases for delegate client creation use case.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user")
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education")
//
// Usage: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Methods:
//   - TestCreateDelegateClientUseCase_Execute_Success: ESPYNA-TEST-ENTITY-DELEGATECLIENT-SUCCESS-v1.0 Basic successful delegate client creation
//   - TestCreateDelegateClientUseCase_Execute_ValidationErrors: ESPYNA-TEST-ENTITY-DELEGATECLIENT-VALIDATION-v1.0 Comprehensive validation error scenarios
//   - TestCreateDelegateClientUseCase_Execute_EntityReferenceErrors: ESPYNA-TEST-ENTITY-DELEGATECLIENT-VALIDATION-v1.0 Entity reference validation tests
package delegate_client

import (
	"context"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/shared/testutil"
	mockAuth "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/auth/mock"
	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock/entity"

	delegateclientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/delegate_client"
)

// Type alias for create delegate client test cases
type CreateDelegateClientTestCase = testutil.GenericTestCase[*delegateclientpb.CreateDelegateClientRequest, *delegateclientpb.CreateDelegateClientResponse]

func createTestUseCaseWithAuth(businessType string, supportsTransaction bool, shouldAuthorize bool) *CreateDelegateClientUseCase {
	repositories := CreateDelegateClientRepositories{
		DelegateClient: entity.NewMockDelegateClientRepository(businessType),
		Delegate:       entity.NewMockDelegateRepository(businessType),
		Client:         entity.NewMockClientRepository(businessType),
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, shouldAuthorize)
	services := CreateDelegateClientServices{
		AuthorizationService: mockAuth.NewDisabledAuth(), // Use disabled auth to match other modules
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
		IDService:            standardServices.IDService,
	}

	return NewCreateDelegateClientUseCase(repositories, services)
}

func TestCreateDelegateClientUseCase_Execute_TableDriven(t *testing.T) {
	// IDs from packages/copya/data/education/
	existingDelegateID := "parent-001"
	existingClientID := "student-001"

	testCases := []CreateDelegateClientTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-ENTITY-DELEGATE-CLIENT-CREATE-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *delegateclientpb.CreateDelegateClientRequest {
				return &delegateclientpb.CreateDelegateClientRequest{
					Data: &delegateclientpb.DelegateClient{
						DelegateId: existingDelegateID,
						ClientId:   existingClientID,
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *delegateclientpb.CreateDelegateClientResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertEqual(t, 1, len(response.Data), "response data length")
				createdRel := response.Data[0]
				testutil.AssertStringEqual(t, existingDelegateID, createdRel.DelegateId, "delegate ID")
				testutil.AssertStringEqual(t, existingClientID, createdRel.ClientId, "client ID")
				testutil.AssertNonEmptyString(t, createdRel.Id, "delegate client ID")
				testutil.AssertFieldSet(t, createdRel.DateCreated, "DateCreated")
			},
		},
		{
			Name:     "WithTransaction",
			TestCode: "ESPYNA-TEST-ENTITY-DELEGATE-CLIENT-CREATE-TRANSACTION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *delegateclientpb.CreateDelegateClientRequest {
				return &delegateclientpb.CreateDelegateClientRequest{
					Data: &delegateclientpb.DelegateClient{
						DelegateId: existingDelegateID,
						ClientId:   existingClientID,
					},
				}
			},
			UseTransaction: true,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *delegateclientpb.CreateDelegateClientResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				createdRel := response.Data[0]
				testutil.AssertStringEqual(t, existingDelegateID, createdRel.DelegateId, "delegate ID")
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-ENTITY-DELEGATE-CLIENT-CREATE-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *delegateclientpb.CreateDelegateClientRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "delegate_client.validation.request_required",
			Assertions: func(t *testing.T, response *delegateclientpb.CreateDelegateClientResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilRequest(t, err)
			},
		},
		{
			Name:     "NilData",
			TestCode: "ESPYNA-TEST-ENTITY-DELEGATE-CLIENT-CREATE-NIL-DATA-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *delegateclientpb.CreateDelegateClientRequest {
				return &delegateclientpb.CreateDelegateClientRequest{Data: nil}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "delegate_client.validation.data_required",
			Assertions: func(t *testing.T, response *delegateclientpb.CreateDelegateClientResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilData(t, err)
			},
		},
		{
			Name:     "EmptyDelegateId",
			TestCode: "ESPYNA-TEST-ENTITY-DELEGATE-CLIENT-CREATE-VALIDATION-EMPTY-DELEGATE-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *delegateclientpb.CreateDelegateClientRequest {
				return &delegateclientpb.CreateDelegateClientRequest{
					Data: &delegateclientpb.DelegateClient{
						DelegateId: "",
						ClientId:   existingClientID,
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "delegate_client.validation.delegate_id_required",
			Assertions: func(t *testing.T, response *delegateclientpb.CreateDelegateClientResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty delegate ID")
			},
		},
		{
			Name:     "EmptyClientId",
			TestCode: "ESPYNA-TEST-ENTITY-DELEGATE-CLIENT-CREATE-VALIDATION-EMPTY-CLIENT-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *delegateclientpb.CreateDelegateClientRequest {
				return &delegateclientpb.CreateDelegateClientRequest{
					Data: &delegateclientpb.DelegateClient{
						DelegateId: existingDelegateID,
						ClientId:   "",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "delegate_client.validation.client_id_required",
			Assertions: func(t *testing.T, response *delegateclientpb.CreateDelegateClientResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty client ID")
			},
		},
		{
			Name:     "SameIds",
			TestCode: "ESPYNA-TEST-ENTITY-DELEGATE-CLIENT-CREATE-VALIDATION-SAME-IDS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *delegateclientpb.CreateDelegateClientRequest {
				return &delegateclientpb.CreateDelegateClientRequest{
					Data: &delegateclientpb.DelegateClient{
						DelegateId: "same-id",
						ClientId:   "same-id",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "delegate_client.validation.same_ids_not_allowed",
			Assertions: func(t *testing.T, response *delegateclientpb.CreateDelegateClientResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "same IDs not allowed")
			},
		},
		{
			Name:     "InvalidDelegateId",
			TestCode: "ESPYNA-TEST-ENTITY-DELEGATE-CLIENT-CREATE-VALIDATION-INVALID-DELEGATE-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *delegateclientpb.CreateDelegateClientRequest {
				return &delegateclientpb.CreateDelegateClientRequest{
					Data: &delegateclientpb.DelegateClient{
						DelegateId: "parent-999",
						ClientId:   existingClientID,
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "delegate_client.errors.delegate_not_found",
			ErrorTags:      map[string]any{"delegateId": "parent-999"},
			Assertions: func(t *testing.T, response *delegateclientpb.CreateDelegateClientResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "invalid delegate ID")
			},
		},
		{
			Name:     "InvalidClientId",
			TestCode: "ESPYNA-TEST-ENTITY-DELEGATE-CLIENT-CREATE-VALIDATION-INVALID-CLIENT-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *delegateclientpb.CreateDelegateClientRequest {
				return &delegateclientpb.CreateDelegateClientRequest{
					Data: &delegateclientpb.DelegateClient{
						DelegateId: existingDelegateID,
						ClientId:   "student-999",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "delegate_client.errors.client_not_found",
			ErrorTags:      map[string]any{"clientId": "student-999"},
			Assertions: func(t *testing.T, response *delegateclientpb.CreateDelegateClientResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "invalid client ID")
			},
		},
		{
			Name:     "DataEnrichment",
			TestCode: "ESPYNA-TEST-ENTITY-DELEGATE-CLIENT-CREATE-ENRICHMENT-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *delegateclientpb.CreateDelegateClientRequest {
				return &delegateclientpb.CreateDelegateClientRequest{
					Data: &delegateclientpb.DelegateClient{
						DelegateId: existingDelegateID,
						ClientId:   existingClientID,
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *delegateclientpb.CreateDelegateClientResponse, err error, useCase interface{}, ctx context.Context) {
				createdRel := response.Data[0]
				testutil.AssertNonEmptyString(t, createdRel.Id, "generated ID")
				testutil.AssertFieldSet(t, createdRel.DateCreated, "DateCreated")
				testutil.AssertFieldSet(t, createdRel.DateCreatedString, "DateCreatedString")
				testutil.AssertTrue(t, createdRel.Active, "Active")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			// Set test code and log execution start
			testutil.SetTestCode(t, tc.TestCode)
			testutil.LogTestExecution(t, tc.TestCode, tc.Name, tc.ExpectSuccess)

			ctx := testutil.CreateTestContext()
			businessType := testutil.GetTestBusinessType()
			useCase := createTestUseCaseWithAuth(businessType, tc.UseTransaction, tc.UseAuth)

			req := tc.SetupRequest(t, businessType)
			response, err := useCase.Execute(ctx, req)

			// Determine actual success/failure
			actualSuccess := err == nil && tc.ExpectSuccess

			if tc.ExpectSuccess {
				testutil.AssertNoError(t, err)
				testutil.AssertNotNil(t, response, "response")
			} else {
				testutil.AssertError(t, err)
				if tc.ExpectedError != "" {
					if tc.ErrorTags != nil {
						testutil.AssertTranslatedErrorWithTags(t, err, tc.ExpectedError, tc.ErrorTags, useCase.services.TranslationService, ctx)
					} else {
						testutil.AssertTranslatedError(t, err, tc.ExpectedError, useCase.services.TranslationService, ctx)
					}
				}
			}

			if tc.Assertions != nil {
				tc.Assertions(t, response, err, useCase, ctx)
			}

			// Log test completion with result
			testutil.LogTestResult(t, tc.TestCode, tc.Name, actualSuccess, err)
		})
	}
}
