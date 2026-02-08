//go:build mock_db && mock_auth

// Package invoice provides table-driven tests for the invoice update use case.
//
// The tests cover various scenarios focused on the Invoice protobuf structure which
// has only Amount as user input, with all other fields (Id, InvoiceNumber, Active,
// DateCreated, DateModified, etc.) being auto-generated. Updates focus on amount
// changes for existing invoices.
//
// Test scenarios include:
// - Success with valid amount update
// - Transaction handling
// - Authorization validation
// - Nil request/data validation
// - Empty ID validation
// - Not found scenarios
// - Amount validation (zero and negative amounts)
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user").
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education").
//
// Usage:
//   - Run all tests: go test -tags="mock_db,mock_auth" ./...
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestUpdateInvoiceUseCase_Execute_TableDriven
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-SUBSCRIPTION-INVOICE-UPDATE-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-SUBSCRIPTION-INVOICE-UPDATE-TRANSACTION-v1.0: WithTransaction
//   - ESPYNA-TEST-SUBSCRIPTION-INVOICE-UPDATE-NIL-REQUEST-v1.0: NilRequest
//   - ESPYNA-TEST-SUBSCRIPTION-INVOICE-UPDATE-NIL-DATA-v1.0: NilData
//   - ESPYNA-TEST-SUBSCRIPTION-INVOICE-UPDATE-VALIDATION-EMPTY-ID-v1.0: EmptyId
//   - ESPYNA-TEST-SUBSCRIPTION-INVOICE-UPDATE-NOT-FOUND-v1.0: NotFound
//   - ESPYNA-TEST-SUBSCRIPTION-INVOICE-UPDATE-VALIDATION-ZERO-AMOUNT-v1.0: ZeroAmount
//   - ESPYNA-TEST-SUBSCRIPTION-INVOICE-UPDATE-VALIDATION-NEGATIVE-AMOUNT-v1.0: NegativeAmount
//
// Data Sources:
//   - All test data is hardcoded with specific invoice IDs and amounts
//   - No external JSON files required since Invoice only has Amount field as input
package invoice

import (
	"context"
	"testing"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/shared/testutil"
	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock/subscription"
	invoicepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/invoice"
)

// Type alias for update invoice test cases
type UpdateInvoiceTestCase = testutil.GenericTestCase[*invoicepb.UpdateInvoiceRequest, *invoicepb.UpdateInvoiceResponse]

// createTestUpdateInvoiceUseCase is a helper function to create the use case with mock dependencies
func createTestUpdateInvoiceUseCase(businessType string, supportsTransaction bool, shouldAuthorize bool) *UpdateInvoiceUseCase {
	mockRepo := subscription.NewMockInvoiceRepository(businessType)

	repositories := UpdateInvoiceRepositories{
		Invoice: mockRepo,
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, shouldAuthorize)
	services := UpdateInvoiceServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
	}

	return NewUpdateInvoiceUseCase(repositories, services)
}

