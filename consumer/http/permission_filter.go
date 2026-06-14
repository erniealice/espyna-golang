package http

// permission_filter.go — fail-CLOSED permission filtering for the mobile app grid.
//
// Relocated verbatim from the service-admin app (internal/infrastructure/input/
// http/permission_filter.go) into espyna consumer/http in Model-A Wave 3. The
// fail-closed invariant (nil perms denies every gated item) is preserved
// byte-for-behaviour — see the inline notes below.

import "github.com/erniealice/pyeza-golang/types"

// FilterAppGroupsByPermissions removes apps the user doesn't have permission to access.
// Items with an empty Permission field are always included.
// Groups with no remaining items after filtering are removed entirely.
func FilterAppGroupsByPermissions(groups []types.AppGridGroup, perms *types.UserPermissions) []types.AppGridGroup {
	// Fail CLOSED (Q-SEC-1 / permission-filter-fail-open): no nil-perms
	// pass-through. The prior `if perms == nil { return groups }` exposed the
	// FULL grid when no permission set was established. A nil *UserPermissions now
	// denies every GATED item — types.UserPermissions.HasCode is nil-receiver-safe
	// (returns false) — while ungated items (Permission == "") still pass, matching
	// the fail-closed invariant in view_adapter.go. The live caller
	// (injectUserPermissions) never reaches here with nil; this is defense-in-depth.
	var filtered []types.AppGridGroup
	for _, group := range groups {
		var items []types.AppGridItem
		for _, item := range group.Items {
			if item.Permission == "" || perms.HasCode(item.Permission) {
				items = append(items, item)
			}
		}
		if len(items) > 0 {
			filtered = append(filtered, types.AppGridGroup{
				Title: group.Title,
				Items: items,
			})
		}
	}
	return filtered
}

// FilterAppsByPermissions filters the flat AllApps list by permissions.
// Items with an empty Permission field are always included.
func FilterAppsByPermissions(apps []types.AppGridItem, perms *types.UserPermissions) []types.AppGridItem {
	// Fail CLOSED (Q-SEC-1 / permission-filter-fail-open): nil perms denies every
	// gated app (HasCode is nil-safe → false) while ungated items still pass. See
	// the companion note in FilterAppGroupsByPermissions.
	var filtered []types.AppGridItem
	for _, app := range apps {
		if app.Permission == "" || perms.HasCode(app.Permission) {
			filtered = append(filtered, app)
		}
	}
	return filtered
}
