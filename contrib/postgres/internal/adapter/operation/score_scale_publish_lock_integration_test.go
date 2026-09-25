//go:build postgresql

package operation

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	"github.com/erniealice/espyna-golang/registry/entityid"
	"github.com/erniealice/espyna-golang/shared/database/operations"
	"github.com/erniealice/espyna-golang/shared/identity"
	enumspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/enums"
	scoreScalePb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/score_scale"
	matrixpb "github.com/erniealice/esqyma/pkg/schema/v1/service/operation/outcome_matrix"
)

// fix2-backend (codex-review-impl2 #1 + #4). TEST_DATABASE_URL-gated; run
// against the CLONE (education2clone20260925a), never education2. All writes
// are inside transactions that are rolled back.

// TestIntegrationScoreScaleDeleteGuardAndPublishVerify: scale deletion
// (generic Delete = active=false) is BAND_LOCKED while any band is referenced
// by a PUBLISHED set; an unreferenced scale deletes; the publish lock rejects a
// foreign-workspace scale; publish verify rejects an entry whose band belongs
// to another scale or another workspace.
func TestIntegrationScoreScaleDeleteGuardAndPublishVerify(t *testing.T) {
	db := openScoreScaleBandLockDB(t)
	ctx := context.Background()
	tm := postgresCore.NewPostgreSQLTransactionManager(db)
	txn, err := tm.StartTransaction(ctx)
	if err != nil {
		t.Fatalf("start ambient transaction: %v", err)
	}
	t.Cleanup(func() { _ = txn.Rollback(context.Background()) })
	txCtx := operations.WithTransaction(ctx, txn)
	tx := txn.(interface{ GetTx() *sql.Tx }).GetTx()

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	id := func(name string) string { return "f2b-it-" + name + "-" + suffix }
	wsID, foreignWs := id("ws"), id("ws-foreign")
	scaleLocked, scaleFree, scaleOther, scaleForeign := id("scale-locked"), id("scale-free"), id("scale-other"), id("scale-foreign")
	bandLocked, bandOther, bandForeign := id("band-locked"), id("band-other"), id("band-foreign")
	criteriaID := id("criteria")
	setPublished, setDraftWrongScale, setDraftForeignBand, setForeignScale := id("set-pub"), id("set-draft-wrong"), id("set-draft-foreign-band"), id("set-foreign-scale")

	exec := func(q string, args ...any) {
		t.Helper()
		if _, err := tx.ExecContext(txCtx, q, args...); err != nil {
			t.Fatalf("fixture %q: %v", q, err)
		}
	}
	exec(`INSERT INTO workspace (id, name, active) VALUES ($1, 'f2b it', true), ($2, 'f2b it foreign', true)`, wsID, foreignWs)
	for _, s := range []struct{ id, ws string }{{scaleLocked, wsID}, {scaleFree, wsID}, {scaleOther, wsID}, {scaleForeign, foreignWs}} {
		exec(`INSERT INTO `+entityid.ScoreScale+`
			(id, scale_group_id, version, version_status, name, scale_kind, input_unit, output_unit, workspace_id, active, created_by)
			VALUES ($1, $1, 1, 'VERSION_STATUS_PUBLISHED', 'f2b scale', 'SCALE_KIND_EXACT_MAP', 'score', 'label', $2, true, 'f2b-it')`, s.id, s.ws)
	}
	for _, b := range []struct{ id, ws, scale string }{{bandLocked, wsID, scaleLocked}, {bandOther, wsID, scaleOther}, {bandForeign, foreignWs, scaleLocked}} {
		exec(`INSERT INTO `+entityid.ScoreScaleBand+`
			(id, workspace_id, active, score_scale_id, sequence_order, input_match, output_value, output_label)
			VALUES ($1, $2, true, $3, 1, $1, 1, 'L')`, b.id, b.ws, b.scale)
	}
	exec(`INSERT INTO `+entityid.OutcomeCriteria+`
		(id, name, criteria_type, min_score, max_score, score_increment, required, active, workspace_id)
		VALUES ($1, 'f2b criterion', 'CRITERIA_TYPE_NUMERIC_SCORE', 0, 8, 1, true, true, $2)`, criteriaID, wsID)
	for _, s := range []struct{ id, status, scale string }{
		{setPublished, "VERSION_STATUS_PUBLISHED", scaleLocked},
		{setDraftWrongScale, "VERSION_STATUS_DRAFT", scaleLocked},
		{setDraftForeignBand, "VERSION_STATUS_DRAFT", scaleLocked},
		{setForeignScale, "VERSION_STATUS_DRAFT", scaleForeign},
	} {
		exec(`INSERT INTO `+ratingDescriptionSetTable+`
			(id, workspace_id, version, version_status, name, score_scale_id, active)
			VALUES ($1, $2, 1, $3, 'f2b set', $4, true)`, s.id, wsID, s.status, s.scale)
	}
	for _, e := range []struct{ set, band string }{{setPublished, bandLocked}, {setDraftWrongScale, bandOther}, {setDraftForeignBand, bandForeign}} {
		exec(`INSERT INTO `+ratingDescriptionSetEntryTable+`
			(id, workspace_id, rating_description_set_id, outcome_criteria_id, score_scale_band_id, description, active)
			VALUES ($1, $2, $3, $4, $5, 'text', true)`, id("entry-"+e.set), wsID, e.set, criteriaID, e.band)
	}

	dbOps := postgresCore.NewWorkspaceAwareOperations(db)
	scaleRepo := NewPostgresScoreScaleRepository(dbOps, "").(*PostgresScoreScaleRepository)
	setRepo := NewPostgresRatingDescriptionSetRepository(dbOps, "").(*PostgresRatingDescriptionSetRepository)
	idnCtx := identity.WithRequestIdentity(txCtx, &identity.RequestIdentity{WorkspaceID: wsID})

	// #1 scale deletion guard.
	if _, err := scaleRepo.DeleteScoreScale(idnCtx, &scoreScalePb.DeleteScoreScaleRequest{Data: &scoreScalePb.ScoreScale{Id: scaleLocked}}); err == nil || !strings.Contains(err.Error(), "BAND_LOCKED") {
		t.Fatalf("delete of scale with a published-referenced band: want BAND_LOCKED, got %v", err)
	}
	var active bool
	if err := tx.QueryRowContext(txCtx, `SELECT active FROM `+entityid.ScoreScale+` WHERE id = $1`, scaleLocked).Scan(&active); err != nil || !active {
		t.Fatalf("locked scale must stay active (active=%v err=%v)", active, err)
	}
	if _, err := scaleRepo.DeleteScoreScale(idnCtx, &scoreScalePb.DeleteScoreScaleRequest{Data: &scoreScalePb.ScoreScale{Id: scaleFree}}); err != nil {
		t.Fatalf("delete of unreferenced scale should succeed, got %v", err)
	}
	// Deactivation payload (active=false) is treated as a locked-field write.
	if !touchesLockedScoreScaleFields(map[string]any{"active": false}) || touchesLockedScoreScaleFields(map[string]any{"active": true, "name": "x"}) {
		t.Fatal("touchesLockedScoreScaleFields: active=false must trigger the guard, active=true must not")
	}

	// Publish lock: own scale locks; foreign-workspace scale → INVALID_CONFIG.
	if got, err := setRepo.LockScoreScaleForPublish(idnCtx, setDraftWrongScale); err != nil || got != scaleLocked {
		t.Fatalf("publish lock own scale: got %q err %v", got, err)
	}
	if _, err := setRepo.LockScoreScaleForPublish(idnCtx, setForeignScale); err == nil || !strings.Contains(err.Error(), "INVALID_CONFIG") {
		t.Fatalf("publish lock foreign scale: want INVALID_CONFIG, got %v", err)
	}
	// Publish verify: entry band from another scale / another workspace → INVALID_CONFIG.
	if err := setRepo.VerifyRatingDescriptionSetPublishScale(idnCtx, setDraftWrongScale, scaleLocked); err == nil || !strings.Contains(err.Error(), "INVALID_CONFIG") {
		t.Fatalf("verify wrong-scale band: want INVALID_CONFIG, got %v", err)
	}
	if err := setRepo.VerifyRatingDescriptionSetPublishScale(idnCtx, setDraftForeignBand, scaleLocked); err == nil || !strings.Contains(err.Error(), "INVALID_CONFIG") {
		t.Fatalf("verify foreign-workspace band: want INVALID_CONFIG, got %v", err)
	}
	// Scale changed between lock and verify → CONFLICT.
	if err := setRepo.VerifyRatingDescriptionSetPublishScale(idnCtx, setPublished, scaleOther); err == nil || !strings.Contains(err.Error(), "CONFLICT") {
		t.Fatalf("verify with a stale scale id: want CONFLICT, got %v", err)
	}
	if err := setRepo.VerifyRatingDescriptionSetPublishScale(idnCtx, setPublished, scaleLocked); err != nil {
		t.Fatalf("verify consistent set: %v", err)
	}

	// #4 set create/update scale ownership validator.
	if err := setRepo.ValidateRatingDescriptionSetScoreScale(idnCtx, scaleForeign); err == nil || !strings.Contains(err.Error(), "INVALID_CONFIG") {
		t.Fatalf("validate foreign scale: want INVALID_CONFIG, got %v", err)
	}
	if err := setRepo.ValidateRatingDescriptionSetScoreScale(idnCtx, scaleLocked); err != nil {
		t.Fatalf("validate own scale: %v", err)
	}
	// Global (workspace_id IS NULL) scale is accepted (outcome_matrix_query.go convention).
	exec(`UPDATE `+entityid.ScoreScale+` SET workspace_id = NULL WHERE id = $1`, scaleOther)
	if err := setRepo.ValidateRatingDescriptionSetScoreScale(idnCtx, scaleOther); err != nil {
		t.Fatalf("validate global scale: %v", err)
	}
	exec(`UPDATE `+entityid.ScoreScale+` SET workspace_id = $2 WHERE id = $1`, scaleOther, wsID)

	// #4 entry write guard: a band owned by another workspace → INVALID_CONFIG.
	entryRepo := NewPostgresRatingDescriptionSetEntryRepository(dbOps, "").(*PostgresRatingDescriptionSetEntryRepository)
	if err := entryRepo.guardRatingDescriptionSetEntryWrite(idnCtx, setDraftWrongScale, bandForeign); err == nil || !strings.Contains(err.Error(), "INVALID_CONFIG") {
		t.Fatalf("entry guard foreign band: want INVALID_CONFIG, got %v", err)
	}
	if err := entryRepo.guardRatingDescriptionSetEntryWrite(idnCtx, setDraftWrongScale, bandLocked); err != nil {
		t.Fatalf("entry guard own band: %v", err)
	}

	// #4 resolver entries: an entry whose band is foreign-owned → INVALID_CONFIG
	// (not silently RESOLVED with fewer descriptions); the clean key resolves.
	q := &PostgresOutcomeMatrixQuery{db: db}
	states := []*cellResolutionState{
		{cell: &matrixpb.CellRatingRef{OutcomeCriteriaId: criteriaID}, setID: setDraftForeignBand},
		{cell: &matrixpb.CellRatingRef{OutcomeCriteriaId: criteriaID}, setID: setPublished},
	}
	if err := q.resolveRatingEntries(txCtx, tx, states, wsID); err != nil {
		t.Fatalf("resolveRatingEntries: %v", err)
	}
	if states[0].status != enumspb.RatingDescriptionResolutionStatus_RATING_DESCRIPTION_RESOLUTION_STATUS_INVALID_CONFIG {
		t.Fatalf("foreign-band entry: want INVALID_CONFIG, got %v", states[0].status)
	}
	if states[1].status != enumspb.RatingDescriptionResolutionStatus_RATING_DESCRIPTION_RESOLUTION_STATUS_RESOLVED || len(states[1].descriptionsOut) != 1 {
		t.Fatalf("own entry: want RESOLVED with 1 description, got %v (%d)", states[1].status, len(states[1].descriptionsOut))
	}
}

