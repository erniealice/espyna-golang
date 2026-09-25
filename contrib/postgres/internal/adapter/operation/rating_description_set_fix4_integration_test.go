//go:build postgresql

package operation

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	setuc "github.com/erniealice/espyna-golang/internal/application/usecases/domain/operation/rating_description_set"
	entryuc "github.com/erniealice/espyna-golang/internal/application/usecases/domain/operation/rating_description_set_entry"
	"github.com/erniealice/espyna-golang/registry/entityid"
	"github.com/erniealice/espyna-golang/shared/identity"
	criteriapb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/outcome_criteria"
	setpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/rating_description_set"
	entrypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/rating_description_set_entry"
	linkpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/rating_description_set_product_plan"
	bandpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/score_scale_band"
	"github.com/lib/pq"
)

// fix4-app (docs/plan/20260925-criterion-descriptors-by-program-year,
// codex-review-impl4.out.md round-3/round-4 disposition #2, "Update-only
// drawer"). TEST_DATABASE_URL-gated; run against the CLONE
// (education2clone20260925a), never education2. Every fixture write happens
// inside a transaction rolled back via t.Cleanup — reuses fix3's helpers
// (fix3StartTx/fix3Seed/fix3RegisteredRepo/fix3UUID), same package.

// --- (a) picker pagination: entry drawer's criteria/band pickers ---------

// TestIntegrationFix4_EntryDrawerPickersReturnAllCriteriaAndBandsOver100
// seeds a FRESH, isolated workspace with 127 outcome_criteria and 133
// score_scale_band rows (both > the adapter's single-page cap of 100) via
// fix3Seed, then drives them through the REAL registered outcome_criteria
// and score_scale_band adapters behind
// GetRatingDescriptionSetEntryDrawerFormPageDataUseCase — the exact code
// path the Add/Edit entry drawer calls. Proves the picker loop added in
// fetchEntryFormPickerData pages past the adapter's 100-row default cap
// against real Postgres, not just the in-package fake-repo unit test.
func TestIntegrationFix4_EntryDrawerPickersReturnAllCriteriaAndBandsOver100(t *testing.T) {
	db := openW3LifecycleIntegrationDB(t)
	txCtx, tx := fix3StartTx(t, db, false)
	ws := fix3UUID(41)
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	scale, set := "fix4-it-scale-"+suffix, "fix4-it-set-"+suffix
	const nBands, nCriteria = 133, 127
	fix3Seed(t, txCtx, tx, ws, scale, set, nBands, nCriteria)

	ctx := identity.WithRequestIdentity(txCtx, &identity.RequestIdentity{UserID: "fix4-it", WorkspaceID: ws})

	criteriaRepo := fix3RegisteredRepo[criteriapb.OutcomeCriteriaDomainServiceServer](t, db, entityid.OutcomeCriteria)
	bandRepo := fix3RegisteredRepo[bandpb.ScoreScaleBandDomainServiceServer](t, db, entityid.ScoreScaleBand)

	uc := entryuc.NewGetRatingDescriptionSetEntryDrawerFormPageDataUseCase(
		entryuc.GetRatingDescriptionSetEntryFormPageDataRepositories{
			OutcomeCriteria: criteriaRepo,
			ScoreScaleBand:  bandRepo,
		},
		entryuc.GetRatingDescriptionSetEntryFormPageDataServices{
			Authorizer:       ports.NewNoOpAuthorizer(),
			Translator:       ports.NewNoOpTranslator(),
			ActionGatekeeper: fix3Gatekeeper(),
		},
	)

	resp, err := uc.Execute(ctx, &entryuc.GetRatingDescriptionSetEntryDrawerFormPageDataRequest{})
	if err != nil {
		t.Fatalf("drawer page data: %v", err)
	}
	if len(resp.Criteria) != nCriteria {
		t.Fatalf("expected all %d criteria back, got %d", nCriteria, len(resp.Criteria))
	}
	if len(resp.Bands) != nBands {
		t.Fatalf("expected all %d bands back, got %d", nBands, len(resp.Bands))
	}

	// The same-package sibling GetRatingDescriptionSetEntryFormPageDataUseCase
	// (:read-gated, backs the read-only matrix) shares fetchEntryFormPickerData
	// — prove it pages identically.
	matrixUC := entryuc.NewGetRatingDescriptionSetEntryFormPageDataUseCase(
		entryuc.GetRatingDescriptionSetEntryFormPageDataRepositories{OutcomeCriteria: criteriaRepo, ScoreScaleBand: bandRepo},
		entryuc.GetRatingDescriptionSetEntryFormPageDataServices{Authorizer: ports.NewNoOpAuthorizer(), Translator: ports.NewNoOpTranslator(), ActionGatekeeper: fix3Gatekeeper()},
	)
	matrixResp, err := matrixUC.Execute(ctx, &entryuc.GetRatingDescriptionSetEntryFormPageDataRequest{})
	if err != nil {
		t.Fatalf("matrix page data: %v", err)
	}
	if len(matrixResp.Criteria) != nCriteria || len(matrixResp.Bands) != nBands {
		t.Fatalf("matrix source: expected %d criteria / %d bands, got %d / %d", nCriteria, nBands, len(matrixResp.Criteria), len(matrixResp.Bands))
	}
}

