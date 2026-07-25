package subscription_group_product_plan_staff

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	productplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_plan"
	productplanstaffpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_plan_staff"
	subscriptiongrouppb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_group"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_group_product_plan_staff"
)

// AssignSubscriptionGroupProductPlanStaffRepositories carries the same repos as
// Create/Update: the class-edge repo plus the eligibility-guard anchors. The
// upsert reuses the Create/Update/Delete use cases verbatim, so the fail-closed
// eligibility guard (active product_plan_staff + plan match + tenancy) stays
// defense-in-depth for the raw API path.
type AssignSubscriptionGroupProductPlanStaffRepositories struct {
	SubscriptionGroupProductPlanStaff pb.SubscriptionGroupProductPlanStaffDomainServiceServer
	ProductPlanStaff                  productplanstaffpb.ProductPlanStaffDomainServiceServer
	ProductPlan                       productplanpb.ProductPlanDomainServiceServer
	SubscriptionGroup                 subscriptiongrouppb.SubscriptionGroupDomainServiceServer
}

type AssignSubscriptionGroupProductPlanStaffServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// AssignSubscriptionGroupProductPlanStaffRequest is the thin upsert input.
// Semantics (§6.3): an ACTIVE edge for (SubscriptionGroupID, ProductPlanID)
//   - StaffID set   → UPDATE that edge's staff/role (change the servicer);
//   - no active edge → CREATE it;
//   - StaffID empty  → soft-deactivate (clear) the active edge, if any.
//
// The section (SubscriptionGroupID) is authoritative from the caller's bound
// path — never a cross-tenant body id (no IDOR). WorkspaceID is the caller's
// bound workspace: it stamps a created edge and scopes the active-edge lookup.
type AssignSubscriptionGroupProductPlanStaffRequest struct {
	WorkspaceID         string
	SubscriptionGroupID string
	ProductPlanID       string
	StaffID             string
	Role                string
}

// AssignOutcome records which branch the upsert took (for the HTMX re-render).
type AssignOutcome string

const (
	AssignOutcomeCreated AssignOutcome = "created"
	AssignOutcomeUpdated AssignOutcome = "updated"
	AssignOutcomeCleared AssignOutcome = "cleared"
	AssignOutcomeNoop    AssignOutcome = "noop"
)

// AssignSubscriptionGroupProductPlanStaffResponse returns the resulting edge
// (nil when cleared/no-op-clear) and the branch taken.
type AssignSubscriptionGroupProductPlanStaffResponse struct {
	Edge    *pb.SubscriptionGroupProductPlanStaff
	Outcome AssignOutcome
}

type AssignSubscriptionGroupProductPlanStaffUseCase struct {
	repositories AssignSubscriptionGroupProductPlanStaffRepositories
	services     AssignSubscriptionGroupProductPlanStaffServices
	create       *CreateSubscriptionGroupProductPlanStaffUseCase
	update       *UpdateSubscriptionGroupProductPlanStaffUseCase
	delete       *DeleteSubscriptionGroupProductPlanStaffUseCase
}

func NewAssignSubscriptionGroupProductPlanStaffUseCase(r AssignSubscriptionGroupProductPlanStaffRepositories, s AssignSubscriptionGroupProductPlanStaffServices) *AssignSubscriptionGroupProductPlanStaffUseCase {
	return &AssignSubscriptionGroupProductPlanStaffUseCase{
		repositories: r,
		services:     s,
		create: NewCreateSubscriptionGroupProductPlanStaffUseCase(CreateSubscriptionGroupProductPlanStaffRepositories{
			SubscriptionGroupProductPlanStaff: r.SubscriptionGroupProductPlanStaff,
			ProductPlanStaff:                  r.ProductPlanStaff,
			ProductPlan:                       r.ProductPlan,
			SubscriptionGroup:                 r.SubscriptionGroup,
		}, CreateSubscriptionGroupProductPlanStaffServices(s)),
		update: NewUpdateSubscriptionGroupProductPlanStaffUseCase(UpdateSubscriptionGroupProductPlanStaffRepositories{
			SubscriptionGroupProductPlanStaff: r.SubscriptionGroupProductPlanStaff,
			ProductPlanStaff:                  r.ProductPlanStaff,
			ProductPlan:                       r.ProductPlan,
			SubscriptionGroup:                 r.SubscriptionGroup,
		}, UpdateSubscriptionGroupProductPlanStaffServices(s)),
		delete: NewDeleteSubscriptionGroupProductPlanStaffUseCase(DeleteSubscriptionGroupProductPlanStaffRepositories{
			SubscriptionGroupProductPlanStaff: r.SubscriptionGroupProductPlanStaff,
		}, DeleteSubscriptionGroupProductPlanStaffServices(s)),
	}
}

