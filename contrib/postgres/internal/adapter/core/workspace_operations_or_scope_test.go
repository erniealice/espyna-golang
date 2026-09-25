//go:build postgresql

package core

// fix3-tenant-or (2026-09-25): a caller FilterLogic_OR must never OR the tenant
// workspace_id predicate away. Before the fix, WorkspaceAwareOperations.List
// prepended workspace_id into the caller's FilterRequest and kept its Logic, so
// PostgresOperations.List emitted `(workspace_id = $1 OR <caller> ...)` and
// returned cross-workspace rows. Now OR requests go through ListWithScope, which
// AND-s the tenant predicate outside the caller's group.

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"testing"

	_ "github.com/lib/pq"

	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	"github.com/erniealice/espyna-golang/shared/database/operations"
	"github.com/erniealice/espyna-golang/shared/identity"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
)

// scopeRecordingInner records how the decorator called it.
type scopeRecordingInner struct {
	stubInner
	listCalled      bool
	listParams      *interfaces.ListParams
	scopedCalled    bool
	scopedScope     []*commonpb.TypedFilter
	scopedParams    *interfaces.ListParams
	scopedTableName string
}

func (s *scopeRecordingInner) List(_ context.Context, _ string, params *interfaces.ListParams) (*interfaces.ListResult, error) {
	s.listCalled = true
	s.listParams = params
	return &interfaces.ListResult{}, nil
}

func (s *scopeRecordingInner) ListWithScope(_ context.Context, tableName string, scope []*commonpb.TypedFilter, params *interfaces.ListParams) (*interfaces.ListResult, error) {
	s.scopedCalled = true
	s.scopedTableName = tableName
	s.scopedScope = scope
	s.scopedParams = params
	return &interfaces.ListResult{}, nil
}

func newScopeTestOps(inner interfaces.DatabaseOperation) *WorkspaceAwareOperations {
	db, _ := sql.Open("postgres", "postgres://localhost/testdb?sslmode=disable")
	return &WorkspaceAwareOperations{
		inner:       inner,
		db:          db,
		columnCache: map[string]map[string]bool{"test_table": {"workspace_id": true}},
	}
}

func strEq(field, value string) *commonpb.TypedFilter {
	return &commonpb.TypedFilter{
		Field: field,
		FilterType: &commonpb.TypedFilter_StringFilter{
			StringFilter: &commonpb.StringFilter{Value: value, Operator: commonpb.StringOperator_STRING_EQUALS, CaseSensitive: true},
		},
	}
}

func assertWorkspaceScope(t *testing.T, scope []*commonpb.TypedFilter, wsID string) {
	t.Helper()
	if len(scope) != 1 {
		t.Fatalf("expected exactly 1 mandatory scope filter, got %d", len(scope))
	}
	f := scope[0]
	sf := f.GetStringFilter()
	if f.GetField() != "workspace_id" || sf == nil || sf.GetValue() != wsID ||
		sf.GetOperator() != commonpb.StringOperator_STRING_EQUALS || !sf.GetCaseSensitive() {
		t.Fatalf("scope filter is not workspace_id == %q (case-sensitive): %v", wsID, f)
	}
}

func TestListORFilterRoutesTenantPredicateOutsideCallerGroup(t *testing.T) {
	inner := &scopeRecordingInner{}
	w := newScopeTestOps(inner)
	caller := &interfaces.ListParams{Filters: &commonpb.FilterRequest{
		Logic:   commonpb.FilterLogic_OR,
		Filters: []*commonpb.TypedFilter{strEq("code", "a"), strEq("name", "b")},
	}}

	if _, err := w.List(newCtxWithWorkspace("ws-own"), "test_table", caller); err != nil {
		t.Fatalf("List: %v", err)
	}
	if inner.listCalled {
		t.Fatal("OR request reached plain inner.List — tenant predicate would be OR-ed with caller filters")
	}
	if !inner.scopedCalled || inner.scopedTableName != "test_table" {
		t.Fatal("OR request did not use the mandatory-scope list path")
	}
	assertWorkspaceScope(t, inner.scopedScope, "ws-own")
	// Caller's group is passed through untouched: tenant filter NOT inside it.
	got := inner.scopedParams.Filters
	if got.GetLogic() != commonpb.FilterLogic_OR || len(got.GetFilters()) != 2 {
		t.Fatalf("caller filter group altered: %v", got)
	}
	for _, f := range got.GetFilters() {
		if f.GetField() == "workspace_id" {
			t.Fatal("workspace_id leaked into the caller's OR group")
		}
	}
}

