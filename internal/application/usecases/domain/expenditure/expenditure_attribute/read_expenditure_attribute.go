package expenditureattribute

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expenditure_attribute"
)

// ReadExpenditureAttributeRepositories groups all repository dependencies
type ReadExpenditureAttributeRepositories struct {
	ExpenditureAttribute pb.ExpenditureAttributeDomainServiceServer
}

// ReadExpenditureAttributeServices groups all business service dependencies
type ReadExpenditureAttributeServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// ReadExpenditureAttributeUseCase handles the business logic for reading an expenditure attribute
type ReadExpenditureAttributeUseCase struct {
	repositories ReadExpenditureAttributeRepositories
	services     ReadExpenditureAttributeServices
}

// NewReadExpenditureAttributeUseCase creates use case with grouped dependencies
func NewReadExpenditureAttributeUseCase(
	repositories ReadExpenditureAttributeRepositories,
	services ReadExpenditureAttributeServices,
) *ReadExpenditureAttributeUseCase {
	return &ReadExpenditureAttributeUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the read expenditure attribute operation
func (uc *ReadExpenditureAttributeUseCase) Execute(ctx context.Context, req *pb.ReadExpenditureAttributeRequest) (*pb.ReadExpenditureAttributeResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityExpenditureAttribute, ports.ActionRead); err != nil {
		return nil, err
	}

	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "expenditure_attribute.validation.id_required", "Expenditure attribute ID is required [DEFAULT]"))
	}

	return uc.repositories.ExpenditureAttribute.ReadExpenditureAttribute(ctx, req)
}
