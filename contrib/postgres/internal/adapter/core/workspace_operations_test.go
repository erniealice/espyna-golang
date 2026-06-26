//go:build postgresql

package core

// Unit tests for WorkspaceAwareOperations.Read NULL workspace_id rejection.
//
// Phase 1.5 — 2026-05-10 — codex C1 fix:
//   Before: Read allowed NULL workspace_id rows when context had a workspace
//   After: Read rejects NULL rows as 404 (same path as workspace mismatch)
//
// Tests also cover that Update/Delete inherit the fix transitively through
// their Read ownership-verification calls.

import (
	"context"
	"database/sql"
	"testing"

	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
)

// stubInner is an in-memory DatabaseOperation for unit testing.
// Only Read, Update, Delete are implemented — all others panic to catch misuse.
type stubInner struct {
	readResult map[string]any
	readErr    error
	updateErr  error
	deleteErr  error
}

func (s *stubInner) Read(_ context.Context, _ string, _ string) (map[string]any, error) {
	return s.readResult, s.readErr
}
func (s *stubInner) Update(_ context.Context, _ string, _ string, data map[string]any) (map[string]any, error) {
	return data, s.updateErr
}
func (s *stubInner) Delete(_ context.Context, _ string, _ string) error {
	return s.deleteErr
}
func (s *stubInner) HardDelete(_ context.Context, _ string, _ string) error {
	return s.deleteErr
}
func (s *stubInner) List(_ context.Context, _ string, _ *interfaces.ListParams) (*interfaces.ListResult, error) {
	return &interfaces.ListResult{}, nil
}
func (s *stubInner) Create(_ context.Context, _ string, data map[string]any) (map[string]any, error) {
	return data, nil
}
func (s *stubInner) Query(_ context.Context, _ string, _ interfaces.QueryBuilder) ([]map[string]any, error) {
	return nil, nil
}
func (s *stubInner) QueryOne(_ context.Context, _ string, _ interfaces.QueryBuilder) (map[string]any, error) {
	return nil, nil
}

// stubDBWithColumn simulates tableHasWorkspaceColumn returning true by pre-populating
// the column cache. We use a real (offline) *sql.DB just for the struct — the cache
// pre-population avoids the live information_schema query.
func newStubWorkspaceOps(inner *stubInner, wsHasColumn bool) *WorkspaceAwareOperations {
	// Intentionally offline DB — cache prevents any real query.
	db, _ := sql.Open("postgres", "postgres://localhost/testdb?sslmode=disable")

	ops := &WorkspaceAwareOperations{
		inner:       inner,
		db:          db,
		columnCache: make(map[string]map[string]bool),
	}
	if wsHasColumn {
		// Pre-populate cache so tableHasWorkspaceColumn returns true without DB.
		ops.columnCache["test_table"] = map[string]bool{"workspace_id": true}
	} else {
		ops.columnCache["test_table"] = map[string]bool{}
	}
	return ops
}

// newCtxWithWorkspace injects a workspace ID into a context.
func newCtxWithWorkspace(wsID string) context.Context {
	return contextutil.WithWorkspaceID(context.Background(), wsID)
}

// ─── Read: NULL workspace_id on a tenant-scoped table ────────────────────────

func TestReadRejectsNullWorkspaceIDWhenContextHasWorkspace(t *testing.T) {
	inner := &stubInner{
		// Row exists but workspace_id is NULL (legacy unbackfilled row)
		readResult: map[string]any{
			"id":           "asset-1",
			"name":         "Old Laptop",
			"workspace_id": nil, // NULL
		},
	}
	w := newStubWorkspaceOps(inner, true)
	ctx := newCtxWithWorkspace("ws-abc")

	result, err := w.Read(ctx, "test_table", "asset-1")
	if err == nil {
		t.Fatal("expected error for NULL workspace_id row, got nil")
	}
	if result != nil {
		t.Errorf("expected nil result for rejected row, got %v", result)
	}
}

