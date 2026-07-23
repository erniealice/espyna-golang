package subscription_group

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_group"
	memberpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_group_member"
	sgppspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_group_product_plan_staff"
	sgwupb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_group_workspace_user"
)

// DeleteSubscriptionGroupRepositories carries the parent repo plus the live
// dependent repos consulted by the referential delete guard. A
// subscription_group (the generic per-period COHORT — a class section, a
// patient panel, a project team) must NOT be deletable while ACTIVE dependents
// still reference its id. The three dependents are every table that carries a
// subscription_group_id FK (esqyma proto grep, 2026-07-23):
//   - subscription_group_member                 (roster membership)
//   - subscription_group_product_plan_staff     (servicing / class edges)
//   - subscription_group_workspace_user         (servicing access grants)
//
// This is a cross-vertical primitive ("parent not deletable while active
// dependents exist"): the CODE stays vertical-neutral; the education tier
// relabels the nouns via lyngua only. The dependent repos are nil-tolerant — a
// nil repo contributes zero to the count so the guard degrades gracefully on a
// wiring gap rather than blocking every delete (the initializer always wires
// all three; see composition/.../domain/subscription.go).
type DeleteSubscriptionGroupRepositories struct {
	SubscriptionGroup pb.SubscriptionGroupDomainServiceServer

	Member        memberpb.SubscriptionGroupMemberDomainServiceServer
	TeachingStaff sgppspb.SubscriptionGroupProductPlanStaffDomainServiceServer
	AccessGrant   sgwupb.SubscriptionGroupWorkspaceUserDomainServiceServer
}

type DeleteSubscriptionGroupServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

type DeleteSubscriptionGroupUseCase struct {
	repositories DeleteSubscriptionGroupRepositories
	services     DeleteSubscriptionGroupServices
}

func NewDeleteSubscriptionGroupUseCase(r DeleteSubscriptionGroupRepositories, s DeleteSubscriptionGroupServices) *DeleteSubscriptionGroupUseCase {
	return &DeleteSubscriptionGroupUseCase{repositories: r, services: s}
}

func (uc *DeleteSubscriptionGroupUseCase) Execute(ctx context.Context, req *pb.DeleteSubscriptionGroupRequest) (*pb.DeleteSubscriptionGroupResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.SubscriptionGroup, Action: entityid.ActionDelete}); err != nil {
		return nil, err
	}
	if req == nil || req.GetData() == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription_group.validation.request_required", "Request is required [DEFAULT]"))
	}
	id := req.GetData().GetId()
	if id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "subscription_group.validation.id_required", "Subscription group ID is required [DEFAULT]"))
	}

	// Referential guard: fail closed while active dependents reference the group.
	// Workspace-scoped: the postgres List auto-scopes by the context workspace,
	// and the in-memory re-filter below re-scopes defensively (so dependents in
	// ANOTHER workspace never block, and the mock-backed unit tests are correct
	// regardless of adapter scoping).
	wsID := contextutil.ExtractWorkspaceIDFromContext(ctx)
	counts, err := uc.countActiveDependents(ctx, wsID, id)
	if err != nil {
		return nil, err
	}
	if counts.total() > 0 {
		return nil, errors.New(uc.blockedReferencesMessage(ctx, counts))
	}

	return uc.repositories.SubscriptionGroup.DeleteSubscriptionGroup(ctx, req)
}

// dependentCounts is the per-dependent-type tally of ACTIVE references that block
// a parent delete. Vertical-neutral field names; the lyngua nouns per tier.
type dependentCounts struct {
	members       int
	teachingStaff int
	accessGrants  int
}

func (d dependentCounts) total() int { return d.members + d.teachingStaff + d.accessGrants }

// groupIDFilter builds the single subscription_group_id equality filter shared by
// every dependent List call. The postgres List operation DEFAULTS to active=true
// unless an explicit "active" filter is supplied (adapter/core/operations.go), so
// this filter already restricts to LIVE rows; the in-memory pass re-asserts it.
func groupIDFilter(groupID string) *commonpb.FilterRequest {
	return &commonpb.FilterRequest{
		Filters: []*commonpb.TypedFilter{{
			Field:      "subscription_group_id",
			FilterType: &commonpb.TypedFilter_StringFilter{StringFilter: &commonpb.StringFilter{Value: groupID, Operator: commonpb.StringOperator_STRING_EQUALS}},
		}},
	}
}

