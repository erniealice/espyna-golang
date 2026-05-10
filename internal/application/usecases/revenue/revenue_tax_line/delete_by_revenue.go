package revenue_tax_line

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	revenuetaxlinepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue_tax_line"
)

// DeleteByRevenueRepositories groups repository dependencies.
type DeleteByRevenueRepositories struct {
	RevenueTaxLine revenuetaxlinepb.RevenueTaxLineDomainServiceServer
}

// DeleteByRevenueServices groups service dependencies.
type DeleteByRevenueServices struct {
	AuthorizationService ports.AuthorizationService
	TranslationService   ports.TranslationService
}

// DeleteByRevenueRevenueTaxLineUseCase deletes all revenue_tax_line rows for a revenue.
// Used internally by ComputeTaxesForRevenue for the idempotency DELETE + INSERT.
type DeleteByRevenueRevenueTaxLineUseCase struct {
	repositories DeleteByRevenueRepositories
	services     DeleteByRevenueServices
}

// NewDeleteByRevenueRevenueTaxLineUseCase creates the use case.
func NewDeleteByRevenueRevenueTaxLineUseCase(
	repositories DeleteByRevenueRepositories,
	services DeleteByRevenueServices,
) *DeleteByRevenueRevenueTaxLineUseCase {
	return &DeleteByRevenueRevenueTaxLineUseCase{repositories: repositories, services: services}
}

// Execute deletes all revenue_tax_line rows for the given revenue.
func (uc *DeleteByRevenueRevenueTaxLineUseCase) Execute(ctx context.Context, revenueID string) error {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityRevenueTaxLine, ports.ActionDelete); err != nil {
		return err
	}
	if revenueID == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"revenue_tax_line.validation.revenue_id_required", "Revenue ID is required [DEFAULT]"))
	}

	q, ok := uc.repositories.RevenueTaxLine.(RevenueTaxLineSystemWriteQueries)
	if !ok {
		return fmt.Errorf("revenue_tax_line repository does not support DeleteByRevenueID")
	}
	return q.DeleteByRevenueID(ctx, revenueID)
}