func TestUpdateInvoiceUseCase_Execute_TableDriven(t *testing.T) {
	// No test data resolvers needed - using hardcoded invoice IDs and amounts only

	testCases := []UpdateInvoiceTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-INVOICE-UPDATE-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *invoicepb.UpdateInvoiceRequest {
				time.Sleep(1 * time.Second) // Ensure DateModified will be different
				return &invoicepb.UpdateInvoiceRequest{
					Data: &invoicepb.Invoice{
						Id:            "invoice-student-001-tuition",
						InvoiceNumber: "TUITION-2024-001-UPDATED",
						Amount:        12000.0,
						Active:        true,
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *invoicepb.UpdateInvoiceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertEqual(t, 1, len(response.Data), "response data length")
				updatedInvoice := response.Data[0]
				testutil.AssertEqual(t, 12000.0, updatedInvoice.Amount, "updated amount")
				testutil.AssertFieldSet(t, updatedInvoice.DateModified, "DateModified")
			},
		},
		{
			Name:     "WithTransaction",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-INVOICE-UPDATE-TRANSACTION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *invoicepb.UpdateInvoiceRequest {
				time.Sleep(1 * time.Second) // Ensure DateModified will be different
				return &invoicepb.UpdateInvoiceRequest{
					Data: &invoicepb.Invoice{
						Id:            "invoice-student-002-tuition",
						InvoiceNumber: "TUITION-2024-002-UPDATED",
						Amount:        15000.0,
						Active:        true,
					},
				}
			},
			UseTransaction: true,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *invoicepb.UpdateInvoiceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				updatedInvoice := response.Data[0]
				testutil.AssertEqual(t, 15000.0, updatedInvoice.Amount, "updated amount")
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-INVOICE-UPDATE-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *invoicepb.UpdateInvoiceRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "invoice.validation.request_required",
			Assertions: func(t *testing.T, response *invoicepb.UpdateInvoiceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilRequest(t, err)
			},
		},
		{
			Name:     "NilData",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-INVOICE-UPDATE-NIL-DATA-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *invoicepb.UpdateInvoiceRequest {
				return &invoicepb.UpdateInvoiceRequest{Data: nil}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "invoice.validation.data_required",
			Assertions: func(t *testing.T, response *invoicepb.UpdateInvoiceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilData(t, err)
			},
		},
		{
			Name:     "EmptyId",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-INVOICE-UPDATE-VALIDATION-EMPTY-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *invoicepb.UpdateInvoiceRequest {
				return &invoicepb.UpdateInvoiceRequest{
					Data: &invoicepb.Invoice{
						Id:            "",
						InvoiceNumber: "EMPTY-ID-TEST",
						Amount:        100.0,
						Active:        true,
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "invoice.validation.id_required",
			Assertions: func(t *testing.T, response *invoicepb.UpdateInvoiceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty ID")
			},
		},
		{
			Name:     "NotFound",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-INVOICE-UPDATE-NOT-FOUND-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *invoicepb.UpdateInvoiceRequest {
				return &invoicepb.UpdateInvoiceRequest{
					Data: &invoicepb.Invoice{
						Id:            "non-existent-invoice",
						InvoiceNumber: "NON-EXISTENT-001",
						Amount:        100.0,
						Active:        true,
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "invoice.errors.not_found",
			ErrorTags:      map[string]any{"invoiceId": "non-existent-invoice"},
			Assertions: func(t *testing.T, response *invoicepb.UpdateInvoiceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertError(t, err)
			},
		},
		{
			Name:     "ZeroAmount",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-INVOICE-UPDATE-VALIDATION-ZERO-AMOUNT-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *invoicepb.UpdateInvoiceRequest {
				return &invoicepb.UpdateInvoiceRequest{
					Data: &invoicepb.Invoice{
						Id:            "invoice-student-001-tuition",
						InvoiceNumber: "TUITION-2024-001-ZERO",
						Amount:        0.0,
						Active:        true,
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "invoice.validation.amount_positive",
			Assertions: func(t *testing.T, response *invoicepb.UpdateInvoiceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "zero amount")
			},
		},
		{
			Name:     "NegativeAmount",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-INVOICE-UPDATE-VALIDATION-NEGATIVE-AMOUNT-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *invoicepb.UpdateInvoiceRequest {
				return &invoicepb.UpdateInvoiceRequest{
					Data: &invoicepb.Invoice{
						Id:            "invoice-student-001-tuition",
						InvoiceNumber: "TUITION-2024-001-NEGATIVE",
						Amount:        -100.0,
						Active:        true,
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "invoice.validation.amount_positive",
			Assertions: func(t *testing.T, response *invoicepb.UpdateInvoiceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "negative amount")
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
			useCase := createTestUpdateInvoiceUseCase(businessType, tc.UseTransaction, tc.UseAuth)

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
					if tc.ExactError {
						testutil.AssertStringEqual(t, tc.ExpectedError, err.Error(), "error message")
					} else if tc.ErrorTags != nil {
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
