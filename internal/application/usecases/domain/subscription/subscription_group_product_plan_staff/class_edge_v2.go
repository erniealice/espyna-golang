package subscription_group_product_plan_staff

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	jobtemplatephasepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template_phase"
	productplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_plan"
	productplanstaffpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_plan_staff"
	sgpppb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_group_product_plan"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_group_product_plan_staff"
)

// v2Repos bundles the repos the class-edge v2 write path needs
// (docs/plan/20260724-section-assignment-merged/espyna.md §2). Distinct from
// eligibilityRepos (the legacy sg/pp/staff guard) — v2Repos resolves the NEW
// (class x eligibility [x phase]) anchors and derives the legacy dual-write.
type v2Repos struct {
	SubscriptionGroupProductPlan sgpppb.SubscriptionGroupProductPlanDomainServiceServer
	ProductPlanStaff             productplanstaffpb.ProductPlanStaffDomainServiceServer
	ProductPlan                  productplanpb.ProductPlanDomainServiceServer
	JobTemplatePhase             jobtemplatephasepb.JobTemplatePhaseDomainServiceServer
}

// resolveClassEdgeV2 mutates `data` in place: when both f12
// (subscription_group_product_plan_id) and f13 (product_plan_staff_id) are
// present, it reads the class + eligibility rows, validates the v2 invariants
// (plan.md §1.2 #1, §2.5's write-side extension), and dual-writes the resolved
// legacy f8/f9/f10 onto `data` so every legacy reader stays correct until M7
// (D-8). A legacy-only write (f12 and f13 both empty) is a no-op passthrough —
// the v2 fields are optional during the migration.
func resolveClassEdgeV2(ctx context.Context, r v2Repos, tr ports.Translator, data *pb.SubscriptionGroupProductPlanStaff) error {
	classID := data.GetSubscriptionGroupProductPlanId()
	ppsID := data.GetProductPlanStaffId()
	if classID == "" && ppsID == "" {
		return nil
	}
	if classID == "" || ppsID == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, tr,
			"subscription_group_product_plan_staff.validation.v2_anchors_required_together",
			"class and eligibility must be set together for a class assignment [DEFAULT]"))
	}
	if r.SubscriptionGroupProductPlan == nil || r.ProductPlanStaff == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, tr,
			"subscription_group_product_plan_staff.errors.v2_validation_unavailable",
			"class-edge validation is not configured [DEFAULT]"))
	}

	classResp, err := r.SubscriptionGroupProductPlan.ReadSubscriptionGroupProductPlan(ctx, &sgpppb.ReadSubscriptionGroupProductPlanRequest{
		Data: &sgpppb.SubscriptionGroupProductPlan{Id: classID},
	})
	if err != nil || classResp == nil || len(classResp.GetData()) == 0 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, tr,
			"subscription_group_product_plan_staff.validation.class_not_found",
			"class not found [DEFAULT]"))
	}
	class := classResp.GetData()[0]
	if !class.GetActive() {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, tr,
			"subscription_group_product_plan_staff.validation.class_not_active",
			"class is not active [DEFAULT]"))
	}

	ppsResp, err := r.ProductPlanStaff.ReadProductPlanStaff(ctx, &productplanstaffpb.ReadProductPlanStaffRequest{
		Data: &productplanstaffpb.ProductPlanStaff{Id: ppsID},
	})
	if err != nil || ppsResp == nil || len(ppsResp.GetData()) == 0 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, tr,
			"subscription_group_product_plan_staff.validation.eligibility_not_found",
			"eligibility row not found [DEFAULT]"))
	}
	ppsRow := ppsResp.GetData()[0]
	if !ppsRow.GetActive() {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, tr,
			"subscription_group_product_plan_staff.validation.eligibility_not_active",
			"eligibility row is not active [DEFAULT]"))
	}

	// plan.md §1.2 #1: pps.product_plan_id == class.product_plan_id.
	if ppsRow.GetProductPlanId() == "" || class.GetProductPlanId() == "" || ppsRow.GetProductPlanId() != class.GetProductPlanId() {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, tr,
			"subscription_group_product_plan_staff.validation.eligibility_offering_mismatch",
			"the eligibility row's offering does not match the class's offering [DEFAULT]"))
	}

	// f14 write-side validation (plan.md §2.5 "Write-side validation"): the
	// phase must belong to the class's own template AND be variant-compatible
	// — a "Music class + Visual-Arts phase" assignment is rejected here.
	if phaseID := data.GetJobTemplatePhaseId(); phaseID != "" {
		if r.JobTemplatePhase == nil {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, tr,
				"subscription_group_product_plan_staff.errors.phase_validation_unavailable",
				"phase validation is not configured [DEFAULT]"))
		}
		phaseResp, err := r.JobTemplatePhase.ReadJobTemplatePhase(ctx, &jobtemplatephasepb.ReadJobTemplatePhaseRequest{
			Data: &jobtemplatephasepb.JobTemplatePhase{Id: phaseID},
		})
		if err != nil || phaseResp == nil || len(phaseResp.GetData()) == 0 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, tr,
				"subscription_group_product_plan_staff.validation.phase_not_found",
				"phase not found [DEFAULT]"))
		}
		phase := phaseResp.GetData()[0]
		if phase.GetJobTemplateId() != class.GetJobTemplateId() {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, tr,
				"subscription_group_product_plan_staff.validation.phase_not_in_class_template",
				"the phase does not belong to the class's curriculum [DEFAULT]"))
		}
		if r.ProductPlan != nil {
			ppResp, err := r.ProductPlan.ReadProductPlan(ctx, &productplanpb.ReadProductPlanRequest{
				Data: &productplanpb.ProductPlan{Id: class.GetProductPlanId()},
			})
			if err == nil && ppResp != nil && len(ppResp.GetData()) > 0 {
				if classVariant := ppResp.GetData()[0].GetProductVariantId(); classVariant != "" && phase.GetOutputProductVariantId() != classVariant {
					return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, tr,
						"subscription_group_product_plan_staff.validation.phase_variant_mismatch",
						"the phase is not compatible with the class's strand [DEFAULT]"))
				}
			}
		}
	}

	// Dual-write (espyna.md §2): resolve legacy f8/f9/f10 from the class + pps
	// rows so every legacy reader stays correct until M7 (D-8). Parity lint:
	// f8/f9/f10 ≡ derived-from-f12/f13.
	data.SubscriptionGroupId = class.GetSubscriptionGroupId()
	data.ProductPlanId = class.GetProductPlanId()
	data.StaffId = ppsRow.GetStaffId()
	return nil
}

