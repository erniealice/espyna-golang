package ap_aging

import (
	"context"
	"errors"

	reportpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/reporting/gross_profit"
	apagingpb "github.com/erniealice/esqyma/pkg/schema/v1/service/reporting/ap_aging"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
)

// GetSimplePayablesAgingReportUseCase is the proto-shaped wrapper for the
// stripped-down "simple" AP aging report — no parameters, returns one row
// per supplier with 5 bucketed centavo amounts + a total.
//
// The underlying postgres adapter method
// (`LedgerReportingAdapter.GetSimplePayablesAgingReport`) takes an empty
// request and returns rows under the gross_profit proto package (`reportpb`
// — that proto file historically carried both the gross-profit and the
// simple-payables-aging shapes, despite the directory name). The wrapping
// gives this method its own request/response under
// `service.reporting.ap_aging`.
type GetSimplePayablesAgingReportUseCase struct {
	reporter             reporter
	authorizationService ports.Authorizer
	translationService   ports.Translator
	actionGatekeeper  *actiongate.ActionGatekeeper
}

// NewGetSimplePayablesAgingReportUseCase wires the use case with nil-safe
// dependency contract.
func NewGetSimplePayablesAgingReportUseCase(
	r reporter,
	authSvc ports.Authorizer,
	i18nSvc ports.Translator,
) *GetSimplePayablesAgingReportUseCase {
	if i18nSvc == nil {
		i18nSvc = ports.NewNoOpTranslator()
	}
	return &GetSimplePayablesAgingReportUseCase{
		reporter:             r,
		authorizationService: authSvc,
		translationService:   i18nSvc,
	}
}

// Execute runs the simple AP aging query under the "reports" + ActionList
// authcheck.
func (uc *GetSimplePayablesAgingReportUseCase) Execute(
	ctx context.Context,
	req *apagingpb.GetSimplePayablesAgingRequest,
) (*apagingpb.GetSimplePayablesAgingResponse, error) {
	if err := uc.actionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: "reports",
		Action: entityid.ActionList,
	}); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.translationService,
			"reports.validation.request_required", "Simple payables aging request is required [DEFAULT]"))
	}
	if uc.reporter == nil {
		return &apagingpb.GetSimplePayablesAgingResponse{Success: true}, nil
	}

	innerResp, err := uc.reporter.GetSimplePayablesAgingReport(ctx, &reportpb.PayablesAgingReportRequest{})
	if err != nil {
		return nil, err
	}
	return translateSimplePayablesAgingResponse(innerResp), nil
}

// translateSimplePayablesAgingResponse copies fields from the legacy
// domain-layer proto response (under the gross_profit package) to the
// service-layer ap_aging proto response.
func translateSimplePayablesAgingResponse(resp *reportpb.PayablesAgingReportResponse) *apagingpb.GetSimplePayablesAgingResponse {
	if resp == nil {
		return &apagingpb.GetSimplePayablesAgingResponse{Success: true}
	}
	out := &apagingpb.GetSimplePayablesAgingResponse{
		Success: resp.GetSuccess(),
	}
	for _, r := range resp.GetData() {
		if r == nil {
			continue
		}
		out.Data = append(out.Data, &apagingpb.SimplePayablesAgingRow{
			SupplierName: r.GetSupplierName(),
			Current:      r.GetCurrent(),
			Days_30:      r.GetDays_30(),
			Days_60:      r.GetDays_60(),
			Days_90:      r.GetDays_90(),
			Over_90:      r.GetOver_90(),
			Total:        r.GetTotal(),
		})
	}
	out.Error = resp.GetError()
	return out
}
