//go:build postgresql

package core

import (
	"context"
	"database/sql"
	"net/url"
	"os"
	"reflect"
	"strings"
	"testing"

	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	"github.com/erniealice/espyna-golang/shared/identity"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	_ "github.com/lib/pq"
)

// This test accepts only the local named clone and configures every new
// connection read-only before querying. No fixture or schema mutation occurs.
func TestColumnlessShadowListLiveReadOnlyParity(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	u, err := url.Parse(dsn)
	if err != nil || (u.Scheme != "postgres" && u.Scheme != "postgresql") ||
		(u.Hostname() != "localhost" && u.Hostname() != "127.0.0.1") ||
		strings.TrimPrefix(u.Path, "/") != "education2clone20260925b" {
		t.Skip("requires local education2clone20260925b URL")
	}
	q := u.Query()
	q.Set("options", "-c default_transaction_read_only=on")
	u.RawQuery = q.Encode()
	db, err := sql.Open("postgres", u.String())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	var readOnly string
	if err := db.QueryRowContext(ctx, "SHOW default_transaction_read_only").Scan(&readOnly); err != nil || readOnly != "on" {
		t.Fatalf("read-only session not established: %q %v", readOnly, err)
	}
	var database string
	if err := db.QueryRowContext(ctx, "SELECT current_database()").Scan(&database); err != nil || database != "education2clone20260925b" {
		t.Fatalf("unexpected database %q: %v", database, err)
	}
	inner := NewPostgresOperations(db)
	params := &interfaces.ListParams{Pagination: &commonpb.PaginationRequest{Limit: 25}}
	for _, table := range []string{"client_attribute", "product_variant"} {
		t.Run(table, func(t *testing.T) {
			var workspace string
			parent := "client"
			if table == "product_variant" {
				parent = "product"
			}
			if err := db.QueryRowContext(ctx, "SELECT workspace_id::text FROM "+parent+" WHERE workspace_id IS NOT NULL LIMIT 1").Scan(&workspace); err != nil {
				t.Skipf("no local %s workspace sample: %v", parent, err)
			}
			caller := identity.WithRequestIdentity(ctx, &identity.RequestIdentity{WorkspaceID: workspace})
			before, err := inner.List(caller, table, params)
			if err != nil {
				t.Fatal(err)
			}
			w := &WorkspaceAwareOperations{inner: inner, db: db, columnCache: map[string]map[string]bool{table: {}}}
			var after *interfaces.ListResult
			logs := shadowCapture(t, func() { after, err = w.List(caller, table, params) })
			if err != nil || !reflect.DeepEqual(before, after) {
				t.Fatalf("shadow changed list result: err=%v", err)
			}
			t.Logf("table=%s row_count=%d shadow_deny_lines=%d", table, len(after.Data), strings.Count(logs, "AUTHZ_WS_COLUMNLESS_SHADOW_DENY"))
			if len(after.Data) > 0 {
				foreign := identity.WithRequestIdentity(ctx, &identity.RequestIdentity{WorkspaceID: "columnless-shadow-nonexistent-workspace"})
				var foreignAfter *interfaces.ListResult
				foreignLogs := shadowCapture(t, func() { foreignAfter, err = w.List(foreign, table, params) })
				if err != nil || !reflect.DeepEqual(before, foreignAfter) {
					t.Fatalf("foreign-workspace shadow changed list result: err=%v", err)
				}
				if !strings.Contains(foreignLogs, "table="+table+" op=list reason=mismatch") {
					t.Fatalf("missing foreign-workspace shadow mismatch")
				}
				t.Logf("table=%s foreign_workspace_shadow_deny_lines=%d", table, strings.Count(foreignLogs, "AUTHZ_WS_COLUMNLESS_SHADOW_DENY"))
			}
		})
	}
}
