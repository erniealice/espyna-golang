package revenue_tax_line

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	revenuetaxlinepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue_tax_line"
)

// ListByRevenueIDQueries is the narrow interface for efficient per-revenue listing.
type ListByRevenueIDQueries interface {
	ListByRevenueID(ctx context.Context, revenueID string) ([]*revenuetaxlinepb.RevenueTaxLine, error)
}

// ListByRevenueRepositories groups repository dependencies.
type ListByRevenueRepositories struct {
	RevenueTaxLine revenuetaxlinepb.RevenueTaxLineDomainServiceServer
}

// ListByRevenueServices groups service dependencies.
type ListByRevenueServices struct {
	Authorizer ports.Authorizer
	Translator ports.Translator
}

// ListByRevenueRevenueTaxLineUseCase returns all revenue_tax_line rows for a revenue.
// Used by the revenue detail page to render the Taxes section.
type ListByRevenueRevenueTaxLineUseCase struct {
	repositories ListByRevenueRepositories
	services     ListByRevenueServices
}

// NewListByRevenueRevenueTaxLineUseCase creates the use case.
func NewListByRevenueRevenueTaxLineUseCase(
	repositories ListByRevenueRepositories,
	services ListByRevenueServices,
) *ListByRevenueRevenueTaxLineUseCase {
	return &ListByRevenueRevenueTaxLineUseCase{repositories: repositories, services: services}
}

// Execute returns all revenue_tax_line rows for the given revenue.
func (uc *ListByRevenueRevenueTaxLineUseCase) Execute(ctx context.Context, revenueID string) ([]*revenuetaxlinepb.RevenueTaxLine, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityRevenueTaxLine, entityid.ActionList); err != nil {
		return nil, err
	}
	if revenueID == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"revenue_tax_line.validation.revenue_id_required", "Revenue ID is required [DEFAULT]"))
	}

	// Prefer the narrow DB query interface.
	q, ok := uc.repositories.RevenueTaxLine.(ListByRevenueIDQueries)
	if ok {
		lines, err := q.ListByRevenueID(ctx, revenueID)
		if err != nil {
			return nil, fmt.Errorf("list revenue_tax_lines by revenue_id: %w", err)
		}
		return lines, nil
	}

	// Fall back to filtered list via the standard proto interface.
	resp, err := uc.repositories.RevenueTaxLine.ListRevenueTaxLines(ctx, &revenuetaxlinepb.ListRevenueTaxLinesRequest{
		Filters: &commonpb.FilterRequest{
			Filters: []*commonpb.TypedFilter{
				{
					Field: "revenue_id",
					FilterType: &commonpb.TypedFilter_StringFilter{
						StringFilter: &commonpb.StringFilter{
							Value:    revenueID,
							Operator: commonpb.StringOperator_STRING_EQUALS,
						},
					},
				},
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("list revenue_tax_lines (fallback): %w", err)
	}
	if resp == nil {
		return nil, nil
	}
	return resp.GetData(), nil
}
