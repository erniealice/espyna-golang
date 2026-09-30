package recovery_document

import (
	"context"
	"sort"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/shared/usecaseerr"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	clientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client"
	recoverydocumentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/recovery_document"
)

// ListRecoverablesAgingRepositories groups repository dependencies.
type ListRecoverablesAgingRepositories struct {
	Balance BalanceRepos
	Client  clientpb.ClientDomainServiceServer
}

// ListRecoverablesAgingServices groups service dependencies.
type ListRecoverablesAgingServices struct {
	Authorizer       ports.Authorizer
	Translator       ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// ListRecoverablesAgingUseCase ages the open balance (> 0) of ISSUED STATEMENT documents by days
// past due as of a date (not-yet-due documents fall in the first band). Permission:
// recovery_document:list.
type ListRecoverablesAgingUseCase struct {
	repositories ListRecoverablesAgingRepositories
	services     ListRecoverablesAgingServices
}

// NewListRecoverablesAgingUseCase creates the use case with grouped dependencies.
func NewListRecoverablesAgingUseCase(r ListRecoverablesAgingRepositories, s ListRecoverablesAgingServices) *ListRecoverablesAgingUseCase {
	return &ListRecoverablesAgingUseCase{repositories: r, services: s}
}

func addAging(r *recoverydocumentpb.RecoverablesAgingRow, days int, amount int64) {
	switch {
	case days <= 30:
		r.Days_0_30 += amount
	case days <= 60:
		r.Days_31_60 += amount
	case days <= 90:
		r.Days_61_90 += amount
	default:
		r.DaysOver_90 += amount
	}
	r.Total += amount
	r.DocumentCount++
}

// Execute returns the aging report: Rows by client+currency, Totals by currency (ClientId nil).
func (uc *ListRecoverablesAgingUseCase) Execute(ctx context.Context, req *recoverydocumentpb.ListRecoverablesAgingRequest) (*recoverydocumentpb.ListRecoverablesAgingResponse, error) {
	out, err := uc.execute(ctx, req)
	if err != nil {
		return nil, usecaseerr.Localize(ctx, uc.services.Translator, "recovery_document", err)
	}
	return out, nil
}

func (uc *ListRecoverablesAgingUseCase) execute(ctx context.Context, req *recoverydocumentpb.ListRecoverablesAgingRequest) (*recoverydocumentpb.ListRecoverablesAgingResponse, error) {
	if err := gate(ctx, uc.services.ActionGatekeeper, entityid.ActionList); err != nil {
		return nil, err
	}
	if req == nil {
		req = &recoverydocumentpb.ListRecoverablesAgingRequest{}
	}
	asOf := req.GetAsOf()
	if asOf == "" {
		asOf = time.Now().UTC().Format(dateLayout)
	}
	asOfT, err := time.Parse(dateLayout, asOf)
	if err != nil {
		return nil, errValidation
	}
	if !blank(req.GetClientId()) {
		if err := verifyClient(ctx, uc.repositories.Client, req.GetClientId()); err != nil {
			return nil, err
		}
	}
	snap, err := LoadSnapshot(ctx, uc.repositories.Balance, req.GetClientId(), asOf)
	if err != nil {
		return nil, err
	}
	type key struct{ client, currency string }
	rows := map[key]*recoverydocumentpb.RecoverablesAgingRow{}
	totals := map[string]*recoverydocumentpb.RecoverablesAgingRow{}
	for _, s := range snap.Statements {
		if s.Balance <= 0 {
			continue
		}
		d := s.Document
		due := d.GetDueDate()
		if due == "" {
			due = d.GetIssueDate()
		}
		days := 0
		if dt, err := time.Parse(dateLayout, due); err == nil {
			days = int(asOfT.Sub(dt).Hours() / 24)
		}
		k := key{d.GetClientId(), d.GetCurrency()}
		if rows[k] == nil {
			client := k.client
			rows[k] = &recoverydocumentpb.RecoverablesAgingRow{ClientId: &client, Currency: k.currency}
		}
		if totals[k.currency] == nil {
			totals[k.currency] = &recoverydocumentpb.RecoverablesAgingRow{Currency: k.currency}
		}
		addAging(rows[k], days, s.Balance)
		addAging(totals[k.currency], days, s.Balance)
	}
	out := &recoverydocumentpb.ListRecoverablesAgingResponse{AsOf: asOf, Success: true}
	for _, r := range rows {
		out.Rows = append(out.Rows, r)
	}
	for _, r := range totals {
		out.Totals = append(out.Totals, r)
	}
	sort.Slice(out.Rows, func(i, j int) bool {
		if out.Rows[i].GetClientId() != out.Rows[j].GetClientId() {
			return out.Rows[i].GetClientId() < out.Rows[j].GetClientId()
		}
		return out.Rows[i].GetCurrency() < out.Rows[j].GetCurrency()
	})
	sort.Slice(out.Totals, func(i, j int) bool { return out.Totals[i].GetCurrency() < out.Totals[j].GetCurrency() })
	return out, nil
}
