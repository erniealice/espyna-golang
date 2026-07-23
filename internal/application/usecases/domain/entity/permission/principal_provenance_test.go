package permission

import (
	"context"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	permissionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/permission"
)

// TestValidatePrincipalProvenance is the CF-5 defense-in-depth guard: when a
// session principal is present in ctx, a body-supplied UserId/GrantedByUserId
// that disagrees with it is rejected; without a principal (service/seed/test)
// the guard is a no-op regardless of the body ids.
func TestValidatePrincipalProvenance(t *testing.T) {
	tr := ports.NewNoOpTranslator()
	withPrincipal := func(uid string) context.Context {
		return contextutil.WithUserID(context.Background(), uid)
	}

	cases := []struct {
		name    string
		ctx     context.Context
		data    *permissionpb.Permission
		wantErr bool
	}{
		{
			name:    "no principal in ctx is a no-op even on mismatch",
			ctx:     context.Background(),
			data:    &permissionpb.Permission{UserId: "someone-else", GrantedByUserId: "another"},
			wantErr: false,
		},
		{
			name:    "matching provenance passes",
			ctx:     withPrincipal("admin-1"),
			data:    &permissionpb.Permission{UserId: "admin-1", GrantedByUserId: "admin-1"},
			wantErr: false,
		},
		{
			name:    "mismatched UserId is rejected",
			ctx:     withPrincipal("admin-1"),
			data:    &permissionpb.Permission{UserId: "victim-2", GrantedByUserId: "admin-1"},
			wantErr: true,
		},
		{
			name:    "mismatched GrantedByUserId is rejected",
			ctx:     withPrincipal("admin-1"),
			data:    &permissionpb.Permission{UserId: "admin-1", GrantedByUserId: "spoofed-3"},
			wantErr: true,
		},
		{
			name:    "empty body ids are not compared (validateInput owns non-empty)",
			ctx:     withPrincipal("admin-1"),
			data:    &permissionpb.Permission{},
			wantErr: false,
		},
		{
			name:    "nil data is a no-op",
			ctx:     withPrincipal("admin-1"),
			data:    nil,
			wantErr: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validatePrincipalProvenance(tc.ctx, tr, tc.data)
			if tc.wantErr && err == nil {
				t.Fatalf("expected rejection, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
		})
	}
}
