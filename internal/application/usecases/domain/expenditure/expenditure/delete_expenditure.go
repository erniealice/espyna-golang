package expenditure

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	expenditurepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expenditure"
)

// DeleteExpenditureRepositories groups all repository dependencies
type DeleteExpenditureRepositories struct {
	Expenditure expenditurepb.ExpenditureDomainServiceServer
}

// DeleteExpenditureServices groups all business service dependencies
type DeleteExpenditureServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
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
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityExpenditure,
		Action: entityid.ActionDelete,
	}); err != nil {
		return nil, err
	}

	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "expenditure.validation.id_required", "Expenditure ID is required [DEFAULT]"))
	}

	return uc.repositories.Expenditure.DeleteExpenditure(ctx, req)
}
