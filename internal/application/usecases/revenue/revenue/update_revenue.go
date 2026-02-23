package revenue

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	revenuepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue"
)

// UpdateRevenueRepositories groups all repository dependencies
type UpdateRevenueRepositories struct {
	Revenue revenuepb.RevenueDomainServiceServer
}

// UpdateRevenueServices groups all business service dependencies
type UpdateRevenueServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// UpdateRevenueUseCase handles the business logic for updating revenues
type UpdateRevenueUseCase struct {
	repositories UpdateRevenueRepositories
	services     UpdateRevenueServices
}

// NewUpdateRevenueUseCase creates use case with grouped dependencies
func NewUpdateRevenueUseCase(
	repositories UpdateRevenueRepositories,
	services UpdateRevenueServices,
) *UpdateRevenueUseCase {
	return &UpdateRevenueUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the update revenue operation
func (uc *UpdateRevenueUseCase) Execute(ctx context.Context, req *revenuepb.UpdateRevenueRequest) (*revenuepb.UpdateRevenueResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityRevenue, ports.ActionUpdate); err != nil {
		return nil, err
	}

	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		var result *revenuepb.UpdateRevenueResponse
		err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
			res, err := uc.executeCore(txCtx, req)
			if err != nil {
				return fmt.Errorf("revenue update failed: %w", err)
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

func (uc *UpdateRevenueUseCase) executeCore(ctx context.Context, req *revenuepb.UpdateRevenueRequest) (*revenuepb.UpdateRevenueResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "revenue.validation.id_required", "Revenue ID is required [DEFAULT]"))
	}

	// Set date_modified
	now := time.Now()
	req.Data.DateModified = &[]int64{now.UnixMilli()}[0]
	req.Data.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return uc.repositories.Revenue.UpdateRevenue(ctx, req)
}
