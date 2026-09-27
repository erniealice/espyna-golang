// Package tenantguard holds the use-case-layer tenant checks for tables that
// carry no workspace_id (user, workspace, workspace_user_role, role_permission).
//
// RBAC (actiongate) decides whether a role may perform a verb; it never decides
// whether the TARGET belongs to the actor's workspace. WorkspaceAwareOperations
// cannot scope these tables either (they have no column), so every by-id
// mutation over them must prove tenant ownership here. All checks fail closed:
// no identity, no selected workspace, an unreadable parent, or a mismatch all
// deny. Reads compare the loaded row in memory, so the result does not depend
// on the AUTHZ_ENFORCE shadow/enforce mode of the adapter decorator.
//
// See docs/wiki/articles/infra-auth-tenant-boundary.md and
// docs/plan/20260927-tenant-boundary-hardening/ (Wave 0).
package tenantguard

import (
	"context"
	"errors"

	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	rolepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/role"
	workspaceuserpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace_user"

	"github.com/erniealice/espyna-golang/shared/identity"
)

// ErrOutsideTenant is returned when the target does not belong to the actor's
// workspace. Callers surface it as a generic not-found so foreign ids are not
// confirmed to exist.
var ErrOutsideTenant = errors.New("not found")

// ErrPlatformOperation is returned for mutations of platform-global rows
// (global user email/active/delete, workspace create/delete) requested from a
// tenant context. Decision Q1 of plan 20260927-tenant-boundary-hardening moves
// these to the control plane.
var ErrPlatformOperation = errors.New("this change is managed by platform operators; deactivate the workspace membership instead")

// WorkspaceUserLister is the subset of the workspace_user repository used for
// membership checks.
type WorkspaceUserLister interface {
	ListWorkspaceUsers(context.Context, *workspaceuserpb.ListWorkspaceUsersRequest) (*workspaceuserpb.ListWorkspaceUsersResponse, error)
}

// WorkspaceUserReader reads one workspace_user by id.
type WorkspaceUserReader interface {
	ReadWorkspaceUser(context.Context, *workspaceuserpb.ReadWorkspaceUserRequest) (*workspaceuserpb.ReadWorkspaceUserResponse, error)
}

// RoleReader reads one role by id.
type RoleReader interface {
	ReadRole(context.Context, *rolepb.ReadRoleRequest) (*rolepb.ReadRoleResponse, error)
}

// ActorWorkspaceID returns the actor's selected workspace, failing closed when
// there is no request identity or no workspace has been selected.
func ActorWorkspaceID(ctx context.Context) (string, error) {
	id, err := identity.RequireWorkspace(ctx)
	if err != nil {
		return "", err
	}
	return id.WorkspaceID, nil
}

// RequireActiveMember proves userID holds an ACTIVE workspace_user row in the
// actor's workspace.
func RequireActiveMember(ctx context.Context, repo WorkspaceUserLister, userID string) error {
	wsID, err := ActorWorkspaceID(ctx)
	if err != nil {
		return err
	}
	if repo == nil || userID == "" {
		return ErrOutsideTenant
	}
	resp, err := repo.ListWorkspaceUsers(ctx, &workspaceuserpb.ListWorkspaceUsersRequest{
		Filters: &commonpb.FilterRequest{Filters: []*commonpb.TypedFilter{
			stringEquals("user_id", userID),
			stringEquals("workspace_id", wsID),
		}},
	})
	if err != nil || resp == nil {
		return ErrOutsideTenant
	}
	// Re-check in memory: never trust the adapter to have applied the filter.
	for _, wu := range resp.GetData() {
		if wu != nil && wu.GetActive() && wu.GetUserId() == userID && wu.GetWorkspaceId() == wsID {
			return nil
		}
	}
	return ErrOutsideTenant
}

// RequireReadableUser allows reading a platform-global user only when it is
// the actor themself (profile, timezone), a control-plane operator is acting,
// or the target is an active member of the actor's workspace.
func RequireReadableUser(ctx context.Context, repo WorkspaceUserLister, userID string) error {
	if IsPlatformOperator(ctx) {
		return nil
	}
	if id, err := identity.Require(ctx); err == nil && id.UserID != "" && id.UserID == userID {
		return nil
	}
	return RequireActiveMember(ctx, repo, userID)
}

