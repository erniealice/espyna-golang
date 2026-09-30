package charge_policy

import (
	"context"

	"github.com/erniealice/espyna-golang/internal/application/shared/listdata"
	"github.com/erniealice/espyna-golang/internal/application/shared/usecaseerr"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	policypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_policy"
	versionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_policy_version"
	enumspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/enums"
)

// ListPickerChargePoliciesRepositories groups the repository dependencies of ListPickerChargePolicies.
type ListPickerChargePoliciesRepositories struct {
	ChargePolicy        policypb.ChargePolicyDomainServiceServer
	ChargePolicyVersion versionpb.ChargePolicyVersionDomainServiceServer
}

// ListPickerChargePoliciesServices groups the service dependencies of ListPickerChargePolicies.
type ListPickerChargePoliciesServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// ListPickerChargePoliciesUseCase returns only ACTIVE policies that have an APPROVED version:
// the package-line picker list. Retired and never-approved policies are excluded (AC-CP-05);
// references already pinned to a retired policy are untouched.
type ListPickerChargePoliciesUseCase struct{ c *core }

// NewListPickerChargePoliciesUseCase creates the use case with grouped dependencies.
func NewListPickerChargePoliciesUseCase(r ListPickerChargePoliciesRepositories, s ListPickerChargePoliciesServices) *ListPickerChargePoliciesUseCase {
	return &ListPickerChargePoliciesUseCase{c: &core{
		repos: Repositories{
			ChargePolicy:        r.ChargePolicy,
			ChargePolicyVersion: r.ChargePolicyVersion,
		},
		svc: Services(s),
	}}
}

// Execute returns every ACTIVE policy with an APPROVED version (a picker needs the full set;
// both reads page through all rows, so nothing is silently capped).
func (uc *ListPickerChargePoliciesUseCase) Execute(ctx context.Context, _ *policypb.ListPickerChargePoliciesRequest) (*policypb.ListPickerChargePoliciesResponse, error) {
	c := uc.c
	if err := c.gate(ctx, entityid.ActionList); err != nil {
		return nil, err
	}
	active, err := listdata.ListAll(func(p *commonpb.PaginationRequest, s *commonpb.SortRequest) ([]*policypb.ChargePolicy, error) {
		resp, err := c.repos.ChargePolicy.GetChargePolicyListPageData(ctx, &policypb.GetChargePolicyListPageDataRequest{Filters: listdata.EqFilter("status", enumspb.ChargePolicyStatus_CHARGE_POLICY_STATUS_ACTIVE.String()), Sort: s, Pagination: p})
		if err != nil {
			return nil, usecaseerr.RepoErr("charge_policy", "list charge_policy", err)
		}
		return resp.GetChargePolicyList(), nil
	})
	if err != nil {
		return nil, err
	}
	approved, err := listdata.ListAll(func(p *commonpb.PaginationRequest, s *commonpb.SortRequest) ([]*versionpb.ChargePolicyVersion, error) {
		resp, err := c.repos.ChargePolicyVersion.ListChargePolicyVersions(ctx, &versionpb.ListChargePolicyVersionsRequest{Filters: listdata.EqFilter("status", enumspbVersionApproved.String()), Sort: s, Pagination: p})
		if err != nil {
			return nil, usecaseerr.RepoErr("charge_policy", "list charge_policy_version", err)
		}
		return resp.GetData(), nil
	})
	if err != nil {
		return nil, err
	}
	has := map[string]bool{}
	for _, v := range approved {
		if v.GetStatus() == enumspbVersionApproved {
			has[v.GetChargePolicyId()] = true
		}
	}
	out := &policypb.ListPickerChargePoliciesResponse{Success: true}
	for _, p := range active {
		if p.GetStatus() == enumspb.ChargePolicyStatus_CHARGE_POLICY_STATUS_ACTIVE && has[p.GetId()] {
			out.Data = append(out.Data, p)
		}
	}
	return out, nil
}
