package domain_specific

import (
	"context"
	"errors"

	revreportpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/reporting/revenue_report"
	dspb "github.com/erniealice/esqyma/pkg/schema/v1/service/reporting/domain_specific"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
)

// GetRevenueReportUseCase is the proto-shaped wrapper for the two-axis
// revenue pivot report.
type GetRevenueReportUseCase struct {
	reporter             reporter
	authorizationService ports.Authorizer
	translationService   ports.Translator
}

// NewGetRevenueReportUseCase wires the use case with nil-safe deps.
func NewGetRevenueReportUseCase(
	r reporter,
	authSvc ports.Authorizer,
	i18nSvc ports.Translator,
) *GetRevenueReportUseCase {
	if i18nSvc == nil {
		i18nSvc = ports.NewNoOpTranslator()
	}
	return &GetRevenueReportUseCase{
		reporter:             r,
		authorizationService: authSvc,
		translationService:   i18nSvc,
	}
}

// Execute runs the revenue pivot under the "reports" + ActionList authcheck.
func (uc *GetRevenueReportUseCase) Execute(
	ctx context.Context,
	req *dspb.GetRevenueReportRequest,
) (*dspb.GetRevenueReportResponse, error) {
	if err := authcheck.Check(
		ctx,
		uc.authorizationService,
		uc.translationService,
		"reports",
		entityid.ActionList,
	); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.translationService,
			"reports.validation.request_required", "Revenue report request is required [DEFAULT]"))
	}
	if uc.reporter == nil {
		return &dspb.GetRevenueReportResponse{Success: true}, nil
	}

	innerReq := translateRevenueReportRequest(req)
	innerResp, err := uc.reporter.GetRevenueReport(ctx, innerReq)
	if err != nil {
		return nil, err
	}
	return translateRevenueReportResponse(innerResp), nil
}

func translateRevenueReportRequest(req *dspb.GetRevenueReportRequest) *revreportpb.RevenueReportRequest {
	if req == nil {
		return nil
	}
	out := &revreportpb.RevenueReportRequest{
		StartDate:         req.StartDate,
		EndDate:           req.EndDate,
		PrimaryDimension:  req.GetPrimaryDimension(),
		RowDimension:      req.GetRowDimension(),
		ProductId:         req.ProductId,
		CollectionId:      req.CollectionId,
		LocationId:        req.LocationId,
		LocationAreaId:    req.LocationAreaId,
		RevenueCategoryId: req.RevenueCategoryId,
		Currency:          req.Currency,
	}
	out.Pagination = req.GetPagination()
	return out
}

func translateRevenueReportResponse(resp *revreportpb.RevenueReportResponse) *dspb.GetRevenueReportResponse {
	if resp == nil {
		return &dspb.GetRevenueReportResponse{Success: true}
	}
	out := &dspb.GetRevenueReportResponse{
		ColumnKeys: append([]string(nil), resp.GetColumnKeys()...),
		Success:    resp.GetSuccess(),
	}
	for _, r := range resp.GetRows() {
		out.Rows = append(out.Rows, translateRevenueReportRow(r))
	}
	if s := resp.GetSummary(); s != nil {
		out.Summary = &dspb.RevenueReportSummary{
			GrandTotal:        s.GetGrandTotal(),
			TotalTransactions: s.GetTotalTransactions(),
			TotalQuantity:     s.GetTotalQuantity(),
			StartDate:         s.StartDate,
			EndDate:           s.EndDate,
			Currency:          s.GetCurrency(),
		}
		for _, c := range s.GetColumnTotals() {
			out.Summary.ColumnTotals = append(out.Summary.ColumnTotals, translateRevenueReportCell(c))
		}
	}
	out.Pagination = resp.GetPagination()
	out.Error = resp.GetError()
	return out
}

func translateRevenueReportRow(r *revreportpb.RevenueReportRow) *dspb.RevenueReportRow {
	if r == nil {
		return nil
	}
	out := &dspb.RevenueReportRow{
		RowKey:              r.GetRowKey(),
		RowId:               r.RowId,
		RowTotal:            r.GetRowTotal(),
		RowTransactionCount: r.GetRowTransactionCount(),
		RowTotalQuantity:    r.GetRowTotalQuantity(),
	}
	for _, c := range r.GetCells() {
		out.Cells = append(out.Cells, translateRevenueReportCell(c))
	}
	return out
}

func translateRevenueReportCell(c *revreportpb.RevenueReportCell) *dspb.RevenueReportCell {
	if c == nil {
		return nil
	}
	return &dspb.RevenueReportCell{
		ColumnKey:        c.GetColumnKey(),
		ColumnId:         c.ColumnId,
		TotalRevenue:     c.GetTotalRevenue(),
		TransactionCount: c.GetTransactionCount(),
		TotalQuantity:    c.GetTotalQuantity(),
	}
}