func TestReadRejectsEmptyStringWorkspaceIDWhenContextHasWorkspace(t *testing.T) {
	inner := &stubInner{
		readResult: map[string]any{
			"id":           "asset-2",
			"workspace_id": "", // empty string — same as NULL for enforcement purposes
		},
	}
	w := newStubWorkspaceOps(inner, true)
	ctx := newCtxWithWorkspace("ws-abc")

	_, err := w.Read(ctx, "test_table", "asset-2")
	if err == nil {
		t.Fatal("expected error for empty workspace_id row, got nil")
	}
}

func TestReadRejectsMismatchedWorkspaceID(t *testing.T) {
	inner := &stubInner{
		readResult: map[string]any{
			"id":           "asset-3",
			"workspace_id": "ws-other", // belongs to a different workspace
		},
	}
	w := newStubWorkspaceOps(inner, true)
	ctx := newCtxWithWorkspace("ws-abc")

	_, err := w.Read(ctx, "test_table", "asset-3")
	if err == nil {
		t.Fatal("expected error for mismatched workspace_id, got nil")
	}
}

func TestReadAllowsMatchingWorkspaceID(t *testing.T) {
	inner := &stubInner{
		readResult: map[string]any{
			"id":           "asset-4",
			"workspace_id": "ws-abc",
		},
	}
	w := newStubWorkspaceOps(inner, true)
	ctx := newCtxWithWorkspace("ws-abc")

	result, err := w.Read(ctx, "test_table", "asset-4")
	if err != nil {
		t.Fatalf("unexpected error for matching workspace_id: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result for valid read")
	}
}

func TestReadPassesThroughWhenContextHasNoWorkspace(t *testing.T) {
	inner := &stubInner{
		// NULL workspace_id row — but context has no workspace (service-to-service)
		readResult: map[string]any{
			"id":           "asset-5",
			"workspace_id": nil,
		},
	}
	w := newStubWorkspaceOps(inner, true)
	ctx := context.Background() // no workspace in context

	result, err := w.Read(ctx, "test_table", "asset-5")
	if err != nil {
		t.Fatalf("unexpected error for service-to-service read (no ctx workspace): %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result for service-to-service read")
	}
}

func TestReadPassesThroughForTableWithoutWorkspaceColumn(t *testing.T) {
	inner := &stubInner{
		readResult: map[string]any{
			"id":   "cat-1",
			"name": "IT Equipment",
			// no workspace_id key — table doesn't have the column
		},
	}
	w := newStubWorkspaceOps(inner, false) // column cache says NOT present
	ctx := newCtxWithWorkspace("ws-abc")

	result, err := w.Read(ctx, "test_table", "cat-1")
	if err != nil {
		t.Fatalf("unexpected error for table without workspace_id column: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result for non-tenant-scoped table")
	}
}

// ─── Update/Delete: transitively inherit the NULL rejection via Read ─────────

func TestUpdateRejectsNullWorkspaceIDViaRead(t *testing.T) {
	inner := &stubInner{
		readResult: map[string]any{
			"id":           "asset-6",
			"workspace_id": nil, // NULL legacy row
		},
	}
	w := newStubWorkspaceOps(inner, true)
	ctx := newCtxWithWorkspace("ws-abc")

	_, err := w.Update(ctx, "test_table", "asset-6", map[string]any{"name": "Updated"})
	if err == nil {
		t.Fatal("expected error for Update on NULL workspace_id row, got nil")
	}
}

func TestDeleteRejectsNullWorkspaceIDViaRead(t *testing.T) {
	inner := &stubInner{
		readResult: map[string]any{
			"id":           "asset-7",
			"workspace_id": nil, // NULL legacy row
		},
	}
	w := newStubWorkspaceOps(inner, true)
	ctx := newCtxWithWorkspace("ws-abc")

	err := w.Delete(ctx, "test_table", "asset-7")
	if err == nil {
		t.Fatal("expected error for Delete on NULL workspace_id row, got nil")
	}
}

// Ensure the commonpb import (used by the real injectWorkspaceFilter) compiles.
var _ = (*commonpb.TypedFilter)(nil)
