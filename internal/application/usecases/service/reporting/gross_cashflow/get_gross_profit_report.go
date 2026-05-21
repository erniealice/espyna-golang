package gross_cashflow

import (
	"context"
	"errors"

	reportpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/reporting/gross_profit"
	gcfpb "github.com/erniealice/esqyma/pkg/schema/v1/service/reporting/gross_cashflow"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
)

// GetGrossProfitReportUseCase is the proto-shaped wrapper for the gross
// profit report. Translates between the new service-layer proto and the
// legacy entity-domain proto (under
// `domain/ledger/reporting/gross_profit`).
type GetGrossProfitReportUseCase struct {
	reporter             reporter
	authorizationService ports.Authorizer
	translationService   ports.Translator
}

// NewGetGrossProfitReportUseCase wires the use case with nil-safe deps.
func NewGetGrossProfitReportUseCase(
	r reporter,
	authSvc ports.Authorizer,
	i18nSvc ports.Translator,
) *GetGrossProfitReportUseCase {
	if i18nSvc == nil {
		i18nSvc = ports.NewNoOpTranslator()
	}
	return &GetGrossProfitReportUseCase{
		reporter:             r,
		authorizationService: authSvc,
		translationService:   i18nSvc,
	}
}

// Execute runs the gross profit query under the "reports" + ActionList
// authcheck.
func (uc *GetGrossProfitReportUseCase) Execute(
	ctx context.Context,
	req *gcfpb.GetGrossProfitRequest,
) (*gcfpb.GetGrossProfitResponse, error) {
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
			"reports.validation.request_required", "Gross profit request is required [DEFAULT]"))
	}
	if uc.reporter == nil {
		return &gcfpb.GetGrossProfitResponse{Success: true}, nil
	}

	innerReq := translateGrossProfitRequest(req)
	innerResp, err := uc.reporter.GetGrossProfitReport(ctx, innerReq)
	if err != nil {
		return nil, err
	}
	return translateGrossProfitResponse(innerResp), nil
}

func translateGrossProfitRequest(req *gcfpb.GetGrossProfitRequest) *reportpb.GrossProfitReportRequest {
	if req == nil {
		return nil
	}
	out := &reportpb.GrossProfitReportRequest{
		StartDate:         req.StartDate,
		EndDate:           req.EndDate,
		GroupBy:           req.GroupBy,
		PeriodGranularity: req.PeriodGranularity,
		ProductId:         req.ProductId,
		LocationId:        req.LocationId,
		RevenueCategoryId: req.RevenueCategoryId,
		Currency:          req.Currency,
	}
	out.Pagination = req.GetPagination()
	return out
}

func translateGrossProfitResponse(resp *reportpb.GrossProfitReportResponse) *gcfpb.GetGrossProfitResponse {
	if resp == nil {
		return &gcfpb.GetGrossProfitResponse{Success: true}
	}
	out := &gcfpb.GetGrossProfitResponse{
		Success: resp.GetSuccess(),
	}
	for _, li := range resp.GetLineItems() {
		out.LineItems = append(out.LineItems, translateGrossProfitLineItem(li))
	}
	if s := resp.GetSummary(); s != nil {
		out.Summary = &gcfpb.GrossProfitSummary{
			TotalRevenue:      s.GetTotalRevenue(),
			TotalDiscount:     s.GetTotalDiscount(),
			NetRevenue:        s.GetNetRevenue(),
			TotalCogs:         s.GetTotalCogs(),
			TotalGrossProfit:  s.GetTotalGrossProfit(),
			OverallMargin:     s.GetOverallMargin(),
			TotalUnitsSold:    s.GetTotalUnitsSold(),
			TotalTransactions: s.GetTotalTransactions(),
			Currency:          s.GetCurrency(),
			StartDate:         s.StartDate,
			EndDate:           s.EndDate,
		}
	}
	out.Pagination = resp.GetPagination()
	out.Error = resp.GetError()
	return out
}

func translateGrossProfitLineItem(li *reportpb.GrossProfitLineItem) *gcfpb.GrossProfitLineItem {
	if li == nil {
		return nil
	}
	return &gcfpb.GrossProfitLineItem{
		GroupKey:          li.GetGroupKey(),
		GroupId:           li.GroupId,
		TotalRevenue:      li.GetTotalRevenue(),
		TotalDiscount:     li.GetTotalDiscount(),
		NetRevenue:        li.GetNetRevenue(),
		CostOfGoodsSold:   li.GetCostOfGoodsSold(),
		GrossProfit:       li.GetGrossProfit(),
		GrossProfitMargin: li.GetGrossProfitMargin(),
		UnitsSold:         li.GetUnitsSold(),
		TransactionCount:  li.GetTransactionCount(),
	}
}