// TestIntegrationFix4_EditDrawerLoadsEntryAndScaleViaUpdateOnlyAuthorization
// is the "Update-only drawer" finding end to end against real Postgres: an
// entry created inside the same isolated workspace is loaded by ExecuteForIDs
// (the Edit GET path's call) through ONLY rating_description_set:update — the
// entry's own :read gate (ReadRatingDescriptionSetEntryUseCase) and the
// parent set's own :read gate (ReadRatingDescriptionSetUseCase) are never
// invoked; this test calls the REAL registered entry/set adapters directly,
// bypassing their use-case gates exactly as production wiring does.
func TestIntegrationFix4_EditDrawerLoadsEntryAndScaleViaUpdateOnlyAuthorization(t *testing.T) {
	db := openW3LifecycleIntegrationDB(t)
	txCtx, tx := fix3StartTx(t, db, false)
	ws := fix3UUID(42)
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	scale, set := "fix4-it-scale2-"+suffix, "fix4-it-set2-"+suffix
	bands, criteria := fix3Seed(t, txCtx, tx, ws, scale, set, 3, 3)

	entryID := "fix4-it-entry-" + suffix
	if _, err := tx.ExecContext(txCtx, `INSERT INTO `+ratingDescriptionSetEntryTable+`
		(id, workspace_id, rating_description_set_id, outcome_criteria_id, score_scale_band_id, description, sequence_order, active)
		VALUES ($1, $2, $3, $4, $5, 'fix4 it entry text', 1, true)`,
		entryID, ws, set, criteria[0], bands[0]); err != nil {
		t.Fatalf("seed entry: %v", err)
	}

	ctx := identity.WithRequestIdentity(txCtx, &identity.RequestIdentity{UserID: "fix4-it", WorkspaceID: ws})

	entryRepo := fix3RegisteredRepo[entrypb.RatingDescriptionSetEntryDomainServiceServer](t, db, entityid.RatingDescriptionSetEntry)
	setRepo := fix3RegisteredRepo[setpb.RatingDescriptionSetDomainServiceServer](t, db, entityid.RatingDescriptionSet)

	uc := entryuc.NewGetRatingDescriptionSetEntryDrawerFormPageDataUseCase(
		entryuc.GetRatingDescriptionSetEntryFormPageDataRepositories{},
		entryuc.GetRatingDescriptionSetEntryFormPageDataServices{
			Authorizer:       ports.NewNoOpAuthorizer(),
			Translator:       ports.NewNoOpTranslator(),
			ActionGatekeeper: fix3Gatekeeper(),
		},
	).WithParentReads(setRepo, entryRepo)

	resp, err := uc.ExecuteForIDs(ctx, "", entryID)
	if err != nil {
		t.Fatalf("edit drawer page data: %v", err)
	}
	if resp.Entry == nil || resp.Entry.ID != entryID {
		t.Fatalf("expected entry %s resolved via the parent-authorized read, got %+v", entryID, resp.Entry)
	}
	if resp.Entry.RatingDescriptionSetId != set || resp.Entry.OutcomeCriteriaId != criteria[0] || resp.Entry.ScoreScaleBandId != bands[0] {
		t.Fatalf("entry snapshot mismatch: %+v", resp.Entry)
	}
	if resp.ScoreScaleId != scale {
		t.Fatalf("expected score_scale_id %s resolved from the parent set, got %s", scale, resp.ScoreScaleId)
	}
}

// --- (b) summary counts: scoped SQL aggregation ---------------------------

