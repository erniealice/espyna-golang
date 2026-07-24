package subscription_group_product_plan_staff

import (
	"context"
	"errors"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	jobtemplatephasepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template_phase"
	productplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_plan"
	productplanstaffpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_plan_staff"
	subscriptiongrouppb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_group"
	sgpppb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_group_product_plan"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_group_product_plan_staff"
)

type UpdateSubscriptionGroupProductPlanStaffRepositories struct {
	SubscriptionGroupProductPlanStaff pb.SubscriptionGroupProductPlanStaffDomainServiceServer
	ProductPlanStaff                  productplanstaffpb.ProductPlanStaffDomainServiceServer
	ProductPlan                       productplanpb.ProductPlanDomainServiceServer
	SubscriptionGroup                 subscriptiongrouppb.SubscriptionGroupDomainServiceServer
	// v2 class-edge anchors (espyna.md §2) — see create_subscription_group_product_plan_staff.go.
	SubscriptionGroupProductPlan sgpppb.SubscriptionGroupProductPlanDomainServiceServer
	JobTemplatePhase             jobtemplatephasepb.JobTemplatePhaseDomainServiceServer
}

type UpdateSubscriptionGroupProductPlanStaffServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

type UpdateSubscriptionGroupProductPlanStaffUseCase struct {
	repositories UpdateSubscriptionGroupProductPlanStaffRepositories
	services     UpdateSubscriptionGroupProductPlanStaffServices
}

func NewUpdateSubscriptionGroupProductPlanStaffUseCase(r UpdateSubscriptionGroupProductPlanStaffRepositories, s UpdateSubscriptionGroupProductPlanStaffServices) *UpdateSubscriptionGroupProductPlanStaffUseCase {
	return &UpdateSubscriptionGroupProductPlanStaffUseCase{repositories: r, services: s}
}

func (uc *UpdateSubscriptionGroupProductPlanStaffUseCase) Execute(ctx context.Context, req *pb.UpdateSubscriptionGroupProductPlanStaffRequest) (*pb.UpdateSubscriptionGroupProductPlanStaffResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.SubscriptionGroupProductPlanStaff, Action: entityid.ActionUpdate}); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.GetId() == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription_group_product_plan_staff.validation.data_required", "Data is required [DEFAULT]"))
	}

	// Fail-closed eligibility guard on the effective (post-update) edge, run in
	// the SAME transaction as the write so the merge-read, the eligibility check
	// and the update commit atomically — a concurrent deactivation of the
	// matching product_plan_staff row cannot slip an ineligible edge through the
	// TOCTOU window (red-team MED). FKs are mutable: merge any FK absent from the
	// update body with the existing row, then re-prove eligibility + plan match +
	// single-workspace tenancy.
	wsID := contextutil.ExtractWorkspaceIDFromContext(ctx)
	now := time.Now()
	ms := now.UnixMilli()
	s := now.Format(time.RFC3339)
	req.Data.DateModified = &ms
	req.Data.DateModifiedString = &s

	writeFn := func(txCtx context.Context) (*pb.UpdateSubscriptionGroupProductPlanStaffResponse, error) {
		effective, err := uc.effectiveEdge(txCtx, req.Data)
		if err != nil {
			return nil, err
		}
		// v2 resolution on the EFFECTIVE (merged) edge — a partial update that
		// only touches, say, role must still re-derive f8/f9/f10 from the
		// persisted class+pps rows (espyna.md §2 dual-write).
		if err := resolveClassEdgeV2(txCtx, uc.v2Repos(), uc.services.Translator, effective); err != nil {
			return nil, err
		}
		// Echo the resolved/merged fields back onto req.Data — the object that
		// actually gets persisted — so a partial update body still writes the
		// correct dual-write + effective v2 anchors. Legacy f8/f9/f10 are always
		// echoed (protojson's zero-value omission already made an omitted body
		// field a no-op passthrough of the existing value, so this is a
		// behavior-preserving explicit write, not a change). f12/f13/f14 are
		// pointer/optional: only echoed when the effective merge actually
		// resolved a non-empty value, so an edge with no v2 anchors at all stays
		// untouched (legacy-only row).
		req.Data.SubscriptionGroupId = effective.GetSubscriptionGroupId()
		req.Data.ProductPlanId = effective.GetProductPlanId()
		req.Data.StaffId = effective.GetStaffId()
		if v := effective.GetSubscriptionGroupProductPlanId(); v != "" {
			req.Data.SubscriptionGroupProductPlanId = &v
		}
		if v := effective.GetProductPlanStaffId(); v != "" {
			req.Data.ProductPlanStaffId = &v
		}
		if effective.JobTemplatePhaseId != nil {
			req.Data.JobTemplatePhaseId = effective.JobTemplatePhaseId
		}

		if err := validateEligibility(txCtx, uc.eligibilityRepos(), uc.services.Translator, wsID, effective); err != nil {
			return nil, err
		}
		if err := checkDuplicateClassEdge(txCtx, uc.repositories.SubscriptionGroupProductPlanStaff, uc.services.Translator,
			effective.GetSubscriptionGroupProductPlanId(), effective.GetProductPlanStaffId(), effective.GetJobTemplatePhaseId(), effective.GetId()); err != nil {
			return nil, err
		}
		return uc.repositories.SubscriptionGroupProductPlanStaff.UpdateSubscriptionGroupProductPlanStaff(txCtx, req)
	}

	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		var result *pb.UpdateSubscriptionGroupProductPlanStaffResponse
		if err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
			res, err := writeFn(txCtx)
			if err != nil {
				return err
			}
			result = res
			return nil
		}); err != nil {
			return nil, err
		}
		return result, nil
	}
	return writeFn(ctx)
}

