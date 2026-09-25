//go:build postgresql

package audit

import (
	"context"
	"strings"
	"testing"

	infraports "github.com/erniealice/espyna-golang/internal/application/ports/infrastructure"
	"github.com/erniealice/espyna-golang/shared/identity"
)

func TestBoundedEntityListLimit(t *testing.T) {
	tests := []struct {
		name string
		in   int
		want int
	}{
		{name: "default zero", in: 0, want: 20},
		{name: "default negative", in: -1, want: 20},
		{name: "requested", in: 42, want: 42},
		{name: "cap", in: 101, want: 100},
		{name: "cap maximum int", in: int(^uint(0) >> 1), want: 100},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := boundedEntityListLimit(tt.in); got != tt.want {
				t.Fatalf("boundedEntityListLimit(%d) = %d, want %d", tt.in, got, tt.want)
			}
		})
	}
}

func TestValidateAuditCursorTokenLengthBoundary(t *testing.T) {
	if err := validateAuditCursorTokenLength(strings.Repeat("a", maxAuditCursorTokenBytes)); err != nil {
		t.Fatalf("boundary token rejected: %v", err)
	}
	if err := validateAuditCursorTokenLength(strings.Repeat("a", maxAuditCursorTokenBytes+1)); err == nil {
		t.Fatal("oversized token accepted")
	}
}

func TestListByEntityRejectsOversizedCursorBeforeDecodeOrSQL(t *testing.T) {
	ctx := identity.WithRequestIdentity(context.Background(), &identity.RequestIdentity{WorkspaceID: "ws-1"})
	adapter := &auditAdapter{} // nil DB deliberately proves the request never reaches SQL.
	_, err := adapter.ListByEntity(ctx, &infraports.ListAuditRequest{
		WorkspaceID: "ws-1",
		CursorToken: strings.Repeat("!", maxAuditCursorTokenBytes+1),
	})
	if err == nil || !strings.Contains(err.Error(), "cursor token exceeds") {
		t.Fatalf("ListByEntity() error = %v, want cursor size rejection", err)
	}
}

// fix3-backend (codex-review-impl3 #4): audit_entry.workspace_id is stamped
// from the trusted request identity, never from request data.
func TestTrustedAuditWorkspaceID(t *testing.T) {
	const trusted = "019ecb8e-d83f-74ab-aa13-5a6c27afd112"
	const other = "019ecb8e-d83f-74ab-aa13-000000000000"
	withWS := func(ws string) context.Context {
		return identity.WithRequestIdentity(context.Background(), &identity.RequestIdentity{UserID: "u", WorkspaceID: ws})
	}
	tests := []struct {
		name      string
		ctx       context.Context
		requested string
		want      string
	}{
		{"generic diff event (no request ws) gets trusted ws", withWS(trusted), "", trusted},
		{"request data cannot redirect to another tenant", withWS(trusted), other, trusted},
		{"matching request ws unchanged", withWS(trusted), trusted, trusted},
		{"no identity keeps caller value (system job)", context.Background(), other, other},
		{"no identity and no value stays NULL", context.Background(), "", ""},
		{"identity without workspace keeps caller value", withWS(""), other, other},
		{"non-UUID trusted ws is not stamped (uuid column)", withWS("ws-1"), "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := trustedAuditWorkspaceID(tt.ctx, tt.requested); got != tt.want {
				t.Fatalf("trustedAuditWorkspaceID() = %q, want %q", got, tt.want)
			}
		})
	}
	if !isCanonicalUUID(trusted) || isCanonicalUUID("ws-1") || isCanonicalUUID("019ecb8e_d83f_74ab_aa13_5a6c27afd112") || isCanonicalUUID("019ecb8e-d83f-74ab-aa13-5a6c27afd11z") {
		t.Fatal("isCanonicalUUID classification wrong")
	}
}
