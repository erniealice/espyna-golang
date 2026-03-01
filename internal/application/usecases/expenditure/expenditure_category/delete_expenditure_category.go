package expenditurecategory

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expenditure_category"
)

// DeleteExpenditureCategoryRepositories groups all repository dependencies
type DeleteExpenditureCategoryRepositories struct {
	ExpenditureCategory pb.ExpenditureCategoryDomainServiceServer
}

// DeleteExpenditureCategoryServices groups all business service dependencies
type DeleteExpenditureCategoryServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// DeleteExpenditureCategoryUseCase handles the business logic for deleting expenditure categories
type DeleteExpenditureCategoryUseCase struct {
	repositories DeleteExpenditureCategoryRepositories
	services     DeleteExpenditureCategoryServices
}

// NewDeleteExpenditureCategoryUseCase creates a new DeleteExpenditureCategoryUseCase
func NewDeleteExpenditureCategoryUseCase(
	repositories DeleteExpenditureCategoryRepositories,
	services DeleteExpenditureCategoryServices,
) *DeleteExpenditureCategoryUseCase {
	return &DeleteExpenditureCategoryUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the delete expenditure category operation
func (uc *DeleteExpenditureCategoryUseCase) Execute(ctx context.Context, req *pb.DeleteExpenditureCategoryRequest) (*pb.DeleteExpenditureCategoryResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityExpenditureCategory, ports.ActionDelete); err != nil {
		return nil, err
	}

	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "expenditure_category.validation.id_required", "Expenditure category ID is required [DEFAULT]"))
	}

	return uc.repositories.ExpenditureCategory.DeleteExpenditureCategory(ctx, req)
}
