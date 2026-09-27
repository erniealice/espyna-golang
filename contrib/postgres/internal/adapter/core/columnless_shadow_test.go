//go:build postgresql

package core

import (
	"bytes"
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"log"
	"reflect"
	"strings"
	"testing"

	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
)

type shadowDBState struct {
	owners  map[string]*string
	queries int
	maxArgs int
	fail    bool
}

type shadowConnector struct{ state *shadowDBState }

func (c *shadowConnector) Connect(context.Context) (driver.Conn, error) {
	return &shadowConn{c.state}, nil
}
func (c *shadowConnector) Driver() driver.Driver { return shadowDriver{c.state} }

type shadowDriver struct{ state *shadowDBState }

func (d shadowDriver) Open(string) (driver.Conn, error) { return &shadowConn{d.state}, nil }

type shadowConn struct{ state *shadowDBState }

func (*shadowConn) Prepare(string) (driver.Stmt, error) { return nil, errors.New("unused") }
func (*shadowConn) Close() error                        { return nil }
func (*shadowConn) Begin() (driver.Tx, error)           { return nil, errors.New("unused") }
func (c *shadowConn) QueryContext(_ context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	c.state.queries++
	if len(args) > c.state.maxArgs {
		c.state.maxArgs = len(args)
	}
	if c.state.fail {
		return nil, errors.New("probe unavailable")
	}
	if !strings.Contains(query, `LEFT JOIN "client" p0 ON c."client_id" = p0.id`) &&
		!strings.Contains(query, `LEFT JOIN "product" p0 ON c."product_id" = p0.id`) {
		return nil, errors.New("unexpected chain SQL")
	}
	var rows [][]driver.Value
	for _, arg := range args {
		id, ok := arg.Value.(string)
		if !ok {
			return nil, errors.New("non-string ID")
		}
		owner, found := c.state.owners[id]
		if !found {
			continue
		}
		if owner == nil {
			rows = append(rows, []driver.Value{id, nil})
		} else {
			rows = append(rows, []driver.Value{id, *owner})
		}
	}
	return &shadowRows{data: rows}, nil
}

type shadowRows struct {
	data   [][]driver.Value
	offset int
}

func (*shadowRows) Columns() []string { return []string{"id", "workspace_id"} }
func (*shadowRows) Close() error      { return nil }
func (r *shadowRows) Next(dest []driver.Value) error {
	if r.offset == len(r.data) {
		return io.EOF
	}
	copy(dest, r.data[r.offset])
	r.offset++
	return nil
}

type shadowListInner struct {
	stubInner
	list *interfaces.ListResult
}

func (s *shadowListInner) List(context.Context, string, *interfaces.ListParams) (*interfaces.ListResult, error) {
	return s.list, nil
}

func shadowTestOps(state *shadowDBState, inner interfaces.DatabaseOperation, table string) *WorkspaceAwareOperations {
	return &WorkspaceAwareOperations{inner: inner, db: sql.OpenDB(&shadowConnector{state}),
		columnCache: map[string]map[string]bool{table: {}}, enforce: true}
}

func shadowCapture(t *testing.T, run func()) string {
	t.Helper()
	var output bytes.Buffer
	old := log.Writer()
	log.SetOutput(&output)
	defer log.SetOutput(old)
	run()
	return output.String()
}

func shadowString(s string) *string { return &s }

