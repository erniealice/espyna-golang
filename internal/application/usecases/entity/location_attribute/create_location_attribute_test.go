//go:build mock_db && mock_auth

// Package location_attribute provides test cases for location_attribute creation use case.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user")
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education")
//
// Usage: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Methods:
//   - TestCreateLocationAttributeUseCase_Execute_Success: ESPYNA-TEST-ENTITY-LOCATIONATTRIBUTE-SUCCESS-v1.0 Basic successful location_attribute creation
//   - TestCreateLocationAttributeUseCase_Execute_ValidationErrors: ESPYNA-TEST-ENTITY-LOCATIONATTRIBUTE-VALIDATION-v1.0 Comprehensive validation error scenarios
//   - TestCreateLocationAttributeUseCase_Execute_EntityReferenceErrors: ESPYNA-TEST-ENTITY-LOCATIONATTRIBUTE-VALIDATION-v1.0 Entity reference validation errors
package location_attribute

import (
	"context"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/shared/testutil"
	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock/common"
	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock/entity"
	locationattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/location_attribute"
)

// Type alias for create location attribute test cases
type CreateLocationAttributeTestCase = testutil.GenericTestCase[*locationattributepb.CreateLocationAttributeRequest, *locationattributepb.CreateLocationAttributeResponse]

func createTestUseCaseWithAuth(businessType string, supportsTransaction bool, shouldAuthorize bool) *CreateLocationAttributeUseCase {
	repositories := CreateLocationAttributeRepositories{
		LocationAttribute: entity.NewMockLocationAttributeRepository(businessType),
		Location:          entity.NewMockLocationRepository(businessType),
		Attribute:         common.NewMockAttributeRepository(businessType),
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, shouldAuthorize)
	services := CreateLocationAttributeServices{
		TransactionService: standardServices.TransactionService,
		TranslationService: standardServices.TranslationService,
		IDService:          standardServices.IDService,
	}

	return NewCreateLocationAttributeUseCase(repositories, services)
}

