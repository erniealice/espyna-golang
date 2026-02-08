//go:build mock_db && mock_auth

package balance

import (
	"context"
	"testing"

	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock/subscription"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	balancepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/balance"
)

func TestGetBalanceListPageDataUseCase_Execute_Success(t *testing.T) {
	// Setup
	ctx := context.Background()

	// Create mock repository with some test data
	mockRepo := subscription.NewMockBalanceRepository("education")

	// Create some test balances first
	balance1, err := mockRepo.CreateBalance(ctx, &balancepb.CreateBalanceRequest{
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
		t.Fatalf("Failed to create test balance 1: %v", err)
	}

	_, err = mockRepo.CreateBalance(ctx, &balancepb.CreateBalanceRequest{
		Data: &balancepb.Balance{
			Amount:         -75.25,
			ClientId:       "client-2",
			SubscriptionId: "subscription-2",
			Currency:       "USD",
			BalanceType:    "debit",
			Active:         false,
		},
	})
	if err != nil {
		t.Fatalf("Failed to create test balance 2: %v", err)
	}

	_, err = mockRepo.CreateBalance(ctx, &balancepb.CreateBalanceRequest{
		Data: &balancepb.Balance{
			Amount:         250.00,
			ClientId:       "client-1",
			SubscriptionId: "subscription-3",
			Currency:       "EUR",
			BalanceType:    "credit",
			Active:         true,
		},
	})
	if err != nil {
		t.Fatalf("Failed to create test balance 3: %v", err)
	}

	// Setup repositories and services
	repos := GetBalanceListPageDataRepositories{
		Balance: mockRepo,
	}
	services := GetBalanceListPageDataServices{
		TransactionService: nil, // No transaction for this test
		TranslationService: nil, // No translation for this test
	}

	// Create use case
	useCase := NewGetBalanceListPageDataUseCase(repos, services)

	// Test case 1: Basic list without filters
	t.Run("BasicList", func(t *testing.T) {
		req := &balancepb.GetBalanceListPageDataRequest{
			Pagination: &commonpb.PaginationRequest{
				Limit: 10,
				Method: &commonpb.PaginationRequest_Offset{
					Offset: &commonpb.OffsetPagination{Page: 1},
				},
			},
		}

		resp, err := useCase.Execute(ctx, req)
		if err != nil {
			t.Fatalf("Execute failed: %v", err)
		}

		if !resp.Success {
			t.Error("Expected success=true")
		}

		if len(resp.BalanceList) < 3 {
			t.Errorf("Expected at least 3 balances, got %d", len(resp.BalanceList))
		}

		if resp.Pagination == nil {
			t.Error("Expected pagination response")
		}
	})

	// Test case 2: Filtering by active status
	t.Run("FilterByActive", func(t *testing.T) {
		req := &balancepb.GetBalanceListPageDataRequest{
			Pagination: &commonpb.PaginationRequest{
				Limit: 10,
				Method: &commonpb.PaginationRequest_Offset{
					Offset: &commonpb.OffsetPagination{Page: 1},
				},
			},
			Filters: &commonpb.FilterRequest{
				Logic: commonpb.FilterLogic_AND,
				Filters: []*commonpb.TypedFilter{
					{
						Field: "active",
						FilterType: &commonpb.TypedFilter_BooleanFilter{
							BooleanFilter: &commonpb.BooleanFilter{
								Value: true,
							},
						},
					},
				},
			},
		}

		resp, err := useCase.Execute(ctx, req)
		if err != nil {
			t.Fatalf("Execute failed: %v", err)
		}

		if !resp.Success {
			t.Error("Expected success=true")
		}

		// Should only get active balances
		for _, balance := range resp.BalanceList {
			if !balance.Active {
				t.Errorf("Expected only active balances, found inactive: %s", balance.Id)
			}
		}
	})

	// Test case 3: Filtering by balance type
	t.Run("FilterByBalanceType", func(t *testing.T) {
		req := &balancepb.GetBalanceListPageDataRequest{
			Pagination: &commonpb.PaginationRequest{
				Limit: 10,
				Method: &commonpb.PaginationRequest_Offset{
					Offset: &commonpb.OffsetPagination{Page: 1},
				},
			},
			Filters: &commonpb.FilterRequest{
				Logic: commonpb.FilterLogic_AND,
				Filters: []*commonpb.TypedFilter{
					{
						Field: "balance_type",
						FilterType: &commonpb.TypedFilter_StringFilter{
							StringFilter: &commonpb.StringFilter{
								Operator: commonpb.StringOperator_STRING_EQUALS,
								Value:    "credit",
							},
						},
					},
				},
			},
		}

		resp, err := useCase.Execute(ctx, req)
		if err != nil {
			t.Fatalf("Execute failed: %v", err)
		}

		if !resp.Success {
			t.Error("Expected success=true")
		}

		// Should only get credit balances
		for _, balance := range resp.BalanceList {
			if balance.BalanceType != "credit" {
				t.Errorf("Expected only credit balances, found: %s", balance.BalanceType)
			}
		}
	})

	// Test case 4: Filtering by amount range
	t.Run("FilterByAmountRange", func(t *testing.T) {
		req := &balancepb.GetBalanceListPageDataRequest{
			Pagination: &commonpb.PaginationRequest{
				Limit: 10,
				Method: &commonpb.PaginationRequest_Offset{
					Offset: &commonpb.OffsetPagination{Page: 1},
				},
			},
			Filters: &commonpb.FilterRequest{
				Logic: commonpb.FilterLogic_AND,
				Filters: []*commonpb.TypedFilter{
					{
						Field: "amount",
						FilterType: &commonpb.TypedFilter_NumberFilter{
							NumberFilter: &commonpb.NumberFilter{
								Operator: commonpb.NumberOperator_NUMBER_GREATER_THAN,
								Value:    100.0,
							},
						},
					},
				},
			},
		}

		resp, err := useCase.Execute(ctx, req)
		if err != nil {
			t.Fatalf("Execute failed: %v", err)
		}

		if !resp.Success {
			t.Error("Expected success=true")
		}

		// Should only get balances greater than 100
		for _, balance := range resp.BalanceList {
			if balance.Amount <= 100.0 {
				t.Errorf("Expected only balances > 100, found: %f", balance.Amount)
			}
		}
	})

	// Test case 5: Search by client ID
	t.Run("SearchByClientId", func(t *testing.T) {
		req := &balancepb.GetBalanceListPageDataRequest{
			Pagination: &commonpb.PaginationRequest{
				Limit: 10,
				Method: &commonpb.PaginationRequest_Offset{
					Offset: &commonpb.OffsetPagination{Page: 1},
				},
			},
			Search: &commonpb.SearchRequest{
				Query: "client-1",
				Options: &commonpb.SearchOptions{
					SearchFields: []string{"client_id"},
				},
			},
		}

		resp, err := useCase.Execute(ctx, req)
		if err != nil {
			t.Fatalf("Execute failed: %v", err)
		}

		if !resp.Success {
			t.Error("Expected success=true")
		}

		// Should find at least the first balance
		found := false
		for _, balance := range resp.BalanceList {
			if balance.Id == balance1.Data[0].Id {
				found = true
				break
			}
		}

		if !found {
			t.Error("Expected to find balance1 in search results")
		}
	})

	// Test case 6: Sorting by amount
	t.Run("SortByAmount", func(t *testing.T) {
		req := &balancepb.GetBalanceListPageDataRequest{
			Pagination: &commonpb.PaginationRequest{
				Limit: 10,
				Method: &commonpb.PaginationRequest_Offset{
					Offset: &commonpb.OffsetPagination{Page: 1},
				},
			},
			Sort: &commonpb.SortRequest{
				Fields: []*commonpb.SortField{
					{
						Field:     "amount",
						Direction: commonpb.SortDirection_DESC,
						NullOrder: commonpb.NullOrder_NULLS_LAST,
					},
				},
			},
		}

		resp, err := useCase.Execute(ctx, req)
		if err != nil {
			t.Fatalf("Execute failed: %v", err)
		}

		if !resp.Success {
			t.Error("Expected success=true")
		}

		// Verify sorting - amounts should be in descending order
		if len(resp.BalanceList) >= 2 {
			for i := 1; i < len(resp.BalanceList); i++ {
				if resp.BalanceList[i-1].Amount < resp.BalanceList[i].Amount {
					t.Errorf("Balances not sorted properly: %f < %f",
						resp.BalanceList[i-1].Amount, resp.BalanceList[i].Amount)
				}
			}
		}
	})

	// Test case 7: Pagination
	t.Run("Pagination", func(t *testing.T) {
		req := &balancepb.GetBalanceListPageDataRequest{
			Pagination: &commonpb.PaginationRequest{
				Limit: 1, // Only get 1 item per page
				Method: &commonpb.PaginationRequest_Offset{
					Offset: &commonpb.OffsetPagination{Page: 1},
				},
			},
		}

		resp, err := useCase.Execute(ctx, req)
		if err != nil {
			t.Fatalf("Execute failed: %v", err)
		}

		if !resp.Success {
			t.Error("Expected success=true")
		}

		if len(resp.BalanceList) != 1 {
			t.Errorf("Expected exactly 1 balance with limit=1, got %d", len(resp.BalanceList))
		}

		if resp.Pagination == nil {
			t.Error("Expected pagination response")
		} else {
			if resp.Pagination.TotalItems < 3 {
				t.Errorf("Expected total items >= 3, got %d", resp.Pagination.TotalItems)
			}
		}
	})

	// Test case 8: Complex filter - Active credit balances for specific client
	t.Run("ComplexFilter", func(t *testing.T) {
		req := &balancepb.GetBalanceListPageDataRequest{
			Pagination: &commonpb.PaginationRequest{
				Limit: 10,
				Method: &commonpb.PaginationRequest_Offset{
					Offset: &commonpb.OffsetPagination{Page: 1},
				},
			},
			Filters: &commonpb.FilterRequest{
				Logic: commonpb.FilterLogic_AND,
				Filters: []*commonpb.TypedFilter{
					{
						Field: "active",
						FilterType: &commonpb.TypedFilter_BooleanFilter{
							BooleanFilter: &commonpb.BooleanFilter{
								Value: true,
							},
						},
					},
					{
						Field: "balance_type",
						FilterType: &commonpb.TypedFilter_StringFilter{
							StringFilter: &commonpb.StringFilter{
								Operator: commonpb.StringOperator_STRING_EQUALS,
								Value:    "credit",
							},
						},
					},
					{
						Field: "client_id",
						FilterType: &commonpb.TypedFilter_StringFilter{
							StringFilter: &commonpb.StringFilter{
								Operator: commonpb.StringOperator_STRING_EQUALS,
								Value:    "client-1",
							},
						},
					},
				},
			},
		}

		resp, err := useCase.Execute(ctx, req)
		if err != nil {
			t.Fatalf("Execute failed: %v", err)
		}

		if !resp.Success {
			t.Error("Expected success=true")
		}

		// Should only get active credit balances for client-1
		for _, balance := range resp.BalanceList {
			if !balance.Active {
				t.Errorf("Expected only active balances, found inactive: %s", balance.Id)
			}
			if balance.BalanceType != "credit" {
				t.Errorf("Expected only credit balances, found: %s", balance.BalanceType)
			}
			if balance.ClientId != "client-1" {
				t.Errorf("Expected only client-1 balances, found: %s", balance.ClientId)
			}
		}
	})
}

func TestGetBalanceListPageDataUseCase_Execute_EmptyList(t *testing.T) {
	// Setup with empty repository
	ctx := context.Background()
	mockRepo := subscription.NewMockBalanceRepository("education")

	repos := GetBalanceListPageDataRepositories{
		Balance: mockRepo,
	}
	services := GetBalanceListPageDataServices{
		TransactionService: nil,
		TranslationService: nil,
	}

	useCase := NewGetBalanceListPageDataUseCase(repos, services)

	req := &balancepb.GetBalanceListPageDataRequest{
		Pagination: &commonpb.PaginationRequest{
			Limit: 10,
			Method: &commonpb.PaginationRequest_Offset{
				Offset: &commonpb.OffsetPagination{Page: 1},
			},
		},
	}

	resp, err := useCase.Execute(ctx, req)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !resp.Success {
		t.Error("Expected success=true")
	}

	if len(resp.BalanceList) != 0 {
		t.Errorf("Expected 0 balances for empty list, got %d", len(resp.BalanceList))
	}

	if resp.Pagination == nil {
		t.Error("Expected pagination response")
	} else if resp.Pagination.TotalItems != 0 {
		t.Errorf("Expected total items = 0, got %d", resp.Pagination.TotalItems)
	}
}

func TestGetBalanceListPageDataUseCase_Execute_ValidationErrors(t *testing.T) {
	// Setup
	ctx := context.Background()
	mockRepo := subscription.NewMockBalanceRepository("education")

	repos := GetBalanceListPageDataRepositories{
		Balance: mockRepo,
	}
	services := GetBalanceListPageDataServices{
		TransactionService: nil,
		TranslationService: nil,
	}

	useCase := NewGetBalanceListPageDataUseCase(repos, services)

	// Test case 1: Nil request
	t.Run("NilRequest", func(t *testing.T) {
		_, err := useCase.Execute(ctx, nil)
		if err == nil {
			t.Error("Expected error for nil request")
		}
	})

	// Test case 2: Invalid pagination limit
	t.Run("InvalidPaginationLimit", func(t *testing.T) {
		req := &balancepb.GetBalanceListPageDataRequest{
			Pagination: &commonpb.PaginationRequest{
				Limit: 150, // Too high
				Method: &commonpb.PaginationRequest_Offset{
					Offset: &commonpb.OffsetPagination{Page: 1},
				},
			},
		}

		_, err := useCase.Execute(ctx, req)
		if err == nil {
			t.Error("Expected error for invalid pagination limit")
		}
	})

	// Test case 3: Invalid filter field
	t.Run("InvalidFilterField", func(t *testing.T) {
		req := &balancepb.GetBalanceListPageDataRequest{
			Pagination: &commonpb.PaginationRequest{
				Limit: 10,
				Method: &commonpb.PaginationRequest_Offset{
					Offset: &commonpb.OffsetPagination{Page: 1},
				},
			},
			Filters: &commonpb.FilterRequest{
				Logic: commonpb.FilterLogic_AND,
				Filters: []*commonpb.TypedFilter{
					{
						Field: "invalid_field",
						FilterType: &commonpb.TypedFilter_StringFilter{
							StringFilter: &commonpb.StringFilter{
								Operator: commonpb.StringOperator_STRING_EQUALS,
								Value:    "test",
							},
						},
					},
				},
			},
		}

		_, err := useCase.Execute(ctx, req)
		if err == nil {
			t.Error("Expected error for invalid filter field")
		}
	})

	// Test case 4: Empty search query
	t.Run("EmptySearchQuery", func(t *testing.T) {
		req := &balancepb.GetBalanceListPageDataRequest{
			Pagination: &commonpb.PaginationRequest{
				Limit: 10,
				Method: &commonpb.PaginationRequest_Offset{
					Offset: &commonpb.OffsetPagination{Page: 1},
				},
			},
			Search: &commonpb.SearchRequest{
				Query: "", // Empty query
			},
		}

		_, err := useCase.Execute(ctx, req)
		if err == nil {
			t.Error("Expected error for empty search query")
		}
	})
}
