package expenditureattribute

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expenditure_attribute"
)

// ReadExpenditureAttributeRepositories groups all repository dependencies
type ReadExpenditureAttributeRepositories struct {
	ExpenditureAttribute pb.ExpenditureAttributeDomainServiceServer
}

// ReadExpenditureAttributeServices groups all business service dependencies
type ReadExpenditureAttributeServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
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
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityExpenditureAttribute, ports.ActionRead); err != nil {
		return nil, err
	}

	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "expenditure_attribute.validation.id_required", "Expenditure attribute ID is required [DEFAULT]"))
	}

	return uc.repositories.ExpenditureAttribute.ReadExpenditureAttribute(ctx, req)
}
