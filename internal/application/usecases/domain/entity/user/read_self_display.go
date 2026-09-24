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

// ReadSelfDisplayUseCase returns the CALLING user's own display fields
// (first/last name, and — via ExecuteWithEmail — email).
//
// It intentionally skips the ActionGatekeeper (mirroring ResolveUserByEmail):
// the user id comes ONLY from the request identity on ctx — there is no id
// parameter — so it cannot read anyone else, and reading your own display
// name/email is not a privileged operation (the sidebar already shows it from
// the session). Staff principals lack user:read, so documents they print need
// this path to stamp "Printed by", and the sidebar profile chip (pyeza
// sidebar01.html via espyna consumer/http.DBUserLoader) needs it so a
// principal without user:read gets their real name/email instead of the
// generic "Signed In" placeholder. Missing identity, a missing row, or an
// inactive user all return an error (fail closed; callers fall back to the
// session identity).
type ReadSelfDisplayUseCase struct {
	repositories ReadSelfDisplayRepositories
}

// NewReadSelfDisplayUseCase creates the use case.
func NewReadSelfDisplayUseCase(repositories ReadSelfDisplayRepositories) *ReadSelfDisplayUseCase {
	return &ReadSelfDisplayUseCase{repositories: repositories}
}

// resolveSelf reads and validates the caller's own user row: the id comes
// only from ctx, the returned row's id is re-checked against it (never trust
// an adapter to have applied the filter), and an inactive/missing row fails
// closed. Shared by Execute and ExecuteWithEmail so both stay consistent.
func (uc *ReadSelfDisplayUseCase) resolveSelf(ctx context.Context) (*userpb.User, error) {
	if uc == nil || uc.repositories.User == nil {
		return nil, errors.New("self display: user repository unavailable")
	}
	userID := strings.TrimSpace(contextutil.ExtractUserIDFromContext(ctx))
	if userID == "" {
		return nil, errors.New("self display: no user on the request")
	}
	resp, err := uc.repositories.User.ReadUser(ctx, &userpb.ReadUserRequest{Data: &userpb.User{Id: userID}})
	if err != nil {
		return nil, err
	}
	for _, u := range resp.GetData() {
		// Re-check the id in code: never trust an adapter to have applied the filter.
		if u.GetId() != userID {
			continue
		}
		if !u.GetActive() {
			return nil, errors.New("self display: user inactive")
		}
		return u, nil
	}
	return nil, errors.New("self display: user not found")
}

// Execute returns the caller's first and last name.
func (uc *ReadSelfDisplayUseCase) Execute(ctx context.Context) (firstName, lastName string, err error) {
	u, err := uc.resolveSelf(ctx)
	if err != nil {
		return "", "", err
	}
	return strings.TrimSpace(u.GetFirstName()), strings.TrimSpace(u.GetLastName()), nil
}

// ExecuteWithEmail returns the caller's first name, last name, and email —
// the same self-only, gate-free semantics as Execute, extended with email for
// callers (the sidebar profile chip) that need a second display line without
// an extra per-request query.
func (uc *ReadSelfDisplayUseCase) ExecuteWithEmail(ctx context.Context) (firstName, lastName, email string, err error) {
	u, err := uc.resolveSelf(ctx)
	if err != nil {
		return "", "", "", err
	}
	return strings.TrimSpace(u.GetFirstName()), strings.TrimSpace(u.GetLastName()), strings.TrimSpace(u.GetEmailAddress()), nil
}
