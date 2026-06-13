package ap_aging

import (
	"context"
	"errors"

	payagingpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/reporting/payables_aging"
	apagingpb "github.com/erniealice/esqyma/pkg/schema/v1/service/reporting/ap_aging"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
)

// GetPayablesAgingReportUseCase is the proto-shaped wrapper over the
// underlying ledger reporting adapter for the parameterized AP aging read.
// Apps consume it via `uc.Service.Reporting.APAging.GetPayablesAgingReport.
// Execute`.
//
// The use case takes the new `apagingpb.GetPayablesAgingRequest` (under
// the `service.reporting.ap_aging` proto package), translates it to the
// legacy `payagingpb.PayablesAgingRequest` shape that the postgres
// LedgerReportingAdapter speaks, invokes the port, and translates the
// response back. The shapes are field-for-field identical — the wrapping
// exists only to give the service-driven domain its own proto package per
// Q-SDM-LEDGER-INTERFACE.
type GetPayablesAgingReportUseCase struct {
	reporter             reporter
	authorizationService ports.Authorizer
	translationService   ports.Translator
	actionGatekeeper  *actiongate.ActionGatekeeper
}

// NewGetPayablesAgingReportUseCase wires the use case. Any dep may be
// nil; Execute degrades to an empty response (no rows, success=true) when
// the reporter is missing.
func NewGetPayablesAgingReportUseCase(
	r reporter,
	authSvc ports.Authorizer,
	i18nSvc ports.Translator,
) *GetPayablesAgingReportUseCase {
	if i18nSvc == nil {
		i18nSvc = ports.NewNoOpTranslator()
	}
	return &GetPayablesAgingReportUseCase{
		reporter:             r,
		authorizationService: authSvc,
		translationService:   i18nSvc,
	}
}

// Execute runs the AP aging query under the "reports" + ActionList authcheck.
func (uc *GetPayablesAgingReportUseCase) Execute(
	ctx context.Context,
	req *apagingpb.GetPayablesAgingRequest,
) (*apagingpb.GetPayablesAgingResponse, error) {
	if err := uc.actionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: "reports",
		Action: entityid.ActionList,
	}); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.translationService,
			"reports.validation.request_required", "Payables aging request is required [DEFAULT]"))
	}
	if uc.reporter == nil {
		return &apagingpb.GetPayablesAgingResponse{Success: true}, nil
	}

	innerReq := translatePayablesAgingRequest(req)
	innerResp, err := uc.reporter.GetPayablesAgingReport(ctx, innerReq)
	if err != nil {
		return nil, err
	}
	return translatePayablesAgingResponse(innerResp), nil
}

// translatePayablesAgingRequest copies fields from the service-layer proto
// request to the legacy domain-layer proto request. The two shapes are
// field-for-field identical; this is mechanical.
func translatePayablesAgingRequest(req *apagingpb.GetPayablesAgingRequest) *payagingpb.PayablesAgingRequest {
	if req == nil {
		return nil
	}
	out := &payagingpb.PayablesAgingRequest{
		AsOfDate:              req.AsOfDate,
		StartDate:             req.StartDate,
		EndDate:               req.EndDate,
		RowDimension:          req.GetRowDimension(),
		SupplierId:            req.SupplierId,
		SupplierCategoryId:    req.SupplierCategoryId,
		LocationId:            req.LocationId,
		LocationAreaId:        req.LocationAreaId,
		ExpenditureCategoryId: req.ExpenditureCategoryId,
		Currency:              req.Currency,
	}
	out.Pagination = req.GetPagination()
	return out
}

// translatePayablesAgingResponse copies fields from the legacy domain-layer
// proto response to the service-layer proto response.
func translatePayablesAgingResponse(resp *payagingpb.PayablesAgingResponse) *apagingpb.GetPayablesAgingResponse {
	if resp == nil {
		return &apagingpb.GetPayablesAgingResponse{Success: true}
	}
	out := &apagingpb.GetPayablesAgingResponse{
		BucketLabels: append([]string(nil), resp.GetBucketLabels()...),
		Success:      resp.GetSuccess(),
	}
	for _, r := range resp.GetRows() {
		out.Rows = append(out.Rows, translatePayablesAgingRow(r))
	}
	if s := resp.GetSummary(); s != nil {
		out.Summary = &apagingpb.PayablesAgingSummary{
			Buckets:               translatePayablesAgingBuckets(s.GetBuckets()),
			GrandTotalOutstanding: s.GetGrandTotalOutstanding(),
			TotalInvoiceCount:     s.GetTotalInvoiceCount(),
			AsOfDate:              s.AsOfDate,
			StartDate:             s.StartDate,
			EndDate:               s.EndDate,
			Currency:              s.GetCurrency(),
		}
	}
	out.Pagination = resp.GetPagination()
	out.Error = resp.GetError()
	return out
}

func translatePayablesAgingRow(r *payagingpb.PayablesAgingRow) *apagingpb.PayablesAgingRow {
	if r == nil {
		return nil
	}
	return &apagingpb.PayablesAgingRow{
		RowKey:           r.GetRowKey(),
		RowId:            r.RowId,
		Buckets:          translatePayablesAgingBuckets(r.GetBuckets()),
		TotalOutstanding: r.GetTotalOutstanding(),
		InvoiceCount:     r.GetInvoiceCount(),
	}
}

func translatePayablesAgingBuckets(b *payagingpb.PayablesAgingBuckets) *apagingpb.PayablesAgingBuckets {
	if b == nil {
		return nil
	}
	return &apagingpb.PayablesAgingBuckets{
		Current:     b.GetCurrent(),
		Days_1_30:   b.GetDays_1_30(),
		Days_31_60:  b.GetDays_31_60(),
		Days_61_90:  b.GetDays_61_90(),
		DaysOver_90: b.GetDaysOver_90(),
	}
}