func TestCreateLocationAttributeUseCase_Execute_TableDriven(t *testing.T) {
	// IDs from packages/copya/data/education/
	existingLocationID := "location-main-building"
	existingAttributeID := "attr_001"

	testCases := []CreateLocationAttributeTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-ENTITY-LOCATION-ATTRIBUTE-CREATE-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *locationattributepb.CreateLocationAttributeRequest {
				return &locationattributepb.CreateLocationAttributeRequest{
					Data: &locationattributepb.LocationAttribute{
						LocationId:  existingLocationID,
						AttributeId: existingAttributeID,
						Value:       "Room 101",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *locationattributepb.CreateLocationAttributeResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertEqual(t, 1, len(response.Data), "response data length")
				createdAttr := response.Data[0]
				testutil.AssertStringEqual(t, "Room 101", createdAttr.Value, "attribute value")
				testutil.AssertStringEqual(t, existingLocationID, createdAttr.LocationId, "location ID")
				testutil.AssertStringEqual(t, existingAttributeID, createdAttr.AttributeId, "attribute ID")
				testutil.AssertNonEmptyString(t, createdAttr.Id, "location attribute ID")
				testutil.AssertFieldSet(t, createdAttr.DateCreated, "DateCreated")
			},
		},
		{
			Name:     "WithTransaction",
			TestCode: "ESPYNA-TEST-ENTITY-LOCATION-ATTRIBUTE-CREATE-TRANSACTION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *locationattributepb.CreateLocationAttributeRequest {
				return &locationattributepb.CreateLocationAttributeRequest{
					Data: &locationattributepb.LocationAttribute{
						LocationId:  existingLocationID,
						AttributeId: existingAttributeID,
						Value:       "Room 102",
					},
				}
			},
			UseTransaction: true,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *locationattributepb.CreateLocationAttributeResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				createdAttr := response.Data[0]
				testutil.AssertStringEqual(t, "Room 102", createdAttr.Value, "attribute value")
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-ENTITY-LOCATION-ATTRIBUTE-CREATE-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *locationattributepb.CreateLocationAttributeRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "location_attribute.validation.request_required",
			Assertions: func(t *testing.T, response *locationattributepb.CreateLocationAttributeResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilRequest(t, err)
			},
		},
		{
			Name:     "NilData",
			TestCode: "ESPYNA-TEST-ENTITY-LOCATION-ATTRIBUTE-CREATE-NIL-DATA-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *locationattributepb.CreateLocationAttributeRequest {
				return &locationattributepb.CreateLocationAttributeRequest{Data: nil}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "location_attribute.validation.data_required",
			Assertions: func(t *testing.T, response *locationattributepb.CreateLocationAttributeResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilData(t, err)
			},
		},
		{
			Name:     "EmptyLocationId",
			TestCode: "ESPYNA-TEST-ENTITY-LOCATION-ATTRIBUTE-CREATE-VALIDATION-EMPTY-LOCATION-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *locationattributepb.CreateLocationAttributeRequest {
				return &locationattributepb.CreateLocationAttributeRequest{
					Data: &locationattributepb.LocationAttribute{
						LocationId:  "",
						AttributeId: existingAttributeID,
						Value:       "test",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "location_attribute.validation.location_id_required",
			Assertions: func(t *testing.T, response *locationattributepb.CreateLocationAttributeResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty location ID")
			},
		},
		{
			Name:     "EmptyAttributeId",
			TestCode: "ESPYNA-TEST-ENTITY-LOCATION-ATTRIBUTE-CREATE-VALIDATION-EMPTY-ATTRIBUTE-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *locationattributepb.CreateLocationAttributeRequest {
				return &locationattributepb.CreateLocationAttributeRequest{
					Data: &locationattributepb.LocationAttribute{
						LocationId:  existingLocationID,
						AttributeId: "",
						Value:       "test",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "location_attribute.validation.attribute_id_required",
			Assertions: func(t *testing.T, response *locationattributepb.CreateLocationAttributeResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty attribute ID")
			},
		},
		{
			Name:     "EmptyValue",
			TestCode: "ESPYNA-TEST-ENTITY-LOCATION-ATTRIBUTE-CREATE-VALIDATION-EMPTY-VALUE-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *locationattributepb.CreateLocationAttributeRequest {
				return &locationattributepb.CreateLocationAttributeRequest{
					Data: &locationattributepb.LocationAttribute{
						LocationId:  existingLocationID,
						AttributeId: existingAttributeID,
						Value:       "",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "location_attribute.validation.value_required",
			Assertions: func(t *testing.T, response *locationattributepb.CreateLocationAttributeResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty value")
			},
		},
		{
			Name:     "InvalidLocationId",
			TestCode: "ESPYNA-TEST-ENTITY-LOCATION-ATTRIBUTE-CREATE-VALIDATION-INVALID-LOCATION-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *locationattributepb.CreateLocationAttributeRequest {
				return &locationattributepb.CreateLocationAttributeRequest{
					Data: &locationattributepb.LocationAttribute{
						LocationId:  "location-999",
						AttributeId: existingAttributeID,
						Value:       "test",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "location_attribute.errors.location_not_found",
			ErrorTags:      map[string]any{"locationId": "location-999"},
			Assertions: func(t *testing.T, response *locationattributepb.CreateLocationAttributeResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "invalid location ID")
			},
		},
		{
			Name:     "InvalidAttributeId",
			TestCode: "ESPYNA-TEST-ENTITY-LOCATION-ATTRIBUTE-CREATE-VALIDATION-INVALID-ATTRIBUTE-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *locationattributepb.CreateLocationAttributeRequest {
				return &locationattributepb.CreateLocationAttributeRequest{
					Data: &locationattributepb.LocationAttribute{
						LocationId:  existingLocationID,
						AttributeId: "attribute-999",
						Value:       "test",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "location_attribute.errors.attribute_not_found",
			ErrorTags:      map[string]any{"attributeId": "attribute-999"},
			Assertions: func(t *testing.T, response *locationattributepb.CreateLocationAttributeResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "invalid attribute ID")
			},
		},
		{
			Name:     "DataEnrichment",
			TestCode: "ESPYNA-TEST-ENTITY-LOCATION-ATTRIBUTE-CREATE-ENRICHMENT-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *locationattributepb.CreateLocationAttributeRequest {
				return &locationattributepb.CreateLocationAttributeRequest{
					Data: &locationattributepb.LocationAttribute{
						LocationId:  existingLocationID,
						AttributeId: existingAttributeID,
						Value:       "Enriched Room",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *locationattributepb.CreateLocationAttributeResponse, err error, useCase interface{}, ctx context.Context) {
				createdAttr := response.Data[0]
				testutil.AssertNonEmptyString(t, createdAttr.Id, "generated ID")
				testutil.AssertFieldSet(t, createdAttr.DateCreated, "DateCreated")
				testutil.AssertFieldSet(t, createdAttr.DateCreatedString, "DateCreatedString")
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