// RequireWorkspaceUserInTenant loads a workspace_user by id and proves it
// belongs to the actor's workspace. It returns the loaded row.
func RequireWorkspaceUserInTenant(ctx context.Context, repo WorkspaceUserReader, workspaceUserID string) (*workspaceuserpb.WorkspaceUser, error) {
	wsID, err := ActorWorkspaceID(ctx)
	if err != nil {
		return nil, err
	}
	if repo == nil || workspaceUserID == "" {
		return nil, ErrOutsideTenant
	}
	resp, err := repo.ReadWorkspaceUser(ctx, &workspaceuserpb.ReadWorkspaceUserRequest{
		Data: &workspaceuserpb.WorkspaceUser{Id: workspaceUserID},
	})
	if err != nil || resp == nil {
		return nil, ErrOutsideTenant
	}
	for _, wu := range resp.GetData() {
		if wu != nil && wu.GetId() == workspaceUserID && wu.GetWorkspaceId() == wsID {
			return wu, nil
		}
	}
	return nil, ErrOutsideTenant
}

// RequireRoleInTenant loads a role by id and proves it belongs to the actor's
// workspace.
func RequireRoleInTenant(ctx context.Context, repo RoleReader, roleID string) error {
	wsID, err := ActorWorkspaceID(ctx)
	if err != nil {
		return err
	}
	if repo == nil || roleID == "" {
		return ErrOutsideTenant
	}
	resp, err := repo.ReadRole(ctx, &rolepb.ReadRoleRequest{Data: &rolepb.Role{Id: roleID}})
	if err != nil || resp == nil {
		return ErrOutsideTenant
	}
	for _, r := range resp.GetData() {
		if r != nil && r.GetId() == roleID && r.GetWorkspaceId() == wsID {
			return nil
		}
	}
	return ErrOutsideTenant
}

// RequireActorWorkspace proves workspaceID is the actor's own selected workspace.
func RequireActorWorkspace(ctx context.Context, workspaceID string) error {
	wsID, err := ActorWorkspaceID(ctx)
	if err != nil {
		return err
	}
	if workspaceID == "" || workspaceID != wsID {
		return ErrOutsideTenant
	}
	return nil
}

func stringEquals(field, value string) *commonpb.TypedFilter {
	return &commonpb.TypedFilter{
		Field: field,
		FilterType: &commonpb.TypedFilter_StringFilter{
			StringFilter: &commonpb.StringFilter{Value: value, Operator: commonpb.StringOperator_STRING_EQUALS},
		},
	}
}

type platformOperatorKey struct{}

// PlatformOperator identifies an audited control-plane caller.
type PlatformOperator struct {
	Operator string
	Reason   string
}

// WithPlatformOperator marks ctx as a control-plane operation. It is for the
// control-plane CLI only (plan P1, decision Q7): the HTTP request path must
// never call it, which TestNoProductionCallerOfWithPlatformOperator enforces.
// Both operator and reason are required; otherwise ctx is returned unchanged
// (fail closed).
func WithPlatformOperator(ctx context.Context, operator, reason string) context.Context {
	if operator == "" || reason == "" {
		return ctx
	}
	return context.WithValue(ctx, platformOperatorKey{}, PlatformOperator{Operator: operator, Reason: reason})
}

// PlatformOperatorFromContext returns the control-plane marker, if present.
func PlatformOperatorFromContext(ctx context.Context) (PlatformOperator, bool) {
	op, ok := ctx.Value(platformOperatorKey{}).(PlatformOperator)
	return op, ok && op.Operator != "" && op.Reason != ""
}

// IsPlatformOperator reports whether ctx carries the control-plane marker.
func IsPlatformOperator(ctx context.Context) bool {
	_, ok := PlatformOperatorFromContext(ctx)
	return ok
}

// RequirePlatformOperator denies platform-global mutations (global user
// email/active/delete, workspace create/delete) unless ctx carries the
// control-plane marker. Tenant requests never carry it.
func RequirePlatformOperator(ctx context.Context) error {
	if IsPlatformOperator(ctx) {
		return nil
	}
	return ErrPlatformOperation
}
