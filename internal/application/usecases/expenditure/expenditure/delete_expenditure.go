package expenditure

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	expenditurepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expenditure"
)

// DeleteExpenditureRepositories groups all repository dependencies
type DeleteExpenditureRepositories struct {
	Expenditure expenditurepb.ExpenditureDomainServiceServer
}

// DeleteExpenditureServices groups all business service dependencies
type DeleteExpenditureServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// DeleteExpenditureUseCase handles the business logic for deleting expenditures
type DeleteExpenditureUseCase struct {
	repositories DeleteExpenditureRepositories
	services     DeleteExpenditureServices
}

// NewDeleteExpenditureUseCase creates a new DeleteExpenditureUseCase
func NewDeleteExpenditureUseCase(
	repositories DeleteExpenditureRepositories,
	services DeleteExpenditureServices,
) *DeleteExpenditureUseCase {
	return &DeleteExpenditureUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the delete expenditure operation
func (uc *DeleteExpenditureUseCase) Execute(ctx context.Context, req *expenditurepb.DeleteExpenditureRequest) (*expenditurepb.DeleteExpenditureResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityExpenditure, ports.ActionDelete); err != nil {
		return nil, err
	}

	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "expenditure.validation.id_required", "Expenditure ID is required [DEFAULT]"))
	}

	return uc.repositories.Expenditure.DeleteExpenditure(ctx, req)
}