// checkDuplicateClassEdge surfaces the esqyma.md §2 partial unique index
// uq_sgpps_class_pps_phase (subscription_group_product_plan_id,
// product_plan_staff_id, COALESCE(job_template_phase_id, ”)) as a friendly
// duplicate error. Only runs when both v2 anchors are present. excludeID skips
// the row being updated (empty on create).
func checkDuplicateClassEdge(ctx context.Context, repo pb.SubscriptionGroupProductPlanStaffDomainServiceServer, tr ports.Translator, classID, ppsID, phaseID, excludeID string) error {
	if classID == "" || ppsID == "" {
		return nil
	}
	resp, err := repo.ListSubscriptionGroupProductPlanStaffs(ctx, &pb.ListSubscriptionGroupProductPlanStaffsRequest{
		Filters: &commonpb.FilterRequest{
			Filters: []*commonpb.TypedFilter{{
				Field:      "subscription_group_product_plan_id",
				FilterType: &commonpb.TypedFilter_StringFilter{StringFilter: &commonpb.StringFilter{Value: classID, Operator: commonpb.StringOperator_STRING_EQUALS}},
			}},
		},
	})
	if err != nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, tr,
			"subscription_group_product_plan_staff.errors.duplicate_check_failed",
			"failed to validate uniqueness [DEFAULT]"))
	}
	for _, row := range resp.GetData() {
		if row == nil || !row.GetActive() {
			continue
		}
		if excludeID != "" && row.GetId() == excludeID {
			continue
		}
		if row.GetSubscriptionGroupProductPlanId() != classID || row.GetProductPlanStaffId() != ppsID {
			continue
		}
		if row.GetJobTemplatePhaseId() != phaseID {
			continue // distinct phase-or-whole slices coexist by design
		}
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, tr,
			"subscription_group_product_plan_staff.validation.duplicate_class_edge",
			"this staff is already assigned to this class for this phase [DEFAULT]"))
	}
	return nil
}

