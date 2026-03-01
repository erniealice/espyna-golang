package expenditureattribute

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expenditure_attribute"
)

// ListExpenditureAttributesRepositories groups all repository dependencies
type ListExpenditureAttributesRepositories struct {
	ExpenditureAttribute pb.ExpenditureAttributeDomainServiceServer
}

// ListExpenditureAttributesServices groups all business service dependencies
type ListExpenditureAttributesServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ListExpenditureAttributesUseCase handles the business logic for listing expenditure attributes
type ListExpenditureAttributesUseCase struct {
	repositories ListExpenditureAttributesRepositories
	services     ListExpenditureAttributesServices
}

// NewListExpenditureAttributesUseCase creates a new ListExpenditureAttributesUseCase
func NewListExpenditureAttributesUseCase(
	repositories ListExpenditureAttributesRepositories,
	services ListExpenditureAttributesServices,
) *ListExpenditureAttributesUseCase {
	return &ListExpenditureAttributesUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list expenditure attributes operation
func (uc *ListExpenditureAttributesUseCase) Execute(ctx context.Context, req *pb.ListExpenditureAttributesRequest) (*pb.ListExpenditureAttributesResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityExpenditureAttribute, ports.ActionList); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "expenditure_attribute.validation.request_required", "Request is required [DEFAULT]"))
	}

	return uc.repositories.ExpenditureAttribute.ListExpenditureAttributes(ctx, req)
}
