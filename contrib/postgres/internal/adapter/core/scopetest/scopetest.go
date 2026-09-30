//go:build postgresql

// Package scopetest is the ONE real-database harness of the S1 known-cost-recovery adapter tests
// (20260927-usage-and-pass-through-charges, build-spec §7 C16). It replaces the five per-package
// copies (expenditure, ledger, revenue, subscription, treasury) of `s1_scope_harness_test.go`.
//
// Contract:
//   - the DSN comes from the TEST_DATABASE_URL environment variable. When it is unset the test
//     SKIPS with a loud message (like every other real-DB suite in this module), unless
//     ESPYNA_REQUIRE_DB_TESTS=1, in which case it FAILS: a provider-tagged run of this plan's lane
//     exports ESPYNA_REQUIRE_DB_TESTS=1 so it cannot look green without running;
//   - the harness runs destructive setup (session_replication_role = replica, seeded workspaces,
//     committed fixtures) so it refuses any database whose name does not match
//     (_test|_scratch|_clone) or is not listed in ESPYNA_TEST_DB_ALLOW (comma-separated names);
//   - session_replication_role = replica needs a database SUPERUSER; an under-privileged role fails
//     the test with that reason (never skips once a DSN is supplied);
//   - Run executes a scenario inside ONE rolled-back transaction (foreign keys disabled for it) so an
//     adapter is exercised without seeding its parent graph: it proves the WORKSPACE PREDICATE and
//     adapter behaviour, not referential integrity;
//   - OpenReplicaPool + SeedCommitted + the returned cleanup give two-connection tests (claim races,
//     gapless numbering) COMMITTED fixtures that see each other, removed again by id prefix.
package scopetest

import (
	"context"
	"database/sql"
	"errors"
	"net/url"
	"os"
	"regexp"
	"strings"
	"testing"

	auditadapter "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/audit"
	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	"github.com/erniealice/espyna-golang/shared/identity"
	"github.com/lib/pq"
)

const (
	// EnvDSN names the environment variable holding the connection string of the clone.
	EnvDSN = "TEST_DATABASE_URL"
	// EnvRequire, when "1", turns an unset EnvDSN from a skip into a failure.
	EnvRequire = "ESPYNA_REQUIRE_DB_TESTS"
	// EnvAllow lists (comma-separated) database names allowed in addition to the name pattern.
	EnvAllow = "ESPYNA_TEST_DB_ALLOW"
	// WsA / WsB are the two workspaces of the scope scenarios.
	WsA = "s1-scope-ws-a"
	WsB = "s1-scope-ws-b"
)

var errRollback = errors.New("scopetest: intentional rollback")

// Harness is one pool on the clone plus the workspace-aware, audited operations and the
// transaction manager the adapters are built from.
type Harness struct {
	DB  *sql.DB
	Ops interfaces.DatabaseOperation
	Tm  interfaces.TransactionManager
}

var safeDBName = regexp.MustCompile(`(_test|_scratch|_clone)`)

// dbName extracts the database name from a URL or keyword/value DSN ("" when absent).
func dbName(dsn string) string {
	if u, err := url.Parse(dsn); err == nil && (u.Scheme == "postgres" || u.Scheme == "postgresql") {
		return strings.TrimPrefix(u.Path, "/")
	}
	for _, f := range strings.Fields(dsn) {
		if v, ok := strings.CutPrefix(f, "dbname="); ok {
			return strings.Trim(v, "'")
		}
	}
	return ""
}

// DSN returns the connection string of the test database, skipping (or failing under
// ESPYNA_REQUIRE_DB_TESTS=1) when unset, and failing when the database name is not one destructive
// setup is allowed on.
func DSN(t testing.TB) string {
	t.Helper()
	dsn := os.Getenv(EnvDSN)
	if dsn == "" {
		if os.Getenv(EnvRequire) == "1" {
			t.Fatalf("%s is unset but %s=1: the real-database tests must run", EnvDSN, EnvRequire)
		}
		t.Skipf("SKIPPED (NOT RUN): %s is unset; real-database tests did not execute (set %s=1 to make this a failure)", EnvDSN, EnvRequire)
	}
	name := dbName(dsn)
	allowed := safeDBName.MatchString(name)
	for _, a := range strings.Split(os.Getenv(EnvAllow), ",") {
		if name != "" && strings.TrimSpace(a) == name {
			allowed = true
		}
	}
	if !allowed {
		t.Fatalf("scopetest: refusing destructive setup on database %q: its name must match (_test|_scratch|_clone) or be listed in %s", name, EnvAllow)
	}
	return dsn
}

func open(t testing.TB, dsn string) *sql.DB {
	t.Helper()
	connector, err := pq.NewConnector(dsn)
	if err != nil {
		t.Fatalf("scopetest: bad %s: %v", EnvDSN, err)
	}
	db := sql.OpenDB(connector)
	t.Cleanup(func() { db.Close() })
	if err := db.Ping(); err != nil {
		t.Fatalf("scopetest: test database is unreachable: %v", err)
	}
	return db
}

