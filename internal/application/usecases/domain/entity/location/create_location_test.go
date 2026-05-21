//go:build mock_db && mock_auth

// Package location provides test cases for location creation use case.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user")
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education")
//
// Usage: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Methods:
//   - TestCreateLocationUseCase_Execute_Success: ESPYNA-TEST-ENTITY-LOCATION-SUCCESS-v1.0 Basic successful location creation
//   - TestCreateLocationUseCase_Execute_ValidationErrors: ESPYNA-TEST-ENTITY-LOCATION-VALIDATION-v1.0 Comprehensive validation error scenarios
//   - TestCreateLocationUseCase_DataEnrichment: ESPYNA-TEST-ENTITY-LOCATION-ENRICHMENT-v1.0 Auto-generated fields verification
package location

import (
	"context"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/shared/testutil"
	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock/entity"
	locationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/location"
)

// Type alias for create location test cases
type CreateLocationTestCase = testutil.GenericTestCase[*locationpb.CreateLocationRequest, *locationpb.CreateLocationResponse]

func createTestUseCaseWithAuth(businessType string, supportsTransaction bool, shouldAuthorize bool) *CreateLocationUseCase {
	repositories := CreateLocationRepositories{
		Location: entity.NewMockLocationRepository(businessType),
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, shouldAuthorize)
	services := CreateLocationServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
		IDService:            standardServices.IDService,
	}

	return NewCreateLocationUseCase(repositories, services)
}

func TestCreateLocationUseCase_Execute_TableDriven(t *testing.T) {
	testCases := []CreateLocationTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-ENTITY-LOCATION-CREATE-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *locationpb.CreateLocationRequest {
				return &locationpb.CreateLocationRequest{
					Data: &locationpb.Location{
						Name:    "Main Campus",
						Address: "123 University Ave",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *locationpb.CreateLocationResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertEqual(t, 1, len(response.Data), "response data length")
				createdLocation := response.Data[0]
				testutil.AssertStringEqual(t, "Main Campus", createdLocation.Name, "location name")
				testutil.AssertStringEqual(t, "123 University Ave", createdLocation.Address, "location address")
				testutil.AssertNonEmptyString(t, createdLocation.Id, "location ID")
				testutil.AssertTrue(t, createdLocation.Active, "location active")
				testutil.AssertFieldSet(t, createdLocation.DateCreated, "DateCreated")
			},
		},
		{
			Name:     "WithTransaction",
			TestCode: "ESPYNA-TEST-ENTITY-LOCATION-CREATE-TRANSACTION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *locationpb.CreateLocationRequest {
				return &locationpb.CreateLocationRequest{
					Data: &locationpb.Location{
						Name:    "Transaction Campus",
						Address: "456 Transaction St",
					},
				}
			},
			UseTransaction: true,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *locationpb.CreateLocationResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				createdLocation := response.Data[0]
				testutil.AssertStringEqual(t, "Transaction Campus", createdLocation.Name, "location name")
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-ENTITY-LOCATION-CREATE-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *locationpb.CreateLocationRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "location.validation.request_required",
			Assertions: func(t *testing.T, response *locationpb.CreateLocationResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilRequest(t, err)
			},
		},
		{
			Name:     "NilData",
			TestCode: "ESPYNA-TEST-ENTITY-LOCATION-CREATE-NIL-DATA-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *locationpb.CreateLocationRequest {
				return &locationpb.CreateLocationRequest{Data: nil}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "location.validation.data_required",
			Assertions: func(t *testing.T, response *locationpb.CreateLocationResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilData(t, err)
			},
		},
		{
			Name:     "EmptyName",
			TestCode: "ESPYNA-TEST-ENTITY-LOCATION-CREATE-VALIDATION-EMPTY-NAME-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *locationpb.CreateLocationRequest {
				return &locationpb.CreateLocationRequest{
					Data: &locationpb.Location{
						Name:    "",
						Address: "123 Street",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "location.validation.name_required",
			Assertions: func(t *testing.T, response *locationpb.CreateLocationResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty name")
			},
		},
		{
			Name:     "EmptyAddress",
			TestCode: "ESPYNA-TEST-ENTITY-LOCATION-CREATE-VALIDATION-EMPTY-ADDRESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *locationpb.CreateLocationRequest {
				return &locationpb.CreateLocationRequest{
					Data: &locationpb.Location{
						Name:    "Campus",
						Address: "",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "location.validation.address_required",
			Assertions: func(t *testing.T, response *locationpb.CreateLocationResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty address")
			},
		},
		{
			Name:     "NameTooShort",
			TestCode: "ESPYNA-TEST-ENTITY-LOCATION-CREATE-VALIDATION-NAME-TOO-SHORT-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *locationpb.CreateLocationRequest {
				return &locationpb.CreateLocationRequest{
					Data: &locationpb.Location{
						Name:    "A",
						Address: "123 Street",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "location.validation.name_too_short",
			Assertions: func(t *testing.T, response *locationpb.CreateLocationResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "name too short")
			},
		},
		{
			Name:     "AddressTooShort",
			TestCode: "ESPYNA-TEST-ENTITY-LOCATION-CREATE-VALIDATION-ADDRESS-TOO-SHORT-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *locationpb.CreateLocationRequest {
				return &locationpb.CreateLocationRequest{
					Data: &locationpb.Location{
						Name:    "Campus",
						Address: "123",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "location.validation.address_too_short",
			Assertions: func(t *testing.T, response *locationpb.CreateLocationResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "address too short")
			},
		},
		{
			Name:     "DataEnrichment",
			TestCode: "ESPYNA-TEST-ENTITY-LOCATION-CREATE-ENRICHMENT-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *locationpb.CreateLocationRequest {
				return &locationpb.CreateLocationRequest{
					Data: &locationpb.Location{
						Name:    "Enriched Campus",
						Address: "456 Enrichment Way",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *locationpb.CreateLocationResponse, err error, useCase interface{}, ctx context.Context) {
				createdLocation := response.Data[0]
				testutil.AssertNonEmptyString(t, createdLocation.Id, "generated ID")
				testutil.AssertTrue(t, createdLocation.Active, "Active")
				testutil.AssertFieldSet(t, createdLocation.DateCreated, "DateCreated")
				testutil.AssertFieldSet(t, createdLocation.DateCreatedString, "DateCreatedString")
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
