package revenuecategory

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue_category"
)

const entityRevenueCategory = "revenue_category"

// CreateRevenueCategoryRepositories groups all repository dependencies
type CreateRevenueCategoryRepositories struct {
	RevenueCategory pb.RevenueCategoryDomainServiceServer
}

// CreateRevenueCategoryServices groups all business service dependencies
type CreateRevenueCategoryServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
}

// CreateRevenueCategoryUseCase handles the business logic for creating revenue categories
type CreateRevenueCategoryUseCase struct {
	repositories CreateRevenueCategoryRepositories
	services     CreateRevenueCategoryServices
}

// NewCreateRevenueCategoryUseCase creates use case with grouped dependencies
func NewCreateRevenueCategoryUseCase(
	repositories CreateRevenueCategoryRepositories,
	services CreateRevenueCategoryServices,
) *CreateRevenueCategoryUseCase {
	return &CreateRevenueCategoryUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the create revenue category operation
func (uc *CreateRevenueCategoryUseCase) Execute(ctx context.Context, req *pb.CreateRevenueCategoryRequest) (*pb.CreateRevenueCategoryResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityRevenueCategory,
		Action: entityid.ActionCreate,
	}); err != nil {
		return nil, err
	}

	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		var result *pb.CreateRevenueCategoryResponse
		err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
			res, err := uc.executeCore(txCtx, req)
			if err != nil {
				return fmt.Errorf("revenue category creation failed: %w", err)
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

func (uc *CreateRevenueCategoryUseCase) executeCore(ctx context.Context, req *pb.CreateRevenueCategoryRequest) (*pb.CreateRevenueCategoryResponse, error) {
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "revenue_category.validation.data_required", "Revenue category data is required [DEFAULT]"))
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

	return uc.repositories.RevenueCategory.CreateRevenueCategory(ctx, req)
}
