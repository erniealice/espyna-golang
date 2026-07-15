//go:build postgresql

package core

// Gate H1 regression tests for the WorkspaceAwareOperations decorator: a
// client-supplied camelCase workspaceId must never coexist with the trusted
// workspace_id (Create) nor survive an Update. Reuses stubInner /
// newStubWorkspaceOps / newCtxWithWorkspace from workspace_operations_test.go.

import (
	"testing"
)

func TestCreateInjectsTrustedWorkspaceOverClientCamelKey(t *testing.T) {
	// stubInner.Create echoes the data map it receives, so the decorator's return
	// value is exactly what the write path would persist.
	inner := &stubInner{}
	w := newStubWorkspaceOps(inner, true)
	ctx := newCtxWithWorkspace("trusted-ws")

	out, err := w.Create(ctx, "test_table", map[string]any{
		"workspaceId": "attacker-ws", // client-supplied camelCase spelling
		"name":        "Academic",
	})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if out["workspace_id"] != "trusted-ws" {
		t.Errorf("expected trusted workspace_id, got %v", out["workspace_id"])
	}
	if _, ok := out["workspaceId"]; ok {
		t.Error("client camelCase workspaceId survived Create (collision vector open)")
	}
}

func TestUpdateStripsBothWorkspaceSpellings(t *testing.T) {
	// Ownership Read must pass: the existing row belongs to the acting workspace.
	inner := &stubInner{readResult: map[string]any{
		"id":           "row-1",
		"workspace_id": "trusted-ws",
	}}
	w := newStubWorkspaceOps(inner, true)
	ctx := newCtxWithWorkspace("trusted-ws")

	out, err := w.Update(ctx, "test_table", "row-1", map[string]any{
		"workspaceId":  "attacker-ws",  // camelCase spelling
		"workspace_id": "attacker2-ws", // snake spelling
		"name":         "Changed",
	})
	if err != nil {
		t.Fatalf("Update returned error: %v", err)
	}
	if _, ok := out["workspaceId"]; ok {
		t.Error("client camelCase workspaceId survived Update (tenant reassignment vector open)")
	}
	if _, ok := out["workspace_id"]; ok {
		t.Error("client snake workspace_id survived Update (tenant reassignment vector open)")
	}
	if out["name"] != "Changed" {
		t.Errorf("non-workspace payload lost on Update: %v", out)
	}
}
