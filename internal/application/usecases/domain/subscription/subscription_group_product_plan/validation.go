package subscription_group_product_plan

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	jobtemplatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template"
	productplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_plan"
	subscriptiongrouppb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_group"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_group_product_plan"
)

// classInvariantRepos bundles the repos the class write-path guard needs.
// subscription_group_product_plan (THE CLASS) binds a section
// (subscription_group), an offering (product_plan) and a curriculum anchor
// (job_template). The guard enforces plan.md §1.2 #1-4:
//
//	(1) pp.plan_id == sg.plan_id — the offering belongs to the section's own plan.
//	(2) template.output_product_id == pp.product_id — job_template_id coherence
//	    (the class's curriculum anchor must actually deliver the offering).
//	(3) all FK rows resolve within the request workspace.
//
// (Invariant #5 — "no staff yet is legal" — is a non-guard: the absence of any
// subscription_group_product_plan_staff row is never checked here.)
type classInvariantRepos struct {
	ProductPlan       productplanpb.ProductPlanDomainServiceServer
	SubscriptionGroup subscriptiongrouppb.SubscriptionGroupDomainServiceServer
	JobTemplate       jobtemplatepb.JobTemplateDomainServiceServer
}

// validateClassInvariants fail-closes unless (1), (2) and (3) all hold for the
// given class data. wsID is the caller's request workspace (must be non-empty).
// JobTemplate is optional (best-effort composition wiring, mirrors the sgpps
// eligibility guard's tolerance for a nil cross-domain repo) — when nil, (2) is
// skipped rather than fail-closed, since a missing wire must not brick every
// class write; the centymo UI's template resolver (plan.md §M0 grounding)
// remains the primary coherence source at creation time.
func validateClassInvariants(ctx context.Context, r classInvariantRepos, tr ports.Translator, wsID string, data *pb.SubscriptionGroupProductPlan) error {
	groupID := data.GetSubscriptionGroupId()
	productPlanID := data.GetProductPlanId()
	templateID := data.GetJobTemplateId()

	if groupID == "" || productPlanID == "" || templateID == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, tr,
			"subscription_group_product_plan.validation.fks_required",
			"subscription group, product plan and job template are required [DEFAULT]"))
	}
	if wsID == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, tr,
			"subscription_group_product_plan.validation.workspace_required",
			"workspace context is required [DEFAULT]"))
	}
	if r.ProductPlan == nil || r.SubscriptionGroup == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, tr,
			"subscription_group_product_plan.errors.validation_unavailable",
			"class validation is not configured [DEFAULT]"))
	}

	// (3a) subscription_group must resolve within the request workspace.
	sgResp, err := r.SubscriptionGroup.ReadSubscriptionGroup(ctx, &subscriptiongrouppb.ReadSubscriptionGroupRequest{
		Data: &subscriptiongrouppb.SubscriptionGroup{Id: groupID},
	})
	if err != nil || sgResp == nil || len(sgResp.GetData()) == 0 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, tr,
			"subscription_group_product_plan.validation.group_not_found",
			"subscription group not found [DEFAULT]"))
	}
	group := sgResp.GetData()[0]
	if group.GetWorkspaceId() == "" || group.GetWorkspaceId() != wsID {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, tr,
			"subscription_group_product_plan.validation.cross_workspace",
			"class must stay within your workspace [DEFAULT]"))
	}

	// (1) product_plan.plan_id == subscription_group.plan_id.
	ppResp, err := r.ProductPlan.ReadProductPlan(ctx, &productplanpb.ReadProductPlanRequest{
		Data: &productplanpb.ProductPlan{Id: productPlanID},
	})
	if err != nil || ppResp == nil || len(ppResp.GetData()) == 0 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, tr,
			"subscription_group_product_plan.validation.product_plan_not_found",
			"product plan not found [DEFAULT]"))
	}
	productPlan := ppResp.GetData()[0]
	if productPlan.GetPlanId() == "" || group.GetPlanId() == "" || productPlan.GetPlanId() != group.GetPlanId() {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, tr,
			"subscription_group_product_plan.validation.plan_mismatch",
			"the offering does not belong to the section's plan [DEFAULT]"))
	}

	// (2) template.output_product_id == pp.product_id — job_template_id coherence.
	// Best-effort: a nil JobTemplate repo skips this leg (composition gap, not a
	// data error); a resolvable template that fails the match is fail-closed.
	if r.JobTemplate != nil {
		jtResp, err := r.JobTemplate.ReadJobTemplate(ctx, &jobtemplatepb.ReadJobTemplateRequest{
			Data: &jobtemplatepb.JobTemplate{Id: templateID},
		})
		if err != nil || jtResp == nil || len(jtResp.GetData()) == 0 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, tr,
				"subscription_group_product_plan.validation.job_template_not_found",
				"job template not found [DEFAULT]"))
		}
		template := jtResp.GetData()[0]
		if template.GetOutputProductId() == "" || template.GetOutputProductId() != productPlan.GetProductId() {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, tr,
				"subscription_group_product_plan.validation.template_mismatch",
				"the job template does not deliver this offering's product [DEFAULT]"))
		}
	}

	return nil
}

// checkDuplicateClass surfaces the (subscription_group_id, product_plan_id)
// unique_together (esqyma.md §1) as a friendly duplicate error rather than a
// raw constraint-violation, mirroring the work_request_type checkCodeUniqueness
// idiom. excludeID skips the row being updated (empty on create).
func checkDuplicateClass(ctx context.Context, repo pb.SubscriptionGroupProductPlanDomainServiceServer, tr ports.Translator, wsID, groupID, productPlanID, excludeID string) error {
	resp, err := repo.ListSubscriptionGroupProductPlans(ctx, &pb.ListSubscriptionGroupProductPlansRequest{
		Filters: &commonpb.FilterRequest{
			Filters: []*commonpb.TypedFilter{{
				Field:      "subscription_group_id",
				FilterType: &commonpb.TypedFilter_StringFilter{StringFilter: &commonpb.StringFilter{Value: groupID, Operator: commonpb.StringOperator_STRING_EQUALS}},
			}},
		},
	})
	if err != nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, tr,
			"subscription_group_product_plan.errors.duplicate_check_failed",
			"failed to validate uniqueness [DEFAULT]"))
	}
	for _, row := range resp.GetData() {
		if row == nil || !row.GetActive() {
			continue
		}
		if excludeID != "" && row.GetId() == excludeID {
			continue
		}
		if row.GetSubscriptionGroupId() != groupID || row.GetProductPlanId() != productPlanID {
			continue
		}
		if wsID != "" && row.GetWorkspaceId() != wsID {
			continue
		}
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, tr,
			"subscription_group_product_plan.validation.duplicate_class",
			"this offering already has a class on this section [DEFAULT]"))
	}
	return nil
}
