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
	"github.com/erniealice/espyna-golang/shared/database/sqlexec"
	"github.com/erniealice/espyna-golang/shared/identity"
	enumspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/enums"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/rating_description_set"
	entrypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/rating_description_set_entry"
	linkpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/rating_description_set_product_plan"
)

// Integration coverage for the W3 follow-up findings (b) and (d)
// (2026-09-25, codex-review-impl1.out.md), plus adapter-level proof of (c)'s
// UpdateRatingDescriptionSetIfDraft never writing lifecycle fields. Uses the
// same TEST_DATABASE_URL / tm.RunInTransaction(...) + intentional-rollback
// harness as subscription_group_document_template_delete_pair_integration_test.go
// — every fixture insert AND the operation under test happen inside ONE
// transaction that is always rolled back, so nothing is ever committed
// (no residue, no cleanup helper needed).

func openW3LifecycleIntegrationDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		t.Skipf("TEST_DATABASE_URL unreachable: %v", err)
	}
	var ok string
	if err := db.QueryRowContext(ctx, "SELECT to_regclass('rating_description_set')::text").Scan(&ok); err != nil || ok == "" {
		db.Close()
		t.Skip("rating_description_set not present")
	}
	t.Cleanup(func() { db.Close() })
	return db
}

const w3lcRollbackMsg = "w3-lifecycle-int: intentional rollback"

// seedW3LifecycleScoreScale seeds one workspace-scoped score_scale +
// score_scale_band pair (minimal NOT NULL columns only).
func seedW3LifecycleScoreScale(t *testing.T, ex sqlexec.DBExecutor, ctx context.Context, scaleID, workspaceID string) {
	t.Helper()
	if _, err := ex.ExecContext(ctx, `INSERT INTO score_scale
		(id, scale_group_id, version, version_status, name, scale_kind, input_unit, output_unit, created_by, workspace_id)
		VALUES ($1, $1, 1, 'VERSION_STATUS_PUBLISHED', 'W3LC Scale', 'SCALE_KIND_NUMERIC', 'points', 'points', 'w3lc-seed', $2)`,
		scaleID, workspaceID); err != nil {
		t.Fatalf("seed score_scale: %v", err)
	}
}

func seedW3LifecycleWorkspace(t *testing.T, ex sqlexec.DBExecutor, ctx context.Context, id string) {
	t.Helper()
	if _, err := ex.ExecContext(ctx, `INSERT INTO workspace (id) VALUES ($1) ON CONFLICT (id) DO NOTHING`, id); err != nil {
		t.Fatalf("seed workspace %s: %v", id, err)
	}
}

// seedW3LifecycleSet seeds one rating_description_set row directly (bypassing
// the adapter's own Create, matching the established fixture-seeding
// convention in this package).
func seedW3LifecycleSet(t *testing.T, ex sqlexec.DBExecutor, ctx context.Context, id, workspaceID, scaleID, status string) {
	t.Helper()
	if _, err := ex.ExecContext(ctx, `INSERT INTO rating_description_set
		(id, workspace_id, version, version_status, name, score_scale_id, active, date_created, date_modified)
		VALUES ($1, $2, 1, $3, 'W3LC Set', $4, true, 0, 0)`,
		id, workspaceID, status, scaleID); err != nil {
		t.Fatalf("seed rating_description_set %s: %v", id, err)
	}
}

// --- (b): entry update must lock+validate the ACTUAL stored parent and
// reject reparenting -------------------------------------------------------

