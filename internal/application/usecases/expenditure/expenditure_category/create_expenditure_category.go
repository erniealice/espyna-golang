package expenditurecategory

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expenditure_category"
)

const entityExpenditureCategory = "expenditure_category"

// CreateExpenditureCategoryRepositories groups all repository dependencies
type CreateExpenditureCategoryRepositories struct {
	ExpenditureCategory pb.ExpenditureCategoryDomainServiceServer
}

// CreateExpenditureCategoryServices groups all business service dependencies
type CreateExpenditureCategoryServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// CreateExpenditureCategoryUseCase handles the business logic for creating expenditure categories
type CreateExpenditureCategoryUseCase struct {
	repositories CreateExpenditureCategoryRepositories
	services     CreateExpenditureCategoryServices
}

// NewCreateExpenditureCategoryUseCase creates use case with grouped dependencies
func NewCreateExpenditureCategoryUseCase(
	repositories CreateExpenditureCategoryRepositories,
	services CreateExpenditureCategoryServices,
) *CreateExpenditureCategoryUseCase {
	return &CreateExpenditureCategoryUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the create expenditure category operation
func (uc *CreateExpenditureCategoryUseCase) Execute(ctx context.Context, req *pb.CreateExpenditureCategoryRequest) (*pb.CreateExpenditureCategoryResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityExpenditureCategory, ports.ActionCreate); err != nil {
		return nil, err
	}

	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		var result *pb.CreateExpenditureCategoryResponse
		err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
			res, err := uc.executeCore(txCtx, req)
			if err != nil {
				return fmt.Errorf("expenditure category creation failed: %w", err)
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

func (uc *CreateExpenditureCategoryUseCase) executeCore(ctx context.Context, req *pb.CreateExpenditureCategoryRequest) (*pb.CreateExpenditureCategoryResponse, error) {
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "expenditure_category.validation.data_required", "Expenditure category data is required [DEFAULT]"))
	}

	now := time.Now()
	if req.Data.Id == "" {
		req.Data.Id = uc.services.IDService.GenerateID()
	}
	req.Data.DateCreated = &[]int64{now.UnixMilli()}[0]
	req.Data.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	req.Data.DateModified = &[]int64{now.UnixMilli()}[0]
	req.Data.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	req.Data.Active = true

	return uc.repositories.ExpenditureCategory.CreateExpenditureCategory(ctx, req)
}
