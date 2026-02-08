//go:build mock_db && mock_auth

// Package invoice provides table-driven tests for the invoice deletion use case.
//
// The tests cover various scenarios, including success, transaction handling,
// nil requests, validation errors, and error handling.
// Each test case is defined in a table with a specific test code, request setup,
// and assertions.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user").
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education").
//
// Usage:
//   - Run all tests: go test -tags="mock_db,mock_auth" ./...
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestDeleteInvoiceUseCase_Execute_TableDriven
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-SUBSCRIPTION-INVOICE-DELETE-SUCCESS-v1.0: Success
//   - ESPYNA-TEST-SUBSCRIPTION-INVOICE-DELETE-WITH-TRANSACTION-v1.0: WithTransaction
//   - ESPYNA-TEST-SUBSCRIPTION-INVOICE-DELETE-NIL-REQUEST-v1.0: NilRequest
//   - ESPYNA-TEST-SUBSCRIPTION-INVOICE-DELETE-EMPTY-ID-v1.0: EmptyId
//   - ESPYNA-TEST-SUBSCRIPTION-INVOICE-DELETE-NOT-FOUND-v1.0: NotFound
//
// Data Sources:
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

// Type alias for delete invoice test cases
type DeleteInvoiceTestCase = testutil.GenericTestCase[*invoicepb.DeleteInvoiceRequest, *invoicepb.DeleteInvoiceResponse]

// createTestDeleteInvoiceUseCase is a helper function to create the use case with mock dependencies
func createTestDeleteInvoiceUseCase(businessType string, supportsTransaction bool, shouldAuthorize bool) *DeleteInvoiceUseCase {
	mockRepo := subscription.NewMockInvoiceRepository(businessType)

	repositories := DeleteInvoiceRepositories{
		Invoice: mockRepo,
	}

	standardServices := testutil.CreateStandardServices(supportsTransaction, shouldAuthorize)
	services := DeleteInvoiceServices{
		AuthorizationService: standardServices.AuthorizationService,
		TransactionService:   standardServices.TransactionService,
		TranslationService:   standardServices.TranslationService,
	}

	return NewDeleteInvoiceUseCase(repositories, services)
}

func TestDeleteInvoiceUseCase_Execute_TableDriven(t *testing.T) {
	testCases := []DeleteInvoiceTestCase{
		{
			Name:     "Success",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-INVOICE-DELETE-SUCCESS-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *invoicepb.DeleteInvoiceRequest {
				return &invoicepb.DeleteInvoiceRequest{
					Data: &invoicepb.Invoice{
						Id: "invoice-student-001-tuition",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *invoicepb.DeleteInvoiceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "successful deletion")
			},
		},
		{
			Name:     "WithTransaction",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-INVOICE-DELETE-WITH-TRANSACTION-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *invoicepb.DeleteInvoiceRequest {
				return &invoicepb.DeleteInvoiceRequest{
					Data: &invoicepb.Invoice{
						Id: "invoice-student-002-tuition",
					},
				}
			},
			UseTransaction: true,
			UseAuth:        true,
			ExpectSuccess:  true,
			Assertions: func(t *testing.T, response *invoicepb.DeleteInvoiceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertTrue(t, response.Success, "successful deletion with transaction")
			},
		},
		{
			Name:     "NilRequest",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-INVOICE-DELETE-NIL-REQUEST-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *invoicepb.DeleteInvoiceRequest {
				return nil
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			Assertions: func(t *testing.T, response *invoicepb.DeleteInvoiceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertErrorForNilRequest(t, err)
				testutil.AssertNil(t, response, "response for nil request")
			},
		},
		{
			Name:     "EmptyId",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-INVOICE-DELETE-EMPTY-ID-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *invoicepb.DeleteInvoiceRequest {
				return &invoicepb.DeleteInvoiceRequest{
					Data: &invoicepb.Invoice{
						Id: "",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			Assertions: func(t *testing.T, response *invoicepb.DeleteInvoiceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertValidationError(t, err, "empty invoice ID")
				testutil.AssertNil(t, response, "response for invalid input")
			},
		},
		{
			Name:     "NotFound",
			TestCode: "ESPYNA-TEST-SUBSCRIPTION-INVOICE-DELETE-NOT-FOUND-v1.0",
			SetupRequest: func(t *testing.T, businessType string) *invoicepb.DeleteInvoiceRequest {
				return &invoicepb.DeleteInvoiceRequest{
					Data: &invoicepb.Invoice{
						Id: "invoice-non-existent-123",
					},
				}
			},
			UseTransaction: false,
			UseAuth:        true,
			ExpectSuccess:  false,
			ExpectedError:  "invoice.errors.not_found",
			ErrorTags:      map[string]any{"invoiceId": "invoice-non-existent-123"},
			Assertions: func(t *testing.T, response *invoicepb.DeleteInvoiceResponse, err error, useCase interface{}, ctx context.Context) {
				testutil.AssertError(t, err)
				testutil.AssertNil(t, response, "response for non-existent invoice")
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
			useCase := createTestDeleteInvoiceUseCase(businessType, tc.UseTransaction, tc.UseAuth)

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
