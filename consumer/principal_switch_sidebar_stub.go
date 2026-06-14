//go:build !postgresql

// Package consumer -- principal_switch_sidebar_stub.go
//
// Build-tag gate: reports that the secure principal-switch rotation primitive
// is NOT compiled into this binary. On non-postgres backends the session
// repository does not satisfy the SwitchPrincipal SessionSwitch adapter.
//
// Moved from apps/service-admin/internal/composition/.

package consumer

// SecureSidebarPrimitiveAvailable reports that the secure principal-switch
// rotation primitive is NOT compiled into this binary.
func SecureSidebarPrimitiveAvailable() bool { return false }
