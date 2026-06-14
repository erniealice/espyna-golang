package workspace

import (
	"context"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	workspacepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace"
)

// ResolveWorkspaceBySlugRepositories groups repository dependencies for the
// slug resolution use case.
type ResolveWorkspaceBySlugRepositories struct {
	Workspace workspacepb.WorkspaceDomainServiceServer
}

// ResolveWorkspaceBySlugServices groups service dependencies for the slug
// resolution use case.
type ResolveWorkspaceBySlugServices struct {
	Translator ports.Translator
}

// ResolveWorkspaceBySlugUseCase resolves a workspace slug to a workspace ID.
//
// This is a pre-authentication use case: the workspace_path middleware calls
// it BEFORE the session's workspace context is established. Therefore it
// intentionally skips the ActionGatekeeper authcheck — the caller is not yet
// authenticated in the target workspace, and the slug-to-id mapping is not
// a privileged operation (the middleware gates access via BindingResolver
// after resolution succeeds).
//
// The query filters on slug (exact match) + active = true to mirror the raw
// SQL it replaces:
//
//	SELECT id FROM workspace WHERE slug = $1 AND active = true LIMIT 1
//
// Returns ("", nil) when no active workspace matches the slug (cache miss).
// Returns ("", error) only on infrastructure failure.
type ResolveWorkspaceBySlugUseCase struct {
	repositories ResolveWorkspaceBySlugRepositories
	services     ResolveWorkspaceBySlugServices
}

// NewResolveWorkspaceBySlugUseCase constructs the slug resolution use case.
func NewResolveWorkspaceBySlugUseCase(
	repositories ResolveWorkspaceBySlugRepositories,
	services ResolveWorkspaceBySlugServices,
) *ResolveWorkspaceBySlugUseCase {
	if services.Translator == nil {
		services.Translator = ports.NewNoOpTranslator()
	}
	return &ResolveWorkspaceBySlugUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute resolves a workspace slug to a workspace ID. Returns ("", nil)
// when no active workspace has the given slug.
func (uc *ResolveWorkspaceBySlugUseCase) Execute(ctx context.Context, slug string) (string, error) {
	if slug == "" {
		return "", nil
	}

	// No authcheck — this is a pre-auth middleware concern (see type doc).

	// Build a ListWorkspaces request with slug + active filters.
	activeTrue := true
	resp, err := uc.repositories.Workspace.ListWorkspaces(ctx, &workspacepb.ListWorkspacesRequest{
		Filters: &commonpb.FilterRequest{
			Filters: []*commonpb.TypedFilter{
				{
					Field: "slug",
					FilterType: &commonpb.TypedFilter_StringFilter{
						StringFilter: &commonpb.StringFilter{
							Value:    slug,
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
			"workspace.errors.slug_resolve_failed",
			"Failed to resolve workspace slug [DEFAULT]",
		)
		return "", fmt.Errorf("%s: %w", translatedError, err)
	}

	data := resp.GetData()
	if len(data) == 0 {
		return "", nil
	}

	return data[0].GetId(), nil
}