func (uc *UpdateSubscriptionGroupProductPlanStaffUseCase) eligibilityRepos() eligibilityRepos {
	return eligibilityRepos{
		ProductPlanStaff:  uc.repositories.ProductPlanStaff,
		ProductPlan:       uc.repositories.ProductPlan,
		SubscriptionGroup: uc.repositories.SubscriptionGroup,
	}
}

func (uc *UpdateSubscriptionGroupProductPlanStaffUseCase) v2Repos() v2Repos {
	return v2Repos{
		SubscriptionGroupProductPlan: uc.repositories.SubscriptionGroupProductPlan,
		ProductPlanStaff:             uc.repositories.ProductPlanStaff,
		ProductPlan:                  uc.repositories.ProductPlan,
		JobTemplatePhase:             uc.repositories.JobTemplatePhase,
	}
}

// effectiveEdge merges the update body over the persisted row so a partial
// update (FK omitted) still validates against the resulting edge. f12/f13/f14
// are proto3 `optional` (explicit-presence pointers): a nil pointer on `in`
// means "omitted, inherit the persisted value"; a non-nil pointer (even to an
// empty string) means "the caller explicitly set/cleared this field" and wins.
func (uc *UpdateSubscriptionGroupProductPlanStaffUseCase) effectiveEdge(ctx context.Context, in *pb.SubscriptionGroupProductPlanStaff) (*pb.SubscriptionGroupProductPlanStaff, error) {
	existingResp, err := uc.repositories.SubscriptionGroupProductPlanStaff.ReadSubscriptionGroupProductPlanStaff(ctx, &pb.ReadSubscriptionGroupProductPlanStaffRequest{
		Data: &pb.SubscriptionGroupProductPlanStaff{Id: in.GetId()},
	})
	if err != nil || existingResp == nil || len(existingResp.GetData()) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription_group_product_plan_staff.errors.not_found", "class edge not found [DEFAULT]"))
	}
	existing := existingResp.GetData()[0]
	merged := &pb.SubscriptionGroupProductPlanStaff{
		Id:                             in.GetId(),
		SubscriptionGroupId:            firstNonEmpty(in.GetSubscriptionGroupId(), existing.GetSubscriptionGroupId()),
		ProductPlanId:                  firstNonEmpty(in.GetProductPlanId(), existing.GetProductPlanId()),
		StaffId:                        firstNonEmpty(in.GetStaffId(), existing.GetStaffId()),
		SubscriptionGroupProductPlanId: firstNonEmptyPtr(in.SubscriptionGroupProductPlanId, existing.SubscriptionGroupProductPlanId),
		ProductPlanStaffId:             firstNonEmptyPtr(in.ProductPlanStaffId, existing.ProductPlanStaffId),
		JobTemplatePhaseId:             firstSetPtr(in.JobTemplatePhaseId, existing.JobTemplatePhaseId),
	}
	return merged, nil
}

// firstNonEmptyPtr picks `in` when present (non-nil), else `existing`.
func firstNonEmptyPtr(in, existing *string) *string {
	if in != nil {
		return in
	}
	return existing
}

// firstSetPtr is an alias of firstNonEmptyPtr kept distinct for the
// job_template_phase_id call site — NULL ("all phases") is a meaningful value
// here, not an absence, so the name documents that a non-nil `in` (even
// pointing at "") always wins over the persisted value.
func firstSetPtr(in, existing *string) *string {
	return firstNonEmptyPtr(in, existing)
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
