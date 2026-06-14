//go:build postgresql

package entity

import (
	"context"

	authpb "github.com/erniealice/esqyma/pkg/schema/v1/service/auth"
)

// PostgresSessionRepository satisfies the PrincipalResolverAdapter interface
// by delegating to an embedded PrincipalResolverAdapter instance. This allows
// the session repository -- which is already wired through the initializer's
// entityRepos.Session type assertion path -- to expose the principal resolution
// methods without adding a separate registry entry.
//
// The resolver uses the same *sql.DB the session repository already holds.

// ResolvePrincipals delegates to the PrincipalResolverAdapter.
func (r *PostgresSessionRepository) ResolvePrincipals(
	ctx context.Context,
	req *authpb.ResolvePrincipalsRequest,
) (*authpb.ResolvePrincipalsResponse, error) {
	return r.principalResolver().ResolvePrincipals(ctx, req)
}

// EnumerateBindingsInWorkspace delegates to the PrincipalResolverAdapter.
func (r *PostgresSessionRepository) EnumerateBindingsInWorkspace(
	ctx context.Context,
	req *authpb.EnumerateBindingsRequest,
) (*authpb.EnumerateBindingsResponse, error) {
	return r.principalResolver().EnumerateBindingsInWorkspace(ctx, req)
}

// LookupSessionPrincipal delegates to the PrincipalResolverAdapter.
func (r *PostgresSessionRepository) LookupSessionPrincipal(
	ctx context.Context,
	req *authpb.LookupSessionPrincipalRequest,
) (*authpb.LookupSessionPrincipalResponse, error) {
	return r.principalResolver().LookupSessionPrincipal(ctx, req)
}

// principalResolver lazily creates a PrincipalResolverAdapter from the
// session repository's existing db handle.
func (r *PostgresSessionRepository) principalResolver() *PrincipalResolverAdapter {
	return NewPrincipalResolverAdapter(r.db)
}
