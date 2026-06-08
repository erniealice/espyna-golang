// Package disbursement — list_advance_disbursements_for_dashboard.go is the
// disbursement-side half of the Advances Dashboard. See the matching
// collection/list_advance_collections_for_dashboard.go for context.
package disbursement

import (
	"context"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	"github.com/erniealice/espyna-golang/registry/entityid"

	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	advancekindpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common/advance_kind"
	disbursementpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/disbursement"
)

// ListAdvanceDisbursementsForDashboardRepositories groups the disbursement repo.
type ListAdvanceDisbursementsForDashboardRepositories struct {
	Disbursement disbursementpb.DisbursementDomainServiceServer
}

// ListAdvanceDisbursementsForDashboardServices groups infra services.
type ListAdvanceDisbursementsForDashboardServices struct {
	Authorizer ports.Authorizer
	Translator ports.Translator
}

// ListAdvanceDisbursementsForDashboardUseCase projects active buying-side
// advance rows into the dashboard row shape.
type ListAdvanceDisbursementsForDashboardUseCase struct {
	repositories ListAdvanceDisbursementsForDashboardRepositories
	services     ListAdvanceDisbursementsForDashboardServices
}

// NewListAdvanceDisbursementsForDashboardUseCase wires the use case.
func NewListAdvanceDisbursementsForDashboardUseCase(
	repos ListAdvanceDisbursementsForDashboardRepositories,
	svcs ListAdvanceDisbursementsForDashboardServices,
) *ListAdvanceDisbursementsForDashboardUseCase {
	return &ListAdvanceDisbursementsForDashboardUseCase{repositories: repos, services: svcs}
}

// Execute returns the disbursement-side Advances Dashboard projection.
func (uc *ListAdvanceDisbursementsForDashboardUseCase) Execute(
	ctx context.Context,
	req *disbursementpb.ListAdvanceDisbursementsForDashboardRequest,
) (*disbursementpb.ListAdvanceDisbursementsForDashboardResponse, error) {
	if req == nil {
		req = &disbursementpb.ListAdvanceDisbursementsForDashboardRequest{}
	}
	_ = req

	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		"treasury_disbursement", entityid.ActionList); err != nil {
		return nil, err
	}

	out := &disbursementpb.ListAdvanceDisbursementsForDashboardResponse{}
	if uc.repositories.Disbursement == nil {
		return out, nil
	}

	resp, err := uc.repositories.Disbursement.ListDisbursements(ctx, &disbursementpb.ListDisbursementsRequest{
		Filters: &commonpb.FilterRequest{},
	})
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return out, nil
	}
	for _, d := range resp.GetData() {
		if d.GetAdvanceKind() == advancekindpb.AdvanceKind_ADVANCE_KIND_NONE {
			continue
		}
		if !isActiveAdvanceForDashboard(d.GetAdvanceStatus()) {
			if d.GetAdvanceStatus() == advancekindpb.AdvanceStatus_ADVANCE_STATUS_FULLY_RECOGNIZED {
				out.FullyRecognizedCount++
			}
			continue
		}
		row := &disbursementpb.AdvanceDisbursementDashboardRow{
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
		row.UtilizationPct = advanceUtilizationPct(row.GetRecognizedAmount(), row.GetTotalAmount())
		out.Rows = append(out.Rows, row)
		out.TotalRemaining += row.GetRemainingAmount()
		out.ActiveCount++
	}
	return out, nil
}

// isActiveAdvanceForDashboard mirrors the collection-side helper.
func isActiveAdvanceForDashboard(s advancekindpb.AdvanceStatus) bool {
	switch s {
	case advancekindpb.AdvanceStatus_ADVANCE_STATUS_ACTIVE,
		advancekindpb.AdvanceStatus_ADVANCE_STATUS_PARTIALLY_SETTLED:
		return true
	}
	return false
}

// advanceUtilizationPct mirrors the collection-side helper.
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
