//go:build mock_db && mock_auth && google && uuidv7

// Package invoice provides table-driven tests for the invoice creation use case.
//
// The tests cover various scenarios focused on the Invoice protobuf structure which
// has only Amount as user input, with all other fields (Id, InvoiceNumber, Active,
// DateCreated, DateModified, etc.) being auto-generated.
//
// Test scenarios include:
// - Success with valid amount
// - Transaction handling
// - Authorization validation
// - Nil request/data validation
// - Amount validation (zero and negative amounts)
// - Data enrichment (auto-generation of fields)
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user").
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education").
//
// Usage:
//   - Run all tests: go test -tags="mock_db,mock_auth" ./...
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestCreateInvoiceUseCase_Execute_TableDriven
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-SUBSCRIPTION-INVOICE-CREATE-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-SUBSCRIPTION-INVOICE-CREATE-TRANSACTION-v1.0: WithTransaction
//   - ESPYNA-TEST-SUBSCRIPTION-INVOICE-CREATE-NIL-REQUEST-v1.0: NilRequest
//   - ESPYNA-TEST-SUBSCRIPTION-INVOICE-CREATE-NIL-DATA-v1.0: NilData
//   - ESPYNA-TEST-SUBSCRIPTION-INVOICE-CREATE-VALIDATION-ZERO-AMOUNT-v1.0: ZeroAmount
//   - ESPYNA-TEST-SUBSCRIPTION-INVOICE-CREATE-VALIDATION-NEGATIVE-AMOUNT-v1.0: NegativeAmount
//   - ESPYNA-TEST-SUBSCRIPTION-INVOICE-CREATE-ENRICHMENT-v1.0: DataEnrichment
//
// Data Sources:
//   - All test data is hardcoded with specific amounts for each test scenario
//   - No external JSON files required since Invoice only has Amount field as input

package invoice

import (
	"context"
	"testing"

	"leapfor.xyz/espyna/internal/application/shared/testutil"
	mockAuth "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/auth/mock"
	mockDb "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock"
	"leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock/subscription"
	"leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/id/uuidv7"
	"leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/translation"
	invoicepb "leapfor.xyz/esqyma/golang/v1/domain/subscription/invoice"
)

// Type alias for create invoice test cases
type CreateInvoiceTestCase = testutil.GenericTestCase[*invoicepb.CreateInvoiceRequest, *invoicepb.CreateInvoiceResponse]

// createTestCreateInvoiceUseCase is a helper function to create the use case with mock dependencies
func createTestCreateInvoiceUseCase(businessType string, supportsTransaction bool) *CreateInvoiceUseCase {
	mockRepo := subscription.NewMockInvoiceRepository(businessType)

	repositories := CreateInvoiceRepositories{
		Invoice: mockRepo,
	}

	services := CreateInvoiceServices{
		AuthorizationService: mockAuth.NewAllowAllAuth(),
		TransactionService:   mockDb.NewMockTransactionService(supportsTransaction),
		TranslationService:   translation.NewLynguaTranslationService(),
		IDService:            uuidv7.NewGoogleUUIDv7Service(),
	}

	return NewCreateInvoiceUseCase(repositories, services)
}

func createTestUseCaseWithAuth(businessType string, supportsTransaction bool, shouldAuthorize bool) *CreateInvoiceUseCase {
	mockRepo := subscription.NewMockInvoiceRepository(businessType)

	repositories := CreateInvoiceRepositories{
		Invoice: mockRepo,
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, shouldAuthorize)
	services := CreateInvoiceServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
		IDService:            standardServices.IDService,
	}

	return NewCreateInvoiceUseCase(repositories, services)
}

