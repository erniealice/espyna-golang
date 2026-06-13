package ar_aging

import (
	"context"
	"errors"

	agingpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/reporting/receivables_aging"
	aragingpb "github.com/erniealice/esqyma/pkg/schema/v1/service/reporting/ar_aging"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
)

// GetReceivablesAgingReportUseCase is the proto-shaped wrapper over the
// underlying ledger reporting adapter for the AR aging read. Apps consume
// it via `uc.Service.Reporting.ARAging.GetReceivablesAgingReport.Execute`.
//
// The use case takes the new `aragingpb.GetReceivablesAgingRequest` (under
// the `service.reporting.ar_aging` proto package), translates it to the
// legacy `agingpb.ReceivablesAgingRequest` shape that the postgres
// LedgerReportingAdapter speaks, invokes the port, and translates the
// response back into `aragingpb.GetReceivablesAgingResponse`. The shapes
// are field-for-field identical — the wrapping exists only to give the
// service-driven domain its own proto package per Q-SDM-LEDGER-INTERFACE.
type GetReceivablesAgingReportUseCase struct {
	reporter             reporter
	authorizationService ports.Authorizer
	translationService   ports.Translator
	actionGatekeeper  *actiongate.ActionGatekeeper
}

// NewGetReceivablesAgingReportUseCase wires the use case. Any dep may be
// nil; Execute degrades to an empty response (no rows, success=true) when
// the reporter is missing.
func NewGetReceivablesAgingReportUseCase(
	r reporter,
	authSvc ports.Authorizer,
	i18nSvc ports.Translator,
	actionGate *actiongate.ActionGatekeeper,
) *GetReceivablesAgingReportUseCase {
	if i18nSvc == nil {
		i18nSvc = ports.NewNoOpTranslator()
	}
	return &GetReceivablesAgingReportUseCase{
		reporter:             r,
		authorizationService: authSvc,
		translationService:   i18nSvc,
		actionGatekeeper:     actionGate,
	}
}

// Execute runs the AR aging query under the "reports" + ActionList authcheck.
func (uc *GetReceivablesAgingReportUseCase) Execute(
	ctx context.Context,
	req *aragingpb.GetReceivablesAgingRequest,
) (*aragingpb.GetReceivablesAgingResponse, error) {
	if err := uc.actionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: "reports",
		Action: entityid.ActionList,
	}); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.translationService,
			"reports.validation.request_required", "Receivables aging request is required [DEFAULT]"))
	}
	if uc.reporter == nil {
		return &aragingpb.GetReceivablesAgingResponse{Success: true}, nil
	}

	innerReq := translateReceivablesAgingRequest(req)
	innerResp, err := uc.reporter.GetReceivablesAgingReport(ctx, innerReq)
	if err != nil {
		return nil, err
	}
	return translateReceivablesAgingResponse(innerResp), nil
}

// translateReceivablesAgingRequest copies fields from the service-layer
// proto request to the legacy domain-layer proto request. The two shapes
// are field-for-field identical; this is mechanical.
func translateReceivablesAgingRequest(req *aragingpb.GetReceivablesAgingRequest) *agingpb.ReceivablesAgingRequest {
	if req == nil {
		return nil
	}
	out := &agingpb.ReceivablesAgingRequest{
		AsOfDate:          req.AsOfDate,
		StartDate:         req.StartDate,
		EndDate:           req.EndDate,
		RowDimension:      req.GetRowDimension(),
		ClientId:          req.ClientId,
		ClientCategoryId:  req.ClientCategoryId,
		LocationId:        req.LocationId,
		LocationAreaId:    req.LocationAreaId,
		RevenueCategoryId: req.RevenueCategoryId,
		Currency:          req.Currency,
	}
	// Pagination is shared common.PaginationRequest — pass through.
	out.Pagination = req.GetPagination()
	return out
}

// translateReceivablesAgingResponse copies fields from the legacy domain-
// layer proto response to the service-layer proto response.
func translateReceivablesAgingResponse(resp *agingpb.ReceivablesAgingResponse) *aragingpb.GetReceivablesAgingResponse {
	if resp == nil {
		return &aragingpb.GetReceivablesAgingResponse{Success: true}
	}
	out := &aragingpb.GetReceivablesAgingResponse{
		BucketLabels: append([]string(nil), resp.GetBucketLabels()...),
		Success:      resp.GetSuccess(),
	}
	for _, r := range resp.GetRows() {
		out.Rows = append(out.Rows, translateReceivablesAgingRow(r))
	}
	if s := resp.GetSummary(); s != nil {
		out.Summary = &aragingpb.ReceivablesAgingSummary{
			Buckets:               translateAgingBuckets(s.GetBuckets()),
			GrandTotalOutstanding: s.GetGrandTotalOutstanding(),
			TotalInvoiceCount:     s.GetTotalInvoiceCount(),
			AsOfDate:              s.AsOfDate,
			StartDate:             s.StartDate,
			EndDate:               s.EndDate,
			Currency:              s.GetCurrency(),
		}
	}
	// Pagination + Error are shared common.* types — pass through.
	out.Pagination = resp.GetPagination()
	out.Error = resp.GetError()
	return out
}

func translateReceivablesAgingRow(r *agingpb.ReceivablesAgingRow) *aragingpb.ReceivablesAgingRow {
	if r == nil {
		return nil
	}
	return &aragingpb.ReceivablesAgingRow{
		RowKey:           r.GetRowKey(),
		RowId:            r.RowId,
		Buckets:          translateAgingBuckets(r.GetBuckets()),
		TotalOutstanding: r.GetTotalOutstanding(),
		InvoiceCount:     r.GetInvoiceCount(),
	}
}

func translateAgingBuckets(b *agingpb.AgingBuckets) *aragingpb.AgingBuckets {
	if b == nil {
		return nil
	}
	return &aragingpb.AgingBuckets{
		Current:     b.GetCurrent(),
		Days_1_30:   b.GetDays_1_30(),
		Days_31_60:  b.GetDays_31_60(),
		Days_61_90:  b.GetDays_61_90(),
		DaysOver_90: b.GetDaysOver_90(),
	}
}
