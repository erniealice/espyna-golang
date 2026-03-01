package expenditure

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	expenditurepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expenditure"
)

// ReadExpenditureRepositories groups all repository dependencies
type ReadExpenditureRepositories struct {
	Expenditure expenditurepb.ExpenditureDomainServiceServer
}

// ReadExpenditureServices groups all business service dependencies
type ReadExpenditureServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ReadExpenditureUseCase handles the business logic for reading an expenditure
type ReadExpenditureUseCase struct {
	repositories ReadExpenditureRepositories
	services     ReadExpenditureServices
}

// NewReadExpenditureUseCase creates use case with grouped dependencies
func NewReadExpenditureUseCase(
	repositories ReadExpenditureRepositories,
	services ReadExpenditureServices,
) *ReadExpenditureUseCase {
	return &ReadExpenditureUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the read expenditure operation
func (uc *ReadExpenditureUseCase) Execute(ctx context.Context, req *expenditurepb.ReadExpenditureRequest) (*expenditurepb.ReadExpenditureResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityExpenditure, ports.ActionRead); err != nil {
		return nil, err
	}

	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "expenditure.validation.id_required", "Expenditure ID is required [DEFAULT]"))
	}

	return uc.repositories.Expenditure.ReadExpenditure(ctx, req)
}
