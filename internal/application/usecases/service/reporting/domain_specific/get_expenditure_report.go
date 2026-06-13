package domain_specific

import (
	"context"
	"errors"

	expreportpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/reporting/expenditure_report"
	dspb "github.com/erniealice/esqyma/pkg/schema/v1/service/reporting/domain_specific"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
)

// GetExpenditureReportUseCase is the proto-shaped wrapper for the two-axis
// expenditure pivot report.
type GetExpenditureReportUseCase struct {
	reporter             reporter
	authorizationService ports.Authorizer
	translationService   ports.Translator
	actionGatekeeper  *actiongate.ActionGatekeeper
}

// NewGetExpenditureReportUseCase wires the use case with nil-safe deps.
func NewGetExpenditureReportUseCase(
	r reporter,
	authSvc ports.Authorizer,
	i18nSvc ports.Translator,
	actionGate *actiongate.ActionGatekeeper,
) *GetExpenditureReportUseCase {
	if i18nSvc == nil {
		i18nSvc = ports.NewNoOpTranslator()
	}
	return &GetExpenditureReportUseCase{
		reporter:             r,
		authorizationService: authSvc,
		translationService:   i18nSvc,
		actionGatekeeper:     actionGate,
	}
}

// Execute runs the expenditure pivot under the "reports" + ActionList
// authcheck.
func (uc *GetExpenditureReportUseCase) Execute(
	ctx context.Context,
	req *dspb.GetExpenditureReportRequest,
) (*dspb.GetExpenditureReportResponse, error) {
	if err := uc.actionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: "reports",
		Action: entityid.ActionList,
	}); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.translationService,
			"reports.validation.request_required", "Expenditure report request is required [DEFAULT]"))
	}
	if uc.reporter == nil {
		return &dspb.GetExpenditureReportResponse{Success: true}, nil
	}

	innerReq := translateExpenditureReportRequest(req)
	innerResp, err := uc.reporter.GetExpenditureReport(ctx, innerReq)
	if err != nil {
		return nil, err
	}
	return translateExpenditureReportResponse(innerResp), nil
}

func translateExpenditureReportRequest(req *dspb.GetExpenditureReportRequest) *expreportpb.ExpenditureReportRequest {
	if req == nil {
		return nil
	}
	out := &expreportpb.ExpenditureReportRequest{
		StartDate:             req.StartDate,
		EndDate:               req.EndDate,
		PrimaryDimension:      req.GetPrimaryDimension(),
		RowDimension:          req.GetRowDimension(),
		ProductId:             req.ProductId,
		CollectionId:          req.CollectionId,
		LocationId:            req.LocationId,
		LocationAreaId:        req.LocationAreaId,
		ExpenditureCategoryId: req.ExpenditureCategoryId,
		Currency:              req.Currency,
		SupplierId:            req.SupplierId,
		ExpenditureType:       req.ExpenditureType,
	}
	out.Pagination = req.GetPagination()
	return out
}

func translateExpenditureReportResponse(resp *expreportpb.ExpenditureReportResponse) *dspb.GetExpenditureReportResponse {
	if resp == nil {
		return &dspb.GetExpenditureReportResponse{Success: true}
	}
	out := &dspb.GetExpenditureReportResponse{
		ColumnKeys: append([]string(nil), resp.GetColumnKeys()...),
		Success:    resp.GetSuccess(),
	}
	for _, r := range resp.GetRows() {
		out.Rows = append(out.Rows, translateExpenditureReportRow(r))
	}
	if s := resp.GetSummary(); s != nil {
		out.Summary = &dspb.ExpenditureReportSummary{
			GrandTotal:        s.GetGrandTotal(),
			TotalTransactions: s.GetTotalTransactions(),
			TotalQuantity:     s.GetTotalQuantity(),
			StartDate:         s.StartDate,
			EndDate:           s.EndDate,
			Currency:          s.GetCurrency(),
		}
		for _, c := range s.GetColumnTotals() {
			out.Summary.ColumnTotals = append(out.Summary.ColumnTotals, translateExpenditureReportCell(c))
		}
	}
	out.Pagination = resp.GetPagination()
	out.Error = resp.GetError()
	return out
}

func translateExpenditureReportRow(r *expreportpb.ExpenditureReportRow) *dspb.ExpenditureReportRow {
	if r == nil {
		return nil
	}
	out := &dspb.ExpenditureReportRow{
		RowKey:              r.GetRowKey(),
		RowId:               r.RowId,
		RowTotal:            r.GetRowTotal(),
		RowTransactionCount: r.GetRowTransactionCount(),
		RowTotalQuantity:    r.GetRowTotalQuantity(),
	}
	for _, c := range r.GetCells() {
		out.Cells = append(out.Cells, translateExpenditureReportCell(c))
	}
	return out
}

func translateExpenditureReportCell(c *expreportpb.ExpenditureReportCell) *dspb.ExpenditureReportCell {
	if c == nil {
		return nil
	}
	return &dspb.ExpenditureReportCell{
		ColumnKey:        c.GetColumnKey(),
		ColumnId:         c.ColumnId,
		TotalExpenditure: c.GetTotalExpenditure(),
		TransactionCount: c.GetTransactionCount(),
		TotalQuantity:    c.GetTotalQuantity(),
	}
}
