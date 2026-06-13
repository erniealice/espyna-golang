package statements

import (
	"context"
	"errors"

	clientstmtpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/reporting/client_statement"
	stmtspb "github.com/erniealice/esqyma/pkg/schema/v1/service/reporting/statements"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
)

// GetClientStatementUseCase is the proto-shaped wrapper for the per-client
// chronological invoice + collection ledger.
type GetClientStatementUseCase struct {
	reporter             reporter
	authorizationService ports.Authorizer
	translationService   ports.Translator
	actionGatekeeper  *actiongate.ActionGatekeeper
}

// NewGetClientStatementUseCase wires the use case with nil-safe deps.
func NewGetClientStatementUseCase(
	r reporter,
	authSvc ports.Authorizer,
	i18nSvc ports.Translator,
	actionGate *actiongate.ActionGatekeeper,
) *GetClientStatementUseCase {
	if i18nSvc == nil {
		i18nSvc = ports.NewNoOpTranslator()
	}
	return &GetClientStatementUseCase{
		reporter:             r,
		authorizationService: authSvc,
		translationService:   i18nSvc,
		actionGatekeeper:     actionGate,
	}
}

// Execute runs the client statement query under the "reports" + ActionList
// authcheck.
func (uc *GetClientStatementUseCase) Execute(
	ctx context.Context,
	req *stmtspb.GetClientStatementRequest,
) (*stmtspb.GetClientStatementResponse, error) {
	if err := uc.actionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: "reports",
		Action: entityid.ActionList,
	}); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.translationService,
			"reports.validation.request_required", "Client statement request is required [DEFAULT]"))
	}
	if uc.reporter == nil {
		return &stmtspb.GetClientStatementResponse{Success: true}, nil
	}

	innerReq := translateClientStatementRequest(req)
	innerResp, err := uc.reporter.GetClientStatement(ctx, innerReq)
	if err != nil {
		return nil, err
	}
	return translateClientStatementResponse(innerResp), nil
}

func translateClientStatementRequest(req *stmtspb.GetClientStatementRequest) *clientstmtpb.ClientStatementRequest {
	if req == nil {
		return nil
	}
	out := &clientstmtpb.ClientStatementRequest{
		ClientId:  req.GetClientId(),
		StartDate: req.StartDate,
		EndDate:   req.EndDate,
		Currency:  req.Currency,
	}
	out.Pagination = req.GetPagination()
	return out
}

func translateClientStatementResponse(resp *clientstmtpb.ClientStatementResponse) *stmtspb.GetClientStatementResponse {
	if resp == nil {
		return &stmtspb.GetClientStatementResponse{Success: true}
	}
	out := &stmtspb.GetClientStatementResponse{
		Success: resp.GetSuccess(),
	}
	for _, e := range resp.GetEntries() {
		if e == nil {
			continue
		}
		out.Entries = append(out.Entries, &stmtspb.StatementEntry{
			Date:            e.GetDate(),
			Type:            e.GetType(),
			ReferenceNumber: e.GetReferenceNumber(),
			Description:     e.GetDescription(),
			Billed:          e.GetBilled(),
			Received:        e.GetReceived(),
			Balance:         e.GetBalance(),
			EntityId:        e.GetEntityId(),
			Status:          e.GetStatus(),
		})
	}
	if s := resp.GetSummary(); s != nil {
		out.Summary = &stmtspb.ClientStatementSummary{
			TotalBilled:        s.GetTotalBilled(),
			TotalReceived:      s.GetTotalReceived(),
			OutstandingBalance: s.GetOutstandingBalance(),
			InvoiceCount:       s.GetInvoiceCount(),
			CollectionCount:    s.GetCollectionCount(),
			Currency:           s.GetCurrency(),
			StartDate:          s.StartDate,
			EndDate:            s.EndDate,
			ClientName:         s.GetClientName(),
		}
	}
	out.Pagination = resp.GetPagination()
	out.Error = resp.GetError()
	return out
}
