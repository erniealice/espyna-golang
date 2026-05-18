package billing_event

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	billingeventpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/billing_event"
)

// SetBillingEventStatusRepositories groups all repository dependencies.
type SetBillingEventStatusRepositories struct {
	BillingEvent billingeventpb.BillingEventDomainServiceServer
}

// SetBillingEventStatusServices groups infra services.
type SetBillingEventStatusServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// SetBillingEventStatusUseCase wraps the proto-domain SetStatus RPC behind a
// Layer-7 use case with auth-check. Phase 3 F7 closure.
type SetBillingEventStatusUseCase struct {
	repositories SetBillingEventStatusRepositories
	services     SetBillingEventStatusServices
}

// NewSetBillingEventStatusUseCase wires the use case.
func NewSetBillingEventStatusUseCase(
	repositories SetBillingEventStatusRepositories,
	services SetBillingEventStatusServices,
) *SetBillingEventStatusUseCase {
	return &SetBillingEventStatusUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the SetStatus operation.
func (uc *SetBillingEventStatusUseCase) Execute(
	ctx context.Context, req *billingeventpb.SetBillingEventStatusRequest,
) (*billingeventpb.SetBillingEventStatusResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		"billing_event", ports.ActionUpdate); err != nil {
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
	return uc.repositories.BillingEvent.SetStatus(ctx, req)
}