// TestIntegrationFix4_EntryAndLinkCountersMatchScopedSQLAggregation drives
// the REAL registered rating_description_set_entry and
// rating_description_set_product_plan adapters' new
// CountActiveRatingDescriptionSetEntriesBySet /
// CountActiveRatingDescriptionSetProductPlansBySet methods (the port
// GetRatingDescriptionSetListSummaryPageDataUseCase now uses) against the
// clone's REAL data and asserts the result against an independent SQL
// GROUP BY oracle — proving the aggregation SQL path is both used (these are
// the exact methods the list-summary use case type-asserts for) and correct,
// without needing a >5,000-entry workspace (the prior capped-scan's specific
// failure mode is closed by construction: aggregation SQL has no page loop
// to cap).
func TestIntegrationFix4_EntryAndLinkCountersMatchScopedSQLAggregation(t *testing.T) {
	db := openW3LifecycleIntegrationDB(t)
	txCtx, tx := fix3StartTx(t, db, true) // REPEATABLE READ: one snapshot for the oracle AND the counter call.

	rows, err := tx.QueryContext(txCtx, `SELECT rating_description_set_id, count(*) FROM `+ratingDescriptionSetEntryTable+`
		WHERE active GROUP BY rating_description_set_id ORDER BY count(*) DESC LIMIT 5`)
	if err != nil {
		t.Fatalf("oracle query: %v", err)
	}
	wantEntryCounts := map[string]int32{}
	var setIDs []string
	for rows.Next() {
		var id string
		var n int32
		if err := rows.Scan(&id, &n); err != nil {
			t.Fatalf("scan: %v", err)
		}
		wantEntryCounts[id] = n
		setIDs = append(setIDs, id)
	}
	rows.Close()
	if len(setIDs) == 0 {
		t.Skip("no active rating_description_set_entry rows on this database")
	}
	t.Logf("scoping to %d real set ids (SQL oracle): %v", len(setIDs), wantEntryCounts)

	wantLinkCounts := map[string]int32{}
	linkRows, err := tx.QueryContext(txCtx, `SELECT rating_description_set_id, count(*) FROM `+ratingDescriptionSetProductPlanTable+`
		WHERE active AND rating_description_set_id = ANY($1) GROUP BY rating_description_set_id`, pq.Array(setIDs))
	if err == nil {
		for linkRows.Next() {
			var id string
			var n int32
			if err := linkRows.Scan(&id, &n); err != nil {
				t.Fatalf("scan: %v", err)
			}
			wantLinkCounts[id] = n
		}
		linkRows.Close()
	}

	entryRepo := fix3RegisteredRepo[*PostgresRatingDescriptionSetEntryRepository](t, db, entityid.RatingDescriptionSetEntry)
	linkRepo := fix3RegisteredRepo[*PostgresRatingDescriptionSetProductPlanRepository](t, db, entityid.RatingDescriptionSetProductPlan)

	gotEntryCounts, err := entryRepo.CountActiveRatingDescriptionSetEntriesBySet(txCtx, setIDs)
	if err != nil {
		t.Fatalf("CountActiveRatingDescriptionSetEntriesBySet: %v", err)
	}
	for _, id := range setIDs {
		if gotEntryCounts[id] != wantEntryCounts[id] {
			t.Fatalf("set %s: entry count = %d, want %d (SQL oracle)", id, gotEntryCounts[id], wantEntryCounts[id])
		}
	}

	gotLinkCounts, err := linkRepo.CountActiveRatingDescriptionSetProductPlansBySet(txCtx, setIDs)
	if err != nil {
		t.Fatalf("CountActiveRatingDescriptionSetProductPlansBySet: %v", err)
	}
	for _, id := range setIDs {
		if gotLinkCounts[id] != wantLinkCounts[id] {
			t.Fatalf("set %s: link count = %d, want %d (SQL oracle)", id, gotLinkCounts[id], wantLinkCounts[id])
		}
	}

	// Scoping: an id NOT in setIDs must never appear, and an empty/nil input
	// must short-circuit to an empty map without executing SQL (no panic on
	// a nil executor path either, since ambient tx IS present here).
	foreign := fix3UUID(43)
	scoped, err := entryRepo.CountActiveRatingDescriptionSetEntriesBySet(txCtx, []string{foreign})
	if err != nil {
		t.Fatalf("scoped call for a foreign id: %v", err)
	}
	if _, ok := scoped[foreign]; ok {
		if scoped[foreign] != 0 {
			t.Fatalf("expected 0 for an id with no active entries, got %d", scoped[foreign])
		}
	}
	empty, err := entryRepo.CountActiveRatingDescriptionSetEntriesBySet(txCtx, nil)
	if err != nil || len(empty) != 0 {
		t.Fatalf("expected an empty map and no error for a nil id slice, got %v, err=%v", empty, err)
	}
}

