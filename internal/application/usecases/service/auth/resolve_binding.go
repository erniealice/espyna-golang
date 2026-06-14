package auth

import (
	"context"
	"errors"
	"strings"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	principaltypepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/principal_type"
	authpb "github.com/erniealice/esqyma/pkg/schema/v1/service/auth"
)

// Sentinel errors for binding resolution. Exported from the use case
// package so consumers can match on them.
var (
	// ErrNoBinding is returned when the user has no active binding in the
	// requested workspace.
	ErrNoBinding = errors.New("resolve_binding: no active binding in workspace")

	// ErrAmbiguousBinding is returned when the user holds multiple active
	// bindings in the workspace and the session principal hint does not
	// uniquely identify one.
	ErrAmbiguousBinding = errors.New("resolve_binding: ambiguous binding (multiple bindings, no session principal match)")
)

// ResolveBindingRepositories groups the adapters this use case consumes.
type ResolveBindingRepositories struct {
	PrincipalResolver PrincipalResolverAdapter
}

// ResolveBindingServices groups infrastructure services.
type ResolveBindingServices struct {
	Translator ports.Translator
}

// ResolveBindingUseCase orchestrates binding resolution for the workspace_path
// middleware: enumerate bindings in a single workspace, then apply the A3
// resolution policy (PickBindingForSession) to select one.
//
// No ActionGatekeeper — this runs in middleware, before handlers.
type ResolveBindingUseCase struct {
	repositories ResolveBindingRepositories
	services     ResolveBindingServices
}

// NewResolveBindingUseCase wires the use case from grouped dependencies.
func NewResolveBindingUseCase(
	repositories ResolveBindingRepositories,
	services ResolveBindingServices,
) *ResolveBindingUseCase {
	return &ResolveBindingUseCase{repositories: repositories, services: services}
}

// Execute enumerates the user's bindings in the workspace and applies the
// A3 resolution policy. Returns the matched binding, or ErrNoBinding /
// ErrAmbiguousBinding.
//
// Parameters are flat (not a custom Go struct) so the composition layer can
// call this method using only proto-typed values without importing the
// internal use case package.
func (uc *ResolveBindingUseCase) Execute(
	ctx context.Context,
	userID, workspaceID string,
	sessionKind principaltypepb.PrincipalType,
	sessionPrincipalID string,
) (*authpb.Principal, error) {
	if uc.repositories.PrincipalResolver == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"auth.errors.service_unavailable",
			"Auth service is not available [DEFAULT]"))
	}
	userID = strings.TrimSpace(userID)
	workspaceID = strings.TrimSpace(workspaceID)
	sessionPrincipalID = strings.TrimSpace(sessionPrincipalID)
	if userID == "" || workspaceID == "" {
		return nil, ErrNoBinding
	}

	resp, err := uc.repositories.PrincipalResolver.EnumerateBindingsInWorkspace(ctx,
		&authpb.EnumerateBindingsRequest{
			UserId:      userID,
			WorkspaceId: workspaceID,
		})
	if err != nil {
		return nil, err
	}

	return PickBindingForSession(resp.GetBindings(), sessionKind, sessionPrincipalID)
}

// PickBindingForSession applies the A3 resolution policy to a list of
// already-enumerated bindings. Pure function — extracted so the decision
// matrix can be exercised by a table-driven unit test without needing a
// live database.
//
// Resolution policy (A3 — security-critical, 2026-05-23):
//
//  1. Exact session-principal match wins (the security-critical "stay in
//     the lane I was in" rule).
//  2. Multi-target delegate hint matches route to chooser (codex RBC#1).
//  3. No session principal hint + exactly one binding (not multi-target
//     delegate) -> auto-pick.
//  4. No session principal hint + multiple bindings -> picker.
//  5. Session hint does not match any binding -> picker.
//  6. Empty bindings -> ErrNoBinding.
//
// Moved from apps/service-admin/internal/infrastructure/input/http/
// principal_loader.go's pickBindingForSession. The logic is IDENTICAL.
func PickBindingForSession(
	bindings []*authpb.Principal,
	sessionPrincipalKind principaltypepb.PrincipalType,
	sessionPrincipalID string,
) (*authpb.Principal, error) {
	if len(bindings) == 0 {
		return nil, ErrNoBinding
	}

	// 2. Exact session-principal match wins.
	if sessionPrincipalKind != principaltypepb.PrincipalType_PRINCIPAL_TYPE_UNSPECIFIED &&
		sessionPrincipalID != "" {
		for _, b := range bindings {
			if b.Type == sessionPrincipalKind && b.PrincipalId == sessionPrincipalID {
				// Multi-target delegate ambiguity check (codex RBC#1).
				if len(b.ActingAsTargets) > 1 {
					return nil, ErrAmbiguousBinding
				}
				return b, nil
			}
		}
		// 5. Session principal hint did not match — bounce to picker.
		return nil, ErrAmbiguousBinding
	}

	// 3. No session principal hint + exactly one binding.
	if len(bindings) == 1 {
		if len(bindings[0].ActingAsTargets) > 1 {
			return nil, ErrAmbiguousBinding
		}
		return bindings[0], nil
	}

	// 4. No session principal hint + multiple bindings → picker.
	return nil, ErrAmbiguousBinding
}