func TestColumnlessShadowByIDPreservesResults(t *testing.T) {
	for _, tc := range []struct {
		name   string
		table  string
		owner  *string
		fail   bool
		reason string
	}{
		{"match", "client_attribute", shadowString("own"), false, ""},
		{"mismatch", "client_attribute", shadowString("other"), false, "mismatch"},
		{"null_parent", "client_attribute", nil, false, "unresolved"},
		{"probe_error", "client_attribute", shadowString("own"), true, "unresolved"},
		{"no_mechanism", "attribute", nil, false, "no_mechanism"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			columnlessNoMechanismLogged.Delete(tc.table)
			row := map[string]any{"id": "row1", "payload": "unchanged"}
			inner := &stubInner{readResult: row}
			state := &shadowDBState{owners: map[string]*string{"row1": tc.owner}, fail: tc.fail}
			w := shadowTestOps(state, inner, tc.table)
			defer w.db.Close()
			ctx := newCtxWithWorkspace("own")
			logs := shadowCapture(t, func() {
				got, err := w.Read(ctx, tc.table, "row1")
				if err != nil || !reflect.DeepEqual(got, row) {
					t.Fatalf("Read changed: row=%v err=%v", got, err)
				}
				updated, err := w.Update(ctx, tc.table, "row1", row)
				if err != nil || !reflect.DeepEqual(updated, row) {
					t.Fatalf("Update changed: row=%v err=%v", updated, err)
				}
				if err := w.Delete(ctx, tc.table, "row1"); err != nil {
					t.Fatalf("Delete changed: %v", err)
				}
				if err := w.HardDelete(ctx, tc.table, "row1"); err != nil {
					t.Fatalf("HardDelete changed: %v", err)
				}
			})
			if tc.reason == "" && strings.Contains(logs, "AUTHZ_WS_COLUMNLESS_SHADOW_DENY") {
				t.Fatalf("match logged deny: %s", logs)
			}
			if tc.reason != "" && !strings.Contains(logs, "AUTHZ_WS_COLUMNLESS_SHADOW_DENY table="+tc.table+" op=read reason="+tc.reason) {
				t.Fatalf("missing token: %s", logs)
			}
			if strings.Contains(logs, "row1") || strings.Contains(logs, "other") {
				t.Fatalf("log exposed row data: %s", logs)
			}
			wantQueries := 4
			if tc.reason == "no_mechanism" {
				wantQueries = 0
				if strings.Count(logs, "reason=no_mechanism") != 1 {
					t.Fatalf("rate limit: %s", logs)
				}
			}
			if state.queries != wantQueries {
				t.Fatalf("queries=%d want %d", state.queries, wantQueries)
			}
		})
	}
}

func TestColumnlessShadowListOneBatchedQueryAndSameRows(t *testing.T) {
	rows := []map[string]any{{"id": "a"}, {"id": "b"}, {"id": "c"}}
	inner := &shadowListInner{list: &interfaces.ListResult{Data: rows, Total: 3}}
	state := &shadowDBState{owners: map[string]*string{"a": shadowString("own"), "b": shadowString("other"), "c": nil}}
	w := shadowTestOps(state, inner, "product_variant")
	defer w.db.Close()
	var got *interfaces.ListResult
	logs := shadowCapture(t, func() {
		var err error
		got, err = w.List(newCtxWithWorkspace("own"), "product_variant", nil)
		if err != nil {
			t.Fatal(err)
		}
	})
	if got != inner.list || !reflect.DeepEqual(got.Data, rows) {
		t.Fatalf("list result changed: %v", got)
	}
	if state.queries != 1 {
		t.Fatalf("3-row page added %d queries, want 1", state.queries)
	}
	for _, reason := range []string{"mismatch", "unresolved"} {
		if !strings.Contains(logs, "table=product_variant op=list reason="+reason) {
			t.Fatalf("missing %s: %s", reason, logs)
		}
	}
	// Even a 65-row page gets only one batch of at most 64 IDs.
	large := make([]map[string]any, 65)
	for i := range large {
		large[i] = map[string]any{"id": "a"}
	}
	inner.list = &interfaces.ListResult{Data: large, Total: 65}
	state.queries = 0
	shadowCapture(t, func() { _, _ = w.List(newCtxWithWorkspace("own"), "product_variant", nil) })
	if state.queries != 1 {
		t.Fatalf("65-row page added %d queries", state.queries)
	}
	if state.maxArgs != columnlessListSampleLimit {
		t.Fatalf("65-row page probed %d IDs, want %d", state.maxArgs, columnlessListSampleLimit)
	}
}

func TestColumnlessShadowListNoMechanismIsRateLimited(t *testing.T) {
	columnlessNoMechanismLogged.Delete("attribute")
	rows := []map[string]any{{"id": "a"}, {"id": "b"}}
	inner := &shadowListInner{list: &interfaces.ListResult{Data: rows, Total: 2}}
	state := &shadowDBState{}
	w := shadowTestOps(state, inner, "attribute")
	defer w.db.Close()
	logs := shadowCapture(t, func() {
		for i := 0; i < 2; i++ {
			got, err := w.List(newCtxWithWorkspace("own"), "attribute", nil)
			if err != nil || got != inner.list || !reflect.DeepEqual(got.Data, rows) {
				t.Fatalf("no-mechanism List changed: %v %v", got, err)
			}
		}
	})
	if state.queries != 0 || strings.Count(logs, "table=attribute op=list reason=no_mechanism") != 1 {
		t.Fatalf("no-mechanism list query/log count: queries=%d logs=%s", state.queries, logs)
	}
}