func TestListORFilterFailsClosedWithoutScopeCapability(t *testing.T) {
	w := newScopeTestOps(&stubInner{}) // no ListWithScope
	caller := &interfaces.ListParams{Filters: &commonpb.FilterRequest{
		Logic:   commonpb.FilterLogic_OR,
		Filters: []*commonpb.TypedFilter{strEq("code", "a")},
	}}
	if _, err := w.List(newCtxWithWorkspace("ws-own"), "test_table", caller); err == nil {
		t.Fatal("expected fail-closed error for OR logic on an inner without mandatory-scope list")
	}
}

func TestListANDFilterKeepsPrependBehaviour(t *testing.T) {
	inner := &scopeRecordingInner{}
	w := newScopeTestOps(inner)
	caller := &interfaces.ListParams{Filters: &commonpb.FilterRequest{
		Filters: []*commonpb.TypedFilter{strEq("code", "a")},
	}}
	if _, err := w.List(newCtxWithWorkspace("ws-own"), "test_table", caller); err != nil {
		t.Fatalf("List: %v", err)
	}
	if inner.scopedCalled || !inner.listCalled {
		t.Fatal("AND request must keep the legacy prepend path")
	}
	fs := inner.listParams.Filters.GetFilters()
	if len(fs) != 2 || inner.listParams.Filters.GetLogic() != commonpb.FilterLogic_AND {
		t.Fatalf("unexpected AND params: %v", inner.listParams.Filters)
	}
	assertWorkspaceScope(t, fs[:1], "ws-own")
	if fs[1].GetField() != "code" {
		t.Fatalf("caller filter order changed: %v", fs)
	}
	if len(caller.Filters.Filters) != 1 {
		t.Fatal("caller params mutated")
	}
}

func TestListScopeValidationParamsCountsScopeInBudget(t *testing.T) {
	caller := &interfaces.ListParams{Filters: &commonpb.FilterRequest{
		Logic:   commonpb.FilterLogic_OR,
		Filters: []*commonpb.TypedFilter{strEq("code", "a")},
	}}
	got := listScopeValidationParams([]*commonpb.TypedFilter{workspaceEqualsFilter("ws")}, caller)
	if len(got.Filters.GetFilters()) != 2 || got.Filters.GetFilters()[0].GetField() != "workspace_id" {
		t.Fatalf("scope not prepended for validation: %v", got.Filters)
	}
	if len(caller.Filters.Filters) != 1 {
		t.Fatal("caller params mutated")
	}
	if listScopeValidationParams(nil, caller) != caller {
		t.Fatal("no-scope validation params must be the caller's params")
	}
	if nilGot := listScopeValidationParams([]*commonpb.TypedFilter{workspaceEqualsFilter("ws")}, nil); len(nilGot.Filters.GetFilters()) != 1 {
		t.Fatal("nil params with scope must validate the scope filter")
	}
}