func TestIntegration_RatingDescriptionSetEntry_UpdateRejectsReparentingToADifferentDraftSet(t *testing.T) {
	db := openW3LifecycleIntegrationDB(t)
	tm := postgresCore.NewPostgreSQLTransactionManager(db)
	wsOps := postgresCore.NewWorkspaceAwareOperations(db)
	entryRepo := NewPostgresRatingDescriptionSetEntryRepository(wsOps, "rating_description_set_entry").(*PostgresRatingDescriptionSetEntryRepository)

	const ws = "w3lc-reparent-ws"
	const scale = "w3lc-reparent-scale"
	const realParent = "w3lc-reparent-real-parent"   // the entry's ACTUAL parent — DRAFT
	const attackerSet = "w3lc-reparent-attacker-set" // a DIFFERENT DRAFT set the caller supplies instead
	const bandID = "w3lc-reparent-band"
	const criterionID = "w3lc-reparent-criterion"
	const entryID = "w3lc-reparent-entry"

	err := tm.RunInTransaction(context.Background(), func(txCtx context.Context) error {
		ex, exErr := sgdtExecutor(txCtx, wsOps)
		if exErr != nil {
			return exErr
		}
		seedW3LifecycleWorkspace(t, ex, txCtx, ws)
		seedW3LifecycleScoreScale(t, ex, txCtx, scale, ws)
		seedW3LifecycleSet(t, ex, txCtx, realParent, ws, scale, ratingDescriptionSetVersionStatusDraft)
		seedW3LifecycleSet(t, ex, txCtx, attackerSet, ws, scale, ratingDescriptionSetVersionStatusDraft)
		if _, err := ex.ExecContext(txCtx, `INSERT INTO score_scale_band (id, workspace_id, score_scale_id, sequence_order, output_label) VALUES ($1, $2, $3, 1, 'Level 1')`, bandID, ws, scale); err != nil {
			return fmt.Errorf("seed band: %w", err)
		}
		if _, err := ex.ExecContext(txCtx, `INSERT INTO outcome_criteria (id, workspace_id, name, active) VALUES ($1, $2, 'W3LC Criterion', true)`, criterionID, ws); err != nil {
			return fmt.Errorf("seed criterion: %w", err)
		}
		// Seed the entry's REAL parent directly as realParent.
		if _, err := ex.ExecContext(txCtx, `INSERT INTO rating_description_set_entry
			(id, workspace_id, rating_description_set_id, outcome_criteria_id, score_scale_band_id, description, active, date_created, date_modified)
			VALUES ($1, $2, $3, $4, $5, 'original text', true, 0, 0)`,
			entryID, ws, realParent, criterionID, bandID); err != nil {
			return fmt.Errorf("seed entry: %w", err)
		}

		ctx := identity.WithRequestIdentity(txCtx, &identity.RequestIdentity{UserID: "w3lc-user", WorkspaceID: ws})

		// Attacker supplies a DIFFERENT (but also DRAFT, same-workspace) set as
		// rating_description_set_id, hoping the guard validates THAT one
		// instead of the entry's real parent.
		_, err := entryRepo.UpdateRatingDescriptionSetEntry(ctx, &entrypb.UpdateRatingDescriptionSetEntryRequest{Data: &entrypb.RatingDescriptionSetEntry{
			Id:                     entryID,
			RatingDescriptionSetId: attackerSet,
			Description:            "attacker text",
		}})
		if err == nil {
			return fmt.Errorf("expected REPARENT_FORBIDDEN, got success")
		}
		if !strings.Contains(err.Error(), "REPARENT_FORBIDDEN") {
			return fmt.Errorf("expected REPARENT_FORBIDDEN, got: %w", err)
		}

		// The entry must be untouched (still under its real parent, original text).
		var storedParent, storedText string
		if scanErr := ex.QueryRowContext(txCtx, `SELECT rating_description_set_id, description FROM rating_description_set_entry WHERE id = $1`, entryID).Scan(&storedParent, &storedText); scanErr != nil {
			return fmt.Errorf("read back entry: %w", scanErr)
		}
		if storedParent != realParent {
			return fmt.Errorf("entry must keep its REAL parent %q, got %q — reparenting succeeded", realParent, storedParent)
		}
		if storedText != "original text" {
			return fmt.Errorf("entry description must be unchanged, got %q", storedText)
		}
		return fmt.Errorf("%s", w3lcRollbackMsg)
	})
	if err == nil || !strings.Contains(err.Error(), w3lcRollbackMsg) {
		t.Fatalf("test body failed (or rollback sentinel missing): %v", err)
	}
}

