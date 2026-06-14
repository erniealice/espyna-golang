package workspace

import (
	"context"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	workspacepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace"
)

// ResolveSlugByWorkspaceIDRepositories groups repository dependencies for the
// ID-to-slug resolution use case.
type ResolveSlugByWorkspaceIDRepositories struct {
	Workspace workspacepb.WorkspaceDomainServiceServer
}

// ResolveSlugByWorkspaceIDServices groups service dependencies for the
// ID-to-slug resolution use case.
type ResolveSlugByWorkspaceIDServices struct {
	Translator ports.Translator
}

// ResolveSlugByWorkspaceIDUseCase resolves a workspace ID to its slug.
//
// This is a pre-authentication use case: it is called from auth_bridge.go
// (homeURLForWorkspaceID, WorkspaceSlugResolver) BEFORE a workspace context
// is established. Therefore it intentionally skips the ActionGatekeeper
// authcheck — mirroring ResolveWorkspaceBySlugUseCase.
//
// The query filters on id (exact match) + active = true to mirror the raw
// SQL it replaces:
//
//	SELECT slug FROM workspace WHERE id = $1 AND active = true LIMIT 1
//
// Returns ("", nil) when no active workspace matches the ID (cache miss).
// Returns ("", error) only on infrastructure failure.
type ResolveSlugByWorkspaceIDUseCase struct {
	repositories ResolveSlugByWorkspaceIDRepositories
	services     ResolveSlugByWorkspaceIDServices
}

// NewResolveSlugByWorkspaceIDUseCase constructs the ID-to-slug resolution use case.
func NewResolveSlugByWorkspaceIDUseCase(
	repositories ResolveSlugByWorkspaceIDRepositories,
	services ResolveSlugByWorkspaceIDServices,
) *ResolveSlugByWorkspaceIDUseCase {
	if services.Translator == nil {
		services.Translator = ports.NewNoOpTranslator()
	}
	return &ResolveSlugByWorkspaceIDUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute resolves a workspace ID to its slug. Returns ("", nil)
// when no active workspace has the given ID.
func (uc *ResolveSlugByWorkspaceIDUseCase) Execute(ctx context.Context, workspaceID string) (string, error) {
	if workspaceID == "" {
		return "", nil
	}

	// No authcheck — this is a pre-auth concern (see type doc).

	// Build a ListWorkspaces request with id + active filters.
	activeTrue := true
	resp, err := uc.repositories.Workspace.ListWorkspaces(ctx, &workspacepb.ListWorkspacesRequest{
		Filters: &commonpb.FilterRequest{
			Filters: []*commonpb.TypedFilter{
				{
					Field: "id",
					FilterType: &commonpb.TypedFilter_StringFilter{
						StringFilter: &commonpb.StringFilter{
							Value:    workspaceID,
							Operator: commonpb.StringOperator_STRING_EQUALS,
						},
					},
				},
				{
					Field: "active",
					FilterType: &commonpb.TypedFilter_BooleanFilter{
						BooleanFilter: &commonpb.BooleanFilter{
							Value: activeTrue,
						},
					},
				},
			},
		},
		Pagination: &commonpb.PaginationRequest{
			Limit: 1,
		},
	})
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"workspace.errors.id_resolve_failed",
			"Failed to resolve workspace ID [DEFAULT]",
		)
		return "", fmt.Errorf("%s: %w", translatedError, err)
	}

	data := resp.GetData()
	if len(data) == 0 {
		return "", nil
	}

	return data[0].GetSlug(), nil
}