// findReactivatableLegacyEdge looks for an INACTIVE row already occupying the
// resolved legacy (subscription_group_id, product_plan_id, staff_id) triple.
// esqyma.md §2 keeps the pre-v2 unique index
// uq_subscription_group_product_plan_staff_1 verbatim through this migration
// ("Existing unique_together ... STAYS through the migration") — it is NOT
// partial-on-active, so a soft-deleted (Cleared) row for the SAME staff on
// the SAME (section, offering) blocks a brand-new INSERT of that legacy
// triple even though the v2, active-only duplicate check
// (checkDuplicateClassEdge, keyed on class+pps+phase) correctly treats the
// slot as available. "Clear, then re-add the same teacher" — including to a
// DIFFERENT phase — is exactly this shape and must not silently 4xx.
// LIVE-FOUND (checkpoint C2 Case 3, 2026-07-24): pool[0]'s Case-1 all-phases
// row, Cleared in Case 2, blocked Case 3's fresh insert for the identical
// (section, offering, pool[0].staff_id) triple on phase A. Returns nil when
// no reactivation target exists — the caller's normal insert path proceeds
// unchanged (a genuine DB error still surfaces loudly).
//
// MUST carry an explicit active=false filter — the same fix already landed
// once in this package under the "C10" finding
// (AssignSubscriptionGroupProductPlanStaffUseCase.findInactiveEdge, this
// file's sibling upsert path, not currently wired to action/assign.go): the
// postgres List operation DEFAULTS to `active = true` unless the caller
// supplies an explicit "active" filter (adapter/core/operations.go List), so
// a subscription_group_id/product_plan_id/staff_id-only query silently
// excludes the very row this function exists to find — the exact miss this
// function suffered on its first pass (confirmed live: 69 "reactivate"
// symbol hits in the compiled binary yet Case 1 still 422'd, because the
// List call never returned the inactive row for the client-side filter to
// see).
func findReactivatableLegacyEdge(ctx context.Context, repo pb.SubscriptionGroupProductPlanStaffDomainServiceServer, sgID, ppID, staffID string) *pb.SubscriptionGroupProductPlanStaff {
	if repo == nil || sgID == "" || ppID == "" || staffID == "" {
		return nil
	}
	resp, err := repo.ListSubscriptionGroupProductPlanStaffs(ctx, &pb.ListSubscriptionGroupProductPlanStaffsRequest{
		Filters: &commonpb.FilterRequest{
			Filters: []*commonpb.TypedFilter{
				{Field: "subscription_group_id", FilterType: &commonpb.TypedFilter_StringFilter{StringFilter: &commonpb.StringFilter{Value: sgID, Operator: commonpb.StringOperator_STRING_EQUALS}}},
				{Field: "product_plan_id", FilterType: &commonpb.TypedFilter_StringFilter{StringFilter: &commonpb.StringFilter{Value: ppID, Operator: commonpb.StringOperator_STRING_EQUALS}}},
				{Field: "staff_id", FilterType: &commonpb.TypedFilter_StringFilter{StringFilter: &commonpb.StringFilter{Value: staffID, Operator: commonpb.StringOperator_STRING_EQUALS}}},
				{Field: "active", FilterType: &commonpb.TypedFilter_BooleanFilter{BooleanFilter: &commonpb.BooleanFilter{Value: false}}},
			},
		},
	})
	if err != nil || resp == nil {
		return nil
	}
	// Re-verify the full match client-side — mirrors findInactiveEdge's own
	// defensive re-filter ("the result is re-filtered in memory... regardless
	// of whether the adapter honors [the filters]"). LIVE-FOUND (espyna unit
	// suite regression, 2026-07-24): trusting server-side filtering alone let
	// this function return an UNRELATED corpse — a test mock (and, in
	// principle, any adapter path) that doesn't fully honor all four filters
	// would otherwise hand back the FIRST inactive row for ANY staff on this
	// legacy triple, not necessarily the one matching `staffID`.
	for _, row := range resp.GetData() {
		if row == nil || row.GetActive() {
			continue
		}
		if row.GetSubscriptionGroupId() != sgID || row.GetProductPlanId() != ppID || row.GetStaffId() != staffID {
			continue
		}
		return row
	}
	return nil
}
