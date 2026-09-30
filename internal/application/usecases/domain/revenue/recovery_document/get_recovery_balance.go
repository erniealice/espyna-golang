package recovery_document

import (
	"context"
	"sort"

	"github.com/erniealice/espyna-golang/internal/application/shared/usecaseerr"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	clientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client"
	recoverydocumentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/recovery_document"
)

// GetRecoveryBalanceRepositories groups repository dependencies.
type GetRecoveryBalanceRepositories struct {
	Balance BalanceRepos
	Client  clientpb.ClientDomainServiceServer
}

// GetRecoveryBalanceServices groups service dependencies.
type GetRecoveryBalanceServices struct {
	Authorizer       ports.Authorizer
	Translator       ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// GetRecoveryBalanceUseCase: balance = Σ ISSUED docs (signed) − Σ APPLIED CASH applications,
// per currency. Permission: recovery_document:list.
type GetRecoveryBalanceUseCase struct {
	repositories GetRecoveryBalanceRepositories
	services     GetRecoveryBalanceServices
}

// NewGetRecoveryBalanceUseCase creates the use case with grouped dependencies.
func NewGetRecoveryBalanceUseCase(r GetRecoveryBalanceRepositories, s GetRecoveryBalanceServices) *GetRecoveryBalanceUseCase {
	return &GetRecoveryBalanceUseCase{repositories: r, services: s}
}

// Execute returns the client's per-currency recovery balance.
func (uc *GetRecoveryBalanceUseCase) Execute(ctx context.Context, req *recoverydocumentpb.GetRecoveryBalanceRequest) (*recoverydocumentpb.GetRecoveryBalanceResponse, error) {
	out, err := uc.execute(ctx, req)
	if err != nil {
		return nil, usecaseerr.Localize(ctx, uc.services.Translator, "recovery_document", err)
	}
	return out, nil
}

func (uc *GetRecoveryBalanceUseCase) execute(ctx context.Context, req *recoverydocumentpb.GetRecoveryBalanceRequest) (*recoverydocumentpb.GetRecoveryBalanceResponse, error) {
	if err := gate(ctx, uc.services.ActionGatekeeper, entityid.ActionList); err != nil {
		return nil, err
	}
	if req == nil || blank(req.GetClientId()) {
		return nil, errValidation
	}
	if err := verifyClient(ctx, uc.repositories.Client, req.GetClientId()); err != nil {
		return nil, err
	}
	snap, err := LoadSnapshot(ctx, uc.repositories.Balance, req.GetClientId(), "")
	if err != nil {
		return nil, err
	}
	by := map[string]*recoverydocumentpb.RecoveryCurrencyBalance{}
	get := func(cur string) *recoverydocumentpb.RecoveryCurrencyBalance {
		if by[cur] == nil {
			by[cur] = &recoverydocumentpb.RecoveryCurrencyBalance{Currency: cur}
		}
		return by[cur]
	}
	for _, s := range snap.Statements {
		get(s.Document.GetCurrency()).Billed += s.Document.GetTotalAmount()
	}
	for _, c := range snap.CreditNotes {
		get(c.GetCurrency()).Billed += c.GetTotalAmount()
	}
	for _, a := range snap.Applications {
		get(a.GetCurrency()).Applied += a.GetAmount()
	}
	out := &recoverydocumentpb.GetRecoveryBalanceResponse{ClientId: req.GetClientId(), Success: true}
	for _, cb := range by {
		cb.Balance = cb.Billed - cb.Applied
		out.ByCurrency = append(out.ByCurrency, cb)
	}
	sort.Slice(out.ByCurrency, func(i, j int) bool { return out.ByCurrency[i].Currency < out.ByCurrency[j].Currency })
	return out, nil
}
