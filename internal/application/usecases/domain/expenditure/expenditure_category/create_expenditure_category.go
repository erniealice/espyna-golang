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

const entityExpenditureCategory = "expenditure_category"

// CreateExpenditureCategoryRepositories groups all repository dependencies
type CreateExpenditureCategoryRepositories struct {
	ExpenditureCategory pb.ExpenditureCategoryDomainServiceServer
}

// CreateExpenditureCategoryServices groups all business service dependencies
type CreateExpenditureCategoryServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	IDGenerator ports.IDGenerator
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
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityExpenditureCategory, entityid.ActionCreate); err != nil {
		return nil, err
	}

	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		var result *pb.CreateExpenditureCategoryResponse
		err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
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
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "expenditure_category.validation.data_required", "Expenditure category data is required [DEFAULT]"))
	}

	now := time.Now()
	if req.Data.Id == "" {
		req.Data.Id = uc.services.IDGenerator.GenerateID()
	}
	req.Data.DateCreated = &[]int64{now.UnixMilli()}[0]
	req.Data.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	req.Data.DateModified = &[]int64{now.UnixMilli()}[0]
	req.Data.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	req.Data.Active = true

	return uc.repositories.ExpenditureCategory.CreateExpenditureCategory(ctx, req)
}
