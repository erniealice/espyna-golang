package billing_event

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	billingeventpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/billing_event"
)

// ListBillingEventsBySubscriptionRepositories groups all repository dependencies.
type ListBillingEventsBySubscriptionRepositories struct {
	BillingEvent billingeventpb.BillingEventDomainServiceServer
}

// ListBillingEventsBySubscriptionServices groups infra services.
type ListBillingEventsBySubscriptionServices struct {
	AuthorizationService ports.AuthorizationService
	TranslationService   ports.TranslationService
}

// ListBillingEventsBySubscriptionUseCase wraps the proto-domain ListBySubscription
// RPC behind a Layer-7 use case with auth-check. Phase 3 F7 closure — replaces
// the raw billingeventpb.BillingEventDomainServiceServer leak from
// SubscriptionUseCases.
type ListBillingEventsBySubscriptionUseCase struct {
	repositories ListBillingEventsBySubscriptionRepositories
	services     ListBillingEventsBySubscriptionServices
}

// NewListBillingEventsBySubscriptionUseCase wires the use case.
func NewListBillingEventsBySubscriptionUseCase(
	repositories ListBillingEventsBySubscriptionRepositories,
	services ListBillingEventsBySubscriptionServices,
) *ListBillingEventsBySubscriptionUseCase {
	return &ListBillingEventsBySubscriptionUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list operation.
func (uc *ListBillingEventsBySubscriptionUseCase) Execute(
	ctx context.Context, req *billingeventpb.ListBillingEventsBySubscriptionRequest,
) (*billingeventpb.ListBillingEventsBySubscriptionResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		"billing_event", ports.ActionList); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"billing_event.validation.request_required", "request is required"))
	}
	if uc.repositories.BillingEvent == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"billing_event.errors.repository_unavailable", "billing event repository not configured"))
	}
	return uc.repositories.BillingEvent.ListBySubscription(ctx, req)
}