// Execute upserts the class edge for (SubscriptionGroupID, ProductPlanID).
//
// Gating model: each WRITE branch delegates to the Create/Update/Delete use
// case, which fail-closed checks its own action permission and (Create/Update)
// runs the eligibility guard inside the write transaction.
//
// The two SHORT-CIRCUIT no-op branches return before reaching any sub-use-case,
// so since 2026-07-25 they carry their OWN explicit gate — delete for
// nothing-to-clear, update for same-value. Before that they had none: an
// unpermissioned principal got a SUCCESS from both, and the same-value branch
// additionally echoed the full persisted edge row back in Edge. Found by the
// 2026-07-25 coverage audit (finding A-G1) and pinned by
// TestAssign_NoopBranches_Denied.
//
// The active-edge lookup is
// workspace-scoped (both by the postgres List's context scoping and the
// in-memory workspace match below), so a caller cannot address another tenant's
// edge. The lookup and the write are not one transaction — a benign TOCTOU
// identical to the existing drawer add/edit flow; the eligibility guard's own
// atomicity is preserved within each sub-use-case.
func (uc *AssignSubscriptionGroupProductPlanStaffUseCase) Execute(ctx context.Context, req *AssignSubscriptionGroupProductPlanStaffRequest) (*AssignSubscriptionGroupProductPlanStaffResponse, error) {
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"subscription_group_product_plan_staff.validation.request_required", "Request is required [DEFAULT]"))
	}
	if req.SubscriptionGroupID == "" || req.ProductPlanID == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"subscription_group_product_plan_staff.validation.assign_target_required",
			"subscription group and product plan are required [DEFAULT]"))
	}

	active, err := uc.findActiveEdge(ctx, req.WorkspaceID, req.SubscriptionGroupID, req.ProductPlanID)
	if err != nil {
		return nil, err
	}

	// Clear: soft-deactivate the active edge (or no-op when none).
	if req.StaffID == "" {
		if active == nil {
			// GATE ADDED 2026-07-25. This branch returns BEFORE uc.delete.Execute,
			// so without its own check an unpermissioned principal reached a
			// SUCCESS response and could probe whether an edge exists. Gate on the
			// verb the short-circuited sub-use-case would have checked.
			if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
				Entity: entityid.SubscriptionGroupProductPlanStaff, Action: entityid.ActionDelete,
			}); err != nil {
				return nil, err
			}
			return &AssignSubscriptionGroupProductPlanStaffResponse{Outcome: AssignOutcomeNoop}, nil
		}
		if _, err := uc.delete.Execute(ctx, &pb.DeleteSubscriptionGroupProductPlanStaffRequest{
			Data: &pb.SubscriptionGroupProductPlanStaff{Id: active.GetId()},
		}); err != nil {
			return nil, err
		}
		return &AssignSubscriptionGroupProductPlanStaffResponse{Outcome: AssignOutcomeCleared}, nil
	}

	// Change the servicer on the existing edge.
	if active != nil {
		if active.GetStaffId() == req.StaffID && active.GetRole() == req.Role {
			// Idempotent: re-saving the same value is a no-op.
			//
			// GATE ADDED 2026-07-25. This was the more serious of the two
			// short-circuits: it returns BEFORE uc.update.Execute AND echoes the
			// FULL persisted edge row back in Edge, so an unpermissioned principal
			// received both a SUCCESS and the row's contents. Gate on the verb the
			// short-circuited sub-use-case would have checked (update — the caller
			// is asserting a desired state on an EXISTING edge).
			if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
				Entity: entityid.SubscriptionGroupProductPlanStaff, Action: entityid.ActionUpdate,
			}); err != nil {
				return nil, err
			}
			return &AssignSubscriptionGroupProductPlanStaffResponse{Edge: active, Outcome: AssignOutcomeNoop}, nil
		}

		// C16 (sibling of C10): reassigning to a DIFFERENT staff whose soft-deleted
		// (group, plan, staff) row survives would mutate the active edge INTO that
		// occupied triple and collide with uq_subscription_group_product_plan_staff_1
		// (no active filter). When such a corpse exists for the TARGET staff,
		// REACTIVATE it and deactivate the prior active edge — never mutate the old
		// row into the collision. Reactivate FIRST so the update use case's
		// in-transaction eligibility guard fail-closes an ineligible reassignment
		// before we touch the current servicer (no window with zero servicer on
		// failure); then soft-deactivate the old edge so exactly one active remains.
		if req.StaffID != active.GetStaffId() {
			inactive, err := uc.findInactiveEdge(ctx, req.WorkspaceID, req.SubscriptionGroupID, req.ProductPlanID, req.StaffID)
			if err != nil {
				return nil, err
			}
			if inactive != nil {
				resp, err := uc.update.Execute(ctx, &pb.UpdateSubscriptionGroupProductPlanStaffRequest{
					Data: &pb.SubscriptionGroupProductPlanStaff{
						Id:                  inactive.GetId(),
						SubscriptionGroupId: req.SubscriptionGroupID,
						ProductPlanId:       req.ProductPlanID,
						StaffId:             req.StaffID,
						Role:                req.Role,
						Active:              true,
					},
				})
				if err != nil {
					return nil, err
				}
				if _, err := uc.delete.Execute(ctx, &pb.DeleteSubscriptionGroupProductPlanStaffRequest{
					Data: &pb.SubscriptionGroupProductPlanStaff{Id: active.GetId()},
				}); err != nil {
					return nil, err
				}
				return &AssignSubscriptionGroupProductPlanStaffResponse{Edge: firstEdge(resp.GetData(), inactive), Outcome: AssignOutcomeUpdated}, nil
			}
		}

		resp, err := uc.update.Execute(ctx, &pb.UpdateSubscriptionGroupProductPlanStaffRequest{
			Data: &pb.SubscriptionGroupProductPlanStaff{
				Id:                  active.GetId(),
				SubscriptionGroupId: req.SubscriptionGroupID,
				ProductPlanId:       req.ProductPlanID,
				StaffId:             req.StaffID,
				Role:                req.Role,
				Active:              true,
			},
		})
		if err != nil {
			return nil, err
		}
		return &AssignSubscriptionGroupProductPlanStaffResponse{Edge: firstEdge(resp.GetData(), active), Outcome: AssignOutcomeUpdated}, nil
	}

	// No active edge for (group, plan). A previously-cleared edge is soft-deleted
	// (active=false), and uq_subscription_group_product_plan_staff_1
	// (group, product_plan, staff) has NO active filter — so the cleared row still
	// occupies the triple and a plain INSERT would collide (422 — C10). When the
	// SAME triple exists inactive, REACTIVATE it (active=true + requested role)
	// instead of inserting a duplicate. The reactivate rides the update use case,
	// so its in-transaction eligibility guard still fail-closes an ineligible
	// reactivation (defense-in-depth); no schema change, soft-delete/audit honored.
	if inactive, err := uc.findInactiveEdge(ctx, req.WorkspaceID, req.SubscriptionGroupID, req.ProductPlanID, req.StaffID); err != nil {
		return nil, err
	} else if inactive != nil {
		resp, err := uc.update.Execute(ctx, &pb.UpdateSubscriptionGroupProductPlanStaffRequest{
			Data: &pb.SubscriptionGroupProductPlanStaff{
				Id:                  inactive.GetId(),
				SubscriptionGroupId: req.SubscriptionGroupID,
				ProductPlanId:       req.ProductPlanID,
				StaffId:             req.StaffID,
				Role:                req.Role,
				Active:              true,
			},
		})
		if err != nil {
			return nil, err
		}
		return &AssignSubscriptionGroupProductPlanStaffResponse{Edge: firstEdge(resp.GetData(), inactive), Outcome: AssignOutcomeCreated}, nil
	}

	// No prior edge at all: create a fresh one.
	resp, err := uc.create.Execute(ctx, &pb.CreateSubscriptionGroupProductPlanStaffRequest{
		Data: &pb.SubscriptionGroupProductPlanStaff{
			WorkspaceId:         req.WorkspaceID,
			SubscriptionGroupId: req.SubscriptionGroupID,
			ProductPlanId:       req.ProductPlanID,
			StaffId:             req.StaffID,
			Role:                req.Role,
		},
	})
	if err != nil {
		return nil, err
	}
	return &AssignSubscriptionGroupProductPlanStaffResponse{Edge: firstEdge(resp.GetData(), nil), Outcome: AssignOutcomeCreated}, nil
}

