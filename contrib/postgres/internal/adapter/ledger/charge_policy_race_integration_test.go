//go:build postgresql

package ledger

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	"github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core/scopetest"
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/shared/usecaseerr"
	"github.com/erniealice/espyna-golang/internal/application/usecases/domain/ledger/charge_policy"
	"github.com/erniealice/espyna-golang/shared/identity"
	versionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/charge_policy_version"
)

// C16 (build-spec §7): two-connection real-database proofs of the charge policy locking claims
// and of the C12 separation-of-duties editor rows. Connections cannot see each other's uncommitted
// rows, so the seed rows are COMMITTED under dedicated workspace ids and removed again afterwards
// (also on failure). The test needs a superuser (FK triggers are disabled only on the seed session)
// on a disposable usage clone (TEST_DATABASE_URL); an unset/unreachable database FAILS the test.

// cpRaceAuthz allows every permission except approve_own (the SoD scenario needs it withheld).
type cpRaceAuthz struct{}

func (cpRaceAuthz) IsEnabled() bool { return true }
func (cpRaceAuthz) HasPermission(_ context.Context, _ string, permission string) (bool, error) {
	return permission != "charge_policy:approve_own", nil
}

type cpRaceEnv struct {
	t          *testing.T
	wsA, wsB   string
	db         *sql.DB
	seed       *sql.Conn
	uc         *charge_policy.UseCases
	versions   *PostgresChargePolicyVersionRepository
	tm         dbTransactor
	polID, ver string
}

func (e *cpRaceEnv) exec(q string, args ...any) {
	e.t.Helper()
	if _, err := e.seed.ExecContext(context.Background(), q, args...); err != nil {
		e.t.Fatalf("charge policy race seed/cleanup: %v\n%s", err, q)
	}
}

func (e *cpRaceEnv) cleanup() {
	for _, ws := range []string{e.wsA, e.wsB} {
		for _, q := range []string{
			`DELETE FROM charge_policy_version_editor WHERE workspace_id = $1`,
			`DELETE FROM charge_policy_posting WHERE workspace_id = $1`,
			`DELETE FROM charge_policy_component WHERE workspace_id = $1`,
			`DELETE FROM charge_policy_version WHERE workspace_id = $1`,
			`DELETE FROM charge_policy WHERE workspace_id = $1`,
			`DELETE FROM workspace WHERE id = $1`,
		} {
			_, _ = e.seed.ExecContext(context.Background(), q, ws)
		}
	}
}

func (e *cpRaceEnv) ctxFor(ws, user string) context.Context {
	ctx := identity.WithRequestIdentity(context.Background(), &identity.RequestIdentity{WorkspaceID: ws})
	return contextutil.WithWorkspaceID(contextutil.WithUserID(ctx, user), ws)
}

func newCPRaceEnv(t *testing.T) *cpRaceEnv {
	t.Helper()
	// scopetest.New: DSN from TEST_DATABASE_URL (must name the clone; unset/unreachable FAILS), and
	// every listed table must exist.
	db := scopetest.New(t, "charge_policy", "charge_policy_version", "charge_policy_version_editor").DB
	seed, err := db.Conn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { seed.Close() })
	if _, err := seed.ExecContext(context.Background(), "SET session_replication_role = replica"); err != nil {
		t.Fatalf("cannot disable FK triggers on the seed session (the test role needs superuser): %v", err)
	}
	e := &cpRaceEnv{t: t, wsA: "cp-race-f2-a", wsB: "cp-race-f2-b", db: db, seed: seed, polID: "cp-race-f2-pol", ver: "cp-race-f2-ver"}
	e.cleanup() // a previous aborted run
	t.Cleanup(e.cleanup)
	e.exec(`INSERT INTO workspace (id) VALUES ($1), ($2)`, e.wsA, e.wsB)
	e.exec(`INSERT INTO charge_policy (id, workspace_id, code, name, status, active) VALUES ($2, $1, 'CP_RACE_F2', 'race', 'CHARGE_POLICY_STATUS_ACTIVE', true)`, e.wsA, e.polID)
	e.exec(`INSERT INTO charge_policy_version (id, workspace_id, charge_policy_id, version_number, status, prepared_by, active) VALUES ($2, $1, $3, 1, 'CHARGE_POLICY_VERSION_STATUS_DRAFT', 'prep', true)`, e.wsA, e.ver, e.polID)

	ops := postgresCore.NewWorkspaceAwareOperations(db) // unaudited: audit partitions are not under test
	e.tm = dbTransactor{postgresCore.NewPostgreSQLTransactionManager(db)}
	e.versions = NewPostgresChargePolicyVersionRepository(ops, "charge_policy_version").(*PostgresChargePolicyVersionRepository)
	e.uc = charge_policy.NewUseCases(charge_policy.Repositories{
		ChargePolicy:              NewPostgresChargePolicyRepository(ops, "charge_policy"),
		ChargePolicyVersion:       e.versions,
		ChargePolicyComponent:     NewPostgresChargePolicyComponentRepository(ops, "charge_policy_component"),
		ChargePolicyPosting:       NewPostgresChargePolicyPostingRepository(ops, "charge_policy_posting"),
		ChargePolicyVersionEditor: NewPostgresChargePolicyVersionEditorRepository(ops, "charge_policy_version_editor"),
	}, charge_policy.Services{
		Authorizer:       ports.NewNoOpAuthorizer(),
		Transactor:       e.tm,
		Translator:       ports.NewNoOpTranslator(),
		IDGenerator:      &raceIDs{prefix: "cp-race-f2"},
		ActionGatekeeper: actiongate.NewActionGatekeeper(cpRaceAuthz{}, ports.NewNoOpTranslator()),
	})
	return e
}

