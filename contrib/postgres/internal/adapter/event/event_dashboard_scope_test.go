//go:build postgresql

package event

import (
	"context"
	"testing"

	"github.com/erniealice/espyna-golang/shared/identity"
)

// U-04 (plan 20260927-tenant-boundary-hardening Wave 0): dashboard queries
// must fail closed instead of widening to every tenant.
func TestDashboardWorkspace_FailsClosed(t *testing.T) {
	inA := identity.WithRequestIdentity(context.Background(), &identity.RequestIdentity{UserID: "u", WorkspaceID: "ws-a"})
	cases := []struct {
		name      string
		ctx       context.Context
		requested string
		ok        bool
	}{
		{"own workspace", inA, "ws-a", true},
		{"empty workspace", inA, "", false},
		{"other workspace", inA, "ws-b", false},
		{"no workspace selected", identity.WithRequestIdentity(context.Background(), &identity.RequestIdentity{UserID: "u"}), "", false},
		{"no identity", context.Background(), "ws-a", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := dashboardWorkspace(tc.ctx, tc.requested)
			if (err == nil) != tc.ok {
				t.Fatalf("dashboardWorkspace(%q) = %q, %v; want ok=%v", tc.requested, got, err, tc.ok)
			}
		})
	}
}