var errF2BRollback = errors.New("f2b: intentional rollback")

// TestIntegrationDescriptorLockInterleaving proves the shared lock order
// (score_scale → score_scale_band → rating_description_set) serializes publish
// against band/scale meaning edits, in both directions, with two concurrent
// transactions and lock_timeout. Uses existing clone rows read-only: the
// transactions only take row locks and always roll back.
func TestIntegrationDescriptorLockInterleaving(t *testing.T) {
	db := openScoreScaleBandLockDB(t)
	var setID, wsID, scaleID, bandID string
	if err := db.QueryRow(`
		SELECT s.id, s.workspace_id, s.score_scale_id, b.id
		FROM `+ratingDescriptionSetTable+` s
		JOIN `+entityid.ScoreScale+` ss ON ss.id = s.score_scale_id
		JOIN `+entityid.ScoreScaleBand+` b ON b.score_scale_id = ss.id
		WHERE s.active ORDER BY s.id, b.id LIMIT 1`).Scan(&setID, &wsID, &scaleID, &bandID); err != nil {
		t.Skipf("no rating_description_set with scale bands to exercise: %v", err)
	}
	tm := postgresCore.NewPostgreSQLTransactionManager(db)
	dbOps := postgresCore.NewWorkspaceAwareOperations(db)
	setRepo := NewPostgresRatingDescriptionSetRepository(dbOps, "").(*PostgresRatingDescriptionSetRepository)
	bandRepo := NewPostgresScoreScaleBandRepository(dbOps, "").(*PostgresScoreScaleBandRepository)
	scaleRepo := NewPostgresScoreScaleRepository(dbOps, "").(*PostgresScoreScaleRepository)
	withWs := func(c context.Context) context.Context {
		return identity.WithRequestIdentity(c, &identity.RequestIdentity{UserID: "f2b", WorkspaceID: wsID})
	}
	setLockTimeout := func(c context.Context) error {
		_, err := setRepo.executor(c).ExecContext(c, `SET LOCAL lock_timeout = '750ms'`)
		return err
	}
	// runB runs fn in a second transaction while the first still holds its locks.
	runB := func(fn func(ctx context.Context) error) error {
		done := make(chan error, 1)
		go func() {
			done <- tm.RunInTransaction(context.Background(), func(txB context.Context) error {
				ctx := withWs(txB)
				if err := setLockTimeout(ctx); err != nil {
					return err
				}
				if err := fn(ctx); err != nil {
					return err
				}
				return errF2BRollback
			})
		}()
		select {
		case err := <-done:
			return err
		case <-time.After(10 * time.Second):
			return fmt.Errorf("B neither timed out nor returned")
		}
	}
	isLockTimeout := func(err error) bool {
		return err != nil && strings.Contains(err.Error(), "lock timeout")
	}

	t.Run("publish holds scale FOR SHARE → band edit blocks", func(t *testing.T) {
		err := tm.RunInTransaction(context.Background(), func(txA context.Context) error {
			ctx := withWs(txA)
			if _, err := setRepo.LockScoreScaleForPublish(ctx, setID); err != nil {
				return fmt.Errorf("A publish lock: %w", err)
			}
			if _, err := setRepo.LockRatingDescriptionSetForUpdate(ctx, setID); err != nil {
				return fmt.Errorf("A set lock: %w", err)
			}
			if errB := runB(func(ctx context.Context) error { return bandRepo.guardScoreScaleBandLockedWrite(ctx, bandID) }); !isLockTimeout(errB) {
				return fmt.Errorf("B band guard must block on A's scale FOR SHARE (lock timeout), got %v", errB)
			}
			if errB := runB(func(ctx context.Context) error { return scaleRepo.guardScoreScaleLockedWrite(ctx, scaleID) }); !isLockTimeout(errB) {
				return fmt.Errorf("B scale guard must block on A's scale FOR SHARE (lock timeout), got %v", errB)
			}
			return errF2BRollback
		})
		if !errors.Is(err, errF2BRollback) {
			t.Fatal(err)
		}
	})

	t.Run("band edit holds scale FOR UPDATE → publish blocks", func(t *testing.T) {
		err := tm.RunInTransaction(context.Background(), func(txA context.Context) error {
			ctx := withWs(txA)
			// The guard may return BAND_LOCKED (the set is published) — the
			// scale and band row locks are held either way until A ends.
			if err := bandRepo.guardScoreScaleBandLockedWrite(ctx, bandID); err != nil && !strings.Contains(err.Error(), "BAND_LOCKED") {
				return fmt.Errorf("A band guard: %w", err)
			}
			if errB := runB(func(ctx context.Context) error {
				_, err := setRepo.LockScoreScaleForPublish(ctx, setID)
				return err
			}); !isLockTimeout(errB) {
				return fmt.Errorf("B publish lock must block on A's scale FOR UPDATE (lock timeout), got %v", errB)
			}
			return errF2BRollback
		})
		if !errors.Is(err, errF2BRollback) {
			t.Fatal(err)
		}
	})

	t.Run("two publishes of one scale do not block each other", func(t *testing.T) {
		err := tm.RunInTransaction(context.Background(), func(txA context.Context) error {
			ctx := withWs(txA)
			if _, err := setRepo.LockScoreScaleForPublish(ctx, setID); err != nil {
				return err
			}
			if errB := runB(func(ctx context.Context) error {
				_, err := setRepo.LockScoreScaleForPublish(ctx, setID)
				return err
			}); !errors.Is(errB, errF2BRollback) {
				return fmt.Errorf("B shared publish lock must be granted, got %v", errB)
			}
			return errF2BRollback
		})
		if !errors.Is(err, errF2BRollback) {
			t.Fatal(err)
		}
	})
}