// Positive control: updating an entry WITHOUT attempting to change its
// rating_description_set_id succeeds while its real (DRAFT) parent stays
// locked/validated.
func TestIntegration_RatingDescriptionSetEntry_UpdateWithoutReparentingSucceeds(t *testing.T) {
	db := openW3LifecycleIntegrationDB(t)
	tm := postgresCore.NewPostgreSQLTransactionManager(db)
	wsOps := postgresCore.NewWorkspaceAwareOperations(db)
	entryRepo := NewPostgresRatingDescriptionSetEntryRepository(wsOps, "rating_description_set_entry").(*PostgresRatingDescriptionSetEntryRepository)

	const ws = "w3lc-noreparent-ws"
	const scale = "w3lc-noreparent-scale"
	const parent = "w3lc-noreparent-parent"
	const bandID = "w3lc-noreparent-band"
	const criterionID = "w3lc-noreparent-criterion"
	const entryID = "w3lc-noreparent-entry"

	err := tm.RunInTransaction(context.Background(), func(txCtx context.Context) error {
		ex, exErr := sgdtExecutor(txCtx, wsOps)
		if exErr != nil {
			return exErr
		}
		seedW3LifecycleWorkspace(t, ex, txCtx, ws)
		seedW3LifecycleScoreScale(t, ex, txCtx, scale, ws)
		seedW3LifecycleSet(t, ex, txCtx, parent, ws, scale, ratingDescriptionSetVersionStatusDraft)
		if _, err := ex.ExecContext(txCtx, `INSERT INTO score_scale_band (id, workspace_id, score_scale_id, sequence_order, output_label) VALUES ($1, $2, $3, 1, 'Level 1')`, bandID, ws, scale); err != nil {
			return fmt.Errorf("seed band: %w", err)
		}
		if _, err := ex.ExecContext(txCtx, `INSERT INTO outcome_criteria (id, workspace_id, name, active) VALUES ($1, $2, 'W3LC Criterion', true)`, criterionID, ws); err != nil {
			return fmt.Errorf("seed criterion: %w", err)
		}
		if _, err := ex.ExecContext(txCtx, `INSERT INTO rating_description_set_entry
			(id, workspace_id, rating_description_set_id, outcome_criteria_id, score_scale_band_id, description, active, date_created, date_modified)
			VALUES ($1, $2, $3, $4, $5, 'original text', true, 0, 0)`,
			entryID, ws, parent, criterionID, bandID); err != nil {
			return fmt.Errorf("seed entry: %w", err)
		}

		ctx := identity.WithRequestIdentity(txCtx, &identity.RequestIdentity{UserID: "w3lc-user", WorkspaceID: ws})
		resp, err := entryRepo.UpdateRatingDescriptionSetEntry(ctx, &entrypb.UpdateRatingDescriptionSetEntryRequest{Data: &entrypb.RatingDescriptionSetEntry{
			Id:          entryID,
			Description: "updated text",
		}})
		if err != nil {
			return fmt.Errorf("expected success, got: %w", err)
		}
		if len(resp.GetData()) != 1 || resp.GetData()[0].GetDescription() != "updated text" {
			return fmt.Errorf("unexpected response: %v", resp)
		}
		return fmt.Errorf("%s", w3lcRollbackMsg)
	})
	if err == nil || !strings.Contains(err.Error(), w3lcRollbackMsg) {
		t.Fatalf("test body failed (or rollback sentinel missing): %v", err)
	}
}

// --- (d): RelinkLocked must validate product_plan + price_schedule tenant
// ownership before inserting --------------------------------------------

// seedW3LifecycleOffering seeds a product+plan+product_plan chain scoped to
// the given workspace.
func seedW3LifecycleOffering(t *testing.T, ex sqlexec.DBExecutor, ctx context.Context, productID, planID, productPlanID, workspaceID string) {
	t.Helper()
	if _, err := ex.ExecContext(ctx, `INSERT INTO product (id, workspace_id, name) VALUES ($1, $2, 'W3LC Product')`, productID, workspaceID); err != nil {
		t.Fatalf("seed product %s: %v", productID, err)
	}
	if _, err := ex.ExecContext(ctx, `INSERT INTO plan (id, workspace_id, name) VALUES ($1, $2, 'W3LC Plan')`, planID, workspaceID); err != nil {
		t.Fatalf("seed plan %s: %v", planID, err)
	}
	if _, err := ex.ExecContext(ctx, `INSERT INTO product_plan (id, product_id, plan_id) VALUES ($1, $2, $3)`, productPlanID, productID, planID); err != nil {
		t.Fatalf("seed product_plan %s: %v", productPlanID, err)
	}
}

