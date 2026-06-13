package expenditurecategory

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expenditure_category"
)

// DeleteExpenditureCategoryRepositories groups all repository dependencies
type DeleteExpenditureCategoryRepositories struct {
	ExpenditureCategory pb.ExpenditureCategoryDomainServiceServer
}

// DeleteExpenditureCategoryServices groups all business service dependencies
type DeleteExpenditureCategoryServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
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
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityExpenditureCategory,
		Action: entityid.ActionDelete,
	}); err != nil {
		return nil, err
	}

	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "expenditure_category.validation.id_required", "Expenditure category ID is required [DEFAULT]"))
	}

	return uc.repositories.ExpenditureCategory.DeleteExpenditureCategory(ctx, req)
}
