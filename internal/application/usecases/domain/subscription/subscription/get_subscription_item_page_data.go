package subscription

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/registry/entityid"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	subscriptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription"
)

type GetSubscriptionItemPageDataRepositories struct {
	Subscription subscriptionpb.SubscriptionDomainServiceServer
}

type GetSubscriptionItemPageDataServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// GetSubscriptionItemPageDataUseCase handles the business logic for getting subscription item page data
type GetSubscriptionItemPageDataUseCase struct {
	repositories GetSubscriptionItemPageDataRepositories
	services     GetSubscriptionItemPageDataServices
}

// NewGetSubscriptionItemPageDataUseCase creates a new GetSubscriptionItemPageDataUseCase
func NewGetSubscriptionItemPageDataUseCase(
	repositories GetSubscriptionItemPageDataRepositories,
	services GetSubscriptionItemPageDataServices,
) *GetSubscriptionItemPageDataUseCase {
	return &GetSubscriptionItemPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get subscription item page data operation
func (uc *GetSubscriptionItemPageDataUseCase) Execute(
	ctx context.Context,
	req *subscriptionpb.GetSubscriptionItemPageDataRequest,
) (*subscriptionpb.GetSubscriptionItemPageDataResponse, error) {
	// Authorization check
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.Subscription,
		Action: entityid.ActionList,
	}); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req.SubscriptionId); err != nil {
		return nil, err
	}

	// Use transaction service if available
	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes subscription item page data retrieval within a transaction
func (uc *GetSubscriptionItemPageDataUseCase) executeWithTransaction(
	ctx context.Context,
	req *subscriptionpb.GetSubscriptionItemPageDataRequest,
) (*subscriptionpb.GetSubscriptionItemPageDataResponse, error) {
	var result *subscriptionpb.GetSubscriptionItemPageDataResponse

	err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
				txCtx,
				uc.services.Translator,
				"subscription.errors.item_page_data_failed",
				"subscription item page data retrieval failed: %w",
			), err)
		}
		result = res
		return nil
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// executeCore contains the core business logic for getting subscription item page data.
// Delegates to the repository's GetSubscriptionItemPageData so the joined Client
// (+ User) and PricePlan (+ Plan) come back populated — a plain ReadSubscription
// returns the bare row and would leave the detail page's Customer + Package
// fields empty.
func (uc *GetSubscriptionItemPageDataUseCase) executeCore(
	ctx context.Context,
	req *subscriptionpb.GetSubscriptionItemPageDataRequest,
) (*subscriptionpb.GetSubscriptionItemPageDataResponse, error) {
	itemResp, err := uc.repositories.Subscription.GetSubscriptionItemPageData(ctx, req)
	if err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.Translator,
			"subscription.errors.item_page_data_failed",
			"failed to retrieve subscription item page data: %w",
		), err)
	}

	if itemResp == nil || itemResp.GetSubscription() == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.Translator,
			"subscription.errors.not_found",
			"subscription not found",
		))
	}

	subscription := itemResp.GetSubscription()
	if subscription.GetId() != req.SubscriptionId {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.Translator,
			"subscription.errors.id_mismatch",
			"retrieved subscription ID does not match requested ID",
		))
	}

	return &subscriptionpb.GetSubscriptionItemPageDataResponse{
		Subscription: subscription,
		Success:      true,
	}, nil
}

// validateInput validates the input request
func (uc *GetSubscriptionItemPageDataUseCase) validateInput(
	ctx context.Context,
	req *subscriptionpb.GetSubscriptionItemPageDataRequest,
) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.Translator,
			"subscription.validation.request_required",
			"request is required",
		))
	}

	if req.SubscriptionId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.Translator,
			"subscription.validation.id_required",
			"subscription ID is required",
		))
	}

	return nil
}

// validateBusinessRules enforces business constraints for reading subscription item page data
func (uc *GetSubscriptionItemPageDataUseCase) validateBusinessRules(
	ctx context.Context,
	subscriptionId string,
) error {
	// Validate subscription ID format
	if len(subscriptionId) < 3 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.Translator,
			"subscription.validation.id_too_short",
			"subscription ID is too short",
		))
	}

	// Additional business rules could be added here:
	// - Check user permissions to access this subscription
	// - Validate subscription belongs to the current user's organization
	// - Check if subscription is in a state that allows viewing
	// - Rate limiting for subscription access
	// - Audit logging requirements

	return nil
}

// Optional: Helper methods for future enhancements

// loadRelatedData loads related entities like plan and client details
// This would be called from executeCore if needed
func (uc *GetSubscriptionItemPageDataUseCase) loadRelatedData(
	ctx context.Context,
	subscription *subscriptionpb.Subscription,
) error {
	// TODO: Implement loading of related data
	// This could involve calls to plan and client repositories
	// to populate the nested plan and client objects if they're not already loaded

	// Example implementation would be:
	// if subscription.Plan == nil && subscription.PlanId != "" {
	//     // Load plan data
	// }
	// if subscription.Client == nil && subscription.ClientId != "" {
	//     // Load client data
	// }

	return nil
}

// applyDataTransformation applies any necessary data transformations for the frontend
func (uc *GetSubscriptionItemPageDataUseCase) applyDataTransformation(
	ctx context.Context,
	subscription *subscriptionpb.Subscription,
) *subscriptionpb.Subscription {
	// TODO: Apply any transformations needed for optimal frontend consumption
	// This could include:
	// - Formatting dates
	// - Computing derived fields
	// - Applying localization
	// - Sanitizing sensitive data

	return subscription
}

// checkAccessPermissions validates user has permission to access this subscription
func (uc *GetSubscriptionItemPageDataUseCase) checkAccessPermissions(
	ctx context.Context,
	subscriptionId string,
) error {
	// TODO: Implement proper access control
	// This could involve:
	// - Checking user role/permissions
	// - Validating subscription belongs to user's organization
	// - Applying multi-tenant access controls

	return nil
}