func seedW3LifecyclePriceSchedule(t *testing.T, ex sqlexec.DBExecutor, ctx context.Context, id, workspaceID string) {
	t.Helper()
	if _, err := ex.ExecContext(ctx, `INSERT INTO price_schedule (id, workspace_id, name) VALUES ($1, $2, 'W3LC AY')`, id, workspaceID); err != nil {
		t.Fatalf("seed price_schedule %s: %v", id, err)
	}
}

func TestIntegration_RatingDescriptionSetProductPlan_RelinkRejectsForeignWorkspaceProductPlan(t *testing.T) {
	db := openW3LifecycleIntegrationDB(t)
	tm := postgresCore.NewPostgreSQLTransactionManager(db)
	wsOps := postgresCore.NewWorkspaceAwareOperations(db)
	linkRepo := NewPostgresRatingDescriptionSetProductPlanRepository(wsOps, "rating_description_set_product_plan").(*PostgresRatingDescriptionSetProductPlanRepository)

	const wsOwn = "w3lc-relink-pp-own-ws"
	const wsForeign = "w3lc-relink-pp-foreign-ws"
	const scale = "w3lc-relink-pp-scale"
	const setPublished = "w3lc-relink-pp-set"
	const ppForeign = "w3lc-relink-pp-foreign-pp"
	const scheduleOwn = "w3lc-relink-pp-own-schedule"

	err := tm.RunInTransaction(context.Background(), func(txCtx context.Context) error {
		ex, exErr := sgdtExecutor(txCtx, wsOps)
		if exErr != nil {
			return exErr
		}
		seedW3LifecycleWorkspace(t, ex, txCtx, wsOwn)
		seedW3LifecycleWorkspace(t, ex, txCtx, wsForeign)
		seedW3LifecycleScoreScale(t, ex, txCtx, scale, wsOwn)
		seedW3LifecycleSet(t, ex, txCtx, setPublished, wsOwn, scale, versionStatusPublishedValue)
		seedW3LifecycleOffering(t, ex, txCtx, "w3lc-relink-pp-foreign-product", "w3lc-relink-pp-foreign-plan", ppForeign, wsForeign)
		seedW3LifecyclePriceSchedule(t, ex, txCtx, scheduleOwn, wsOwn)

		ctx := identity.WithRequestIdentity(txCtx, &identity.RequestIdentity{UserID: "w3lc-user", WorkspaceID: wsOwn})
		_, err := linkRepo.RelinkLocked(ctx, &linkpb.RatingDescriptionSetProductPlan{
			Id: "w3lc-relink-pp-newlink", ProductPlanId: ppForeign, PriceScheduleId: scheduleOwn, RatingDescriptionSetId: setPublished,
		}, nil, "")
		if err == nil {
			return fmt.Errorf("expected forbidden: product_plan belongs to a foreign workspace")
		}
		if !strings.Contains(err.Error(), "forbidden") {
			return fmt.Errorf("expected a forbidden error, got: %w", err)
		}

		var residue int
		if scanErr := ex.QueryRowContext(txCtx, `SELECT count(*) FROM rating_description_set_product_plan WHERE product_plan_id = $1`, ppForeign).Scan(&residue); scanErr != nil {
			return fmt.Errorf("count residue: %w", scanErr)
		}
		if residue != 0 {
			return fmt.Errorf("expected no link row inserted for the foreign product_plan, found %d", residue)
		}
		return fmt.Errorf("%s", w3lcRollbackMsg)
	})
	if err == nil || !strings.Contains(err.Error(), w3lcRollbackMsg) {
		t.Fatalf("test body failed (or rollback sentinel missing): %v", err)
	}
}

