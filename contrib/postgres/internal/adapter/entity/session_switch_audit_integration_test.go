//go:build postgresql

package entity

import (
	"context"
	"database/sql"
	"os"
	"testing"

	audittrailenums "github.com/erniealice/esqyma/pkg/schema/v1/domain/audit_trail/enums"
	_ "github.com/lib/pq"
	"uuid"
)

// TestIntegrationWriteSwitchAuditRow inserts for real and rolls back. Before the fix
// the enum names went into smallint columns: strict mode always returned the error
// and best-effort mode swallowed it, so no row was ever written.
func TestIntegrationWriteSwitchAuditRow(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	ctx := context.Background()

	for name, strict := range map[string]bool{"strict": true, "best effort": false} {
		t.Run(name, func(t *testing.T) {
			tx, err := db.BeginTx(ctx, nil)
			if err != nil {
				t.Skipf("database unavailable: %v", err)
			}
			defer func() { _ = tx.Rollback() }()
			entityID := uuid.NewV7().String()
			row := switchAuditRow{
				WorkspaceID:  uuid.NewV7().String(),
				UserID:       "switch-audit-test",
				EntityID:     entityID,
				UseCaseLabel: "switch_principal",
				Reason:       "test",
				RequireAudit: strict,
			}
			if err := writeSwitchAuditRow(ctx, tx, row); err != nil {
				t.Fatalf("write: %v", err)
			}
			var actorType, action int32
			var transactionID int64
			err = tx.QueryRowContext(ctx, `SELECT actor_type, action, transaction_id FROM audit_trail.audit_entry
				WHERE entity_type = 'session' AND entity_id = $1 AND actor_id = 'switch-audit-test'`, entityID).Scan(&actorType, &action, &transactionID)
			if err != nil {
				t.Fatalf("the audit row must exist: %v", err)
			}
			if actorType != int32(audittrailenums.ActorType_ACTOR_TYPE_USER) || action != int32(audittrailenums.AuditAction_AUDIT_ACTION_UPDATE) {
				t.Errorf("actor_type=%d action=%d", actorType, action)
			}
			if transactionID == 0 {
				t.Error("transaction_id must name the writing transaction")
			}
		})
	}
}

func TestRefererWithoutQuery(t *testing.T) {
	for in, want := range map[string]string{
		"":                              "",
		"http://x.test/w/a/list":        "http://x.test/w/a/list",
		"http://x.test/w/a/list?q=name": "http://x.test/w/a/list",
		"http://x.test/w/a/list#frag":   "http://x.test/w/a/list",
		"http://x.test/w/a?q=1#frag":    "http://x.test/w/a",
	} {
		if got := refererWithoutQuery(in); got != want {
			t.Errorf("%q: got %q", in, got)
		}
	}
}
