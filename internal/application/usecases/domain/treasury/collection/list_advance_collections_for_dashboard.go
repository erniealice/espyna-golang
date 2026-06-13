// Package collection — list_advance_collections_for_dashboard.go is the
// collection-side half of the Advances Dashboard (Plan B Phase 3 of
// 20260517-advance-cash-events). Originally lived in
// internal/application/usecases/treasury/get_advances_dashboard.go as a
// cross-entity use case backed by proto/v1/domain/treasury/advances_dashboard/.
//
// 20260518-hexagonal-strict-adherence Phase 1.C-iii relocates the collection
// half here so the proto contract + use case + caller all align on the
// collection entity boundary. The matching disbursement half lives at
// usecases/treasury/disbursement/list_advance_disbursements_for_dashboard.go.
package collection

import (
	"context"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"

	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	advancekindpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common/advance_kind"
	collectionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection"
)

// ListAdvanceCollectionsForDashboardRepositories groups the collection repo.
type ListAdvanceCollectionsForDashboardRepositories struct {
	Collection collectionpb.CollectionDomainServiceServer
}

// ListAdvanceCollectionsForDashboardServices groups infra services.
type ListAdvanceCollectionsForDashboardServices struct {
	Authorizer ports.Authorizer
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// ListAdvanceCollectionsForDashboardUseCase projects active selling-side
// advance rows (treasury_collection rows whose advance_kind != NONE) into the
// dashboard row shape, plus per-side totals + active/fully-recognized counts.
type ListAdvanceCollectionsForDashboardUseCase struct {
	repositories ListAdvanceCollectionsForDashboardRepositories
	services     ListAdvanceCollectionsForDashboardServices
}

// NewListAdvanceCollectionsForDashboardUseCase wires the use case.
func NewListAdvanceCollectionsForDashboardUseCase(
	repos ListAdvanceCollectionsForDashboardRepositories,
	svcs ListAdvanceCollectionsForDashboardServices,
) *ListAdvanceCollectionsForDashboardUseCase {
	return &ListAdvanceCollectionsForDashboardUseCase{repositories: repos, services: svcs}
}

// Execute returns the collection-side Advances Dashboard projection.
func (uc *ListAdvanceCollectionsForDashboardUseCase) Execute(
	ctx context.Context,
	req *collectionpb.ListAdvanceCollectionsForDashboardRequest,
) (*collectionpb.ListAdvanceCollectionsForDashboardResponse, error) {
	if req == nil {
		req = &collectionpb.ListAdvanceCollectionsForDashboardRequest{}
	}
	_ = req // request fields are reserved for future as-of filtering.

	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: "treasury_collection",
		Action: entityid.ActionList,
	}); err != nil {
		return nil, err
	}

	out := &collectionpb.ListAdvanceCollectionsForDashboardResponse{}
	if uc.repositories.Collection == nil {
		return out, nil
	}

	resp, err := uc.repositories.Collection.ListCollections(ctx, &collectionpb.ListCollectionsRequest{
		Filters: &commonpb.FilterRequest{},
	})
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return out, nil
	}
	for _, c := range resp.GetData() {
		if c.GetAdvanceKind() == advancekindpb.AdvanceKind_ADVANCE_KIND_NONE {
			continue
		}
		if !isActiveAdvanceForDashboard(c.GetAdvanceStatus()) {
			if c.GetAdvanceStatus() == advancekindpb.AdvanceStatus_ADVANCE_STATUS_FULLY_RECOGNIZED {
				out.FullyRecognizedCount++
			}
			continue
		}
		row := &collectionpb.AdvanceCollectionDashboardRow{
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
		row.UtilizationPct = advanceUtilizationPct(row.GetRecognizedAmount(), row.GetTotalAmount())
		out.Rows = append(out.Rows, row)
		out.TotalRemaining += row.GetRemainingAmount()
		out.ActiveCount++
	}
	return out, nil
}

// isActiveAdvanceForDashboard reports whether the status represents an "open"
// advance for dashboard purposes — ACTIVE or PARTIALLY_SETTLED.
func isActiveAdvanceForDashboard(s advancekindpb.AdvanceStatus) bool {
	switch s {
	case advancekindpb.AdvanceStatus_ADVANCE_STATUS_ACTIVE,
		advancekindpb.AdvanceStatus_ADVANCE_STATUS_PARTIALLY_SETTLED:
		return true
	}
	return false
}

// advanceUtilizationPct returns recognized/total as a 0-100 percentage. Safe
// on zero totals (returns 0).
func advanceUtilizationPct(recognized, total int64) float32 {
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
