package auth

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	authpb "github.com/erniealice/esqyma/pkg/schema/v1/service/auth"
)

// LookupSessionPrincipalRepositories groups the adapters this use case consumes.
type LookupSessionPrincipalRepositories struct {
	PrincipalResolver PrincipalResolverAdapter
}

// LookupSessionPrincipalServices groups infrastructure services.
type LookupSessionPrincipalServices struct {
	Translator ports.Translator
}

// LookupSessionPrincipalUseCase reads the (principal_type, principal_id,
// acting_as_*) from the session row by token. Replaces the raw-SQL
// lookupSessionPrincipalFull in composition/session_principal.go.
//
// No ActionGatekeeper — this runs in middleware (session resolution),
// before any per-action authorization.
type LookupSessionPrincipalUseCase struct {
	repositories LookupSessionPrincipalRepositories
	services     LookupSessionPrincipalServices
}

// NewLookupSessionPrincipalUseCase wires the use case from grouped dependencies.
func NewLookupSessionPrincipalUseCase(
	repositories LookupSessionPrincipalRepositories,
	services LookupSessionPrincipalServices,
) *LookupSessionPrincipalUseCase {
	return &LookupSessionPrincipalUseCase{repositories: repositories, services: services}
}

// Execute reads the session principal information for the given token.
// Returns a zero response (UNSPECIFIED kind) on miss/error — callers
// treat that as the "no hint" sentinel and fall through to single-binding /
// picker behaviour, which is fail-closed (no auto-elevation).
func (uc *LookupSessionPrincipalUseCase) Execute(
	ctx context.Context,
	req *authpb.LookupSessionPrincipalRequest,
) (*authpb.LookupSessionPrincipalResponse, error) {
	if uc.repositories.PrincipalResolver == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"auth.errors.service_unavailable",
			"Auth service is not available [DEFAULT]"))
	}
	if req == nil || req.GetToken() == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"auth.validation.request_required",
			"Session principal lookup request is required [DEFAULT]"))
	}

	return uc.repositories.PrincipalResolver.LookupSessionPrincipal(ctx, req)
}
