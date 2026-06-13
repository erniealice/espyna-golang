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

// ListExpendituresRepositories groups all repository dependencies
type ListExpendituresRepositories struct {
	Expenditure expenditurepb.ExpenditureDomainServiceServer
}

// ListExpendituresServices groups all business service dependencies
type ListExpendituresServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
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
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityExpenditure,
		Action: entityid.ActionList,
	}); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "expenditure.validation.request_required", "Request is required [DEFAULT]"))
	}

	return uc.repositories.Expenditure.ListExpenditures(ctx, req)
}
