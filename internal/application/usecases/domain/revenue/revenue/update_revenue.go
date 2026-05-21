package revenue

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	revenuepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue"
)

// UpdateRevenueRepositories groups all repository dependencies
type UpdateRevenueRepositories struct {
	Revenue revenuepb.RevenueDomainServiceServer
}

// UpdateRevenueServices groups all business service dependencies
type UpdateRevenueServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
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
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityRevenue, ports.ActionUpdate); err != nil {
		return nil, err
	}

	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		var result *revenuepb.UpdateRevenueResponse
		err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
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
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "revenue.validation.id_required", "Revenue ID is required [DEFAULT]"))
	}

	// Set date_modified
	now := time.Now()
	req.Data.DateModified = &[]int64{now.UnixMilli()}[0]
	req.Data.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return uc.repositories.Revenue.UpdateRevenue(ctx, req)
}