// findActiveEdge returns the single ACTIVE class edge for (groupID, productPlanID)
// within wsID, or nil when none. The List filter is applied server-side when the
// adapter honors it; the result is re-filtered in memory so the lookup is correct
// regardless (mirrors hasActiveEligibility) and never leaks across workspaces.
func (uc *AssignSubscriptionGroupProductPlanStaffUseCase) findActiveEdge(ctx context.Context, wsID, groupID, productPlanID string) (*pb.SubscriptionGroupProductPlanStaff, error) {
	resp, err := uc.repositories.SubscriptionGroupProductPlanStaff.ListSubscriptionGroupProductPlanStaffs(ctx, &pb.ListSubscriptionGroupProductPlanStaffsRequest{
		Filters: &commonpb.FilterRequest{
			Filters: []*commonpb.TypedFilter{{
				Field:      "subscription_group_id",
				FilterType: &commonpb.TypedFilter_StringFilter{StringFilter: &commonpb.StringFilter{Value: groupID, Operator: commonpb.StringOperator_STRING_EQUALS}},
			}},
		},
	})
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, nil
	}
	for _, row := range resp.GetData() {
		if row == nil || !row.GetActive() {
			continue
		}
		if row.GetSubscriptionGroupId() != groupID || row.GetProductPlanId() != productPlanID {
			continue
		}
		if wsID != "" && row.GetWorkspaceId() != wsID {
			continue
		}
		return row, nil
	}
	return nil, nil
}