func TestIntegration_RatingDescriptionSetProductPlan_RelinkRejectsForeignWorkspacePriceSchedule(t *testing.T) {
	db := openW3LifecycleIntegrationDB(t)
	tm := postgresCore.NewPostgreSQLTransactionManager(db)
	wsOps := postgresCore.NewWorkspaceAwareOperations(db)
	linkRepo := NewPostgresRatingDescriptionSetProductPlanRepository(wsOps, "rating_description_set_product_plan").(*PostgresRatingDescriptionSetProductPlanRepository)

	const wsOwn = "w3lc-relink-ps-own-ws"
	const wsForeign = "w3lc-relink-ps-foreign-ws"
	const scale = "w3lc-relink-ps-scale"
	const setPublished = "w3lc-relink-ps-set"
	const ppOwn = "w3lc-relink-ps-own-pp"
	const scheduleForeign = "w3lc-relink-ps-foreign-schedule"

	err := tm.RunInTransaction(context.Background(), func(txCtx context.Context) error {
		ex, exErr := sgdtExecutor(txCtx, wsOps)
		if exErr != nil {
			return exErr
		}
		seedW3LifecycleWorkspace(t, ex, txCtx, wsOwn)
		seedW3LifecycleWorkspace(t, ex, txCtx, wsForeign)
		seedW3LifecycleScoreScale(t, ex, txCtx, scale, wsOwn)
		seedW3LifecycleSet(t, ex, txCtx, setPublished, wsOwn, scale, versionStatusPublishedValue)
		seedW3LifecycleOffering(t, ex, txCtx, "w3lc-relink-ps-own-product", "w3lc-relink-ps-own-plan", ppOwn, wsOwn)
		seedW3LifecyclePriceSchedule(t, ex, txCtx, scheduleForeign, wsForeign)

		ctx := identity.WithRequestIdentity(txCtx, &identity.RequestIdentity{UserID: "w3lc-user", WorkspaceID: wsOwn})
		_, err := linkRepo.RelinkLocked(ctx, &linkpb.RatingDescriptionSetProductPlan{
			Id: "w3lc-relink-ps-newlink", ProductPlanId: ppOwn, PriceScheduleId: scheduleForeign, RatingDescriptionSetId: setPublished,
		}, nil, "")
		if err == nil {
			return fmt.Errorf("expected forbidden: price_schedule belongs to a foreign workspace")
		}
		if !strings.Contains(err.Error(), "forbidden") {
			return fmt.Errorf("expected a forbidden error, got: %w", err)
		}

		var residue int
		if scanErr := ex.QueryRowContext(txCtx, `SELECT count(*) FROM rating_description_set_product_plan WHERE price_schedule_id = $1`, scheduleForeign).Scan(&residue); scanErr != nil {
			return fmt.Errorf("count residue: %w", scanErr)
		}
		if residue != 0 {
			return fmt.Errorf("expected no link row inserted for the foreign price_schedule, found %d", residue)
		}
		return fmt.Errorf("%s", w3lcRollbackMsg)
	})
	if err == nil || !strings.Contains(err.Error(), w3lcRollbackMsg) {
		t.Fatalf("test body failed (or rollback sentinel missing): %v", err)
	}
}

