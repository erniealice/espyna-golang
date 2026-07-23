package role_permission

import (
	"context"

	securityports "github.com/erniealice/espyna-golang/internal/application/ports/security"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	clientportalgrantpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client_portal_grant"
	delegateclientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/delegate_client"
	delegatesupplierpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/delegate_supplier"
	supplierportalgrantpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/supplier_portal_grant"
	workspaceuserpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace_user"
	workspaceuserrolepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace_user_role"
)

// roleBindingInvalidationRepos carries every grant-table read seam a
// role_permission grant/revoke must enumerate to reach EVERY binding kind the
// permission loader caches — not just workspace_user_role (CF-1).
//
// The permission loader keys its per-identity cache on
// (userID, workspaceID, bindingKind, bindingID, actingAsClientID,
// actingAsSupplierID) (consumer/http/loaders.go permissionCacheKey), and each of
// the SIX binding kinds resolves its role→permission grant chain differently
// (contrib/postgres rbac/permission_query.go), so the cache-key bindingID means
// a DIFFERENT id per kind:
//
//	kind 1/2 operator-owner/staff  bindingID = workspace_user.id
//	kind 7   staff                 bindingID = staff.id           (roles via workspace_user_role)
//	kind 3   client (portal)       bindingID = client_portal_grant.id     (cpg.role_id)
//	kind 5   supplier (portal)     bindingID = supplier_portal_grant.id   (spg.role_id)
//	kind 4   client delegate       bindingID = delegate.id                (delegate_client.role_id)
//	kind 6   supplier delegate     bindingID = delegate.id                (delegate_supplier.role_id)
//
// The original invalidator only enumerated workspace_user_role and evicted by
// InvalidateBinding(workspace_user_id) — which matches ONLY kinds 1/2. A revoke
// on a role held through a portal, delegate, or staff binding therefore lingered
// in that principal's cache until the loader TTL expired (CF-1: bounded
// over-authorization on the revoke direction). This struct lets the invalidator
// reach the other four representations.
//
// Every field is nil-safe: an absent repo simply skips its chain. Invalidation
// is best-effort — the grant/revoke itself already committed, and a missed
// eviction reverts only to the pre-existing loader-TTL staleness bound.
type roleBindingInvalidationRepos struct {
	WorkspaceUserRole   workspaceuserrolepb.WorkspaceUserRoleDomainServiceServer
	WorkspaceUser       workspaceuserpb.WorkspaceUserDomainServiceServer
	ClientPortalGrant   clientportalgrantpb.ClientPortalGrantDomainServiceServer
	SupplierPortalGrant supplierportalgrantpb.SupplierPortalGrantDomainServiceServer
	DelegateClient      delegateclientpb.DelegateClientDomainServiceServer
	DelegateSupplier    delegatesupplierpb.DelegateSupplierDomainServiceServer
}

// invalidateRoleBindings evicts the cached RBAC permission codes of EVERY
// binding kind currently holding the given role, so a role→permission grant or
// revoke is visible on those principals' very next request (P10/D2 — grants
// apply immediately, no TTL wait, no restart) — closing the CF-1 gap where the
// original workspace_user_role-only sweep left portal/delegate/staff principals
// stale until TTL.
//
// Precise (P10/D2 audit-gate: no over-clear): each eviction targets exactly one
// binding id or exactly one login user — never a whole-cache flush and never an
// unrelated principal. Enumeration is scoped to the ACTIVE grant rows for the
// role (the role assignment itself is unchanged by a role_permission mutation),
// so the affected set is the small set of current role-holders. Nil-safe and
// best-effort throughout: a nil invalidator, an empty role id, an unwired repo,
// or a list error is a silent no-op.
func invalidateRoleBindings(
	ctx context.Context,
	repos roleBindingInvalidationRepos,
	inv securityports.PermissionCacheInvalidator,
	roleID string,
) {
	if inv == nil || roleID == "" {
		return
	}
	invalidateOperatorAndStaffBindings(ctx, repos.WorkspaceUserRole, repos.WorkspaceUser, inv, roleID)
	invalidateClientPortalBindings(ctx, repos.ClientPortalGrant, inv, roleID)
	invalidateSupplierPortalBindings(ctx, repos.SupplierPortalGrant, inv, roleID)
	invalidateClientDelegateBindings(ctx, repos.DelegateClient, inv, roleID)
	invalidateSupplierDelegateBindings(ctx, repos.DelegateSupplier, inv, roleID)
}

