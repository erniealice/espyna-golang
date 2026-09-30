package charge_policy

import (
	"context"

	"github.com/erniealice/espyna-golang/internal/application/shared/usecaseerr"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	postingpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_policy_posting"
)

// ListChargePolicyPostingsRepositories groups the repository dependencies of ListChargePolicyPostings.
type ListChargePolicyPostingsRepositories struct {
	ChargePolicyPosting postingpb.ChargePolicyPostingDomainServiceServer
}

// ListChargePolicyPostingsServices groups the service dependencies of ListChargePolicyPostings.
type ListChargePolicyPostingsServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// ListChargePolicyPostingsUseCase lists the postings of one version (scope charge_policy_version_id).
type ListChargePolicyPostingsUseCase struct{ c *core }

// NewListChargePolicyPostingsUseCase creates the use case with grouped dependencies.
func NewListChargePolicyPostingsUseCase(r ListChargePolicyPostingsRepositories, s ListChargePolicyPostingsServices) *ListChargePolicyPostingsUseCase {
	return &ListChargePolicyPostingsUseCase{c: &core{
		repos: Repositories{
			ChargePolicyPosting: r.ChargePolicyPosting,
		},
		svc: Services(s),
	}}
}

// Execute lists the postings of the scoped version (paginated by the request).
func (uc *ListChargePolicyPostingsUseCase) Execute(ctx context.Context, req *postingpb.ListChargePolicyPostingsRequest) (*postingpb.ListChargePolicyPostingsResponse, error) {
	c := uc.c
	if err := c.gate(ctx, entityid.ActionRead); err != nil {
		return nil, err
	}
	if req == nil || blank(req.GetChargePolicyVersionId()) {
		return nil, c.refuse(ctx, CodeValidation, "charge policy version id is required")
	}
	filters, err := c.scopeFilter(ctx, req.GetFilters(), "charge_policy_version_id", req.GetChargePolicyVersionId())
	if err != nil {
		return nil, err
	}
	scoped := &postingpb.ListChargePolicyPostingsRequest{Search: req.Search, Filters: filters, Sort: req.Sort, Pagination: req.Pagination, ChargePolicyVersionId: req.ChargePolicyVersionId}
	resp, err := c.repos.ChargePolicyPosting.ListChargePolicyPostings(ctx, scoped)
	if err != nil {
		return nil, usecaseerr.RepoErr("charge_policy", "list charge_policy_posting", err, req.GetChargePolicyVersionId())
	}
	return resp, nil
}
