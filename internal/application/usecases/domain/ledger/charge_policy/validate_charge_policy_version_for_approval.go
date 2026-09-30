package charge_policy

import (
	"context"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	accountpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/account"
	componentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_policy_component"
	postingpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_policy_posting"
	versionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_policy_version"
)

// ValidateChargePolicyVersionForApprovalRepositories groups the repository dependencies of ValidateChargePolicyVersionForApproval.
type ValidateChargePolicyVersionForApprovalRepositories struct {
	ChargePolicyVersion   versionpb.ChargePolicyVersionDomainServiceServer
	ChargePolicyComponent componentpb.ChargePolicyComponentDomainServiceServer
	ChargePolicyPosting   postingpb.ChargePolicyPostingDomainServiceServer
	Account               accountpb.AccountDomainServiceServer
}

// ValidateChargePolicyVersionForApprovalServices groups the service dependencies of ValidateChargePolicyVersionForApproval.
type ValidateChargePolicyVersionForApprovalServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// ValidateChargePolicyVersionForApprovalUseCase returns the checklist items and S1 matrix verdict
// for the approval dialog. It only reads; approval re-runs the same validation inside its transaction.
type ValidateChargePolicyVersionForApprovalUseCase struct{ c *core }

// NewValidateChargePolicyVersionForApprovalUseCase creates the use case with grouped dependencies.
func NewValidateChargePolicyVersionForApprovalUseCase(r ValidateChargePolicyVersionForApprovalRepositories, s ValidateChargePolicyVersionForApprovalServices) *ValidateChargePolicyVersionForApprovalUseCase {
	return &ValidateChargePolicyVersionForApprovalUseCase{c: &core{
		repos: Repositories{
			ChargePolicyVersion:   r.ChargePolicyVersion,
			ChargePolicyComponent: r.ChargePolicyComponent,
			ChargePolicyPosting:   r.ChargePolicyPosting,
			Account:               r.Account,
		},
		svc: Services(s),
	}}
}

// Execute evaluates the version's approval readiness.
func (uc *ValidateChargePolicyVersionForApprovalUseCase) Execute(ctx context.Context, req *versionpb.ValidateChargePolicyVersionForApprovalRequest) (*versionpb.ValidateChargePolicyVersionForApprovalResponse, error) {
	c := uc.c
	if err := c.gate(ctx, entityid.ActionRead); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, c.refuse(ctx, CodeValidation, "request is required")
	}
	v, err := c.readVersion(ctx, req.GetChargePolicyVersionId())
	if err != nil {
		return nil, err
	}
	res, err := c.validate(ctx, v)
	if err != nil {
		return nil, err
	}
	return &versionpb.ValidateChargePolicyVersionForApprovalResponse{Checklist: res, Success: true}, nil
}
