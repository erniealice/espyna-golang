package expenditurecategory

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expenditure_category"
)

// ListExpenditureCategoriesRepositories groups all repository dependencies
type ListExpenditureCategoriesRepositories struct {
	ExpenditureCategory pb.ExpenditureCategoryDomainServiceServer
}

// ListExpenditureCategoriesServices groups all business service dependencies
type ListExpenditureCategoriesServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// ListExpenditureCategoriesUseCase handles the business logic for listing expenditure categories
type ListExpenditureCategoriesUseCase struct {
	repositories ListExpenditureCategoriesRepositories
	services     ListExpenditureCategoriesServices
}

// NewListExpenditureCategoriesUseCase creates a new ListExpenditureCategoriesUseCase
func NewListExpenditureCategoriesUseCase(
	repositories ListExpenditureCategoriesRepositories,
	services ListExpenditureCategoriesServices,
) *ListExpenditureCategoriesUseCase {
	return &ListExpenditureCategoriesUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list expenditure categories operation
func (uc *ListExpenditureCategoriesUseCase) Execute(ctx context.Context, req *pb.ListExpenditureCategoriesRequest) (*pb.ListExpenditureCategoriesResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityExpenditureCategory, ports.ActionList); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "expenditure_category.validation.request_required", "Request is required [DEFAULT]"))
	}

	return uc.repositories.ExpenditureCategory.ListExpenditureCategories(ctx, req)
}
