//go:build postgresql

package core

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"sort"
	"strings"
	"testing"

	_ "github.com/lib/pq"
)

// TestTableColumnsIgnoreOtherSchemas reproduces the Supabase collision that broke
// enrollment create on MMIS prod (2026-10-23): realtime.subscription shares its
// name with public.subscription, and the unscoped information_schema read merged
// both column sets, so Create's RETURNING scan failed ("expected 24 destination
// arguments in Scan, not 33"). A same-named table in a shadow schema must not
// leak into getTableColumns, getTableColumnTypes, or Create. All DDL runs inside
// one transaction that is rolled back (zero residue).
func TestTableColumnsIgnoreOtherSchemas(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	txManager := NewPostgreSQLTransactionManager(db)
	ops := &PostgresOperations{db: db}
	rollback := errors.New("rollback")

	err = txManager.RunInTransaction(context.Background(), func(ctx context.Context) error {
		exec := ops.getExecutor(ctx)
		for _, stmt := range []string{
			`CREATE TABLE _test_schema_scope (id TEXT PRIMARY KEY, name TEXT, active BOOL DEFAULT true, date_created TIMESTAMPTZ DEFAULT NOW(), date_modified TIMESTAMPTZ DEFAULT NOW())`,
			`CREATE SCHEMA _test_schema_scope_shadow`,
			`CREATE TABLE _test_schema_scope_shadow._test_schema_scope (id BIGINT, claims JSONB, filters TEXT[], created_at TIMESTAMP)`,
		} {
			if _, err := exec.ExecContext(ctx, stmt); err != nil {
				t.Fatalf("setup %q: %v", stmt, err)
			}
		}

		cols, err := ops.getTableColumns(ctx, "_test_schema_scope")
		if err != nil {
			t.Fatalf("getTableColumns: %v", err)
		}
		if got, want := strings.Join(cols, ","), "id,name,active,date_created,date_modified"; got != want {
			t.Errorf("getTableColumns = %s, want %s", got, want)
		}

		types, err := ops.getTableColumnTypes(ctx, "_test_schema_scope")
		if err != nil {
			t.Fatalf("getTableColumnTypes: %v", err)
		}
		names := make([]string, 0, len(types))
		for name := range types {
			names = append(names, name)
		}
		sort.Strings(names)
		if got, want := strings.Join(names, ","), "active,date_created,date_modified,id,name"; got != want {
			t.Errorf("getTableColumnTypes keys = %s, want %s", got, want)
		}
		if types["id"] != "text" {
			t.Errorf("id type = %q, want text (shadow BIGINT leaked)", types["id"])
		}

		if _, err := ops.Create(ctx, "_test_schema_scope", map[string]any{"id": "schema-scope-1", "name": "kept"}); err != nil {
			t.Errorf("Create with a same-named shadow table: %v", err)
		}
		return rollback
	})
	if !errors.Is(err, rollback) {
		t.Fatalf("expected rollback sentinel, got %v", err)
	}
}
