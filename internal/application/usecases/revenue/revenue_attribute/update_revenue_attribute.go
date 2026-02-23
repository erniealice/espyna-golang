package revenueattribute

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue_attribute"
)

// UpdateRevenueAttributeRepositories groups all repository dependencies
type UpdateRevenueAttributeRepositories struct {
	RevenueAttribute pb.RevenueAttributeDomainServiceServer
}

// UpdateRevenueAttributeServices groups all business service dependencies
type UpdateRevenueAttributeServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// UpdateRevenueAttributeUseCase handles the business logic for updating revenue attributes
type UpdateRevenueAttributeUseCase struct {
	repositories UpdateRevenueAttributeRepositories
	services     UpdateRevenueAttributeServices
}

// NewUpdateRevenueAttributeUseCase creates use case with grouped dependencies
func NewUpdateRevenueAttributeUseCase(
	repositories UpdateRevenueAttributeRepositories,
	services UpdateRevenueAttributeServices,
) *UpdateRevenueAttributeUseCase {
	return &UpdateRevenueAttributeUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the update revenue attribute operation
func (uc *UpdateRevenueAttributeUseCase) Execute(ctx context.Context, req *pb.UpdateRevenueAttributeRequest) (*pb.UpdateRevenueAttributeResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityRevenueAttribute, ports.ActionUpdate); err != nil {
		return nil, err
	}

	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		var result *pb.UpdateRevenueAttributeResponse
		err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
			res, err := uc.executeCore(txCtx, req)
			if err != nil {
				return fmt.Errorf("revenue attribute update failed: %w", err)
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

func (uc *UpdateRevenueAttributeUseCase) executeCore(ctx context.Context, req *pb.UpdateRevenueAttributeRequest) (*pb.UpdateRevenueAttributeResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "revenue_attribute.validation.id_required", "Revenue attribute ID is required [DEFAULT]"))
	}

	now := time.Now()
	req.Data.DateModified = &[]int64{now.UnixMilli()}[0]
	req.Data.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return uc.repositories.RevenueAttribute.UpdateRevenueAttribute(ctx, req)
}
