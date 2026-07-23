package permission

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	permissionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/permission"
)

// validatePrincipalProvenance is a defense-in-depth guard (CF-5) shared by the
// permission create/update use cases: when the request carries a resolved
// session principal in ctx, the permission DEFINITION's provenance ids
// (UserId / GrantedByUserId) supplied on the body MUST match that principal.
//
// The sole wired HTTP handler already FORCES both ids from the signed session
// ctx (entydad permission/action/action.go via ExtractUserIDFromContext), so on
// the wired path UserId == GrantedByUserId == principal and this guard never
// fires. It exists for a future/internal caller that forwards a body-supplied id
// — which would otherwise mis-attribute the definition's provenance AND
// mis-target the post-write cache eviction (which keys on req.Data.UserId,
// see invalidateProvenanceCache). REJECT rather than silently overwrite: the
// mismatch is a caller bug worth surfacing, and rejecting cannot mis-record who
// authored the change.
//
// Precondition — "when a session principal exists in ctx": a context with no
// resolved principal (service-to-service, seed, or test callers) yields
// ExtractUserIDFromContext == "" and the guard is a no-op, leaving those paths
// unaffected. Each id is compared only when non-empty (validateInput already
// enforces non-empty on the wired create/update path).
func validatePrincipalProvenance(ctx context.Context, translator ports.Translator, data *permissionpb.Permission) error {
	if data == nil {
		return nil
	}
	principal := contextutil.ExtractUserIDFromContext(ctx)
	if principal == "" {
		return nil
	}
	if data.UserId != "" && data.UserId != principal {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, translator,
			"permission.validation.provenance_user_mismatch",
			"Permission provenance user does not match the acting principal [DEFAULT]"))
	}
	if data.GrantedByUserId != "" && data.GrantedByUserId != principal {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, translator,
			"permission.validation.provenance_granted_by_mismatch",
			"Permission granted-by user does not match the acting principal [DEFAULT]"))
	}
	return nil
}
