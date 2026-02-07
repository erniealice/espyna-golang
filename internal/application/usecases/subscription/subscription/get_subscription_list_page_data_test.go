//go:build mock_db && mock_auth

package subscription

import (
	"context"
	"testing"

	"leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock/subscription"
	commonpb "leapfor.xyz/esqyma/golang/v1/domain/common"
	subscriptionpb "leapfor.xyz/esqyma/golang/v1/domain/subscription/subscription"
)

func TestGetSubscriptionListPageDataUseCase_Execute_Success(t *testing.T) {
	// Setup
	ctx := context.Background()

	// Create mock repository with some test data
	mockRepo := subscription.NewMockSubscriptionRepository("education")

	// Create some test subscriptions first
	subscription1, err := mockRepo.CreateSubscription(ctx, &subscriptionpb.CreateSubscriptionRequest{
		Data: &subscriptionpb.Subscription{
			Name:        "Test Subscription 1",
			PricePlanId: "plan-1",
			ClientId:    "client-1",
			Active:      true,
		},
	})
	if err != nil {
		t.Fatalf("Failed to create test subscription 1: %v", err)
	}

	_, err = mockRepo.CreateSubscription(ctx, &subscriptionpb.CreateSubscriptionRequest{
		Data: &subscriptionpb.Subscription{
			Name:        "Test Subscription 2",
			PricePlanId: "plan-2",
			ClientId:    "client-2",
			Active:      false,
		},
	})
	if err != nil {
		t.Fatalf("Failed to create test subscription 2: %v", err)
	}

	// Setup repositories and services
	repos := GetSubscriptionListPageDataRepositories{
		Subscription: mockRepo,
	}
	services := GetSubscriptionListPageDataServices{
		TransactionService: nil, // No transaction for this test
		TranslationService: nil, // No translation for this test
	}

	// Create use case
	useCase := NewGetSubscriptionListPageDataUseCase(repos, services)

	// Test case 1: Basic list without filters
	t.Run("BasicList", func(t *testing.T) {
		req := &subscriptionpb.GetSubscriptionListPageDataRequest{
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

		if len(resp.SubscriptionList) < 2 {
			t.Errorf("Expected at least 2 subscriptions, got %d", len(resp.SubscriptionList))
		}

		if resp.Pagination == nil {
			t.Error("Expected pagination response")
		}
	})

	// Test case 2: Filtering by active status
	t.Run("FilterByActive", func(t *testing.T) {
		req := &subscriptionpb.GetSubscriptionListPageDataRequest{
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

		// Should only get active subscriptions
		for _, sub := range resp.SubscriptionList {
			if !sub.Active {
				t.Errorf("Expected only active subscriptions, found inactive: %s", sub.Id)
			}
		}
	})

	// Test case 3: Search by name
	t.Run("SearchByName", func(t *testing.T) {
		req := &subscriptionpb.GetSubscriptionListPageDataRequest{
			Pagination: &commonpb.PaginationRequest{
				Limit: 10,
				Method: &commonpb.PaginationRequest_Offset{
					Offset: &commonpb.OffsetPagination{Page: 1},
				},
			},
			Search: &commonpb.SearchRequest{
				Query: "Test Subscription 1",
				Options: &commonpb.SearchOptions{
					SearchFields: []string{"name"},
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

		// Should find at least the first subscription
		found := false
		for _, sub := range resp.SubscriptionList {
			if sub.Id == subscription1.Data[0].Id {
				found = true
				break
			}
		}

		if !found {
			t.Error("Expected to find subscription1 in search results")
		}
	})

	// Test case 4: Sorting by name
	t.Run("SortByName", func(t *testing.T) {
		req := &subscriptionpb.GetSubscriptionListPageDataRequest{
			Pagination: &commonpb.PaginationRequest{
				Limit: 10,
				Method: &commonpb.PaginationRequest_Offset{
					Offset: &commonpb.OffsetPagination{Page: 1},
				},
			},
			Sort: &commonpb.SortRequest{
				Fields: []*commonpb.SortField{
					{
						Field:     "name",
						Direction: commonpb.SortDirection_ASC,
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

		// Verify sorting - names should be in ascending order
		if len(resp.SubscriptionList) >= 2 {
			for i := 1; i < len(resp.SubscriptionList); i++ {
				if resp.SubscriptionList[i-1].Name > resp.SubscriptionList[i].Name {
					t.Errorf("Subscriptions not sorted properly: %s > %s",
						resp.SubscriptionList[i-1].Name, resp.SubscriptionList[i].Name)
				}
			}
		}
	})

	// Test case 5: Pagination
	t.Run("Pagination", func(t *testing.T) {
		req := &subscriptionpb.GetSubscriptionListPageDataRequest{
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

		if len(resp.SubscriptionList) != 1 {
			t.Errorf("Expected exactly 1 subscription with limit=1, got %d", len(resp.SubscriptionList))
		}

		if resp.Pagination == nil {
			t.Error("Expected pagination response")
		} else {
			if resp.Pagination.TotalItems < 2 {
				t.Errorf("Expected total items >= 2, got %d", resp.Pagination.TotalItems)
			}
		}
	})
}

func TestGetSubscriptionListPageDataUseCase_Execute_EmptyList(t *testing.T) {
	// Setup with empty repository
	ctx := context.Background()
	mockRepo := subscription.NewMockSubscriptionRepository("education")

	repos := GetSubscriptionListPageDataRepositories{
		Subscription: mockRepo,
	}
	services := GetSubscriptionListPageDataServices{
		TransactionService: nil,
		TranslationService: nil,
	}

	useCase := NewGetSubscriptionListPageDataUseCase(repos, services)

	req := &subscriptionpb.GetSubscriptionListPageDataRequest{
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

	if len(resp.SubscriptionList) != 0 {
		t.Errorf("Expected 0 subscriptions for empty list, got %d", len(resp.SubscriptionList))
	}

	if resp.Pagination == nil {
		t.Error("Expected pagination response")
	} else if resp.Pagination.TotalItems != 0 {
		t.Errorf("Expected total items = 0, got %d", resp.Pagination.TotalItems)
	}
}
