//go:build mock_db && mock_auth

package balance

import (
	"context"
	"testing"

	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock/subscription"
	balancepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/balance"
)

func TestGetBalanceItemPageDataUseCase_Execute_Success(t *testing.T) {
	// Setup
	ctx := context.Background()

	// Create mock repository with some test data
	mockRepo := subscription.NewMockBalanceRepository("education")

	// Create a test balance
	createResp, err := mockRepo.CreateBalance(ctx, &balancepb.CreateBalanceRequest{
		Data: &balancepb.Balance{
			Amount:         150.75,
			ClientId:       "client-1",
			SubscriptionId: "subscription-1",
			Currency:       "USD",
			BalanceType:    "credit",
			Active:         true,
		},
	})
	if err != nil {
		t.Fatalf("Failed to create test balance: %v", err)
	}

	testBalanceId := createResp.Data[0].Id

	// Setup repositories and services
	repos := GetBalanceItemPageDataRepositories{
		Balance: mockRepo,
	}
	services := GetBalanceItemPageDataServices{
		TransactionService: nil, // No transaction for this test
		TranslationService: nil, // No translation for this test
	}

	// Create use case
	useCase := NewGetBalanceItemPageDataUseCase(repos, services)

	// Test case 1: Successful retrieval
	t.Run("SuccessfulRetrieval", func(t *testing.T) {
		req := &balancepb.GetBalanceItemPageDataRequest{
			BalanceId: testBalanceId,
		}

		resp, err := useCase.Execute(ctx, req)
		if err != nil {
			t.Fatalf("Execute failed: %v", err)
		}

		if !resp.Success {
			t.Error("Expected success=true")
		}

		if resp.Balance == nil {
			t.Fatal("Expected balance data")
		}

		if resp.Balance.Id != testBalanceId {
			t.Errorf("Expected balance ID %s, got %s", testBalanceId, resp.Balance.Id)
		}

		if resp.Balance.Amount != 150.75 {
			t.Errorf("Expected amount 150.75, got %f", resp.Balance.Amount)
		}

		if resp.Balance.ClientId != "client-1" {
			t.Errorf("Expected client_id 'client-1', got '%s'", resp.Balance.ClientId)
		}

		if resp.Balance.SubscriptionId != "subscription-1" {
			t.Errorf("Expected subscription_id 'subscription-1', got '%s'", resp.Balance.SubscriptionId)
		}

		if resp.Balance.Currency != "USD" {
			t.Errorf("Expected currency 'USD', got '%s'", resp.Balance.Currency)
		}

		if resp.Balance.BalanceType != "credit" {
			t.Errorf("Expected balance_type 'credit', got '%s'", resp.Balance.BalanceType)
		}

		if !resp.Balance.Active {
			t.Error("Expected active=true")
		}
	})

	// Test case 2: Data transformation validation
	t.Run("DataTransformation", func(t *testing.T) {
		// Create a balance without currency to test default assignment
		createResp2, err := mockRepo.CreateBalance(ctx, &balancepb.CreateBalanceRequest{
			Data: &balancepb.Balance{
				Amount:         -50.00,
				ClientId:       "client-2",
				SubscriptionId: "subscription-2",
				Currency:       "", // Empty currency
				BalanceType:    "", // Empty balance type
				Active:         true,
			},
		})
		if err != nil {
			t.Fatalf("Failed to create test balance 2: %v", err)
		}

		req := &balancepb.GetBalanceItemPageDataRequest{
			BalanceId: createResp2.Data[0].Id,
		}

		resp, err := useCase.Execute(ctx, req)
		if err != nil {
			t.Fatalf("Execute failed: %v", err)
		}

		if !resp.Success {
			t.Error("Expected success=true")
		}

		// Check that default currency was applied
		if resp.Balance.Currency != "USD" {
			t.Errorf("Expected default currency 'USD', got '%s'", resp.Balance.Currency)
		}

		// Check that balance type was auto-determined based on amount
		if resp.Balance.BalanceType != "debit" {
			t.Errorf("Expected balance_type 'debit' for negative amount, got '%s'", resp.Balance.BalanceType)
		}
	})

	// Test case 3: Financial data validation
	t.Run("FinancialDataValidation", func(t *testing.T) {
		// Create a balance with valid financial data
		createResp3, err := mockRepo.CreateBalance(ctx, &balancepb.CreateBalanceRequest{
			Data: &balancepb.Balance{
				Amount:         999.99,
				ClientId:       "client-3",
				SubscriptionId: "subscription-3",
				Currency:       "EUR",
				BalanceType:    "credit",
				Active:         true,
			},
		})
		if err != nil {
			t.Fatalf("Failed to create test balance 3: %v", err)
		}

		req := &balancepb.GetBalanceItemPageDataRequest{
			BalanceId: createResp3.Data[0].Id,
		}

		resp, err := useCase.Execute(ctx, req)
		if err != nil {
			t.Fatalf("Execute failed: %v", err)
		}

		if !resp.Success {
			t.Error("Expected success=true")
		}

		// Verify all financial data is preserved correctly
		if resp.Balance.Amount != 999.99 {
			t.Errorf("Expected amount 999.99, got %f", resp.Balance.Amount)
		}

		if resp.Balance.Currency != "EUR" {
			t.Errorf("Expected currency 'EUR', got '%s'", resp.Balance.Currency)
		}

		if resp.Balance.BalanceType != "credit" {
			t.Errorf("Expected balance_type 'credit', got '%s'", resp.Balance.BalanceType)
		}
	})
}