func newHarness(t testing.TB, db *sql.DB, tables []string) *Harness {
	t.Helper()
	for _, table := range tables {
		var present sql.NullString
		if err := db.QueryRow("SELECT to_regclass('public." + table + "')::text").Scan(&present); err != nil || !present.Valid {
			t.Fatalf("scopetest: table %s is not present in %s (schema release 2026.09.7 not applied): %v", table, dbName(os.Getenv(EnvDSN)), err)
		}
	}
	return &Harness{
		DB:  db,
		Ops: postgresCore.NewAuditedWorkspaceAwareOperations(db, auditadapter.New(db)),
		Tm:  postgresCore.NewPostgreSQLTransactionManager(db),
	}
}

// New opens the clone and requires every listed table to exist.
func New(t testing.TB, tables ...string) *Harness {
	t.Helper()
	return newHarness(t, open(t, DSN(t)), tables)
}

// NewReplica opens ANOTHER independent pool (its own connections, so its transactions contend with
// the other harnesses' for real) whose sessions run with session_replication_role = replica (FK
// triggers off; superuser), for committed fixtures without parent graphs.
func NewReplica(t testing.TB, tables ...string) *Harness {
	t.Helper()
	dsn := DSN(t)
	const opt = "-c session_replication_role=replica"
	if u, err := url.Parse(dsn); err == nil && (u.Scheme == "postgres" || u.Scheme == "postgresql") {
		q := u.Query()
		q.Set("options", opt)
		u.RawQuery = q.Encode()
		dsn = u.String()
	} else {
		dsn += " options='" + opt + "'"
	}
	h := newHarness(t, open(t, dsn), tables)
	var role string
	if err := h.DB.QueryRow("SHOW session_replication_role").Scan(&role); err != nil || role != "replica" {
		t.Fatalf("scopetest: cannot set session_replication_role=replica (needs a superuser): role=%q err=%v", role, err)
	}
	return h
}

// Ctx returns a context carrying a trusted request identity of the workspace ("" = no workspace).
func Ctx(base context.Context, workspaceID string) context.Context {
	return identity.WithRequestIdentity(base, &identity.RequestIdentity{WorkspaceID: workspaceID})
}

// Run executes fn with contexts for workspace A, workspace B and no workspace inside ONE
// transaction that is always rolled back. base is the transaction context (direct executor use).
// FK triggers are disabled for the transaction; a database that cannot do so fails the test.
func (h *Harness) Run(t *testing.T, fn func(base, ctxA, ctxB, ctxNone context.Context)) {
	t.Helper()
	err := h.Tm.RunInTransaction(context.Background(), func(base context.Context) error {
		exec := postgresCore.TxExecutor(base, h.Ops)
		if exec == nil {
			t.Fatal("scopetest: no transaction executor")
		}
		if _, err := exec.ExecContext(base, "SET LOCAL session_replication_role = replica"); err != nil {
			t.Fatalf("scopetest: cannot disable FK triggers (needs a superuser): %v", err)
		}
		for _, ws := range []string{WsA, WsB} {
			if _, err := exec.ExecContext(base, `INSERT INTO workspace (id) VALUES ($1) ON CONFLICT (id) DO NOTHING`, ws); err != nil {
				t.Fatalf("scopetest: seed workspace %s: %v", ws, err)
			}
		}
		fn(base, Ctx(base, WsA), Ctx(base, WsB), Ctx(base, ""))
		return errRollback
	})
	if !errors.Is(err, errRollback) {
		t.Fatalf("scopetest: scenario did not complete: %v", err)
	}
}

// Committed is a set of committed fixtures removed again when the test ends (and before it starts,
// so an aborted earlier run leaves nothing behind). Rows are matched by id prefix only.
type Committed struct {
	h      *Harness
	prefix string
	tables []string
}

// SeedCommitted registers cleanup for every table it will touch (rows whose id starts with
// idPrefix are deleted before and after the test), then returns the handle to Exec inserts on.
// h must be a replica pool (NewReplica) so inserts need no parent rows.
func SeedCommitted(t testing.TB, h *Harness, idPrefix string, tables ...string) *Committed {
	t.Helper()
	if idPrefix == "" || len(idPrefix) < 4 {
		t.Fatal("scopetest: a distinctive id prefix is required for committed fixtures")
	}
	c := &Committed{h: h, prefix: idPrefix, tables: tables}
	c.cleanup(t)
	t.Cleanup(func() { c.cleanup(t) })
	return c
}

func (c *Committed) cleanup(t testing.TB) {
	// The audited adapters also write audit_trail.audit_entry rows for committed fixtures; remove
	// those by the same id prefix so a run leaves the clone exactly as it found it.
	if _, err := c.h.DB.Exec(`DELETE FROM audit_trail.audit_entry WHERE entity_id LIKE $1`, c.prefix+"%"); err != nil {
		t.Errorf("scopetest: cleanup audit_trail.audit_entry: %v", err)
	}
	for _, tbl := range c.tables {
		if _, err := c.h.DB.Exec(`DELETE FROM `+tbl+` WHERE id LIKE $1`, c.prefix+"%"); err != nil {
			t.Errorf("scopetest: cleanup %s: %v", tbl, err)
		}
	}
}

// Exec runs one fixture statement (committed immediately).
func (c *Committed) Exec(t testing.TB, q string, args ...any) {
	t.Helper()
	if _, err := c.h.DB.Exec(q, args...); err != nil {
		t.Fatalf("scopetest: fixture %q: %v", q, err)
	}
}

// Str returns a pointer to s (optional proto string fields in fixtures).
func Str(s string) *string { return &s }
