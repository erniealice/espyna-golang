package workspace

import (
	"context"
	"errors"
	"regexp"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
)

// slugFormatRegex enforces the canonical workspace-slug format: lowercase
// alphanumeric segments separated by single hyphens, no leading/trailing
// hyphen, no double hyphens. Mirrors the workspace_slug_format CHECK
// constraint added in migration 20260522000000_workspace_slug.sql.
var slugFormatRegex = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

const (
	slugMinLen = 3
	slugMaxLen = 30
)

// ReservedSlugProvider is the port through which the validation use case
// receives the application-policy list of reserved workspace slugs. The
// reserved list lives in the composition layer (service-admin) because the
// reservations are URL-routing decisions, not domain invariants. This port
// keeps espyna decoupled from those routing decisions.
//
// Implementations must be safe for concurrent use.
type ReservedSlugProvider interface {
	// IsReserved returns true when slug (already case-folded by the caller)
	// collides with a top-level URL path or other reservation that prevents
	// it from being used as a workspace identifier.
	IsReserved(slug string) bool
}

// ValidateSlugRequest is the input to the slug validation use case.
type ValidateSlugRequest struct {
	Slug string
}

// ValidateSlugResponse reports validation success. On failure, Execute returns
// a translated error instead — there is no per-field error map today.
type ValidateSlugResponse struct {
	Success bool
}

// ValidateSlugServices groups the use case's service dependencies.
type ValidateSlugServices struct {
	Translator    ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	ReservedSlugs ReservedSlugProvider
}

// ValidateSlugUseCase enforces the four workspace-slug rules:
//  1. Slug is non-empty.
//  2. Length is between slugMinLen and slugMaxLen.
//  3. Format matches ^[a-z0-9]+(?:-[a-z0-9]+)*$.
//  4. Slug is not in the application-level reserved list.
//
// The use case does NOT check DB-level uniqueness — that constraint is
// enforced by the workspace_slug_unique index in the database; callers
// should treat a unique-violation error from the workspace repository as
// the user-visible "slug already taken" failure.
//
// Added 2026-05-22 per Phase P-1 of docs/plan/20260521-workspace-keyed-routing.
type ValidateSlugUseCase struct {
	services ValidateSlugServices
}

// NewValidateSlugUseCase constructs a ValidateSlug use case. A nil
// ReservedSlugs provider is permitted (e.g. for tests that only want the
// format checks); when nil, the reserved-word rule is skipped.
func NewValidateSlugUseCase(services ValidateSlugServices) *ValidateSlugUseCase {
	if services.Translator == nil {
		services.Translator = ports.NewNoOpTranslator()
	}
	return &ValidateSlugUseCase{services: services}
}

// Execute runs the four checks in order and returns the first failure as a
// translated error. The signature follows the espyna 5-step Execute pattern
// (input → business rules → enrichment → transaction → core) — this is a
// pure-validation use case so the enrichment and transaction steps are
// no-ops.
func (uc *ValidateSlugUseCase) Execute(ctx context.Context, req *ValidateSlugRequest) (*ValidateSlugResponse, error) {
	// Step 1: input validation
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "workspace.validation.request_required", "Request is required for workspaces [DEFAULT]"))
	}
	if req.Slug == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "workspace.validation.slug_required", "Workspace slug is required [DEFAULT]"))
	}

	// Step 2: business rule — length
	if len(req.Slug) < slugMinLen {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "workspace.validation.slug_too_short", "Workspace slug must be at least 3 characters long [DEFAULT]"))
	}
	if len(req.Slug) > slugMaxLen {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "workspace.validation.slug_too_long", "Workspace slug cannot exceed 30 characters [DEFAULT]"))
	}

	// Step 3: business rule — format
	if !slugFormatRegex.MatchString(req.Slug) {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "workspace.validation.slug_format_invalid", "Workspace slug must contain only lowercase letters, numbers and hyphens [DEFAULT]"))
	}

	// Step 4: business rule — reserved-word list (skipped when no provider wired)
	if uc.services.ReservedSlugs != nil && uc.services.ReservedSlugs.IsReserved(req.Slug) {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "workspace.validation.slug_reserved", "Workspace slug collides with a reserved URL path [DEFAULT]"))
	}

	return &ValidateSlugResponse{Success: true}, nil
}
