package subscription

import (
	"context"
	"errors"
	"fmt"

	"leapfor.xyz/espyna/internal/application/ports"
	contextutil "leapfor.xyz/espyna/internal/application/shared/context"
	subscriptionpb "leapfor.xyz/esqyma/golang/v1/domain/subscription/subscription"
)

type ListSubscriptionsRepositories struct {
	Subscription subscriptionpb.SubscriptionDomainServiceServer
}

type ListSubscriptionsServices struct {
	TransactionService ports.TransactionService
	TranslationService ports.TranslationService
}

// ListSubscriptionsUseCase handles the business logic for listing subscriptions
type ListSubscriptionsUseCase struct {
	repositories ListSubscriptionsRepositories
	services     ListSubscriptionsServices
}

// NewListSubscriptionsUseCase creates a new ListSubscriptionsUseCase
func NewListSubscriptionsUseCase(
	repositories ListSubscriptionsRepositories,
	services ListSubscriptionsServices,
) *ListSubscriptionsUseCase {
	return &ListSubscriptionsUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list subscriptions operation
func (uc *ListSubscriptionsUseCase) Execute(ctx context.Context, req *subscriptionpb.ListSubscriptionsRequest) (*subscriptionpb.ListSubscriptionsResponse, error) {
	// Extract business type at start
	// businessType := uc.getBusinessTypeFromContext(ctx)

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Use transaction service if available
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes subscription listing within a transaction (or simply calls core if no transaction needed)
func (uc *ListSubscriptionsUseCase) executeWithTransaction(ctx context.Context, req *subscriptionpb.ListSubscriptionsRequest) (*subscriptionpb.ListSubscriptionsResponse, error) {
	var result *subscriptionpb.ListSubscriptionsResponse

	// For read operations, we might not strictly need a transaction, but we use the service for consistency
	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "subscription.errors.list_failed", "subscription listing failed: %w"), err)
		}
		result = res
		return nil
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// executeCore contains the core business logic for listing subscriptions
func (uc *ListSubscriptionsUseCase) executeCore(ctx context.Context, req *subscriptionpb.ListSubscriptionsRequest) (*subscriptionpb.ListSubscriptionsResponse, error) {
	resp, err := uc.repositories.Subscription.ListSubscriptions(ctx, req)
	if err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "subscription.errors.list_failed", "subscription listing failed: %w"), err)
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *ListSubscriptionsUseCase) validateInput(ctx context.Context, req *subscriptionpb.ListSubscriptionsRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "subscription.validation.request_required", "request is required"))
	}

	// Note: ListSubscriptionsRequest is empty in the protobuf definition
	// Additional filtering/pagination parameters can be validated here if added later

	return nil
}
