//go:build mock_db && mock_auth

// Package invoice provides table-driven tests for the invoice read use case.
//
// The tests cover various scenarios, including success, transaction handling,
// nil requests, validation errors, and not-found cases.
// Each test case is defined in a table with a specific test code, request setup,
// and assertions.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user").
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education").
//
// Usage:
//   - Run all tests: go test -tags="mock_db,mock_auth" ./...
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestReadInvoiceUseCase_Execute_TableDriven
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-SUBSCRIPTION-INVOICE-READ-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-SUBSCRIPTION-INVOICE-READ-TRANSACTION-v1.0: WithTransaction
//   - ESPYNA-TEST-SUBSCRIPTION-INVOICE-READ-NIL-REQUEST-v1.0: NilRequest
//   - ESPYNA-TEST-SUBSCRIPTION-INVOICE-READ-EMPTY-ID-v1.0: EmptyId
//   - ESPYNA-TEST-SUBSCRIPTION-INVOICE-READ-NOT-FOUND-v1.0: NotFound
//
// Data Sources:
//   - Hardcoded test invoice IDs for simplicity
//   - Mock data: packages/copya/data/{businessType}/invoice.json
//   - Translations: packages/lyngua/translations/{languageCode}/{businessType}/invoice.json

package invoice

import (
	"context"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/shared/testutil"
	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock/subscription"
	invoicepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/invoice"
)

// Type alias for read invoice test cases
type ReadInvoiceTestCase = testutil.GenericTestCase[*invoicepb.ReadInvoiceRequest, *invoicepb.ReadInvoiceResponse]

func createTestReadInvoiceUseCaseWithAuth(businessType string, supportsTransaction bool, shouldAuthorize bool) *ReadInvoiceUseCase {
	mockInvoiceRepo := subscription.NewMockInvoiceRepository(businessType)

	repositories := ReadInvoiceRepositories{
		Invoice: mockInvoiceRepo,
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, shouldAuthorize)
	services := ReadInvoiceServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
	}

	return NewReadInvoiceUseCase(repositories, services)
}

func TestReadInvoiceUseCase_Execute_TableDriven(t *testing.T) {
	// Use hardcoded test IDs instead of complex resolvers for simplicity
	existingInvoiceId := "invoice-test-123"
	nonExistentInvoiceId := "invoice-nonexistent-789"

	testCases := []ReadInvoiceTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-INVOICE-READ-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *invoicepb.ReadInvoiceRequest {
				return &invoicepb.ReadInvoiceRequest{
					Data: &invoicepb.Invoice{
						Id: existingInvoiceId,
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *invoicepb.ReadInvoiceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				testutil.AssertEqual(t, 1, len(response.Data), "response data length")
				invoice := response.Data[0]
				testutil.AssertStringEqual(t, existingInvoiceId, invoice.Id, "invoice ID")
				testutil.AssertNonEmptyString(t, invoice.InvoiceNumber, "invoice number")
				testutil.AssertTrue(t, invoice.Amount > 0, "invoice amount should be positive")
				testutil.AssertTrue(t, invoice.Active, "invoice should be active")
				testutil.AssertFieldSet(t, invoice.DateCreated, "DateCreated")
				testutil.AssertFieldSet(t, invoice.DateCreatedString, "DateCreatedString")
			},
		},
		{
			Name:     "WithTransaction",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-INVOICE-READ-TRANSACTION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *invoicepb.ReadInvoiceRequest {
				return &invoicepb.ReadInvoiceRequest{
					Data: &invoicepb.Invoice{
						Id: existingInvoiceId,
					},
				}
			},
			UseTransaction: true,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *invoicepb.ReadInvoiceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "success")
				invoice := response.Data[0]
				testutil.AssertStringEqual(t, existingInvoiceId, invoice.Id, "invoice ID")
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-INVOICE-READ-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *invoicepb.ReadInvoiceRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "invoice.validation.request_required",
			Assertions: func(t *testing.T, response *invoicepb.ReadInvoiceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilRequest(t, err)
			},
		},
		{
			Name:     "EmptyId",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-INVOICE-READ-EMPTY-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *invoicepb.ReadInvoiceRequest {
				return &invoicepb.ReadInvoiceRequest{
					Data: &invoicepb.Invoice{Id: ""},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "invoice.validation.id_required",
			Assertions: func(t *testing.T, response *invoicepb.ReadInvoiceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty ID")
			},
		},
		{
			Name:     "NotFound",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-INVOICE-READ-NOT-FOUND-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *invoicepb.ReadInvoiceRequest {
				return &invoicepb.ReadInvoiceRequest{
					Data: &invoicepb.Invoice{Id: nonExistentInvoiceId},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "invoice.errors.not_found",
			ErrorTags:      map[string]any{"invoiceId": nonExistentInvoiceId},
			Assertions: func(t *testing.T, response *invoicepb.ReadInvoiceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertError(t, err)
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
			useCase := createTestReadInvoiceUseCaseWithAuth(businessType, tc.UseTransaction, tc.UseAuth)

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