// Positive control: an owned product_plan + owned price_schedule pair links
// successfully.
func TestIntegration_RatingDescriptionSetProductPlan_RelinkSucceedsForOwnedPair(t *testing.T) {
	db := openW3LifecycleIntegrationDB(t)
	tm := postgresCore.NewPostgreSQLTransactionManager(db)
	wsOps := postgresCore.NewWorkspaceAwareOperations(db)
	linkRepo := NewPostgresRatingDescriptionSetProductPlanRepository(wsOps, "rating_description_set_product_plan").(*PostgresRatingDescriptionSetProductPlanRepository)

	// ws must be a real UUID string (not the package's usual human-readable
	// fixture id): this test reaches RelinkLocked's success path, which now
	// (codex-review-impl2.out.md finding 9) writes a semantic
	// audit_trail.audit_entry row whose workspace_id column is typed uuid —
	// production workspace ids are always real UUIDs (uuidv7), so this only
	// tightens the fixture to match reality. Every OTHER id in this test stays
	// a plain string; only audit_entry.workspace_id is uuid-typed.
	const ws = "00000000-0000-0000-0000-00000000a001"
	const scale = "w3lc-relink-own-scale"
	const setPublished = "w3lc-relink-own-set"
	const pp = "w3lc-relink-own-pp"
	const schedule = "w3lc-relink-own-schedule"

	err := tm.RunInTransaction(context.Background(), func(txCtx context.Context) error {
		ex, exErr := sgdtExecutor(txCtx, wsOps)
		if exErr != nil {
			return exErr
		}
		seedW3LifecycleWorkspace(t, ex, txCtx, ws)
		seedW3LifecycleScoreScale(t, ex, txCtx, scale, ws)
		seedW3LifecycleSet(t, ex, txCtx, setPublished, ws, scale, versionStatusPublishedValue)
		seedW3LifecycleOffering(t, ex, txCtx, "w3lc-relink-own-product", "w3lc-relink-own-plan", pp, ws)
		seedW3LifecyclePriceSchedule(t, ex, txCtx, schedule, ws)

		ctx := identity.WithRequestIdentity(txCtx, &identity.RequestIdentity{UserID: "w3lc-user", WorkspaceID: ws})
		item, err := linkRepo.RelinkLocked(ctx, &linkpb.RatingDescriptionSetProductPlan{
			Id: "w3lc-relink-own-newlink", ProductPlanId: pp, PriceScheduleId: schedule, RatingDescriptionSetId: setPublished,
		}, nil, "")
		if err != nil {
			return fmt.Errorf("expected success for an owned product_plan + price_schedule pair, got: %w", err)
		}
		if !item.GetActive() {
			return fmt.Errorf("expected the new link to be active")
		}

		var count int
		if scanErr := ex.QueryRowContext(txCtx, `SELECT count(*) FROM rating_description_set_product_plan WHERE product_plan_id = $1 AND price_schedule_id = $2 AND active`, pp, schedule).Scan(&count); scanErr != nil {
			return fmt.Errorf("count inserted link: %w", scanErr)
		}
		if count != 1 {
			return fmt.Errorf("expected exactly one active link row, found %d", count)
		}
		return fmt.Errorf("%s", w3lcRollbackMsg)
	})
	if err == nil || !strings.Contains(err.Error(), w3lcRollbackMsg) {
		t.Fatalf("test body failed (or rollback sentinel missing): %v", err)
	}
}

// --- (c), adapter-level proof: UpdateRatingDescriptionSetIfDraft never
// writes lifecycle fields, even when the caller's payload carries them ----

func TestIntegration_RatingDescriptionSet_LockForUpdateReturnsActualStatus(t *testing.T) {
	db := openW3LifecycleIntegrationDB(t)
	tm := postgresCore.NewPostgreSQLTransactionManager(db)
	wsOps := postgresCore.NewWorkspaceAwareOperations(db)
	setRepo := NewPostgresRatingDescriptionSetRepository(wsOps, "rating_description_set").(*PostgresRatingDescriptionSetRepository)

	const ws = "w3lc-lock-ws"
	const scale = "w3lc-lock-scale"
	const draftSet = "w3lc-lock-draft-set"
	const publishedSet = "w3lc-lock-published-set"

	err := tm.RunInTransaction(context.Background(), func(txCtx context.Context) error {
		ex, exErr := sgdtExecutor(txCtx, wsOps)
		if exErr != nil {
			return exErr
		}
		seedW3LifecycleWorkspace(t, ex, txCtx, ws)
		seedW3LifecycleScoreScale(t, ex, txCtx, scale, ws)
		seedW3LifecycleSet(t, ex, txCtx, draftSet, ws, scale, ratingDescriptionSetVersionStatusDraft)
		seedW3LifecycleSet(t, ex, txCtx, publishedSet, ws, scale, versionStatusPublishedValue)

		ctx := identity.WithRequestIdentity(txCtx, &identity.RequestIdentity{UserID: "w3lc-user", WorkspaceID: ws})
		draftStatus, err := setRepo.LockRatingDescriptionSetForUpdate(ctx, draftSet)
		if err != nil {
			return fmt.Errorf("lock draft set: %w", err)
		}
		if draftStatus != enumspb.VersionStatus_VERSION_STATUS_DRAFT {
			return fmt.Errorf("expected DRAFT, got %v", draftStatus)
		}
		publishedStatus, err := setRepo.LockRatingDescriptionSetForUpdate(ctx, publishedSet)
		if err != nil {
			return fmt.Errorf("lock published set: %w", err)
		}
		if publishedStatus != enumspb.VersionStatus_VERSION_STATUS_PUBLISHED {
			return fmt.Errorf("expected PUBLISHED, got %v", publishedStatus)
		}
		return fmt.Errorf("%s", w3lcRollbackMsg)
	})
	if err == nil || !strings.Contains(err.Error(), w3lcRollbackMsg) {
		t.Fatalf("test body failed (or rollback sentinel missing): %v", err)
	}
}