func TestCreateInvoiceUseCase_Execute_TableDriven(t *testing.T) {
	// No test data resolvers needed - using hardcoded amounts only

	testCases := []CreateInvoiceTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-INVOICE-CREATE-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *invoicepb.CreateInvoiceRequest {
				return &invoicepb.CreateInvoiceRequest{
					Data: &invoicepb.Invoice{
						Amount: 10000.0,
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *invoicepb.CreateInvoiceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertEqual(t, 1, len(response.Data), "response data length")
				createdInvoice := response.Data[0]
				testutil.AssertEqual(t, 10000.0, createdInvoice.Amount, "invoice amount")
				testutil.AssertNonEmptyString(t, createdInvoice.Id, "invoice ID")
				testutil.AssertNonEmptyString(t, createdInvoice.InvoiceNumber, "invoice number")
				testutil.AssertTrue(t, createdInvoice.Active, "invoice active status")
				testutil.AssertFieldSet(t, createdInvoice.DateCreated, "DateCreated")
				testutil.AssertFieldSet(t, createdInvoice.DateCreatedString, "DateCreatedString")
			},
		},
		{
			Name:     "WithTransaction",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-INVOICE-CREATE-TRANSACTION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *invoicepb.CreateInvoiceRequest {
				return &invoicepb.CreateInvoiceRequest{
					Data: &invoicepb.Invoice{
						Amount: 5000.0,
					},
				}
			},
			UseTransaction: true,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *invoicepb.CreateInvoiceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				createdInvoice := response.Data[0]
				testutil.AssertEqual(t, 5000.0, createdInvoice.Amount, "invoice amount")
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-INVOICE-CREATE-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *invoicepb.CreateInvoiceRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "invoice.validation.request_required",
			Assertions: func(t *testing.T, response *invoicepb.CreateInvoiceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilRequest(t, err)
			},
		},
		{
			Name:     "NilData",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-INVOICE-CREATE-NIL-DATA-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *invoicepb.CreateInvoiceRequest {
				return &invoicepb.CreateInvoiceRequest{Data: nil}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "invoice.validation.data_required",
			Assertions: func(t *testing.T, response *invoicepb.CreateInvoiceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilData(t, err)
			},
		},
		{
			Name:     "ZeroAmount",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-INVOICE-CREATE-VALIDATION-ZERO-AMOUNT-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *invoicepb.CreateInvoiceRequest {
				return &invoicepb.CreateInvoiceRequest{
					Data: &invoicepb.Invoice{
						Amount: 0.0,
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "invoice.validation.amount_required",
			Assertions: func(t *testing.T, response *invoicepb.CreateInvoiceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "zero amount")
			},
		},
		{
			Name:     "NegativeAmount",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-INVOICE-CREATE-VALIDATION-NEGATIVE-AMOUNT-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *invoicepb.CreateInvoiceRequest {
				return &invoicepb.CreateInvoiceRequest{
					Data: &invoicepb.Invoice{
						Amount: -100.0,
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "invoice.validation.amount_invalid",
			Assertions: func(t *testing.T, response *invoicepb.CreateInvoiceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "negative amount")
			},
		},
		{
			Name:     "DataEnrichment",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-INVOICE-CREATE-ENRICHMENT-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *invoicepb.CreateInvoiceRequest {
				return &invoicepb.CreateInvoiceRequest{
					Data: &invoicepb.Invoice{
						Amount: 1500.0,
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *invoicepb.CreateInvoiceResponse, err error, useCase interface{}, ctx context.Context) {
				createdInvoice := response.Data[0]
				testutil.AssertEqual(t, 1500.0, createdInvoice.Amount, "invoice amount")
				testutil.AssertNonEmptyString(t, createdInvoice.Id, "generated ID")
				testutil.AssertNonEmptyString(t, createdInvoice.InvoiceNumber, "generated invoice number")
				testutil.AssertFieldSet(t, createdInvoice.DateCreated, "DateCreated")
				testutil.AssertFieldSet(t, createdInvoice.DateCreatedString, "DateCreatedString")
				testutil.AssertTrue(t, createdInvoice.Active, "Active")
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
