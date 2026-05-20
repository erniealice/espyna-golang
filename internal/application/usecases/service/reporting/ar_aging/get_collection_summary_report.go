package ar_aging

import (
	"context"
	"errors"

	collsumpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/reporting/collection_summary"
	aragingpb "github.com/erniealice/esqyma/pkg/schema/v1/service/reporting/ar_aging"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
)

// GetCollectionSummaryReportUseCase is the proto-shaped wrapper for the
// AR-side cash receipts pivot report. Mirrors the pattern in
// `get_receivables_aging_report.go`: takes the new service-layer proto
// request, translates to the legacy domain-layer proto, invokes the
// reporter, translates the response back.
type GetCollectionSummaryReportUseCase struct {
	reporter             reporter
	authorizationService ports.AuthorizationService
	translationService   ports.TranslationService
}

// NewGetCollectionSummaryReportUseCase wires the use case with nil-safe
// dependency contract (same as GetReceivablesAgingReportUseCase).
func NewGetCollectionSummaryReportUseCase(
	r reporter,
	authSvc ports.AuthorizationService,
	i18nSvc ports.TranslationService,
) *GetCollectionSummaryReportUseCase {
	if i18nSvc == nil {
		i18nSvc = ports.NewNoOpTranslationService()
	}
	return &GetCollectionSummaryReportUseCase{
		reporter:             r,
		authorizationService: authSvc,
		translationService:   i18nSvc,
	}
}

// Execute runs the collection summary pivot under the "reports" + ActionList
// authcheck.
func (uc *GetCollectionSummaryReportUseCase) Execute(
	ctx context.Context,
	req *aragingpb.GetCollectionSummaryRequest,
) (*aragingpb.GetCollectionSummaryResponse, error) {
	if err := authcheck.Check(
		ctx,
		uc.authorizationService,
		uc.translationService,
		"reports",
		ports.ActionList,
	); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.translationService,
			"reports.validation.request_required", "Collection summary request is required [DEFAULT]"))
	}
	if uc.reporter == nil {
		return &aragingpb.GetCollectionSummaryResponse{Success: true}, nil
	}

	innerReq := translateCollectionSummaryRequest(req)
	innerResp, err := uc.reporter.GetCollectionSummaryReport(ctx, innerReq)
	if err != nil {
		return nil, err
	}
	return translateCollectionSummaryResponse(innerResp), nil
}

// translateCollectionSummaryRequest copies the new service-layer proto
// request into the legacy domain-layer proto request shape. The two are
// field-for-field identical.
func translateCollectionSummaryRequest(req *aragingpb.GetCollectionSummaryRequest) *collsumpb.CollectionSummaryRequest {
	if req == nil {
		return nil
	}
	out := &collsumpb.CollectionSummaryRequest{
		StartDate:          req.StartDate,
		EndDate:            req.EndDate,
		PrimaryDimension:   req.GetPrimaryDimension(),
		RowDimension:       req.GetRowDimension(),
		ClientId:           req.ClientId,
		ClientCategoryId:   req.ClientCategoryId,
		LocationId:         req.LocationId,
		LocationAreaId:     req.LocationAreaId,
		CollectionMethodId: req.CollectionMethodId,
		Currency:           req.Currency,
		CollectionType:     req.CollectionType,
	}
	// Pagination is shared common.PaginationRequest — pass through.
	out.Pagination = req.GetPagination()
	return out
}

// translateCollectionSummaryResponse copies the legacy domain-layer proto
// response into the new service-layer proto response shape.
func translateCollectionSummaryResponse(resp *collsumpb.CollectionSummaryResponse) *aragingpb.GetCollectionSummaryResponse {
	if resp == nil {
		return &aragingpb.GetCollectionSummaryResponse{Success: true}
	}
	out := &aragingpb.GetCollectionSummaryResponse{
		ColumnKeys: append([]string(nil), resp.GetColumnKeys()...),
		Success:    resp.GetSuccess(),
	}
	for _, r := range resp.GetRows() {
		out.Rows = append(out.Rows, translateCollectionSummaryRow(r))
	}
	if s := resp.GetSummary(); s != nil {
		out.Summary = &aragingpb.CollectionSummarySummary{
			GrandTotal:        s.GetGrandTotal(),
			TotalTransactions: s.GetTotalTransactions(),
			StartDate:         s.StartDate,
			EndDate:           s.EndDate,
			Currency:          s.GetCurrency(),
		}
		for _, c := range s.GetColumnTotals() {
			out.Summary.ColumnTotals = append(out.Summary.ColumnTotals, translateCollectionSummaryCell(c))
		}
	}
	// Pagination + Error are shared common.* types — pass through.
	out.Pagination = resp.GetPagination()
	out.Error = resp.GetError()
	return out
}

func translateCollectionSummaryRow(r *collsumpb.CollectionSummaryRow) *aragingpb.CollectionSummaryRow {
	if r == nil {
		return nil
	}
	out := &aragingpb.CollectionSummaryRow{
		RowKey:              r.GetRowKey(),
		RowId:               r.RowId,
		RowTotal:            r.GetRowTotal(),
		RowTransactionCount: r.GetRowTransactionCount(),
	}
	for _, c := range r.GetCells() {
		out.Cells = append(out.Cells, translateCollectionSummaryCell(c))
	}
	return out
}

func translateCollectionSummaryCell(c *collsumpb.CollectionSummaryCell) *aragingpb.CollectionSummaryCell {
	if c == nil {
		return nil
	}
	return &aragingpb.CollectionSummaryCell{
		ColumnKey:        c.GetColumnKey(),
		ColumnId:         c.ColumnId,
		TotalCollected:   c.GetTotalCollected(),
		TransactionCount: c.GetTransactionCount(),
	}
}
