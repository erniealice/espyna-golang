package expenditureattribute

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expenditure_attribute"
)

// DeleteExpenditureAttributeRepositories groups all repository dependencies
type DeleteExpenditureAttributeRepositories struct {
	ExpenditureAttribute pb.ExpenditureAttributeDomainServiceServer
}

// DeleteExpenditureAttributeServices groups all business service dependencies
type DeleteExpenditureAttributeServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// DeleteExpenditureAttributeUseCase handles the business logic for deleting expenditure attributes
type DeleteExpenditureAttributeUseCase struct {
	repositories DeleteExpenditureAttributeRepositories
	services     DeleteExpenditureAttributeServices
}

// NewDeleteExpenditureAttributeUseCase creates a new DeleteExpenditureAttributeUseCase
func NewDeleteExpenditureAttributeUseCase(
	repositories DeleteExpenditureAttributeRepositories,
	services DeleteExpenditureAttributeServices,
) *DeleteExpenditureAttributeUseCase {
	return &DeleteExpenditureAttributeUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the delete expenditure attribute operation
func (uc *DeleteExpenditureAttributeUseCase) Execute(ctx context.Context, req *pb.DeleteExpenditureAttributeRequest) (*pb.DeleteExpenditureAttributeResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityExpenditureAttribute, ports.ActionDelete); err != nil {
		return nil, err
	}

	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "expenditure_attribute.validation.id_required", "Expenditure attribute ID is required [DEFAULT]"))
	}

	return uc.repositories.ExpenditureAttribute.DeleteExpenditureAttribute(ctx, req)
}
