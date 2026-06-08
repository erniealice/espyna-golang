package domain_specific

import (
	"context"
	"errors"

	disbreportpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/reporting/disbursement_report"
	dspb "github.com/erniealice/esqyma/pkg/schema/v1/service/reporting/domain_specific"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
)

// GetDisbursementReportUseCase is the proto-shaped wrapper for the two-axis
// disbursement pivot report.
type GetDisbursementReportUseCase struct {
	reporter             reporter
	authorizationService ports.Authorizer
	translationService   ports.Translator
}

// NewGetDisbursementReportUseCase wires the use case with nil-safe deps.
func NewGetDisbursementReportUseCase(
	r reporter,
	authSvc ports.Authorizer,
	i18nSvc ports.Translator,
) *GetDisbursementReportUseCase {
	if i18nSvc == nil {
		i18nSvc = ports.NewNoOpTranslator()
	}
	return &GetDisbursementReportUseCase{
		reporter:             r,
		authorizationService: authSvc,
		translationService:   i18nSvc,
	}
}

// Execute runs the disbursement pivot under the "reports" + ActionList
// authcheck.
func (uc *GetDisbursementReportUseCase) Execute(
	ctx context.Context,
	req *dspb.GetDisbursementReportRequest,
) (*dspb.GetDisbursementReportResponse, error) {
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
			"reports.validation.request_required", "Disbursement report request is required [DEFAULT]"))
	}
	if uc.reporter == nil {
		return &dspb.GetDisbursementReportResponse{Success: true}, nil
	}

	innerReq := translateDisbursementReportRequest(req)
	innerResp, err := uc.reporter.GetDisbursementReport(ctx, innerReq)
	if err != nil {
		return nil, err
	}
	return translateDisbursementReportResponse(innerResp), nil
}

func translateDisbursementReportRequest(req *dspb.GetDisbursementReportRequest) *disbreportpb.DisbursementReportRequest {
	if req == nil {
		return nil
	}
	out := &disbreportpb.DisbursementReportRequest{
		StartDate:             req.StartDate,
		EndDate:               req.EndDate,
		PrimaryDimension:      req.GetPrimaryDimension(),
		RowDimension:          req.GetRowDimension(),
		SupplierId:            req.SupplierId,
		SupplierCategoryId:    req.SupplierCategoryId,
		LocationId:            req.LocationId,
		LocationAreaId:        req.LocationAreaId,
		ExpenditureCategoryId: req.ExpenditureCategoryId,
		Currency:              req.Currency,
		DisbursementType:      req.DisbursementType,
		DisbursementMethodId:  req.DisbursementMethodId,
		ExpenditureId:         req.ExpenditureId,
	}
	out.Pagination = req.GetPagination()
	return out
}

func translateDisbursementReportResponse(resp *disbreportpb.DisbursementReportResponse) *dspb.GetDisbursementReportResponse {
	if resp == nil {
		return &dspb.GetDisbursementReportResponse{Success: true}
	}
	out := &dspb.GetDisbursementReportResponse{
		ColumnKeys: append([]string(nil), resp.GetColumnKeys()...),
		Success:    resp.GetSuccess(),
	}
	for _, r := range resp.GetRows() {
		out.Rows = append(out.Rows, translateDisbursementReportRow(r))
	}
	if s := resp.GetSummary(); s != nil {
		out.Summary = &dspb.DisbursementReportSummary{
			GrandTotal:        s.GetGrandTotal(),
			TotalTransactions: s.GetTotalTransactions(),
			TotalQuantity:     s.GetTotalQuantity(),
			StartDate:         s.StartDate,
			EndDate:           s.EndDate,
			Currency:          s.GetCurrency(),
		}
		for _, c := range s.GetColumnTotals() {
			out.Summary.ColumnTotals = append(out.Summary.ColumnTotals, translateDisbursementReportCell(c))
		}
	}
	out.Pagination = resp.GetPagination()
	out.Error = resp.GetError()
	return out
}

func translateDisbursementReportRow(r *disbreportpb.DisbursementReportRow) *dspb.DisbursementReportRow {
	if r == nil {
		return nil
	}
	out := &dspb.DisbursementReportRow{
		RowKey:              r.GetRowKey(),
		RowId:               r.RowId,
		RowTotal:            r.GetRowTotal(),
		RowTransactionCount: r.GetRowTransactionCount(),
		RowTotalQuantity:    r.GetRowTotalQuantity(),
	}
	for _, c := range r.GetCells() {
		out.Cells = append(out.Cells, translateDisbursementReportCell(c))
	}
	return out
}

func translateDisbursementReportCell(c *disbreportpb.DisbursementReportCell) *dspb.DisbursementReportCell {
	if c == nil {
		return nil
	}
	return &dspb.DisbursementReportCell{
		ColumnKey:         c.GetColumnKey(),
		ColumnId:          c.ColumnId,
		TotalDisbursement: c.GetTotalDisbursement(),
		TransactionCount:  c.GetTransactionCount(),
		TotalQuantity:     c.GetTotalQuantity(),
	}
}
