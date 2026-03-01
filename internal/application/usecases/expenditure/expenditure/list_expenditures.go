package expenditure

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	expenditurepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expenditure"
)

// ListExpendituresRepositories groups all repository dependencies
type ListExpendituresRepositories struct {
	Expenditure expenditurepb.ExpenditureDomainServiceServer
}

// ListExpendituresServices groups all business service dependencies
type ListExpendituresServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ListExpendituresUseCase handles the business logic for listing expenditures
type ListExpendituresUseCase struct {
	repositories ListExpendituresRepositories
	services     ListExpendituresServices
}

// NewListExpendituresUseCase creates a new ListExpendituresUseCase
func NewListExpendituresUseCase(
	repositories ListExpendituresRepositories,
	services ListExpendituresServices,
) *ListExpendituresUseCase {
	return &ListExpendituresUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list expenditures operation
func (uc *ListExpendituresUseCase) Execute(ctx context.Context, req *expenditurepb.ListExpendituresRequest) (*expenditurepb.ListExpendituresResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityExpenditure, ports.ActionList); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "expenditure.validation.request_required", "Request is required [DEFAULT]"))
	}

	return uc.repositories.Expenditure.ListExpenditures(ctx, req)
}