func TestIntegration_RatingDescriptionSet_UpdateIfDraftNeverWritesLifecycleFields(t *testing.T) {
	db := openW3LifecycleIntegrationDB(t)
	tm := postgresCore.NewPostgreSQLTransactionManager(db)
	wsOps := postgresCore.NewWorkspaceAwareOperations(db)
	setRepo := NewPostgresRatingDescriptionSetRepository(wsOps, "rating_description_set").(*PostgresRatingDescriptionSetRepository)

	const ws = "w3lc-update-ws"
	const scale = "w3lc-update-scale"
	const draftSet = "w3lc-update-draft-set"

	err := tm.RunInTransaction(context.Background(), func(txCtx context.Context) error {
		ex, exErr := sgdtExecutor(txCtx, wsOps)
		if exErr != nil {
			return exErr
		}
		seedW3LifecycleWorkspace(t, ex, txCtx, ws)
		seedW3LifecycleScoreScale(t, ex, txCtx, scale, ws)
		seedW3LifecycleSet(t, ex, txCtx, draftSet, ws, scale, ratingDescriptionSetVersionStatusDraft)

		ctx := identity.WithRequestIdentity(txCtx, &identity.RequestIdentity{UserID: "w3lc-user", WorkspaceID: ws})
		if _, err := setRepo.LockRatingDescriptionSetForUpdate(ctx, draftSet); err != nil {
			return fmt.Errorf("lock: %w", err)
		}

		// Simulate a caller/payload that STILL carries lifecycle fields (as if
		// the use-case-layer strip were somehow bypassed) — the adapter must
		// strip them unconditionally regardless.
		staleSupersedes := "some-other-set"
		req := &pb.UpdateRatingDescriptionSetRequest{Data: &pb.RatingDescriptionSet{
			Id:            draftSet,
			Name:          "Renamed By Test",
			VersionStatus: enumspb.VersionStatus_VERSION_STATUS_PUBLISHED,
			Version:       99,
			SupersedesId:  &staleSupersedes,
		}}
		item, err := setRepo.UpdateRatingDescriptionSetIfDraft(ctx, req)
		if err != nil {
			return fmt.Errorf("update: %w", err)
		}
		if item.GetName() != "Renamed By Test" {
			return fmt.Errorf("expected name to be written, got %q", item.GetName())
		}

		var status string
		var version int
		var supersedes sql.NullString
		if scanErr := ex.QueryRowContext(txCtx, `SELECT version_status, version, supersedes_id FROM rating_description_set WHERE id = $1`, draftSet).Scan(&status, &version, &supersedes); scanErr != nil {
			return fmt.Errorf("read back: %w", scanErr)
		}
		if status != ratingDescriptionSetVersionStatusDraft {
			return fmt.Errorf("version_status must remain DRAFT (never written from the payload), got %q", status)
		}
		if version != 1 {
			return fmt.Errorf("version must remain 1 (never written from the payload), got %d", version)
		}
		if supersedes.Valid {
			return fmt.Errorf("supersedes_id must remain NULL (never written from the payload), got %q", supersedes.String)
		}
		return fmt.Errorf("%s", w3lcRollbackMsg)
	})
	if err == nil || !strings.Contains(err.Error(), w3lcRollbackMsg) {
		t.Fatalf("test body failed (or rollback sentinel missing): %v", err)
	}
}
