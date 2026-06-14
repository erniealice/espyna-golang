//go:build postgresql

// Package consumer -- principal_switch_sidebar_postgresql.go
//
// Build-tag gate: reports that the secure principal-switch rotation primitive
// is compiled into this binary. Under the `postgresql` build tag the
// contrib/postgres session adapter satisfies the SwitchPrincipal SessionSwitch
// adapter.
//
// Moved from apps/service-admin/internal/composition/.

package consumer

// SecureSidebarPrimitiveAvailable reports that the secure principal-switch
// rotation primitive is compiled into this binary.
func SecureSidebarPrimitiveAvailable() bool { return true }
