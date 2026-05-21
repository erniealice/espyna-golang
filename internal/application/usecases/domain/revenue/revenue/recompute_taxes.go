package revenue

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	userctx "github.com/erniealice/espyna-golang/internal/application/shared/context"
	computepkg "github.com/erniealice/espyna-golang/internal/application/usecases/domain/tax/compute_taxes_for_revenue"

	revenuepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue"
)

// RecomputeTaxesRepositories groups repositories needed for tax recomputation.
type RecomputeTaxesRepositories struct {
	Revenue revenuepb.RevenueDomainServiceServer
}

// RecomputeTaxesServices groups services for tax recomputation.
type RecomputeTaxesServices struct {
	Authorizer ports.Authorizer
	Translator ports.Translator
}

// RecomputeTaxesRequest is the input to RecomputeTaxes.
type RecomputeTaxesRequest struct {
	// RevenueID is the revenue to recompute taxes for.
	RevenueID string
	// WorkspaceID is the owning workspace — required for multi-tenant guard.
	WorkspaceID string
	// AsOf, when set, overrides the revenue_date for historical recompute.
	// Zero value means use revenue.revenue_date.
	AsOf time.Time
}

// RecomputeTaxesResponse is the result of RecomputeTaxes.
type RecomputeTaxesResponse struct {
	LinesCount int
}

// RecomputeTaxesUseCase is the admin use case for recomputing taxes on a revenue.
// Recompute is a privileged, audited action — it requires a confirmed,
// non-settled revenue and no outstanding WithholdingCertificates.
// The blocking rules are enforced by ComputeTaxesForRevenue (IsRecompute=true).
type RecomputeTaxesUseCase struct {
	repositories RecomputeTaxesRepositories
	services     RecomputeTaxesServices
	compute      *computepkg.ComputeTaxesForRevenueUseCase
}

// NewRecomputeTaxesUseCase creates the use case.
func NewRecomputeTaxesUseCase(
	repositories RecomputeTaxesRepositories,
	services RecomputeTaxesServices,
	compute *computepkg.ComputeTaxesForRevenueUseCase,
) *RecomputeTaxesUseCase {
	return &RecomputeTaxesUseCase{
		repositories: repositories,
		services:     services,
		compute:      compute,
	}
}

// SetComputeTaxes installs the ComputeTaxesForRevenue use case after construction.
// Used by the composition layer to wire tax computation into the revenue domain
// without requiring a circular import at initialization time.
//
// Safe to call with nil — disables tax computation (no-op, no warning).
func (uc *RecomputeTaxesUseCase) SetComputeTaxes(compute *computepkg.ComputeTaxesForRevenueUseCase) {
	if uc == nil {
		return
	}
	uc.compute = compute
}

// Execute runs the admin tax recompute flow.
func (uc *RecomputeTaxesUseCase) Execute(
	ctx context.Context,
	req *RecomputeTaxesRequest,
) (*RecomputeTaxesResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		"revenue_tax_line", ports.ActionCreate); err != nil {
		return nil, err
	}

	if req == nil || req.RevenueID == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"revenue_tax_line.validation.revenue_id_required",
			"Revenue ID is required [DEFAULT]"))
	}

	// Resolve workspace_id: prefer request field → context.
	workspaceID := req.WorkspaceID
	if workspaceID == "" {
		workspaceID = userctx.ExtractWorkspaceIDFromContext(ctx)
	}
	if workspaceID == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"revenue_tax_line.validation.workspace_id_required",
			"Workspace ID is required for tax recomputation [DEFAULT]"))
	}

	if uc.compute == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"revenue_tax_line.errors.compute_unavailable",
			"Tax compute use case is not configured [DEFAULT]"))
	}

	res, err := uc.compute.Execute(ctx, &computepkg.ComputeTaxesRequest{
		RevenueID:   req.RevenueID,
		WorkspaceID: workspaceID,
		AsOf:        req.AsOf,
		IsRecompute: true,
	})
	if err != nil {
		return nil, fmt.Errorf("recompute_taxes: %w", err)
	}

	return &RecomputeTaxesResponse{LinesCount: len(res.Lines)}, nil
}
