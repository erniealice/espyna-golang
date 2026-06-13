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

// ReadExpenditureRepositories groups all repository dependencies
type ReadExpenditureRepositories struct {
	Expenditure expenditurepb.ExpenditureDomainServiceServer
}

// ReadExpenditureServices groups all business service dependencies
type ReadExpenditureServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
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
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityExpenditure,
		Action: entityid.ActionRead,
	}); err != nil {
		return nil, err
	}

	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "expenditure.validation.id_required", "Expenditure ID is required [DEFAULT]"))
	}

	return uc.repositories.Expenditure.ReadExpenditure(ctx, req)
}
