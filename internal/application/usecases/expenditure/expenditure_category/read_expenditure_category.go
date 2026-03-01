package expenditurecategory

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expenditure_category"
)

// ReadExpenditureCategoryRepositories groups all repository dependencies
type ReadExpenditureCategoryRepositories struct {
	ExpenditureCategory pb.ExpenditureCategoryDomainServiceServer
}

// ReadExpenditureCategoryServices groups all business service dependencies
type ReadExpenditureCategoryServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
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
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityExpenditureCategory, ports.ActionRead); err != nil {
		return nil, err
	}

	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "expenditure_category.validation.id_required", "Expenditure category ID is required [DEFAULT]"))
	}

	return uc.repositories.ExpenditureCategory.ReadExpenditureCategory(ctx, req)
}
