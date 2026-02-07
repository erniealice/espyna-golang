//go:build mock_db && mock_auth

// Package client_attribute provides test cases for client attribute creation use case.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user")
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education")
//
// Usage: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Methods:
//   - TestCreateClientAttributeUseCase_Execute_Success: ESPYNA-TEST-ENTITY-CLIENTATTRIBUTE-SUCCESS-v1.0 Basic successful client attribute creation
//   - TestCreateClientAttributeUseCase_Execute_ValidationErrors: ESPYNA-TEST-ENTITY-CLIENTATTRIBUTE-VALIDATION-v1.0 Comprehensive validation error scenarios
//   - TestCreateClientAttributeUseCase_Execute_EntityReferenceErrors: ESPYNA-TEST-ENTITY-CLIENTATTRIBUTE-VALIDATION-v1.0 Entity reference validation tests
package client_attribute

import (
	"context"
	"testing"

	"leapfor.xyz/espyna/internal/application/shared/testutil"
	"leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock/common"
	"leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock/entity"
	clientattributepb "leapfor.xyz/esqyma/golang/v1/domain/entity/client_attribute"
)

// Type alias for create client attribute test cases
type CreateClientAttributeTestCase = testutil.GenericTestCase[*clientattributepb.CreateClientAttributeRequest, *clientattributepb.CreateClientAttributeResponse]

func createTestUseCaseWithAuth(businessType string, supportsTransaction bool, shouldAuthorize bool) *CreateClientAttributeUseCase {
	repositories := CreateClientAttributeRepositories{
		ClientAttribute: entity.NewMockClientAttributeRepository(businessType),
		Client:          entity.NewMockClientRepository(businessType),
		Attribute:       common.NewMockAttributeRepository(businessType),
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, shouldAuthorize)
	services := CreateClientAttributeServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
		IDService:            standardServices.IDService,
	}

	return NewCreateClientAttributeUseCase(repositories, services)
}