// TestListORFilterExcludesForeignWorkspaceRowsLive runs against a real database
// (TEST_DATABASE_URL — the education2clone20260925a lane, never education2)
// inside a transaction that is ALWAYS rolled back. It seeds a second workspace
// plus one `category` row per workspace, then lists as the first workspace with
// an OR filter that matches the foreign row.
func TestListORFilterExcludesForeignWorkspaceRowsLive(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	const (
		ownWS     = "fix3-or-test-ws-own"
		foreignWS = "fix3-or-test-ws-foreign"
		ownRow    = "fix3-or-test-cat-own"
		foreignRw = "fix3-or-test-cat-foreign"
	)
	rollback := errors.New("always roll back")
	txManager := NewPostgreSQLTransactionManager(db)
	inner := NewPostgresOperations(db)
	w := &WorkspaceAwareOperations{inner: inner, db: db, columnCache: map[string]map[string]bool{}}

	err = txManager.RunInTransaction(context.Background(), func(txCtx context.Context) error {
		tx, ok := operations.GetTransactionFromContext(txCtx)
		if !ok {
			t.Fatal("no transaction in context")
		}
		exec := tx.(*PostgreSQLTransaction).tx
		for _, ws := range []string{ownWS, foreignWS} {
			if _, err := exec.ExecContext(txCtx, `INSERT INTO workspace (id, name) VALUES ($1, $1)`, ws); err != nil {
				t.Fatalf("seed workspace: %v", err)
			}
		}
		if _, err := exec.ExecContext(txCtx,
			`INSERT INTO category (id, name, code, workspace_id) VALUES ($1,'own','FIX3-OWN',$2), ($3,'foreign','FIX3-FOREIGN',$4)`,
			ownRow, ownWS, foreignRw, foreignWS); err != nil {
			t.Fatalf("seed category: %v", err)
		}

		ctx := identity.WithRequestIdentity(txCtx, &identity.RequestIdentity{WorkspaceID: ownWS})
		list := func(codes ...string) map[string]bool {
			t.Helper()
			fs := make([]*commonpb.TypedFilter, 0, len(codes))
			for _, c := range codes {
				fs = append(fs, strEq("code", c))
			}
			res, err := w.List(ctx, "category", &interfaces.ListParams{Filters: &commonpb.FilterRequest{Logic: commonpb.FilterLogic_OR, Filters: fs}})
			if err != nil {
				t.Fatalf("List: %v", err)
			}
			ids := map[string]bool{}
			for _, r := range res.Data {
				ids[r["id"].(string)] = true
			}
			if int(res.Total) != len(res.Data) {
				t.Fatalf("COUNT(*) %d disagrees with page rows %d (tenant predicate must scope the count too)", res.Total, len(res.Data))
			}
			return ids
		}

		// Attack: OR filter matching ONLY the foreign row.
		if got := list("FIX3-FOREIGN"); len(got) != 0 {
			t.Fatalf("cross-workspace leak: OR filter returned %v", got)
		}
		// Attack + control: OR matching both rows → only the own row.
		if got := list("FIX3-FOREIGN", "FIX3-OWN"); len(got) != 1 || !got[ownRow] {
			t.Fatalf("expected only own row, got %v", got)
		}

		// Regression witness: the pre-fix composition (prepend into the OR group)
		// DOES leak on this data — proves the test data exercises the bug.
		legacy := w.injectWorkspaceFilter(&interfaces.ListParams{Filters: &commonpb.FilterRequest{
			Logic: commonpb.FilterLogic_OR, Filters: []*commonpb.TypedFilter{strEq("code", "FIX3-FOREIGN")},
		}}, ownWS)
		res, err := inner.List(ctx, "category", legacy)
		if err != nil {
			t.Fatalf("legacy List: %v", err)
		}
		leaked := false
		for _, r := range res.Data {
			if r["id"] == foreignRw {
				leaked = true
			}
		}
		if !leaked {
			t.Fatal("regression witness: legacy prepend-into-OR path no longer leaks; test data does not exercise the bug")
		}
		return rollback
	})
	if !errors.Is(err, rollback) {
		t.Fatalf("expected forced rollback, got %v", err)
	}
	var n int
	if err := db.QueryRow(`SELECT count(*) FROM category WHERE id IN ($1,$2)`, ownRow, foreignRw).Scan(&n); err != nil || n != 0 {
		t.Fatalf("rollback check: n=%d err=%v", n, err)
	}
}
