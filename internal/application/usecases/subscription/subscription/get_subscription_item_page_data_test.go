//go:build mock_db && mock_auth

package subscription

import (
	"context"
	"testing"

	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock/subscription"
	subscriptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription"
)

func TestGetSubscriptionItemPageDataUseCase_Execute_Success(t *testing.T) {
	// Setup
	ctx := context.Background()

	// Create mock repository with test data
	mockRepo := subscription.NewMockSubscriptionRepository("education")

	// Create a test subscription first
	createResp, err := mockRepo.CreateSubscription(ctx, &subscriptionpb.CreateSubscriptionRequest{
		Data: &subscriptionpb.Subscription{
			Name:        "Test Subscription",
			PricePlanId: "plan-1",
			ClientId:    "client-1",
			Active:      true,
		},
	})
	if err != nil {
		t.Fatalf("Failed to create test subscription: %v", err)
	}

	subscriptionId := createResp.Data[0].Id

	// Setup repositories and services
	repos := GetSubscriptionItemPageDataRepositories{
		Subscription: mockRepo,
	}
	services := GetSubscriptionItemPageDataServices{
		TransactionService: nil, // No transaction for this test
		TranslationService: nil, // No translation for this test
	}

	// Create use case
	useCase := NewGetSubscriptionItemPageDataUseCase(repos, services)

	// Test successful retrieval
	req := &subscriptionpb.GetSubscriptionItemPageDataRequest{
		SubscriptionId: subscriptionId,
	}

	resp, err := useCase.Execute(ctx, req)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !resp.Success {
		t.Error("Expected success=true")
	}

	if resp.Subscription == nil {
		t.Fatal("Expected subscription data")
	}

	if resp.Subscription.Id != subscriptionId {
		t.Errorf("Expected subscription ID %s, got %s", subscriptionId, resp.Subscription.Id)
	}

	if resp.Subscription.Name != "Test Subscription" {
		t.Errorf("Expected subscription name 'Test Subscription', got %s", resp.Subscription.Name)
	}

	if resp.Subscription.PricePlanId != "plan-1" {
		t.Errorf("Expected price plan ID 'plan-1', got %s", resp.Subscription.PricePlanId)
	}

	if resp.Subscription.ClientId != "client-1" {
		t.Errorf("Expected client ID 'client-1', got %s", resp.Subscription.ClientId)
	}

	if !resp.Subscription.Active {
		t.Error("Expected subscription to be active")
	}
}

func TestGetSubscriptionItemPageDataUseCase_Execute_NotFound(t *testing.T) {
	// Setup with empty repository
	ctx := context.Background()
	mockRepo := subscription.NewMockSubscriptionRepository("education")

	repos := GetSubscriptionItemPageDataRepositories{
		Subscription: mockRepo,
	}
	services := GetSubscriptionItemPageDataServices{
		TransactionService: nil,
		TranslationService: nil,
	}

	useCase := NewGetSubscriptionItemPageDataUseCase(repos, services)

	// Test with non-existent subscription ID
	req := &subscriptionpb.GetSubscriptionItemPageDataRequest{
		SubscriptionId: "non-existent-id",
	}

	_, err := useCase.Execute(ctx, req)
	if err == nil {
		t.Error("Expected error for non-existent subscription")
	}
}

func TestGetSubscriptionItemPageDataUseCase_Execute_ValidationErrors(t *testing.T) {
	// Setup
	ctx := context.Background()
	mockRepo := subscription.NewMockSubscriptionRepository("education")

	repos := GetSubscriptionItemPageDataRepositories{
		Subscription: mockRepo,
	}
	services := GetSubscriptionItemPageDataServices{
		TransactionService: nil,
		TranslationService: nil,
	}

	useCase := NewGetSubscriptionItemPageDataUseCase(repos, services)

	// Test case 1: Nil request
	t.Run("NilRequest", func(t *testing.T) {
		_, err := useCase.Execute(ctx, nil)
		if err == nil {
			t.Error("Expected error for nil request")
		}
	})

	// Test case 2: Empty subscription ID
	t.Run("EmptySubscriptionId", func(t *testing.T) {
		req := &subscriptionpb.GetSubscriptionItemPageDataRequest{
			SubscriptionId: "",
		}

		_, err := useCase.Execute(ctx, req)
		if err == nil {
			t.Error("Expected error for empty subscription ID")
		}
	})

	// Test case 3: Short subscription ID
	t.Run("ShortSubscriptionId", func(t *testing.T) {
		req := &subscriptionpb.GetSubscriptionItemPageDataRequest{
			SubscriptionId: "ab", // Too short
		}

		_, err := useCase.Execute(ctx, req)
		if err == nil {
			t.Error("Expected error for too short subscription ID")
		}
	})
}
