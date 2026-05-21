package statements

import (
	"context"
	"errors"

	suppstmtpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/reporting/supplier_statement"
	stmtspb "github.com/erniealice/esqyma/pkg/schema/v1/service/reporting/statements"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
)

// GetSupplierStatementUseCase is the proto-shaped wrapper for the
// per-supplier chronological bill + disbursement ledger.
type GetSupplierStatementUseCase struct {
	reporter             reporter
	authorizationService ports.Authorizer
	translationService   ports.Translator
}

// NewGetSupplierStatementUseCase wires the use case with nil-safe deps.
func NewGetSupplierStatementUseCase(
	r reporter,
	authSvc ports.Authorizer,
	i18nSvc ports.Translator,
) *GetSupplierStatementUseCase {
	if i18nSvc == nil {
		i18nSvc = ports.NewNoOpTranslator()
	}
	return &GetSupplierStatementUseCase{
		reporter:             r,
		authorizationService: authSvc,
		translationService:   i18nSvc,
	}
}

// Execute runs the supplier statement query under the "reports" +
// ActionList authcheck.
func (uc *GetSupplierStatementUseCase) Execute(
	ctx context.Context,
	req *stmtspb.GetSupplierStatementRequest,
) (*stmtspb.GetSupplierStatementResponse, error) {
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
			"reports.validation.request_required", "Supplier statement request is required [DEFAULT]"))
	}
	if uc.reporter == nil {
		return &stmtspb.GetSupplierStatementResponse{Success: true}, nil
	}

	innerReq := translateSupplierStatementRequest(req)
	innerResp, err := uc.reporter.GetSupplierStatement(ctx, innerReq)
	if err != nil {
		return nil, err
	}
	return translateSupplierStatementResponse(innerResp), nil
}

func translateSupplierStatementRequest(req *stmtspb.GetSupplierStatementRequest) *suppstmtpb.SupplierStatementRequest {
	if req == nil {
		return nil
	}
	out := &suppstmtpb.SupplierStatementRequest{
		SupplierId: req.GetSupplierId(),
		StartDate:  req.StartDate,
		EndDate:    req.EndDate,
		Currency:   req.Currency,
	}
	out.Pagination = req.GetPagination()
	return out
}

func translateSupplierStatementResponse(resp *suppstmtpb.SupplierStatementResponse) *stmtspb.GetSupplierStatementResponse {
	if resp == nil {
		return &stmtspb.GetSupplierStatementResponse{Success: true}
	}
	out := &stmtspb.GetSupplierStatementResponse{
		Success: resp.GetSuccess(),
	}
	for _, e := range resp.GetEntries() {
		if e == nil {
			continue
		}
		out.Entries = append(out.Entries, &stmtspb.SupplierStatementEntry{
			Date:            e.GetDate(),
			Type:            e.GetType(),
			ReferenceNumber: e.GetReferenceNumber(),
			Description:     e.GetDescription(),
			Billed:          e.GetBilled(),
			Paid:            e.GetPaid(),
			Balance:         e.GetBalance(),
			EntityId:        e.GetEntityId(),
			Status:          e.GetStatus(),
		})
	}
	if s := resp.GetSummary(); s != nil {
		out.Summary = &stmtspb.SupplierStatementSummary{
			TotalBilled:        s.GetTotalBilled(),
			TotalPaid:          s.GetTotalPaid(),
			OutstandingBalance: s.GetOutstandingBalance(),
			BillCount:          s.GetBillCount(),
			PaymentCount:       s.GetPaymentCount(),
			Currency:           s.GetCurrency(),
			StartDate:          s.StartDate,
			EndDate:            s.EndDate,
			SupplierName:       s.GetSupplierName(),
		}
	}
	out.Pagination = resp.GetPagination()
	out.Error = resp.GetError()
	return out
}
