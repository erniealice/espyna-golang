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

type CreateSubscriptionGroupProductPlanStaffRepositories struct {
	SubscriptionGroupProductPlanStaff pb.SubscriptionGroupProductPlanStaffDomainServiceServer
	// Eligibility guard anchors (red-team HIGH #5).
	ProductPlanStaff  productplanstaffpb.ProductPlanStaffDomainServiceServer
	ProductPlan       productplanpb.ProductPlanDomainServiceServer
	SubscriptionGroup subscriptiongrouppb.SubscriptionGroupDomainServiceServer
	// v2 class-edge anchors (espyna.md §2) — best-effort: a nil
	// SubscriptionGroupProductPlan repo simply leaves legacy-only (f12/f13
	// empty) writes unaffected; a v2 write with a nil repo fail-closes inside
	// resolveClassEdgeV2.
	SubscriptionGroupProductPlan sgpppb.SubscriptionGroupProductPlanDomainServiceServer
	JobTemplatePhase             jobtemplatephasepb.JobTemplatePhaseDomainServiceServer
}

type CreateSubscriptionGroupProductPlanStaffServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

type CreateSubscriptionGroupProductPlanStaffUseCase struct {
	repositories CreateSubscriptionGroupProductPlanStaffRepositories
	services     CreateSubscriptionGroupProductPlanStaffServices
}

func NewCreateSubscriptionGroupProductPlanStaffUseCase(r CreateSubscriptionGroupProductPlanStaffRepositories, s CreateSubscriptionGroupProductPlanStaffServices) *CreateSubscriptionGroupProductPlanStaffUseCase {
	return &CreateSubscriptionGroupProductPlanStaffUseCase{repositories: r, services: s}
}

func (uc *CreateSubscriptionGroupProductPlanStaffUseCase) Execute(ctx context.Context, req *pb.CreateSubscriptionGroupProductPlanStaffRequest) (*pb.CreateSubscriptionGroupProductPlanStaffResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.SubscriptionGroupProductPlanStaff, Action: entityid.ActionCreate}); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription_group_product_plan_staff.validation.data_required", "Data is required [DEFAULT]"))
	}

	// Fail-closed eligibility guard, run in the SAME transaction as the write so a
	// concurrent deactivation/deletion of the matching product_plan_staff row
	// cannot slip an ineligible edge in between the check and the insert
	// (red-team MED — eligibility snapshot not atomic). enrich() is a pure
	// in-memory step, so it runs before the transaction.
	wsID := contextutil.ExtractWorkspaceIDFromContext(ctx)
	uc.enrich(req.Data)

	writeFn := func(txCtx context.Context) (*pb.CreateSubscriptionGroupProductPlanStaffResponse, error) {
		// v2 resolution FIRST — mutates req.Data's legacy f8/f9/f10 from the
		// class+pps rows (espyna.md §2 dual-write) before the legacy eligibility
		// guard runs, so that guard sees the resolved values on a v2 write.
		if err := resolveClassEdgeV2(txCtx, uc.v2Repos(), uc.services.Translator, req.Data); err != nil {
			return nil, err
		}
		if err := validateEligibility(txCtx, uc.eligibilityRepos(), uc.services.Translator, wsID, req.Data); err != nil {
			return nil, err
		}
		if err := checkDuplicateClassEdge(txCtx, uc.repositories.SubscriptionGroupProductPlanStaff, uc.services.Translator,
			req.Data.GetSubscriptionGroupProductPlanId(), req.Data.GetProductPlanStaffId(), req.Data.GetJobTemplatePhaseId(), ""); err != nil {
			return nil, err
		}
		// Reactivate-on-legacy-collision (class_edge_v2.go's
		// findReactivatableLegacyEdge) — the pre-v2 unique index on (legacy
		// subscription_group_id, product_plan_id, staff_id) stays live through
		// this migration (esqyma.md §2) and is not partial-on-active, so a
		// Cleared row for this exact staff on this (section, offering) would
		// otherwise 4xx a fresh INSERT even though the v2 duplicate check above
		// already proved the (class, pps, phase) slot is free.
		//
		// SCOPED TO GENUINE V2 WRITES ONLY (both v2 anchors present — the same
		// precondition resolveClassEdgeV2/checkDuplicateClassEdge already use):
		// a plain legacy-only write (e.g. via AssignSubscriptionGroupProductPlanStaffUseCase,
		// this file's sibling upsert path — assign_subscription_group_product_plan_staff.go
		// — which already runs its OWN pre-check, findInactiveEdge, before ever
		// calling Create) must not have this second, independent reactivation
		// decision layered underneath it — LIVE-FOUND (espyna unit regression,
		// 2026-07-24): TestAssign_ChangeServicer_CorpseRoundtrip failed because
		// an unscoped check here intercepted a legacy-only "assign a
		// never-before-seen staff" call before it ever reached the plain
		// Create the test expects.
		if req.Data.GetSubscriptionGroupProductPlanId() != "" && req.Data.GetProductPlanStaffId() != "" {
			if reuse := findReactivatableLegacyEdge(txCtx, uc.repositories.SubscriptionGroupProductPlanStaff,
				req.Data.GetSubscriptionGroupId(), req.Data.GetProductPlanId(), req.Data.GetStaffId()); reuse != nil {
				return uc.reactivate(txCtx, reuse.GetId(), req.Data)
			}
		}
		return uc.repositories.SubscriptionGroupProductPlanStaff.CreateSubscriptionGroupProductPlanStaff(txCtx, req)
	}

	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		var result *pb.CreateSubscriptionGroupProductPlanStaffResponse
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

