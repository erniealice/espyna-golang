package subscription

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	subscriptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription"
)

type DeleteSubscriptionRepositories struct {
	Subscription subscriptionpb.SubscriptionDomainServiceServer
}

type DeleteSubscriptionServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService ports.TransactionService
	TranslationService ports.TranslationService
}

// DeleteSubscriptionUseCase handles the business logic for deleting subscriptions
type DeleteSubscriptionUseCase struct {
	repositories DeleteSubscriptionRepositories
	services     DeleteSubscriptionServices
}

// NewDeleteSubscriptionUseCase creates a new DeleteSubscriptionUseCase
func NewDeleteSubscriptionUseCase(
	repositories DeleteSubscriptionRepositories,
	services DeleteSubscriptionServices,
) *DeleteSubscriptionUseCase {
	return &DeleteSubscriptionUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the delete subscription operation
func (uc *DeleteSubscriptionUseCase) Execute(ctx context.Context, req *subscriptionpb.DeleteSubscriptionRequest) (*subscriptionpb.DeleteSubscriptionResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntitySubscription, ports.ActionDelete); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		return nil, err
	}

	// Use transaction service if available
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes subscription deletion within a transaction
func (uc *DeleteSubscriptionUseCase) executeWithTransaction(ctx context.Context, req *subscriptionpb.DeleteSubscriptionRequest) (*subscriptionpb.DeleteSubscriptionResponse, error) {
	var result *subscriptionpb.DeleteSubscriptionResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return err
		}
		result = res
		return nil
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// executeCore contains the core business logic for deleting a subscription
func (uc *DeleteSubscriptionUseCase) executeCore(ctx context.Context, req *subscriptionpb.DeleteSubscriptionRequest) (*subscriptionpb.DeleteSubscriptionResponse, error) {
	// First, check if the subscription exists
	_, err := uc.repositories.Subscription.ReadSubscription(ctx, &subscriptionpb.ReadSubscriptionRequest{
		Data: &subscriptionpb.Subscription{Id: req.Data.Id},
	})
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "subscription.errors.not_found", ""))
	}

	resp, err := uc.repositories.Subscription.DeleteSubscription(ctx, req)
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "subscription.errors.deletion_failed", ""))
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *DeleteSubscriptionUseCase) validateInput(ctx context.Context, req *subscriptionpb.DeleteSubscriptionRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "subscription.validation.request_required", ""))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "subscription.validation.data_required", ""))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "subscription.validation.id_required", ""))
	}
	return nil
}

// validateBusinessRules enforces business constraints for deleting subscriptions
func (uc *DeleteSubscriptionUseCase) validateBusinessRules(ctx context.Context, subscription *subscriptionpb.Subscription) error {
	// Validate subscription ID format
	if len(subscription.Id) < 3 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "subscription.validation.id_too_short", ""))
	}

	return nil
}
