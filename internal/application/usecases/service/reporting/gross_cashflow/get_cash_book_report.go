package gross_cashflow

import (
	"context"
	"errors"

	reportpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/reporting/gross_profit"
	gcfpb "github.com/erniealice/esqyma/pkg/schema/v1/service/reporting/gross_cashflow"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
)

// GetCashBookReportUseCase is the proto-shaped wrapper for the cash book
// report — a chronological list of receipts + expenses.
type GetCashBookReportUseCase struct {
	reporter             reporter
	authorizationService ports.Authorizer
	translationService   ports.Translator
}

// NewGetCashBookReportUseCase wires the use case with nil-safe deps.
func NewGetCashBookReportUseCase(
	r reporter,
	authSvc ports.Authorizer,
	i18nSvc ports.Translator,
) *GetCashBookReportUseCase {
	if i18nSvc == nil {
		i18nSvc = ports.NewNoOpTranslator()
	}
	return &GetCashBookReportUseCase{
		reporter:             r,
		authorizationService: authSvc,
		translationService:   i18nSvc,
	}
}

// Execute runs the cash book query under the "reports" + ActionList
// authcheck.
func (uc *GetCashBookReportUseCase) Execute(
	ctx context.Context,
	req *gcfpb.GetCashBookRequest,
) (*gcfpb.GetCashBookResponse, error) {
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
			"reports.validation.request_required", "Cash book request is required [DEFAULT]"))
	}
	if uc.reporter == nil {
		return &gcfpb.GetCashBookResponse{Success: true}, nil
	}

	innerReq := &reportpb.CashBookReportRequest{Limit: req.Limit}
	innerResp, err := uc.reporter.GetCashBookReport(ctx, innerReq)
	if err != nil {
		return nil, err
	}
	return translateCashBookResponse(innerResp), nil
}

func translateCashBookResponse(resp *reportpb.CashBookReportResponse) *gcfpb.GetCashBookResponse {
	if resp == nil {
		return &gcfpb.GetCashBookResponse{Success: true}
	}
	out := &gcfpb.GetCashBookResponse{
		Success: resp.GetSuccess(),
	}
	for _, r := range resp.GetData() {
		if r == nil {
			continue
		}
		out.Data = append(out.Data, &gcfpb.CashBookRow{
			TxDate:      r.GetTxDate(),
			Description: r.GetDescription(),
			Reference:   r.GetReference(),
			TxType:      r.GetTxType(),
			Amount:      r.GetAmount(),
		})
	}
	out.Error = resp.GetError()
	return out
}
