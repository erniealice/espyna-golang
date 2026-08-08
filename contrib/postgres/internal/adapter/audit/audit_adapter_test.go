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
