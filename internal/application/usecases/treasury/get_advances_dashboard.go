// Package treasury holds the workspace-level Advances Dashboard use case
// (Plan B Phase 3 — 20260517-advance-cash-events).
//
// The dashboard projects active treasury_collection + treasury_disbursement
// rows whose advance_kind != NONE into the centymo view's row shape, plus
// per-side totals + active/fully-recognized counts.
package treasury

import (
	"context"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"

	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	advancekindpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common/advance_kind"
	advancesdashboardpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/advances_dashboard"
	collectionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection"
	disbursementpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/disbursement"
)

// GetAdvancesDashboardRepositories groups the cross-domain repository deps.
type GetAdvancesDashboardRepositories struct {
	TreasuryCollection   collectionpb.CollectionDomainServiceServer
	TreasuryDisbursement disbursementpb.DisbursementDomainServiceServer
}

// GetAdvancesDashboardServices groups infra services.
type GetAdvancesDashboardServices struct {
	AuthorizationService ports.AuthorizationService
	TranslationService   ports.TranslationService
}

// GetAdvancesDashboardUseCase aggregates active advance rows from both
// treasury_collection (selling) + treasury_disbursement (buying) sides.
type GetAdvancesDashboardUseCase struct {
	repositories GetAdvancesDashboardRepositories
	services     GetAdvancesDashboardServices
}

// NewGetAdvancesDashboardUseCase wires the use case.
func NewGetAdvancesDashboardUseCase(
	repos GetAdvancesDashboardRepositories,
	svcs GetAdvancesDashboardServices,
) *GetAdvancesDashboardUseCase {
	return &GetAdvancesDashboardUseCase{repositories: repos, services: svcs}
}

// Execute returns the workspace-level Advances Dashboard projection.
//
// Implementation notes:
//   - Uses existing List adapter methods (no new adapter surface required).
//   - Filters to advance_kind != NONE in memory (the postgres adapter's filter
//     surface doesn't expose a NOT_EQUALS on enum fields uniformly).
//   - workspace_id filtering happens via the adapter's standard scope policy
//     when an authenticated session is in context.
func (uc *GetAdvancesDashboardUseCase) Execute(
	ctx context.Context,
	req *advancesdashboardpb.GetAdvancesDashboardRequest,
) (*advancesdashboardpb.GetAdvancesDashboardResponse, error) {
	if req == nil {
		req = &advancesdashboardpb.GetAdvancesDashboardRequest{}
	}
	_ = req // request fields are informational; reserved for future tz/as-of filtering.

	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		"treasury_collection", ports.ActionList); err != nil {
		return nil, err
	}
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		"treasury_disbursement", ports.ActionList); err != nil {
		return nil, err
	}

	out := &advancesdashboardpb.GetAdvancesDashboardResponse{}

	// Selling-side rows (Inflows on the workspace).
	if uc.repositories.TreasuryCollection != nil {
		resp, err := uc.repositories.TreasuryCollection.ListCollections(ctx, &collectionpb.ListCollectionsRequest{
			Filters: &commonpb.FilterRequest{},
		})
		if err != nil {
			return nil, err
		}
		if resp != nil {
			for _, c := range resp.GetData() {
				if c.GetAdvanceKind() == advancekindpb.AdvanceKind_ADVANCE_KIND_NONE {
					continue
				}
				if !isActiveAdvance(c.GetAdvanceStatus()) {
					if c.GetAdvanceStatus() == advancekindpb.AdvanceStatus_ADVANCE_STATUS_FULLY_RECOGNIZED {
						out.InflowFullyRecognized++
						out.FullyRecognizedCount++
					}
					continue
				}
				row := &advancesdashboardpb.AdvancesDashboardRow{
					AdvanceId:        c.GetId(),
					ReferenceNumber:  c.GetReferenceNumber(),
					CounterpartyName: c.GetName(),
					Kind:             c.GetAdvanceKind(),
					Status:           c.GetAdvanceStatus(),
					Currency:         c.GetCurrency(),
					TotalAmount:      c.GetAdvanceTotalAmount(),
					RemainingAmount:  c.GetAdvanceRemainingAmount(),
					RecognizedAmount: c.GetAdvanceRecognizedAmount(),
					StartDate:        c.GetAdvanceStartDate(),
					EndDate:          c.GetAdvanceEndDate(),
				}
				row.UtilizationPct = utilizationPct(row.GetRecognizedAmount(), row.GetTotalAmount())
				out.InflowRows = append(out.InflowRows, row)
				out.InflowTotalRemaining += row.GetRemainingAmount()
				out.InflowActiveCount++
				out.TotalPrepaid += row.GetRemainingAmount()
				out.ActiveCount++
			}
		}
	}

	// Buying-side rows (Outflows on the workspace).
	if uc.repositories.TreasuryDisbursement != nil {
		resp, err := uc.repositories.TreasuryDisbursement.ListDisbursements(ctx, &disbursementpb.ListDisbursementsRequest{
			Filters: &commonpb.FilterRequest{},
		})
		if err != nil {
			return nil, err
		}
		if resp != nil {
			for _, d := range resp.GetData() {
				if d.GetAdvanceKind() == advancekindpb.AdvanceKind_ADVANCE_KIND_NONE {
					continue
				}
				if !isActiveAdvance(d.GetAdvanceStatus()) {
					if d.GetAdvanceStatus() == advancekindpb.AdvanceStatus_ADVANCE_STATUS_FULLY_RECOGNIZED {
						out.OutflowFullyRecognized++
						out.FullyRecognizedCount++
					}
					continue
				}
				row := &advancesdashboardpb.AdvancesDashboardRow{
					AdvanceId:        d.GetId(),
					ReferenceNumber:  d.GetReferenceNumber(),
					CounterpartyName: d.GetName(),
					Kind:             d.GetAdvanceKind(),
					Status:           d.GetAdvanceStatus(),
					Currency:         d.GetCurrency(),
					TotalAmount:      d.GetAdvanceTotalAmount(),
					RemainingAmount:  d.GetAdvanceRemainingAmount(),
					RecognizedAmount: d.GetAdvanceRecognizedAmount(),
					StartDate:        d.GetAdvanceStartDate(),
					EndDate:          d.GetAdvanceEndDate(),
				}
				row.UtilizationPct = utilizationPct(row.GetRecognizedAmount(), row.GetTotalAmount())
				out.OutflowRows = append(out.OutflowRows, row)
				out.OutflowTotalRemaining += row.GetRemainingAmount()
				out.OutflowActiveCount++
				out.TotalDeferred += row.GetRemainingAmount()
				out.ActiveCount++
			}
		}
	}

	return out, nil
}

// isActiveAdvance reports whether the status represents an "open" advance for
// dashboard purposes — ACTIVE or PARTIALLY_SETTLED.
func isActiveAdvance(s advancekindpb.AdvanceStatus) bool {
	switch s {
	case advancekindpb.AdvanceStatus_ADVANCE_STATUS_ACTIVE,
		advancekindpb.AdvanceStatus_ADVANCE_STATUS_PARTIALLY_SETTLED:
		return true
	}
	return false
}

// utilizationPct returns recognized/total as a 0-100 percentage. Safe on zero
// totals (returns 0).
func utilizationPct(recognized, total int64) float32 {
	if total <= 0 {
		return 0
	}
	v := float32(recognized) * 100 / float32(total)
	if v < 0 {
		return 0
	}
	if v > 100 {
		return 100
	}
	return v
}
