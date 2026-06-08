package expenditurecategory

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expenditure_category"
)

// ReadExpenditureCategoryRepositories groups all repository dependencies
type ReadExpenditureCategoryRepositories struct {
	ExpenditureCategory pb.ExpenditureCategoryDomainServiceServer
}

// ReadExpenditureCategoryServices groups all business service dependencies
type ReadExpenditureCategoryServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// ReadExpenditureCategoryUseCase handles the business logic for reading an expenditure category
type ReadExpenditureCategoryUseCase struct {
	repositories ReadExpenditureCategoryRepositories
	services     ReadExpenditureCategoryServices
}

// NewReadExpenditureCategoryUseCase creates use case with grouped dependencies
func NewReadExpenditureCategoryUseCase(
	repositories ReadExpenditureCategoryRepositories,
	services ReadExpenditureCategoryServices,
) *ReadExpenditureCategoryUseCase {
	return &ReadExpenditureCategoryUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the read expenditure category operation
func (uc *ReadExpenditureCategoryUseCase) Execute(ctx context.Context, req *pb.ReadExpenditureCategoryRequest) (*pb.ReadExpenditureCategoryResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityExpenditureCategory, entityid.ActionRead); err != nil {
		return nil, err
	}

	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "expenditure_category.validation.id_required", "Expenditure category ID is required [DEFAULT]"))
	}

	return uc.repositories.ExpenditureCategory.ReadExpenditureCategory(ctx, req)
}