// holdVersionLock takes the version row lock in its own transaction and holds it until release().
func (e *cpRaceEnv) holdVersionLock(ws string) (release func(), done <-chan error) {
	locked := make(chan struct{})
	rel := make(chan struct{})
	res := make(chan error, 1)
	go func() {
		res <- e.tm.ExecuteInTransaction(e.ctxFor(ws, "holder"), func(txCtx context.Context) error {
			if _, err := e.versions.LockChargePolicyVersionForUpdate(txCtx, e.ver); err != nil {
				close(locked)
				return err
			}
			close(locked)
			<-rel
			return nil
		})
	}()
	<-locked
	return func() { close(rel) }, res
}

// The version row lock serialises a draft edit behind a concurrent holder (approval / edit), and
// the lock statement carries the workspace predicate: a foreign workspace neither blocks nor locks.
func TestChargePolicyVersionLockSerialisesAcrossConnections(t *testing.T) {
	e := newCPRaceEnv(t)

	t.Run("a concurrent draft edit waits for the holder", func(t *testing.T) {
		release, held := e.holdVersionLock(e.wsA)
		scope := "edited under contention"
		edited := make(chan error, 1)
		go func() {
			_, err := e.uc.UpdateChargePolicyVersion.Execute(e.ctxFor(e.wsA, "editor2"), &versionpb.UpdateChargePolicyVersionRequest{Data: &versionpb.ChargePolicyVersion{Id: e.ver, AssessmentScope: &scope}})
			edited <- err
		}()
		select {
		case err := <-edited:
			t.Fatalf("the edit must wait for the row lock, returned early: %v", err)
		case <-time.After(500 * time.Millisecond):
		}
		release()
		if err := <-held; err != nil {
			t.Fatalf("holder: %v", err)
		}
		select {
		case err := <-edited:
			if err != nil {
				t.Fatalf("edit after release: %v", err)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("the edit never resumed after the lock was released")
		}
	})

	t.Run("a foreign workspace is reported not found without waiting (C5)", func(t *testing.T) {
		release, held := e.holdVersionLock(e.wsA)
		defer func() { release(); <-held }()
		got := make(chan error, 1)
		go func() {
			got <- e.tm.ExecuteInTransaction(e.ctxFor(e.wsB, "intruder"), func(txCtx context.Context) error {
				_, err := e.versions.LockChargePolicyVersionForUpdate(txCtx, e.ver)
				return err
			})
		}()
		select {
		case err := <-got:
			if err == nil || !strings.Contains(err.Error(), "not found") {
				t.Fatalf("foreign lock must report not found, got %v", err)
			}
		case <-time.After(1500 * time.Millisecond):
			t.Fatal("a foreign-workspace lock blocked behind another tenant's row lock (missing workspace predicate)")
		}
	})
}

// C12 against the real tables: every editor is recorded once (UNIQUE(version,user)), and an editor
// approving without approve_own is refused before anything is approved.
func TestChargePolicyEditorsAreSelfForApprovalRealDB(t *testing.T) {
	e := newCPRaceEnv(t)
	scope := "first"
	for i := 0; i < 2; i++ { // the second edit by the same user must not violate the unique constraint
		if _, err := e.uc.UpdateChargePolicyVersion.Execute(e.ctxFor(e.wsA, "editor2"), &versionpb.UpdateChargePolicyVersionRequest{Data: &versionpb.ChargePolicyVersion{Id: e.ver, AssessmentScope: &scope}}); err != nil {
			t.Fatalf("edit %d: %v", i, err)
		}
	}
	var n int
	if err := e.db.QueryRow(`SELECT count(*) FROM charge_policy_version_editor WHERE charge_policy_version_id = $1 AND user_id = 'editor2'`, e.ver).Scan(&n); err != nil || n != 1 {
		t.Fatalf("editor rows for editor2 = %d (%v), want exactly 1", n, err)
	}
	_, err := e.uc.ApproveChargePolicyVersion.Execute(e.ctxFor(e.wsA, "editor2"), &versionpb.ApproveChargePolicyVersionRequest{ChargePolicyVersionId: e.ver})
	if !usecaseerr.IsCode(err, charge_policy.CodeSelfApproval) {
		t.Fatalf("an editor approving without approve_own must be refused as self_approval, got %v", err)
	}
	var status string
	if err := e.db.QueryRow(`SELECT status FROM charge_policy_version WHERE id = $1`, e.ver).Scan(&status); err != nil || status != "CHARGE_POLICY_VERSION_STATUS_DRAFT" {
		t.Fatalf("version status = %q (%v), must stay DRAFT", status, err)
	}
	// The foreign workspace cannot see the editors (tenant scope of the SoD lookup).
	if _, err := e.uc.ApproveChargePolicyVersion.Execute(e.ctxFor(e.wsB, "boss"), &versionpb.ApproveChargePolicyVersionRequest{ChargePolicyVersionId: e.ver}); err == nil {
		t.Fatal("a foreign workspace must not approve another tenant's version")
	}
}
