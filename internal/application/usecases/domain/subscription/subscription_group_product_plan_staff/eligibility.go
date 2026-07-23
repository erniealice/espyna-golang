package subscription_group_product_plan_staff

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	productplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_plan"
	productplanstaffpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_plan_staff"
	subscriptiongrouppb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_group"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_group_product_plan_staff"
)

// eligibilityRepos bundles the repos the class-edge write-path guard needs.
// subscription_group_product_plan_staff binds a section (subscription_group), a
// per-plan subject (product_plan), and a teacher (staff). The guard enforces
// (red-team HIGH #5 — "eligibility fail-loud"):
//
//	(a) an ACTIVE product_plan_staff eligibility row exists for
//	    (product_plan_id, staff_id) in the request workspace;
//	(b) product_plan.plan_id == subscription_group.plan_id (the class-edge
//	    self-validation the storybook §5.2 documents);
//	(c) all FK rows resolve within the request workspace.
//
// product_plan itself carries no workspace_id column, so its tenancy (and the
// staff's, for this pairing) is proven by the workspace-scoped product_plan_staff
// eligibility row that references both product_plan_id and staff_id.
type eligibilityRepos struct {
	ProductPlanStaff  productplanstaffpb.ProductPlanStaffDomainServiceServer
	ProductPlan       productplanpb.ProductPlanDomainServiceServer
	SubscriptionGroup subscriptiongrouppb.SubscriptionGroupDomainServiceServer
}

// validateEligibility fail-closes unless (a), (b) and (c) all hold for the given
// class-edge data. wsID is the caller's request workspace (must be non-empty).
func validateEligibility(ctx context.Context, r eligibilityRepos, tr ports.Translator, wsID string, data *pb.SubscriptionGroupProductPlanStaff) error {
	groupID := data.GetSubscriptionGroupId()
	productPlanID := data.GetProductPlanId()
	staffID := data.GetStaffId()

	if groupID == "" || productPlanID == "" || staffID == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, tr,
			"subscription_group_product_plan_staff.validation.fks_required",
			"subscription group, product plan and staff are required [DEFAULT]"))
	}
	if wsID == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, tr,
			"subscription_group_product_plan_staff.validation.workspace_required",
			"workspace context is required [DEFAULT]"))
	}
	if r.ProductPlanStaff == nil || r.ProductPlan == nil || r.SubscriptionGroup == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, tr,
			"subscription_group_product_plan_staff.errors.eligibility_unavailable",
			"eligibility validation is not configured [DEFAULT]"))
	}

	// (a) ACTIVE product_plan_staff eligibility for (product_plan_id, staff_id)
	// within the request workspace.
	if !hasActiveEligibility(ctx, r.ProductPlanStaff, wsID, productPlanID, staffID) {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, tr,
			"subscription_group_product_plan_staff.validation.staff_not_eligible",
			"staff is not eligible for this product plan [DEFAULT]"))
	}

	// (c) subscription_group must resolve within the request workspace.
	sgResp, err := r.SubscriptionGroup.ReadSubscriptionGroup(ctx, &subscriptiongrouppb.ReadSubscriptionGroupRequest{
		Data: &subscriptiongrouppb.SubscriptionGroup{Id: groupID},
	})
	if err != nil || sgResp == nil || len(sgResp.GetData()) == 0 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, tr,
			"subscription_group_product_plan_staff.validation.group_not_found",
			"subscription group not found [DEFAULT]"))
	}
	group := sgResp.GetData()[0]
	if group.GetWorkspaceId() == "" || group.GetWorkspaceId() != wsID {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, tr,
			"subscription_group_product_plan_staff.validation.cross_workspace",
			"class edge must stay within your workspace [DEFAULT]"))
	}

	// (b) product_plan.plan_id == subscription_group.plan_id.
	ppResp, err := r.ProductPlan.ReadProductPlan(ctx, &productplanpb.ReadProductPlanRequest{
		Data: &productplanpb.ProductPlan{Id: productPlanID},
	})
	if err != nil || ppResp == nil || len(ppResp.GetData()) == 0 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, tr,
			"subscription_group_product_plan_staff.validation.product_plan_not_found",
			"product plan not found [DEFAULT]"))
	}
	productPlan := ppResp.GetData()[0]
	if productPlan.GetPlanId() == "" || group.GetPlanId() == "" || productPlan.GetPlanId() != group.GetPlanId() {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, tr,
			"subscription_group_product_plan_staff.validation.plan_mismatch",
			"product plan does not belong to the section's plan [DEFAULT]"))
	}

	return nil
}

// hasActiveEligibility returns true iff an ACTIVE product_plan_staff row exists
// for (product_plan_id, staff_id) within wsID. The result set is filtered
// in-memory so the check is correct regardless of whether the adapter honors
// list filters (the postgres provider already workspace-scopes List; the small
// per-workspace row count keeps the scan cheap).
func hasActiveEligibility(ctx context.Context, repo productplanstaffpb.ProductPlanStaffDomainServiceServer, wsID, productPlanID, staffID string) bool {
	resp, err := repo.ListProductPlanStaffs(ctx, &productplanstaffpb.ListProductPlanStaffsRequest{})
	if err != nil || resp == nil {
		return false
	}
	for _, row := range resp.GetData() {
		if row == nil {
			continue
		}
		if row.GetActive() &&
			row.GetProductPlanId() == productPlanID &&
			row.GetStaffId() == staffID &&
			row.GetWorkspaceId() == wsID {
			return true
		}
	}
	return false
}
