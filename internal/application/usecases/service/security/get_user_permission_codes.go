// Package security hosts the service-driven Security use cases.
//
// Per docs/plan/20260518-hexagonal-strict-adherence/proto-service.md (Q7
// service-driven domain category) and the follow-up migration plan
// docs/plan/20260520-service-domain-migration/, PermissionQuery is a
// cross-cutting RBAC concern: it reads across permission + role +
// workspace_user_role with DENY-wins semantics, and owns no aggregate of
// its own. That makes it a service-driven domain, not entity-driven.
//
// Its proto contract lives at `proto/v1/service/security/permission_query.proto`.
// This use case is the read surface (GetUserPermissionCodes) consumed by the
// service-admin HTTP permission loader.
package security

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	securityports "github.com/erniealice/espyna-golang/internal/application/ports/security"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	securitypb "github.com/erniealice/esqyma/pkg/schema/v1/service/security"
)

// GetUserPermissionCodesRepositories groups the infrastructure dependencies
// of the use case. PermissionQuery is the narrow RBAC port — there are no
// proto domain repositories because PermissionQuery is not entity-driven.
type GetUserPermissionCodesRepositories struct {
	PermissionQuery securityports.PermissionQuery
}

// GetUserPermissionCodesServices groups application services.
// Translator is used for error messages; no Authorizer
// because permission-query callers ARE the authorization layer — gating
// the lookup itself on RBAC would be circular.
type GetUserPermissionCodesServices struct {
	Translator ports.Translator
}

// GetUserPermissionCodesUseCase resolves the effective ALLOW permission
// codes for a user/workspace pair, honoring DENY-wins semantics from the
// underlying PermissionQuery.
type GetUserPermissionCodesUseCase struct {
	repositories GetUserPermissionCodesRepositories
	services     GetUserPermissionCodesServices
}

// NewGetUserPermissionCodesUseCase wires the use case from grouped
// dependencies.
func NewGetUserPermissionCodesUseCase(
	repositories GetUserPermissionCodesRepositories,
	services GetUserPermissionCodesServices,
) *GetUserPermissionCodesUseCase {
	return &GetUserPermissionCodesUseCase{repositories: repositories, services: services}
}

// Execute performs the RBAC permission-code lookup.
//
// Translation flow: the proto-shaped request becomes the port's two-string
// signature; the port's []string response is wrapped in the proto-shaped
// response. Returns an empty (non-nil) Response with empty PermissionCodes
// when the underlying port is unregistered, so HTTP callers degrade
// gracefully (the permission loader treats an empty slice as "no codes
// loaded — disable sidebar filtering").
//
// Note: no authcheck.Check call. PermissionQuery IS the building block the
// authorization layer is built on — gating the lookup itself on a
// permission check would be circular. Callers (service-admin HTTP
// permission loader) invoke this during session bootstrap, before any
// per-action authorization is even possible.
func (uc *GetUserPermissionCodesUseCase) Execute(
	ctx context.Context,
	req *securitypb.GetUserPermissionCodesRequest,
) (*securitypb.GetUserPermissionCodesResponse, error) {
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"security.validation.request_required", "request is required"))
	}
	if uc.repositories.PermissionQuery == nil {
		// No RBAC provider — return empty response so the permission
		// loader treats this as "no codes" rather than failing.
		return &securitypb.GetUserPermissionCodesResponse{PermissionCodes: []string{}}, nil
	}

	// Plumb the proto-shaped binding hint (added 2026-05-24 per A2 /
	// WKR-P0-2) into the port. The port already documents the
	// "zero values = legacy union" fall-back, so passing through the
	// generated zero values when the caller didn't set them preserves
	// backwards compatibility transparently.
	//
	// Delegate target scoping (A2-followup, codex A2-P0-1 fix):
	// acting_as_client_id / acting_as_supplier_id are required for
	// CLIENT_DELEGATE / SUPPLIER_DELEGATE bindings respectively; missing
	// values for those kinds cause the port to fail closed (empty
	// result). For non-delegate kinds the acting-as values are ignored.
	codes, err := uc.repositories.PermissionQuery.GetUserPermissionCodes(
		ctx,
		req.GetUserId(),
		req.GetWorkspaceId(),
		int32(req.GetBindingKind()),
		req.GetBindingId(),
		req.GetActingAsClientId(),
		req.GetActingAsSupplierId(),
	)
	if err != nil {
		return nil, fmt.Errorf(
			contextutil.GetTranslatedMessageWithContext(
				ctx, uc.services.Translator,
				"security.errors.permission_lookup_failed",
				"failed to get user permission codes: %w"),
			err,
		)
	}
	if codes == nil {
		codes = []string{}
	}
	return &securitypb.GetUserPermissionCodesResponse{PermissionCodes: codes}, nil
}
