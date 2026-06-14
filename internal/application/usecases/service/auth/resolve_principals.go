package auth

import (
	"context"
	"errors"
	"strings"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	authpb "github.com/erniealice/esqyma/pkg/schema/v1/service/auth"
)

// PrincipalResolverAdapter is the narrow extension interface the
// ResolvePrincipals and ResolveBinding use cases consume from the session
// repository. The concrete *PostgresSessionRepository (or the standalone
// PrincipalResolverAdapter in contrib/postgres/internal/adapter/entity/
// principal_resolver.go) satisfies it.
//
// Under mock-db / non-postgres builds the type assertion fails and
// the adapter stays nil — Execute nil-guards and fails closed with
// auth.errors.service_unavailable.
type PrincipalResolverAdapter interface {
	ResolvePrincipals(ctx context.Context, req *authpb.ResolvePrincipalsRequest) (*authpb.ResolvePrincipalsResponse, error)
	EnumerateBindingsInWorkspace(ctx context.Context, req *authpb.EnumerateBindingsRequest) (*authpb.EnumerateBindingsResponse, error)
	LookupSessionPrincipal(ctx context.Context, req *authpb.LookupSessionPrincipalRequest) (*authpb.LookupSessionPrincipalResponse, error)
}

// ResolvePrincipalsRepositories groups the adapters this use case consumes.
type ResolvePrincipalsRepositories struct {
	PrincipalResolver PrincipalResolverAdapter
}

// ResolvePrincipalsServices groups infrastructure services. No Authorizer —
// per the package invariant in usecases.go, principal resolution runs
// pre-auth (login flow).
type ResolvePrincipalsServices struct {
	Translator ports.Translator
}

// ResolvePrincipalsUseCase orchestrates principal resolution for the login flow:
// enumerate ALL bindings for a user across all workspaces and all five grant tables.
//
// No ActionGatekeeper — this runs pre-auth (the user has authenticated but
// has not yet selected a principal/workspace; we need the list of bindings
// to present the chooser or auto-route).
type ResolvePrincipalsUseCase struct {
	repositories ResolvePrincipalsRepositories
	services     ResolvePrincipalsServices
}

// NewResolvePrincipalsUseCase wires the use case from grouped dependencies.
func NewResolvePrincipalsUseCase(
	repositories ResolvePrincipalsRepositories,
	services ResolvePrincipalsServices,
) *ResolvePrincipalsUseCase {
	return &ResolvePrincipalsUseCase{repositories: repositories, services: services}
}

// Execute runs the principal resolution.
func (uc *ResolvePrincipalsUseCase) Execute(
	ctx context.Context,
	req *authpb.ResolvePrincipalsRequest,
) (*authpb.ResolvePrincipalsResponse, error) {
	if uc.repositories.PrincipalResolver == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"auth.errors.service_unavailable",
			"Auth service is not available [DEFAULT]"))
	}
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"auth.validation.request_required",
			"Principal resolution request is required [DEFAULT]"))
	}
	if strings.TrimSpace(req.GetUserId()) == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"auth.validation.user_id_required",
			"User ID is required for principal resolution [DEFAULT]"))
	}

	return uc.repositories.PrincipalResolver.ResolvePrincipals(ctx, req)
}