func (uc *CreateSubscriptionGroupProductPlanStaffUseCase) eligibilityRepos() eligibilityRepos {
	return eligibilityRepos{
		ProductPlanStaff:  uc.repositories.ProductPlanStaff,
		ProductPlan:       uc.repositories.ProductPlan,
		SubscriptionGroup: uc.repositories.SubscriptionGroup,
	}
}

func (uc *CreateSubscriptionGroupProductPlanStaffUseCase) v2Repos() v2Repos {
	return v2Repos{
		SubscriptionGroupProductPlan: uc.repositories.SubscriptionGroupProductPlan,
		ProductPlanStaff:             uc.repositories.ProductPlanStaff,
		ProductPlan:                  uc.repositories.ProductPlan,
		JobTemplatePhase:             uc.repositories.JobTemplatePhase,
	}
}

// reactivate turns a Create into an Update against a previously-Cleared row
// occupying the same legacy triple (findReactivatableLegacyEdge) — same
// legacy identity (subscription_group_id/product_plan_id/staff_id), new v2
// anchors (class/pps/phase) and role, active again. Wrapped in the Create
// response shape (both share `{Data []*SubscriptionGroupProductPlanStaff;
// Success bool}` per the postgres adapter) so callers of Execute see no
// difference between "inserted fresh" and "reactivated".
func (uc *CreateSubscriptionGroupProductPlanStaffUseCase) reactivate(ctx context.Context, existingID string, data *pb.SubscriptionGroupProductPlanStaff) (*pb.CreateSubscriptionGroupProductPlanStaffResponse, error) {
	now := time.Now()
	ms := now.UnixMilli()
	s := now.Format(time.RFC3339)
	// LIVE-FOUND (checkpoint C2, Case 1 repeat run, 2026-07-24): a bare
	// pass-through of data.JobTemplatePhaseId is WRONG here. action/assign.go
	// leaves that pointer nil (never sets it) whenever the caller submits "All
	// Phases" (phaseID==""), which is correct for a FRESH INSERT (an omitted
	// column simply defaults to SQL NULL) but wrong for THIS UPDATE-shaped
	// reactivation: an omitted/nil optional field in an UPDATE means "leave
	// the persisted value untouched" (the same explicit-presence convention
	// UpdateSubscriptionGroupProductPlanStaffUseCase.effectiveEdge documents
	// for this exact field), so the target row's STALE phase value (from
	// whatever it held before being Cleared) silently survived reactivation
	// instead of being cleared to NULL. Fix: always write an EXPLICIT pointer
	// — mirrors action/assign.go's own Edit-branch, which always calls
	// strPtr(phaseID) (even for "") for precisely this reason.
	phaseValue := data.GetJobTemplatePhaseId()
	update := &pb.SubscriptionGroupProductPlanStaff{
		Id:                             existingID,
		Active:                         true,
		Role:                           data.GetRole(),
		SubscriptionGroupId:            data.GetSubscriptionGroupId(),
		ProductPlanId:                  data.GetProductPlanId(),
		StaffId:                        data.GetStaffId(),
		SubscriptionGroupProductPlanId: data.SubscriptionGroupProductPlanId,
		ProductPlanStaffId:             data.ProductPlanStaffId,
		JobTemplatePhaseId:             &phaseValue,
		DateModified:                   &ms,
		DateModifiedString:             &s,
	}
	updResp, err := uc.repositories.SubscriptionGroupProductPlanStaff.UpdateSubscriptionGroupProductPlanStaff(ctx, &pb.UpdateSubscriptionGroupProductPlanStaffRequest{Data: update})
	if err != nil {
		return nil, err
	}
	return &pb.CreateSubscriptionGroupProductPlanStaffResponse{Data: updResp.GetData(), Success: updResp.GetSuccess()}, nil
}

func (uc *CreateSubscriptionGroupProductPlanStaffUseCase) enrich(data *pb.SubscriptionGroupProductPlanStaff) {
	now := time.Now()
	if data.Id == "" && uc.services.IDGenerator != nil {
		data.Id = uc.services.IDGenerator.GenerateID()
	}
	data.Active = true
	ms := now.UnixMilli()
	s := now.Format(time.RFC3339)
	data.DateCreated = &ms
	data.DateCreatedString = &s
	data.DateModified = &ms
	data.DateModifiedString = &s
}