func TestGetBalanceItemPageDataUseCase_Execute_NotFound(t *testing.T) {
	// Setup with empty repository
	ctx := context.Background()
	mockRepo := subscription.NewMockBalanceRepository("education")

	repos := GetBalanceItemPageDataRepositories{
		Balance: mockRepo,
	}
	services := GetBalanceItemPageDataServices{
		TransactionService: nil,
		TranslationService: nil,
	}

	useCase := NewGetBalanceItemPageDataUseCase(repos, services)

	req := &balancepb.GetBalanceItemPageDataRequest{
		BalanceId: "non-existent-id",
	}

	_, err := useCase.Execute(ctx, req)
	if err == nil {
		t.Error("Expected error for non-existent balance")
	}
}

func TestGetBalanceItemPageDataUseCase_Execute_ValidationErrors(t *testing.T) {
	// Setup
	ctx := context.Background()
	mockRepo := subscription.NewMockBalanceRepository("education")

	repos := GetBalanceItemPageDataRepositories{
		Balance: mockRepo,
	}
	services := GetBalanceItemPageDataServices{
		TransactionService: nil,
		TranslationService: nil,
	}

	useCase := NewGetBalanceItemPageDataUseCase(repos, services)

	// Test case 1: Nil request
	t.Run("NilRequest", func(t *testing.T) {
		_, err := useCase.Execute(ctx, nil)
		if err == nil {
			t.Error("Expected error for nil request")
		}
	})

	// Test case 2: Empty balance ID
	t.Run("EmptyBalanceId", func(t *testing.T) {
		req := &balancepb.GetBalanceItemPageDataRequest{
			BalanceId: "",
		}

		_, err := useCase.Execute(ctx, req)
		if err == nil {
			t.Error("Expected error for empty balance ID")
		}
	})

	// Test case 3: Too short balance ID
	t.Run("TooShortBalanceId", func(t *testing.T) {
		req := &balancepb.GetBalanceItemPageDataRequest{
			BalanceId: "ab", // Too short
		}

		_, err := useCase.Execute(ctx, req)
		if err == nil {
			t.Error("Expected error for too short balance ID")
		}
	})
}

func TestGetBalanceItemPageDataUseCase_FinancialDataValidation(t *testing.T) {
	// Setup
	ctx := context.Background()
	mockRepo := subscription.NewMockBalanceRepository("education")

	repos := GetBalanceItemPageDataRepositories{
		Balance: mockRepo,
	}
	services := GetBalanceItemPageDataServices{
		TransactionService: nil,
		TranslationService: nil,
	}

	useCase := NewGetBalanceItemPageDataUseCase(repos, services)

	// Test various financial data scenarios
	testCases := []struct {
		name             string
		amount           float64
		currency         string
		balanceType      string
		expectedError    bool
		expectedCurrency string
		expectedType     string
	}{
		{
			name:             "ValidPositiveCredit",
			amount:           100.50,
			currency:         "USD",
			balanceType:      "credit",
			expectedError:    false,
			expectedCurrency: "USD",
			expectedType:     "credit",
		},
		{
			name:             "ValidNegativeDebit",
			amount:           -75.25,
			currency:         "EUR",
			balanceType:      "debit",
			expectedError:    false,
			expectedCurrency: "EUR",
			expectedType:     "debit",
		},
		{
			name:             "ZeroAmount",
			amount:           0.0,
			currency:         "GBP",
			balanceType:      "credit",
			expectedError:    false,
			expectedCurrency: "GBP",
			expectedType:     "credit",
		},
		{
			name:             "EmptyCurrency",
			amount:           50.0,
			currency:         "",
			balanceType:      "credit",
			expectedError:    false,
			expectedCurrency: "USD", // Default
			expectedType:     "credit",
		},
		{
			name:             "EmptyBalanceType",
			amount:           -25.0,
			currency:         "USD",
			balanceType:      "",
			expectedError:    false,
			expectedCurrency: "USD",
			expectedType:     "debit", // Auto-determined
		},
		{
			name:          "InvalidCurrency",
			amount:        100.0,
			currency:      "INVALID", // Too long
			balanceType:   "credit",
			expectedError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create test balance
			createResp, err := mockRepo.CreateBalance(ctx, &balancepb.CreateBalanceRequest{
				Data: &balancepb.Balance{
					Amount:         tc.amount,
					ClientId:       "client-test",
					SubscriptionId: "subscription-test",
					Currency:       tc.currency,
					BalanceType:    tc.balanceType,
					Active:         true,
				},
			})
			if err != nil {
				t.Fatalf("Failed to create test balance: %v", err)
			}

			req := &balancepb.GetBalanceItemPageDataRequest{
				BalanceId: createResp.Data[0].Id,
			}

			resp, err := useCase.Execute(ctx, req)

			if tc.expectedError {
				if err == nil {
					t.Errorf("Expected error for test case %s", tc.name)
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error for test case %s: %v", tc.name, err)
			}

			if !resp.Success {
				t.Errorf("Expected success=true for test case %s", tc.name)
			}

			if resp.Balance.Amount != tc.amount {
				t.Errorf("Expected amount %f, got %f for test case %s", tc.amount, resp.Balance.Amount, tc.name)
			}

			if resp.Balance.Currency != tc.expectedCurrency {
				t.Errorf("Expected currency '%s', got '%s' for test case %s", tc.expectedCurrency, resp.Balance.Currency, tc.name)
			}

			if resp.Balance.BalanceType != tc.expectedType {
				t.Errorf("Expected balance_type '%s', got '%s' for test case %s", tc.expectedType, resp.Balance.BalanceType, tc.name)
			}
		})
	}
}