// invalidateOperatorAndStaffBindings covers the workspace_user_role-sourced
// kinds. For each active WUR row on the role it evicts:
//   - kinds 1/2 (operator-owner/staff) precisely via InvalidateBinding on the
//     workspace_user.id the WUR row carries (unchanged from the original sweep,
//     no over-clear); and
//   - kind 7 (staff), whose cache-key bindingID is staff.id — NOT
//     workspace_user.id — so InvalidateBinding(workspace_user_id) alone misses
//     it. That facet is resolved to its login user (workspace_user.user_id) and
//     evicted representation-agnostically via InvalidateUser, which covers every
//     entry that login user holds regardless of how its bindingID is encoded.
//     The staff CTE draws its roles from the SAME workspace_user_role rows as the
//     operator CTE, so the same login user legitimately holds the role under both
//     representations. Resolving the login user is nil-safe: skipped when the
//     workspace_user repo is unwired, leaving the operator eviction intact.
func invalidateOperatorAndStaffBindings(
	ctx context.Context,
	wurRepo workspaceuserrolepb.WorkspaceUserRoleDomainServiceServer,
	wuRepo workspaceuserpb.WorkspaceUserDomainServiceServer,
	inv securityports.PermissionCacheInvalidator,
	roleID string,
) {
	if wurRepo == nil {
		return
	}
	resp, err := wurRepo.ListWorkspaceUserRoles(ctx, &workspaceuserrolepb.ListWorkspaceUserRolesRequest{
		Filters: roleIDFilter(roleID),
	})
	if err != nil || resp == nil {
		return
	}
	for _, row := range resp.GetData() {
		if row == nil || !row.GetActive() {
			continue
		}
		// Defense-in-depth: re-check the role id in memory in case the adapter
		// ignores the server-side filter (never widen past the target role).
		if row.GetRoleId() != roleID {
			continue
		}
		wuID := row.GetWorkspaceUserId()
		if wuID == "" {
			continue
		}
		inv.InvalidateBinding(wuID)
		if uid := resolveWorkspaceUserLoginID(ctx, wuRepo, wuID); uid != "" {
			inv.InvalidateUser(uid)
		}
	}
}

// invalidateClientPortalBindings evicts CLIENT (kind 3) portal principals holding
// the role. The role grant lives on client_portal_grant.role_id and the
// cache-key bindingID for a CLIENT principal is the client_portal_grant.id, so
// InvalidateBinding on that id is the precise (no over-clear) eviction.
func invalidateClientPortalBindings(
	ctx context.Context,
	repo clientportalgrantpb.ClientPortalGrantDomainServiceServer,
	inv securityports.PermissionCacheInvalidator,
	roleID string,
) {
	if repo == nil {
		return
	}
	resp, err := repo.ListClientPortalGrants(ctx, &clientportalgrantpb.ListClientPortalGrantsRequest{
		Filters: roleIDFilter(roleID),
	})
	if err != nil || resp == nil {
		return
	}
	for _, row := range resp.GetData() {
		if row == nil || !row.GetActive() || row.GetRoleId() != roleID {
			continue
		}
		if id := row.GetId(); id != "" {
			inv.InvalidateBinding(id)
		}
	}
}

// invalidateSupplierPortalBindings is the SUPPLIER (kind 5) analogue of
// invalidateClientPortalBindings: bindingID = supplier_portal_grant.id.
func invalidateSupplierPortalBindings(
	ctx context.Context,
	repo supplierportalgrantpb.SupplierPortalGrantDomainServiceServer,
	inv securityports.PermissionCacheInvalidator,
	roleID string,
) {
	if repo == nil {
		return
	}
	resp, err := repo.ListSupplierPortalGrants(ctx, &supplierportalgrantpb.ListSupplierPortalGrantsRequest{
		Filters: roleIDFilter(roleID),
	})
	if err != nil || resp == nil {
		return
	}
	for _, row := range resp.GetData() {
		if row == nil || !row.GetActive() || row.GetRoleId() != roleID {
			continue
		}
		if id := row.GetId(); id != "" {
			inv.InvalidateBinding(id)
		}
	}
}