// scopeMatch is the shared in-memory predicate: keep only ACTIVE rows whose
// subscription_group_id matches and (when wsID is known) whose workspace matches.
func scopeMatch(active bool, rowGroupID, rowWsID, groupID, wsID string) bool {
	if !active {
		return false
	}
	if rowGroupID != groupID {
		return false
	}
	if wsID != "" && rowWsID != wsID {
		return false
	}
	return true
}

func (uc *DeleteSubscriptionGroupUseCase) countActiveDependents(ctx context.Context, wsID, groupID string) (dependentCounts, error) {
	var c dependentCounts

	if uc.repositories.Member != nil {
		resp, err := uc.repositories.Member.ListSubscriptionGroupMembers(ctx, &memberpb.ListSubscriptionGroupMembersRequest{Filters: groupIDFilter(groupID)})
		if err != nil {
			return c, err
		}
		for _, row := range resp.GetData() {
			if row != nil && scopeMatch(row.GetActive(), row.GetSubscriptionGroupId(), row.GetWorkspaceId(), groupID, wsID) {
				c.members++
			}
		}
	}

	if uc.repositories.TeachingStaff != nil {
		resp, err := uc.repositories.TeachingStaff.ListSubscriptionGroupProductPlanStaffs(ctx, &sgppspb.ListSubscriptionGroupProductPlanStaffsRequest{Filters: groupIDFilter(groupID)})
		if err != nil {
			return c, err
		}
		for _, row := range resp.GetData() {
			if row != nil && scopeMatch(row.GetActive(), row.GetSubscriptionGroupId(), row.GetWorkspaceId(), groupID, wsID) {
				c.teachingStaff++
			}
		}
	}

	if uc.repositories.AccessGrant != nil {
		resp, err := uc.repositories.AccessGrant.ListSubscriptionGroupWorkspaceUsers(ctx, &sgwupb.ListSubscriptionGroupWorkspaceUsersRequest{Filters: groupIDFilter(groupID)})
		if err != nil {
			return c, err
		}
		for _, row := range resp.GetData() {
			if row != nil && scopeMatch(row.GetActive(), row.GetSubscriptionGroupId(), row.GetWorkspaceId(), groupID, wsID) {
				c.accessGrants++
			}
		}
	}

	return c, nil
}

// blockedReferencesMessage renders the fail-closed message enumerating the active
// dependent counts. Each nonzero dimension contributes a "<n> <noun>" clause via a
// per-dependent-type lyngua key (singular/plural).
//
// Go defaults are deliberately VERTICAL-NEUTRAL (code-generic principle: no
// vertical nouns in code — "group", not "section"; "staff assignment", not
// "teacher"). The lyngua keys carry the per-tier vocabulary (general → section /
// teaching staff; education → enrolled student / teacher). NOTE: the use-case
// Translator is currently the port NoOp (getServices in composition/core/
// usecases.go retired the provider), so at runtime this returns the Go defaults —
// the lyngua keys are structurally correct and activate if the use-case
// translator is re-wired; until then the education noun override lands via the
// (label-backed) view layer as a follow-up.
func (uc *DeleteSubscriptionGroupUseCase) blockedReferencesMessage(ctx context.Context, c dependentCounts) string {
	var clauses []string
	if c.members > 0 {
		clauses = append(clauses, fmt.Sprintf("%d %s", c.members, uc.noun(ctx, "subscription_group.validation.dependent_member", "member", "members", c.members)))
	}
	if c.teachingStaff > 0 {
		clauses = append(clauses, fmt.Sprintf("%d %s", c.teachingStaff, uc.noun(ctx, "subscription_group.validation.dependent_teaching_staff", "staff assignment", "staff assignments", c.teachingStaff)))
	}
	if c.accessGrants > 0 {
		clauses = append(clauses, fmt.Sprintf("%d %s", c.accessGrants, uc.noun(ctx, "subscription_group.validation.dependent_access_grant", "access grant", "access grants", c.accessGrants)))
	}
	dependents := strings.Join(clauses, ", ")
	return contextutil.GetTranslatedMessageWithContextAndTags(
		ctx,
		uc.services.Translator,
		"subscription_group.validation.delete_blocked_references",
		map[string]interface{}{"dependents": dependents},
		"Cannot delete this group while it still has active dependents: {dependents}. Remove them first.",
	)
}

// noun resolves a per-dependent-type noun, choosing the singular ("_one") or
// plural ("_many") key by count. The education tier overrides these nouns (via
// lyngua) once the use-case translator is re-wired.
func (uc *DeleteSubscriptionGroupUseCase) noun(ctx context.Context, baseKey, defOne, defMany string, n int) string {
	if n == 1 {
		return contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, baseKey+"_one", defOne)
	}
	return contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, baseKey+"_many", defMany)
}
