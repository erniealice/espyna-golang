package identity

import (
	"context"
	"testing"
)

func TestWorkspaceRowScopeMarker(t *testing.T) {
	ctx := context.Background()
	if HasWorkspaceRowScope(ctx) {
		t.Fatal("unmarked context must report false (fail closed)")
	}
	if !HasWorkspaceRowScope(WithWorkspaceRowScope(ctx)) {
		t.Fatal("marked context must report true")
	}
	if WorkspaceRowScopePermission != "approval_scope:workspace" {
		t.Fatalf("permission code drifted: %q", WorkspaceRowScopePermission)
	}
}