// invalidateClientDelegateBindings evicts CLIENT_DELEGATE (kind 4) principals.
// The role grant lives on delegate_client.role_id, but the cache-key bindingID
// for a delegate principal is the delegate.id (the acting-as client scopes the
// per-target grant, and enters the key as actingAsClientID). InvalidateBinding
// matches on bindingID ALONE, so a single InvalidateBinding(delegate_id) clears
// every acting-as-target entry that delegate holds — exactly the set affected by
// a change to a role reachable through any of its delegate_client rows.
func invalidateClientDelegateBindings(
	ctx context.Context,
	repo delegateclientpb.DelegateClientDomainServiceServer,
	inv securityports.PermissionCacheInvalidator,
	roleID string,
) {
	if repo == nil {
		return
	}
	resp, err := repo.ListDelegateClients(ctx, &delegateclientpb.ListDelegateClientsRequest{
		Filters: roleIDFilter(roleID),
	})
	if err != nil || resp == nil {
		return
	}
	for _, row := range resp.GetData() {
		if row == nil || !row.GetActive() || row.GetRoleId() != roleID {
			continue
		}
		if did := row.GetDelegateId(); did != "" {
			inv.InvalidateBinding(did)
		}
	}
}

// invalidateSupplierDelegateBindings is the SUPPLIER_DELEGATE (kind 6) analogue:
// role grant on delegate_supplier.role_id, cache-key bindingID = delegate.id.
func invalidateSupplierDelegateBindings(
	ctx context.Context,
	repo delegatesupplierpb.DelegateSupplierDomainServiceServer,
	inv securityports.PermissionCacheInvalidator,
	roleID string,
) {
	if repo == nil {
		return
	}
	resp, err := repo.ListDelegateSuppliers(ctx, &delegatesupplierpb.ListDelegateSuppliersRequest{
		Filters: roleIDFilter(roleID),
	})
	if err != nil || resp == nil {
		return
	}
	for _, row := range resp.GetData() {
		if row == nil || !row.GetActive() || row.GetRoleId() != roleID {
			continue
		}
		if did := row.GetDelegateId(); did != "" {
			inv.InvalidateBinding(did)
		}
	}
}

// roleIDFilter builds the single-field role_id equality filter shared by every
// grant-table enumeration. The List seams default to active=true (they carry no
// explicit "active" filter here), which is exactly what is wanted: a
// role_permission mutation affects the bindings that ACTIVELY hold the role, and
// a soft-deleted grant no longer confers it. Each caller additionally re-checks
// GetActive()/GetRoleId() in memory as defense-in-depth against an adapter that
// ignores the server-side filter.
func roleIDFilter(roleID string) *commonpb.FilterRequest {
	return &commonpb.FilterRequest{
		Filters: []*commonpb.TypedFilter{{
			Field: "role_id",
			FilterType: &commonpb.TypedFilter_StringFilter{
				StringFilter: &commonpb.StringFilter{Value: roleID, Operator: commonpb.StringOperator_STRING_EQUALS},
			},
		}},
	}
}

// resolveWorkspaceUserLoginID reads a workspace_user by id and returns its login
// user_id (or "" when unwired/unreadable/empty). Used to reach the staff (kind 7)
// cache representation, whose bindingID is staff.id rather than the
// workspace_user.id the WUR row carries. Best-effort and nil-safe.
func resolveWorkspaceUserLoginID(
	ctx context.Context,
	repo workspaceuserpb.WorkspaceUserDomainServiceServer,
	wuID string,
) string {
	if repo == nil || wuID == "" {
		return ""
	}
	resp, err := repo.ReadWorkspaceUser(ctx, &workspaceuserpb.ReadWorkspaceUserRequest{
		Data: &workspaceuserpb.WorkspaceUser{Id: wuID},
	})
	if err != nil || resp == nil {
		return ""
	}
	for _, row := range resp.GetData() {
		if row != nil && row.GetUserId() != "" {
			return row.GetUserId()
		}
	}
	return ""
}
