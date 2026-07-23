package consumer

// subscription_group_assign.go — public entrypoint for the group-centric
// class-edge upsert (subscription_group_product_plan_staff "assign").
//
// The AssignSubscriptionGroupProductPlanStaff use-case (the thin coordinator
// over Create/Update/Delete that keeps the fail-closed eligibility guard as
// defense-in-depth) lives in
// internal/application/usecases/domain/subscription/subscription_group_product_plan_staff
// and is reachable on the aggregate as
// Subscription.SubscriptionGroupProductPlanStaff.AssignSubscriptionGroupProductPlanStaff.
// Consumer apps (separate Go modules) cannot import that internal package to
// construct its request struct, so this thin pass-through exposes the call with
// no logic of its own — it builds the request and delegates to the REAL
// use-case Execute. The workspace id is read from the request context (the
// caller's bound workspace), never from a caller-supplied argument, so the
// upsert can never be pointed at another tenant's rows.

import (
	"context"
	"fmt"

	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	assignuc "github.com/erniealice/espyna-golang/internal/application/usecases/domain/subscription/subscription_group_product_plan_staff"
)

// AssignGroupServicer upserts the active class edge for
// (subscriptionGroupID, productPlanID):
//   - staffID set + an active edge exists → update its staff/role;
//   - staffID set + no active edge        → create it;
//   - staffID empty                       → soft-deactivate (clear) the edge.
//
// It returns the branch the upsert took ("created"|"updated"|"cleared"|"noop").
// The group id is authoritative from the caller's signed path; the workspace
// id is taken from ctx (the caller's bound workspace). ctx must carry an
// authorized principal — each branch action-gates its own create/update/delete
// permission inside the use case (Layer-4 backstop).
func AssignGroupServicer(ctx context.Context, uc *UseCases, subscriptionGroupID, productPlanID, staffID, role string) (string, error) {
	if uc == nil || uc.Subscription == nil ||
		uc.Subscription.SubscriptionGroupProductPlanStaff == nil ||
		uc.Subscription.SubscriptionGroupProductPlanStaff.AssignSubscriptionGroupProductPlanStaff == nil {
		return "", fmt.Errorf("assign group servicer: use-case not wired on the subscription aggregate")
	}
	resp, err := uc.Subscription.SubscriptionGroupProductPlanStaff.AssignSubscriptionGroupProductPlanStaff.Execute(ctx, &assignuc.AssignSubscriptionGroupProductPlanStaffRequest{
		WorkspaceID:         contextutil.ExtractWorkspaceIDFromContext(ctx),
		SubscriptionGroupID: subscriptionGroupID,
		ProductPlanID:       productPlanID,
		StaffID:             staffID,
		Role:                role,
	})
	if err != nil {
		return "", err
	}
	if resp == nil {
		return "", nil
	}
	return string(resp.Outcome), nil
}
