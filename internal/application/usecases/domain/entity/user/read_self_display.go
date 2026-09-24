package user

import (
	"context"
	"errors"
	"strings"

	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	userpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/user"
)

// ReadSelfDisplayRepositories groups repository dependencies for the
// caller's own display-name read.
type ReadSelfDisplayRepositories struct {
	User userpb.UserDomainServiceServer
}

// ReadSelfDisplayUseCase returns the CALLING user's own first and last name.
//
// It intentionally skips the ActionGatekeeper (mirroring ResolveUserByEmail):
// the user id comes ONLY from the request identity on ctx — there is no id
// parameter — so it cannot read anyone else, and reading your own display name
// is not a privileged operation (the sidebar already shows it from the session).
// Staff principals lack user:read, so documents they print need this path to
// stamp "Printed by". Missing identity, a missing row, or an inactive user all
// return an error (fail closed; callers fall back to the session identity).
type ReadSelfDisplayUseCase struct {
	repositories ReadSelfDisplayRepositories
}

// NewReadSelfDisplayUseCase creates the use case.
func NewReadSelfDisplayUseCase(repositories ReadSelfDisplayRepositories) *ReadSelfDisplayUseCase {
	return &ReadSelfDisplayUseCase{repositories: repositories}
}

// Execute returns the caller's first and last name.
func (uc *ReadSelfDisplayUseCase) Execute(ctx context.Context) (firstName, lastName string, err error) {
	if uc == nil || uc.repositories.User == nil {
		return "", "", errors.New("self display: user repository unavailable")
	}
	userID := strings.TrimSpace(contextutil.ExtractUserIDFromContext(ctx))
	if userID == "" {
		return "", "", errors.New("self display: no user on the request")
	}
	resp, err := uc.repositories.User.ReadUser(ctx, &userpb.ReadUserRequest{Data: &userpb.User{Id: userID}})
	if err != nil {
		return "", "", err
	}
	for _, u := range resp.GetData() {
		// Re-check the id in code: never trust an adapter to have applied the filter.
		if u.GetId() != userID {
			continue
		}
		if !u.GetActive() {
			return "", "", errors.New("self display: user inactive")
		}
		return strings.TrimSpace(u.GetFirstName()), strings.TrimSpace(u.GetLastName()), nil
	}
	return "", "", errors.New("self display: user not found")
}