// findInactiveEdge returns the single SOFT-DELETED (active=false) class edge for
// the EXACT (groupID, productPlanID, staffID) triple within wsID, or nil when
// none. uq_subscription_group_product_plan_staff_1 has no active filter, so at
// most one row exists per triple regardless of active state; a cleared row still
// occupies the constraint. Reactivating that row (rather than inserting) is the
// C10 fix. Matching includes staffID (unlike findActiveEdge, which scopes to the
// offering): only a prior assignment of the SAME staff may be reactivated; a
// never-before-assigned staff has no soft-deleted row and creates cleanly.
//
// The List MUST carry an explicit active=false BooleanFilter: the postgres List
// operation DEFAULTS to `active = true` unless the caller supplies an explicit
// "active" filter (adapter/core/operations.go List), so a subscription_group_id-
// only query would silently exclude the soft-deleted row and this lookup would
// never find it — falling through to the plain INSERT that collides (the exact
// C10 422). With active=false the soft-deleted row is returned; the result is
// re-filtered in memory (staff match, tenancy) and never leaks across workspaces.
func (uc *AssignSubscriptionGroupProductPlanStaffUseCase) findInactiveEdge(ctx context.Context, wsID, groupID, productPlanID, staffID string) (*pb.SubscriptionGroupProductPlanStaff, error) {
	resp, err := uc.repositories.SubscriptionGroupProductPlanStaff.ListSubscriptionGroupProductPlanStaffs(ctx, &pb.ListSubscriptionGroupProductPlanStaffsRequest{
		Filters: &commonpb.FilterRequest{
			Filters: []*commonpb.TypedFilter{
				{
					Field:      "subscription_group_id",
					FilterType: &commonpb.TypedFilter_StringFilter{StringFilter: &commonpb.StringFilter{Value: groupID, Operator: commonpb.StringOperator_STRING_EQUALS}},
				},
				{
					Field:      "active",
					FilterType: &commonpb.TypedFilter_BooleanFilter{BooleanFilter: &commonpb.BooleanFilter{Value: false}},
				},
			},
		},
	})
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, nil
	}
	for _, row := range resp.GetData() {
		if row == nil || row.GetActive() {
			continue
		}
		if row.GetSubscriptionGroupId() != groupID || row.GetProductPlanId() != productPlanID || row.GetStaffId() != staffID {
			continue
		}
		if wsID != "" && row.GetWorkspaceId() != wsID {
			continue
		}
		return row, nil
	}
	return nil, nil
}

// firstEdge returns the first non-nil edge from a write response, falling back to
// the pre-write row when the adapter echoes no data.
func firstEdge(data []*pb.SubscriptionGroupProductPlanStaff, fallback *pb.SubscriptionGroupProductPlanStaff) *pb.SubscriptionGroupProductPlanStaff {
	for _, e := range data {
		if e != nil {
			return e
		}
	}
	return fallback
}
