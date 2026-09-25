//go:build postgresql

package operation

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	_ "github.com/lib/pq"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	"github.com/erniealice/espyna-golang/registry/entityid"
	"github.com/erniealice/espyna-golang/shared/database/operations"
	"github.com/erniealice/espyna-golang/shared/identity"
	enumspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/enums"
	scoreScalePb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/score_scale"
	scoreScaleBandPb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/score_scale_band"
)

// openScoreScaleBandLockDB is TEST_DATABASE_URL-gated, same convention as
// openResolveCellRatingDescriptionsDB (outcome_matrix_rating_resolution_integration_test.go).
func openScoreScaleBandLockDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Skipf("TEST_DATABASE_URL unusable: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		t.Skipf("TEST_DATABASE_URL unreachable: %v", err)
	}
	var table sql.NullString
	if err := db.QueryRowContext(ctx,
		"SELECT to_regclass('public.rating_description_set_entry')::text").Scan(&table); err != nil || !table.Valid {
		db.Close()
		t.Skip("rating_description_set_entry not present; run against the expanded schema (PD 20260925 migration)")
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// strp is a small *string helper so this file doesn't need to import
// google.golang.org/protobuf/proto just for proto.String.
func strp(s string) *string { return &s }

// TestIntegrationScoreScaleBandLock is a rollback-only fixture exercising
// codex-review-impl1.out.md item 2 / schema-proposal.md §9.3 "Immutable
// meaning": once a band is referenced by a rating_description_set_entry of a
// PUBLISHED (or DEPRECATED) set, UpdateScoreScaleBand/DeleteScoreScaleBand
// (and UpdateScoreScale for scale_kind/input_min/input_max) must reject
// changes to the band's/scale's locked fields with BAND_LOCKED, while
// label/sequence edits and writes to an UNREFERENCED band/scale stay allowed.
//
// Everything in this test — fixture rows AND the guarded repository calls —
// runs on ONE ambient transaction (postgresCore.NewPostgreSQLTransactionManager
// + operations.WithTransaction), rolled back at the end: the guard requires an
// ambient *sql.Tx (fail closed otherwise, mirroring
// PostgresRatingDescriptionSetEntryRepository's write guard), so a separate
// fixture-only tx would be invisible to it under READ COMMITTED.
func TestIntegrationScoreScaleBandLock(t *testing.T) {
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
	id := func(name string) string { return "ssbl-it-" + name + "-" + suffix }

	wsID := id("ws")
	scaleGroupID := id("scale-group")
	scaleID := id("scale")
	bandLockedID := id("band-locked")
	bandFreeID := id("band-free")
	criteriaID := id("criteria")
	setPublishedID := id("set-published")

	exec := func(query string, args ...any) {
		t.Helper()
		if _, err := tx.ExecContext(txCtx, query, args...); err != nil {
			t.Fatalf("fixture statement %q: %v", query, err)
		}
	}

	exec(`INSERT INTO workspace (id, name, active) VALUES ($1, 'ssbl resolver it', true)`, wsID)
	exec(`INSERT INTO `+entityid.ScoreScale+`
		(id, scale_group_id, version, version_status, name, scale_kind, input_unit, output_unit, workspace_id, active, created_by)
		VALUES ($1, $2, 1, 'VERSION_STATUS_PUBLISHED', 'ssbl scale', 'SCALE_KIND_EXACT_MAP', 'score', 'label', $3, true, 'ssbl-it')`,
		scaleID, scaleGroupID, wsID)
	exec(`INSERT INTO `+entityid.ScoreScaleBand+`
		(id, workspace_id, active, score_scale_id, sequence_order, input_match, output_value, output_label)
		VALUES ($1, $2, true, $3, 1, 'locked-band', 1, 'Locked Band')`,
		bandLockedID, wsID, scaleID)
	exec(`INSERT INTO `+entityid.ScoreScaleBand+`
		(id, workspace_id, active, score_scale_id, sequence_order, input_match, output_value, output_label)
		VALUES ($1, $2, true, $3, 2, 'free-band', 2, 'Free Band')`,
		bandFreeID, wsID, scaleID)
	exec(`INSERT INTO `+entityid.OutcomeCriteria+`
		(id, name, criteria_type, min_score, max_score, score_increment, required, active, workspace_id)
		VALUES ($1, 'ssbl criterion', 'CRITERIA_TYPE_NUMERIC_SCORE', 0, 8, 1, true, true, $2)`,
		criteriaID, wsID)
	exec(`INSERT INTO `+ratingDescriptionSetTable+`
		(id, workspace_id, version, version_status, name, score_scale_id, active)
		VALUES ($1, $2, 1, 'VERSION_STATUS_PUBLISHED', 'ssbl set', $3, true)`,
		setPublishedID, wsID, scaleID)
	exec(`INSERT INTO `+ratingDescriptionSetEntryTable+`
		(id, workspace_id, rating_description_set_id, outcome_criteria_id, score_scale_band_id, description, active)
		VALUES ($1, $2, $3, $4, $5, 'Locked band text', true)`,
		id("entry"), wsID, setPublishedID, criteriaID, bandLockedID)

	dbOps := postgresCore.NewWorkspaceAwareOperations(db)
	bandRepo := NewPostgresScoreScaleBandRepository(dbOps, "").(*PostgresScoreScaleBandRepository)
	scaleRepo := NewPostgresScoreScaleRepository(dbOps, "").(*PostgresScoreScaleRepository)
	idnCtx := identity.WithRequestIdentity(txCtx, &identity.RequestIdentity{WorkspaceID: wsID})

	// 1. Locked field (input_match) on the REFERENCED band -> BAND_LOCKED, rejected.
	_, err = bandRepo.UpdateScoreScaleBand(idnCtx, &scoreScaleBandPb.UpdateScoreScaleBandRequest{
		Data: &scoreScaleBandPb.ScoreScaleBand{Id: bandLockedID, InputMatch: strp("changed-locked-value")},
	})
	if err == nil || !strings.Contains(err.Error(), "BAND_LOCKED") {
		t.Fatalf("locked-field update on referenced band: want BAND_LOCKED, got %v", err)
	}

	// 2. Label/sequence edit on the SAME referenced band -> allowed (schema-proposal.md §9.3
	// "Label/sequence edits stay allowed").
	if _, err := bandRepo.UpdateScoreScaleBand(idnCtx, &scoreScaleBandPb.UpdateScoreScaleBandRequest{
		Data: &scoreScaleBandPb.ScoreScaleBand{Id: bandLockedID, OutputLabel: "Renamed Locked Band"},
	}); err != nil {
		t.Fatalf("label edit on referenced band should be allowed, got: %v", err)
	}

	// 3. Delete the referenced band -> BAND_LOCKED, rejected.
	if _, err := bandRepo.DeleteScoreScaleBand(idnCtx, &scoreScaleBandPb.DeleteScoreScaleBandRequest{
		Data: &scoreScaleBandPb.ScoreScaleBand{Id: bandLockedID},
	}); err == nil || !strings.Contains(err.Error(), "BAND_LOCKED") {
		t.Fatalf("delete of referenced band: want BAND_LOCKED, got %v", err)
	}

	// 4. Locked field on the UNREFERENCED band -> allowed.
	if _, err := bandRepo.UpdateScoreScaleBand(idnCtx, &scoreScaleBandPb.UpdateScoreScaleBandRequest{
		Data: &scoreScaleBandPb.ScoreScaleBand{Id: bandFreeID, InputMatch: strp("changed-free-value")},
	}); err != nil {
		t.Fatalf("locked-field update on unreferenced band should be allowed, got: %v", err)
	}

	// 5. Delete the UNREFERENCED band -> allowed.
	if _, err := bandRepo.DeleteScoreScaleBand(idnCtx, &scoreScaleBandPb.DeleteScoreScaleBandRequest{
		Data: &scoreScaleBandPb.ScoreScaleBand{Id: bandFreeID},
	}); err != nil {
		t.Fatalf("delete of unreferenced band should be allowed, got: %v", err)
	}

	// 6. Scale-level guard: scale_kind change while the scale still has a
	// referenced band (bandLockedID; step 3's delete was correctly rejected,
	// so it still exists) -> BAND_LOCKED.
	if _, err := scaleRepo.UpdateScoreScale(idnCtx, &scoreScalePb.UpdateScoreScaleRequest{
		Data: &scoreScalePb.ScoreScale{Id: scaleID, ScaleKind: enumspb.ScaleKind_SCALE_KIND_RANGE_MAP},
	}); err == nil || !strings.Contains(err.Error(), "BAND_LOCKED") {
		t.Fatalf("scale_kind update while a band is locked: want BAND_LOCKED, got %v", err)
	}

	// 7. A non-locked scale field (name) -> allowed even though the scale has
	// a locked band.
	if _, err := scaleRepo.UpdateScoreScale(idnCtx, &scoreScalePb.UpdateScoreScaleRequest{
		Data: &scoreScalePb.ScoreScale{Id: scaleID, Name: "ssbl scale renamed"},
	}); err != nil {
		t.Fatalf("non-locked scale field update should be allowed, got: %v", err)
	}
}
