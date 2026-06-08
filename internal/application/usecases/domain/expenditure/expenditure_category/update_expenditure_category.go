package expenditurecategory

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expenditure_category"
)

// UpdateExpenditureCategoryRepositories groups all repository dependencies
type UpdateExpenditureCategoryRepositories struct {
	ExpenditureCategory pb.ExpenditureCategoryDomainServiceServer
}

// UpdateExpenditureCategoryServices groups all business service dependencies
type UpdateExpenditureCategoryServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// UpdateExpenditureCategoryUseCase handles the business logic for updating expenditure categories
type UpdateExpenditureCategoryUseCase struct {
	repositories UpdateExpenditureCategoryRepositories
	services     UpdateExpenditureCategoryServices
}

// NewUpdateExpenditureCategoryUseCase creates use case with grouped dependencies
func NewUpdateExpenditureCategoryUseCase(
	repositories UpdateExpenditureCategoryRepositories,
	services UpdateExpenditureCategoryServices,
) *UpdateExpenditureCategoryUseCase {
	return &UpdateExpenditureCategoryUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the update expenditure category operation
func (uc *UpdateExpenditureCategoryUseCase) Execute(ctx context.Context, req *pb.UpdateExpenditureCategoryRequest) (*pb.UpdateExpenditureCategoryResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityExpenditureCategory, entityid.ActionUpdate); err != nil {
		return nil, err
	}

	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		var result *pb.UpdateExpenditureCategoryResponse
		err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
			res, err := uc.executeCore(txCtx, req)
			if err != nil {
				return fmt.Errorf("expenditure category update failed: %w", err)
			}
			result = res
			return nil
		})
		if err != nil {
			return nil, err
		}
		return result, nil
	}

	return uc.executeCore(ctx, req)
}

func (uc *UpdateExpenditureCategoryUseCase) executeCore(ctx context.Context, req *pb.UpdateExpenditureCategoryRequest) (*pb.UpdateExpenditureCategoryResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "expenditure_category.validation.id_required", "Expenditure category ID is required [DEFAULT]"))
	}

	now := time.Now()
	req.Data.DateModified = &[]int64{now.UnixMilli()}[0]
	req.Data.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return uc.repositories.ExpenditureCategory.UpdateExpenditureCategory(ctx, req)
}