func TestCreateClientAttributeUseCase_Execute_TableDriven(t *testing.T) {
	// IDs from packages/copya/data/education/
	existingClientID := "student-001"
	existingAttributeID := "attr_001"

	testCases := []CreateClientAttributeTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-ENTITY-CLIENT-ATTRIBUTE-CREATE-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *clientattributepb.CreateClientAttributeRequest {
				return &clientattributepb.CreateClientAttributeRequest{
					Data: &clientattributepb.ClientAttribute{
						ClientId:    existingClientID,
						AttributeId: existingAttributeID,
						Value:       "Test Value",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *clientattributepb.CreateClientAttributeResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertEqual(t, 1, len(response.Data), "response data length")
				createdAttr := response.Data[0]
				testutil.AssertStringEqual(t, "Test Value", createdAttr.Value, "attribute value")
				testutil.AssertStringEqual(t, existingClientID, createdAttr.ClientId, "client ID")
				testutil.AssertStringEqual(t, existingAttributeID, createdAttr.AttributeId, "attribute ID")
				testutil.AssertNonEmptyString(t, createdAttr.Id, "client attribute ID")
				testutil.AssertFieldSet(t, createdAttr.DateCreated, "DateCreated")
			},
		},
		{
			Name:     "WithTransaction",
			TestCode: "ESPYNA-TEST-ENTITY-CLIENT-ATTRIBUTE-CREATE-TRANSACTION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *clientattributepb.CreateClientAttributeRequest {
				return &clientattributepb.CreateClientAttributeRequest{
					Data: &clientattributepb.ClientAttribute{
						ClientId:    existingClientID,
						AttributeId: existingAttributeID,
						Value:       "Transaction Value",
					},
				}
			},
			UseTransaction: true,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *clientattributepb.CreateClientAttributeResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				createdAttr := response.Data[0]
				testutil.AssertStringEqual(t, "Transaction Value", createdAttr.Value, "attribute value")
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-ENTITY-CLIENT-ATTRIBUTE-CREATE-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *clientattributepb.CreateClientAttributeRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "client_attribute.validation.request_required",
			Assertions: func(t *testing.T, response *clientattributepb.CreateClientAttributeResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilRequest(t, err)
			},
		},
		{
			Name:     "NilData",
			TestCode: "ESPYNA-TEST-ENTITY-CLIENT-ATTRIBUTE-CREATE-NIL-DATA-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *clientattributepb.CreateClientAttributeRequest {
				return &clientattributepb.CreateClientAttributeRequest{Data: nil}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "client_attribute.validation.data_required",
			Assertions: func(t *testing.T, response *clientattributepb.CreateClientAttributeResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilData(t, err)
			},
		},
		{
			Name:     "EmptyClientId",
			TestCode: "ESPYNA-TEST-ENTITY-CLIENT-ATTRIBUTE-CREATE-VALIDATION-EMPTY-CLIENT-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *clientattributepb.CreateClientAttributeRequest {
				return &clientattributepb.CreateClientAttributeRequest{
					Data: &clientattributepb.ClientAttribute{
						ClientId:    "",
						AttributeId: existingAttributeID,
						Value:       "test",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "client_attribute.validation.client_id_required",
			Assertions: func(t *testing.T, response *clientattributepb.CreateClientAttributeResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty client ID")
			},
		},
		{
			Name:     "EmptyAttributeId",
			TestCode: "ESPYNA-TEST-ENTITY-CLIENT-ATTRIBUTE-CREATE-VALIDATION-EMPTY-ATTRIBUTE-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *clientattributepb.CreateClientAttributeRequest {
				return &clientattributepb.CreateClientAttributeRequest{
					Data: &clientattributepb.ClientAttribute{
						ClientId:    existingClientID,
						AttributeId: "",
						Value:       "test",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "client_attribute.validation.attribute_id_required",
			Assertions: func(t *testing.T, response *clientattributepb.CreateClientAttributeResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty attribute ID")
			},
		},
		{
			Name:     "EmptyValue",
			TestCode: "ESPYNA-TEST-ENTITY-CLIENT-ATTRIBUTE-CREATE-VALIDATION-EMPTY-VALUE-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *clientattributepb.CreateClientAttributeRequest {
				return &clientattributepb.CreateClientAttributeRequest{
					Data: &clientattributepb.ClientAttribute{
						ClientId:    existingClientID,
						AttributeId: existingAttributeID,
						Value:       "",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "client_attribute.validation.value_required",
			Assertions: func(t *testing.T, response *clientattributepb.CreateClientAttributeResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty value")
			},
		},
		{
			Name:     "InvalidClientId",
			TestCode: "ESPYNA-TEST-ENTITY-CLIENT-ATTRIBUTE-CREATE-VALIDATION-INVALID-CLIENT-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *clientattributepb.CreateClientAttributeRequest {
				return &clientattributepb.CreateClientAttributeRequest{
					Data: &clientattributepb.ClientAttribute{
						ClientId:    "student-999",
						AttributeId: existingAttributeID,
						Value:       "test",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "client_attribute.errors.referenced_client_not_found",
			ErrorTags:      map[string]any{"clientId": "student-999"},
			Assertions: func(t *testing.T, response *clientattributepb.CreateClientAttributeResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "invalid client ID")
			},
		},
		{
			Name:     "InvalidAttributeId",
			TestCode: "ESPYNA-TEST-ENTITY-CLIENT-ATTRIBUTE-CREATE-VALIDATION-INVALID-ATTRIBUTE-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *clientattributepb.CreateClientAttributeRequest {
				return &clientattributepb.CreateClientAttributeRequest{
					Data: &clientattributepb.ClientAttribute{
						ClientId:    existingClientID,
						AttributeId: "attribute-999",
						Value:       "test",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "client_attribute.errors.referenced_attribute_not_found",
			ErrorTags:      map[string]any{"attributeId": "attribute-999"},
			Assertions: func(t *testing.T, response *clientattributepb.CreateClientAttributeResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "invalid attribute ID")
			},
		},
		{
			Name:     "DataEnrichment",
			TestCode: "ESPYNA-TEST-ENTITY-CLIENT-ATTRIBUTE-CREATE-ENRICHMENT-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *clientattributepb.CreateClientAttributeRequest {
				return &clientattributepb.CreateClientAttributeRequest{
					Data: &clientattributepb.ClientAttribute{
						ClientId:    existingClientID,
						AttributeId: existingAttributeID,
						Value:       "Enriched Value",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *clientattributepb.CreateClientAttributeResponse, err error, useCase interface{}, ctx context.Context) {
				createdAttr := response.Data[0]
				testutil.AssertNonEmptyString(t, createdAttr.Id, "generated ID")
				testutil.AssertFieldSet(t, createdAttr.DateCreated, "DateCreated")
				testutil.AssertFieldSet(t, createdAttr.DateCreatedString, "DateCreatedString")
				testutil.AssertTrue(t, createdAttr.Active, "Active")
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
