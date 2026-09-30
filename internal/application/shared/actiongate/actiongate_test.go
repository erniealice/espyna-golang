package actiongate

import (
	"context"
	"errors"
	"testing"

	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
)

type denyAuthorizer struct{}

func (denyAuthorizer) IsEnabled() bool { return true }
func (denyAuthorizer) HasPermission(context.Context, string, string) (bool, error) {
	return false, nil
}

// strictDenyAuthorizer also exposes the deny-capable strict path used by CheckStrict.
type strictDenyAuthorizer struct{ denyAuthorizer }

func (strictDenyAuthorizer) HasPermissionStrict(context.Context, string, string) (bool, error) {
	return false, nil
}

// R4 m11: a use-case denial (shadow Check and CheckStrict alike) carries the stable code
// "permission_denied" so a view renders PermissionDenied instead of the generic error.
func TestDenialsCarryPermissionDeniedCode(t *testing.T) {
	ctx := contextutil.WithUserID(context.Background(), "u1")
	req := &CheckActionRequest{Entity: entityid.ChargePolicy, Action: entityid.ActionRead}
	cases := map[string]error{
		"Check":       NewActionGatekeeper(denyAuthorizer{}, nil).Check(ctx, req),
		"CheckStrict": NewActionGatekeeper(strictDenyAuthorizer{}, nil).CheckStrict(ctx, req),
	}
	for name, err := range cases {
		var ce interface{ ErrorCode() string }
		if err == nil || !errors.As(err, &ce) || ce.ErrorCode() != "permission_denied" {
			t.Errorf("%s: want coded permission_denied, got %v", name, err)
		}
		if err != nil && err.Error() != "Permission denied" {
			t.Errorf("%s: message must be unchanged, got %q", name, err.Error())
		}
	}
}
