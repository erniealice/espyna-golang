package user

import (
	"context"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	userpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/user"
)

// ResolveUserByEmailRepositories groups repository dependencies for the
// email-to-user-ID resolution use case.
type ResolveUserByEmailRepositories struct {
	User userpb.UserDomainServiceServer
}

// ResolveUserByEmailServices groups service dependencies for the
// email-to-user-ID resolution use case.
type ResolveUserByEmailServices struct {
	Translator ports.Translator
}

// ResolveUserByEmailUseCase resolves a user email address to a user ID.
//
// This is a pre-authentication use case: it is called from auth_bridge.go
// (UserIDByEmail closure) during the login flow. Therefore it intentionally
// skips the ActionGatekeeper authcheck — mirroring
// ResolveWorkspaceBySlugUseCase.
//
// The query filters on email_address (exact match) + active = true to mirror
// the raw SQL it replaces:
//
//	SELECT id FROM "user" WHERE email_address = $1 AND active = true LIMIT 1
//
// Returns ("", nil) when no active user matches the email (cache miss).
// Returns ("", error) only on infrastructure failure.
type ResolveUserByEmailUseCase struct {
	repositories ResolveUserByEmailRepositories
	services     ResolveUserByEmailServices
}

// NewResolveUserByEmailUseCase constructs the email-to-user-ID resolution use case.
func NewResolveUserByEmailUseCase(
	repositories ResolveUserByEmailRepositories,
	services ResolveUserByEmailServices,
) *ResolveUserByEmailUseCase {
	if services.Translator == nil {
		services.Translator = ports.NewNoOpTranslator()
	}
	return &ResolveUserByEmailUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute resolves a user email address to a user ID. Returns ("", nil)
// when no active user has the given email.
func (uc *ResolveUserByEmailUseCase) Execute(ctx context.Context, email string) (string, error) {
	if email == "" {
		return "", nil
	}

	// No authcheck — this is a pre-auth concern (see type doc).

	// Build a ListUsers request with email_address + active filters.
	activeTrue := true
	resp, err := uc.repositories.User.ListUsers(ctx, &userpb.ListUsersRequest{
		Filters: &commonpb.FilterRequest{
			Filters: []*commonpb.TypedFilter{
				{
					Field: "email_address",
					FilterType: &commonpb.TypedFilter_StringFilter{
						StringFilter: &commonpb.StringFilter{
							Value:    email,
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
			"user.errors.email_resolve_failed",
			"Failed to resolve user by email [DEFAULT]",
		)
		return "", fmt.Errorf("%s: %w", translatedError, err)
	}

	data := resp.GetData()
	if len(data) == 0 {
		return "", nil
	}

	return data[0].GetId(), nil
}