// TestIntegrationFix4_ListSummaryUseCaseUsesScopedCounters exercises the
// FULL GetRatingDescriptionSetListSummaryPageDataUseCase (the list page's
// sole data source) against the REAL registered set/entry/link adapters in
// the clone's default workspace, and cross-checks every row's EntryCount/
// LinkCount against the same SQL oracle used above — proving the use case's
// constructor-time type assertion actually wires the counter port in a real
// composition, not just in the fake-repo unit test.
func TestIntegrationFix4_ListSummaryUseCaseUsesScopedCounters(t *testing.T) {
	db := openW3LifecycleIntegrationDB(t)
	txCtx, tx := fix3StartTx(t, db, true)

	var ws string
	if err := tx.QueryRowContext(txCtx, `SELECT workspace_id FROM `+ratingDescriptionSetTable+` WHERE active GROUP BY workspace_id ORDER BY count(*) DESC LIMIT 1`).Scan(&ws); err != nil {
		t.Skipf("no active rating_description_set rows on this database: %v", err)
	}
	ctx := identity.WithRequestIdentity(txCtx, &identity.RequestIdentity{UserID: "fix4-it", WorkspaceID: ws})

	setRepo := fix3RegisteredRepo[setpb.RatingDescriptionSetDomainServiceServer](t, db, entityid.RatingDescriptionSet)
	entryRepo := fix3RegisteredRepo[entrypb.RatingDescriptionSetEntryDomainServiceServer](t, db, entityid.RatingDescriptionSetEntry)
	linkRepo := fix3RegisteredRepo[linkpb.RatingDescriptionSetProductPlanDomainServiceServer](t, db, entityid.RatingDescriptionSetProductPlan)

	uc := setuc.NewGetRatingDescriptionSetListSummaryPageDataUseCase(
		setuc.GetRatingDescriptionSetListSummaryPageDataRepositories{
			RatingDescriptionSet:            setRepo,
			RatingDescriptionSetEntry:       entryRepo,
			RatingDescriptionSetProductPlan: linkRepo,
		},
		setuc.GetRatingDescriptionSetListSummaryPageDataServices{
			Authorizer:       ports.NewNoOpAuthorizer(),
			Translator:       ports.NewNoOpTranslator(),
			ActionGatekeeper: fix3Gatekeeper(),
		},
	)
	resp, err := uc.Execute(ctx, nil)
	if err != nil {
		t.Fatalf("list summary: %v", err)
	}
	if len(resp.Rows) == 0 {
		t.Skip("no rows returned for this workspace")
	}

	setIDs := make([]string, 0, len(resp.Rows))
	for _, row := range resp.Rows {
		setIDs = append(setIDs, row.Set.GetId())
	}
	entryOracle, err := sqlGroupCount(txCtx, tx, ratingDescriptionSetEntryTable, setIDs)
	if err != nil {
		t.Fatalf("entry oracle: %v", err)
	}
	linkOracle, err := sqlGroupCount(txCtx, tx, ratingDescriptionSetProductPlanTable, setIDs)
	if err != nil {
		t.Fatalf("link oracle: %v", err)
	}
	for _, row := range resp.Rows {
		id := row.Set.GetId()
		if row.EntryCount != entryOracle[id] {
			t.Fatalf("set %s: use-case EntryCount=%d, SQL oracle=%d", id, row.EntryCount, entryOracle[id])
		}
		if row.LinkCount != linkOracle[id] {
			t.Fatalf("set %s: use-case LinkCount=%d, SQL oracle=%d", id, row.LinkCount, linkOracle[id])
		}
	}
	t.Logf("verified EntryCount/LinkCount against SQL oracle for %d sets in workspace %s", len(resp.Rows), ws)
}

func sqlGroupCount(ctx context.Context, tx *sql.Tx, table string, setIDs []string) (map[string]int32, error) {
	out := map[string]int32{}
	if len(setIDs) == 0 {
		return out, nil
	}
	rows, err := tx.QueryContext(ctx, `SELECT rating_description_set_id, count(*) FROM `+table+`
		WHERE active AND rating_description_set_id = ANY($1) GROUP BY rating_description_set_id`, pq.Array(setIDs))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		var n int32
		if err := rows.Scan(&id, &n); err != nil {
			return nil, err
		}
		out[id] = n
	}
	return out, rows.Err()
}
