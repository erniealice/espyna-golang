package subscription

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	subscriptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription"
)

type ReadSubscriptionRepositories struct {
	Subscription subscriptionpb.SubscriptionDomainServiceServer
}

type ReadSubscriptionServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService ports.TransactionService
	TranslationService ports.TranslationService
}

// ReadSubscriptionUseCase handles the business logic for reading subscriptions
type ReadSubscriptionUseCase struct {
	repositories ReadSubscriptionRepositories
	services     ReadSubscriptionServices
}

// NewReadSubscriptionUseCase creates a new ReadSubscriptionUseCase
func NewReadSubscriptionUseCase(
	repositories ReadSubscriptionRepositories,
	services ReadSubscriptionServices,
) *ReadSubscriptionUseCase {
	return &ReadSubscriptionUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the read subscription operation
func (uc *ReadSubscriptionUseCase) Execute(ctx context.Context, req *subscriptionpb.ReadSubscriptionRequest) (*subscriptionpb.ReadSubscriptionResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntitySubscription, ports.ActionRead); err != nil {
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

// executeWithTransaction executes subscription reading within a transaction (or simply calls core if no transaction needed)
func (uc *ReadSubscriptionUseCase) executeWithTransaction(ctx context.Context, req *subscriptionpb.ReadSubscriptionRequest) (*subscriptionpb.ReadSubscriptionResponse, error) {
	var result *subscriptionpb.ReadSubscriptionResponse

	// For read operations, we might not strictly need a transaction, but we use the service for consistency
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

// executeCore contains the core business logic for reading a subscription
func (uc *ReadSubscriptionUseCase) executeCore(ctx context.Context, req *subscriptionpb.ReadSubscriptionRequest) (*subscriptionpb.ReadSubscriptionResponse, error) {
	// Call repository
	resp, err := uc.repositories.Subscription.ReadSubscription(ctx, req)
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "subscription.errors.not_found", ""))
	}
	if resp == nil || len(resp.Data) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "subscription.errors.not_found", ""))
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *ReadSubscriptionUseCase) validateInput(ctx context.Context, req *subscriptionpb.ReadSubscriptionRequest) error {
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

// validateBusinessRules enforces business constraints for reading subscriptions
func (uc *ReadSubscriptionUseCase) validateBusinessRules(ctx context.Context, subscription *subscriptionpb.Subscription) error {
	// Validate subscription ID format
	if len(subscription.Id) < 3 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "subscription.validation.id_too_short", ""))
	}

	return nil
}
