package charge_policy

import (
	"context"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	policypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_policy"
	versionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_policy_version"
	enumspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/enums"
)

// ResolveChargePolicyRepositories groups the repository dependencies of ResolveChargePolicy.
type ResolveChargePolicyRepositories struct {
	ChargePolicy        policypb.ChargePolicyDomainServiceServer
	ChargePolicyVersion versionpb.ChargePolicyVersionDomainServiceServer
}

// ResolveChargePolicyServices groups the service dependencies of ResolveChargePolicy.
type ResolveChargePolicyServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// ResolveChargePolicyUseCase resolves package-line policy -> (product default, S2: none in S1)
// -> the policy's current APPROVED version (AC-CP-07). A retired policy or one without an
// APPROVED version is a named refusal (retired / no_approved_version), never a silent fallback.
type ResolveChargePolicyUseCase struct{ c *core }

// NewResolveChargePolicyUseCase creates the use case with grouped dependencies.
func NewResolveChargePolicyUseCase(r ResolveChargePolicyRepositories, s ResolveChargePolicyServices) *ResolveChargePolicyUseCase {
	return &ResolveChargePolicyUseCase{c: &core{
		repos: Repositories{
			ChargePolicy:        r.ChargePolicy,
			ChargePolicyVersion: r.ChargePolicyVersion,
		},
		svc: Services(s),
	}}
}

// Resolution sources (ResolveChargePolicyResponse.Source).
const (
	SourceNone        = "none"         // no policy configured: legacy path
	SourcePackageLine = "package_line" // product_price_plan.charge_policy_id
	// SourceProductDefault is reserved for S2 (product.charge_policy_id); S1 never returns it.
	SourceProductDefault = "product_default"
)

// Execute resolves the effective policy version of a package line under the charge_policy:read
// gate.
func (uc *ResolveChargePolicyUseCase) Execute(ctx context.Context, req *policypb.ResolveChargePolicyRequest) (*policypb.ResolveChargePolicyResponse, error) {
	if err := uc.c.gate(ctx, entityid.ActionRead); err != nil {
		return nil, err
	}
	return uc.ResolveUngated(ctx, req)
}

// ResolveUngated is the resolution rule of Execute without the charge_policy:read gate. It is an
// intra-domain helper for a use case that COMPOSES this rule under its own gate (subscription
// create, under subscription:create): such a caller must not inherit charge_policy:read, so an
// actor who may create a subscription can create it on an opted-in plan without the policy read
// grant (R4 m9). It stays workspace-scoped (the repositories require the trusted workspace) and
// keeps every named refusal. Never expose it to a presentation layer.
func (uc *ResolveChargePolicyUseCase) ResolveUngated(ctx context.Context, req *policypb.ResolveChargePolicyRequest) (*policypb.ResolveChargePolicyResponse, error) {
	c := uc.c
	if req == nil || blank(req.GetPackageLineChargePolicyId()) {
		// S1: the product default is not consulted (S2), so no line policy = legacy path.
		return &policypb.ResolveChargePolicyResponse{Source: SourceNone, Success: true}, nil
	}
	p, err := c.readPolicy(ctx, req.GetPackageLineChargePolicyId())
	if err != nil {
		return nil, err
	}
	if p.GetStatus() == enumspb.ChargePolicyStatus_CHARGE_POLICY_STATUS_RETIRED {
		return nil, c.refuse(ctx, CodeRetired, "charge policy %s is retired", p.GetId())
	}
	vs, err := c.listVersions(ctx, p.GetId())
	if err != nil {
		return nil, err
	}
	cur := currentApproved(vs)
	if cur == nil {
		return nil, c.refuse(ctx, CodeNoApprovedVersion, "charge policy %s has no approved version", p.GetId())
	}
	return &policypb.ResolveChargePolicyResponse{Source: SourcePackageLine, Policy: p, Version: cur, Success: true}, nil
}
