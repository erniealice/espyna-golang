// Package consumer — session_principal.go
//
// SessionPrincipal type definition and helper for session-row principal
// identification. Framework-portable type that carries the binding
// identification from a session row without coupling to the composition
// layer.
//
// Moved from apps/service-admin/internal/composition/ 2026-06-14.
package consumer

import (
	pyezarender "github.com/erniealice/pyeza-golang/render"
)

// SessionPrincipal carries the full binding identification for a session
// row: which kind of binding the user is acting as, which grant row that
// binding identifies, and (for delegate kinds) which per-target row they
// are currently acting through.
//
// Returned by the LookupSessionPrincipal use case (the typed-stack
// replacement for the old lookupSessionPrincipalFull raw-SQL helper).
// The acting-as fields are populated only for delegate principals;
// non-delegate kinds leave them empty.
type SessionPrincipal struct {
	Kind               pyezarender.PrincipalType
	PrincipalID        string
	ActingAsClientID   string
	ActingAsSupplierID string
}

// IsZero reports whether the principal lookup returned no session
// information at all -- used by callers to short-circuit.
func (p SessionPrincipal) IsZero() bool {
	return p.Kind == pyezarender.PrincipalTypeUnspecified && p.PrincipalID == ""
}
