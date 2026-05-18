package prepayment

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	prepaymentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/prepayment"
)

// ListPrepaymentsRepositories groups all repository dependencies
type ListPrepaymentsRepositories struct {
	Prepayment prepaymentpb.PrepaymentDomainServiceServer
}

// ListPrepaymentsServices groups all business service dependencies
type ListPrepaymentsServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ListPrepaymentsUseCase handles the business logic for listing prepayments
type ListPrepaymentsUseCase struct {
	repositories ListPrepaymentsRepositories
	services     ListPrepaymentsServices
}

// NewListPrepaymentsUseCase creates a new ListPrepaymentsUseCase
func NewListPrepaymentsUseCase(
	repositories ListPrepaymentsRepositories,
	services ListPrepaymentsServices,
) *ListPrepaymentsUseCase {
	return &ListPrepaymentsUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list prepayments operation
func (uc *ListPrepaymentsUseCase) Execute(ctx context.Context, req *prepaymentpb.ListPrepaymentsRequest) (*prepaymentpb.ListPrepaymentsResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityPrepayment, ports.ActionList); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "prepayment.validation.request_required", "Request is required [DEFAULT]"))
	}

	if uc.repositories.Prepayment == nil {
		return nil, errors.New("prepayment repository is not available")
	}
	return uc.repositories.Prepayment.ListPrepayments(ctx, req)
}
