//go:build postgresql

package operation

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"
	"time"

	auditadapter "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/audit"
	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	setuc "github.com/erniealice/espyna-golang/internal/application/usecases/domain/operation/rating_description_set"
	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/common/transactions"
	"github.com/erniealice/espyna-golang/registry"
	"github.com/erniealice/espyna-golang/registry/entityid"
	"github.com/erniealice/espyna-golang/shared/database/operations"
	"github.com/erniealice/espyna-golang/shared/identity"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	setpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/rating_description_set"
	entrypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/rating_description_set_entry"
	linkpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/rating_description_set_product_plan"
	bandpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/score_scale_band"
)

// fix3-backend (codex-review-impl3 #2, #4, #6). TEST_DATABASE_URL-gated; run
// against the CLONE (education2clone20260925a), never education2. Every
// fixture write happens inside a transaction that is rolled back; the one
// top-level use-case call (audit-failure atomicity) is designed to roll back
// by itself and is followed by a residue check.

func fix3UUID(tag int) string {
	return fmt.Sprintf("00000000-0000-4000-8%03d-%012d", tag%1000, time.Now().UnixNano()%1_000_000_000_000)
}

func fix3Transactor(db *sql.DB) ports.Transactor {
	return transactions.NewTransactionServiceAdapter(postgresCore.NewPostgreSQLTransactionManager(db))
}

func fix3Gatekeeper() *actiongate.ActionGatekeeper {
	return actiongate.NewActionGatekeeper(ports.NewNoOpAuthorizer(), ports.NewNoOpTranslator())
}

func fix3RegisteredRepo[T any](t *testing.T, db *sql.DB, entity string) T {
	t.Helper()
	repoAny, err := registry.CreateRepository("postgresql", entity, db, "")
	if err != nil {
		t.Fatalf("registered %s factory: %v", entity, err)
	}
	repo, ok := repoAny.(T)
	if !ok {
		t.Fatalf("registered %s factory returned %T", entity, repoAny)
	}
	return repo
}

func fix3IDSort() *commonpb.SortRequest {
	return &commonpb.SortRequest{Fields: []*commonpb.SortField{{Field: "id", Direction: commonpb.SortDirection_ASC}}}
}

func fix3Page(limit, page int32) *commonpb.PaginationRequest {
	return &commonpb.PaginationRequest{Limit: limit, Method: &commonpb.PaginationRequest_Offset{Offset: &commonpb.OffsetPagination{Page: page}}}
}

func fix3EqualsFilter(field, value string) *commonpb.FilterRequest {
	return &commonpb.FilterRequest{Filters: []*commonpb.TypedFilter{{
		Field: field,
		FilterType: &commonpb.TypedFilter_StringFilter{StringFilter: &commonpb.StringFilter{
			Value: value, Operator: commonpb.StringOperator_STRING_EQUALS, CaseSensitive: true,
		}},
	}}}
}

// fix3Seed inserts a workspace, one scale with nBands bands, nCriteria
// criteria and one DRAFT set on that scale. Returns (bandIDs, criteriaIDs).
func fix3Seed(t *testing.T, ctx context.Context, tx *sql.Tx, ws, scale, set string, nBands, nCriteria int) ([]string, []string) {
	t.Helper()
	exec := func(q string, args ...any) {
		t.Helper()
		if _, err := tx.ExecContext(ctx, q, args...); err != nil {
			t.Fatalf("fixture %q: %v", q, err)
		}
	}
	exec(`INSERT INTO workspace (id, name, active) VALUES ($1, 'fix3 it', true)`, ws)
	exec(`INSERT INTO `+entityid.ScoreScale+`
		(id, scale_group_id, version, version_status, name, scale_kind, input_unit, output_unit, workspace_id, active, created_by)
		VALUES ($1, $1, 1, 'VERSION_STATUS_PUBLISHED', 'fix3 scale', 'SCALE_KIND_EXACT_MAP', 'score', 'label', $2, true, 'fix3-it')`, scale, ws)
	bands := make([]string, nBands)
	for i := range bands {
		bands[i] = fmt.Sprintf("%s-band-%03d", scale, i)
		exec(`INSERT INTO `+entityid.ScoreScaleBand+`
			(id, workspace_id, active, score_scale_id, sequence_order, input_match, output_value, output_label)
			VALUES ($1, $2, true, $3, $4, $5, $6, 'L')`, bands[i], ws, scale, i+1, fmt.Sprintf("%d", i+1), float64(i+1))
	}
	criteria := make([]string, nCriteria)
	for i := range criteria {
		criteria[i] = fmt.Sprintf("%s-crit-%03d", scale, i)
		exec(`INSERT INTO `+entityid.OutcomeCriteria+`
			(id, name, criteria_type, min_score, max_score, score_increment, required, active, workspace_id)
			VALUES ($1, 'fix3 criterion', 'CRITERIA_TYPE_NUMERIC_SCORE', 0, 8, 1, true, true, $2)`, criteria[i], ws)
	}
	exec(`INSERT INTO `+ratingDescriptionSetTable+`
		(id, workspace_id, version, version_status, name, score_scale_id, active)
		VALUES ($1, $2, 1, 'VERSION_STATUS_DRAFT', 'fix3 set', $3, true)`, set, ws, scale)
	return bands, criteria
}

func fix3StartTx(t *testing.T, db *sql.DB, repeatableRead bool) (context.Context, *sql.Tx) {
	t.Helper()
	tm := postgresCore.NewPostgreSQLTransactionManager(db)
	txn, err := tm.StartTransaction(context.Background())
	if err != nil {
		t.Fatalf("start tx: %v", err)
	}
	t.Cleanup(func() { _ = txn.Rollback(context.Background()) })
	txCtx := operations.WithTransaction(context.Background(), txn)
	tx := txn.(interface{ GetTx() *sql.Tx }).GetTx()
	if repeatableRead {
		// One snapshot for the SQL oracle AND every adapter page (other
		// sessions may write the clone concurrently).
		if _, err := tx.ExecContext(txCtx, `SET TRANSACTION ISOLATION LEVEL REPEATABLE READ`); err != nil {
			t.Fatalf("set isolation: %v", err)
		}
	}
	return txCtx, tx
}

// --- #2 pagination ------------------------------------------------------

// TestIntegrationFix3_EntryListPaginatesClone pages every active entry of the
// clone's workspace through the REAL registered entry adapter at 50/page:
// ceil(n/50) pages, no duplicates, total_items == SQL count (128 at review
// time), and ListRatingDescriptionSetEntries returns the same pages.
func TestIntegrationFix3_EntryListPaginatesClone(t *testing.T) {
	db := openW3LifecycleIntegrationDB(t)
	txCtx, tx := fix3StartTx(t, db, true)

	var ws string
	var want int
	if err := tx.QueryRowContext(txCtx, `SELECT workspace_id, count(*) FROM `+ratingDescriptionSetEntryTable+`
		WHERE active GROUP BY workspace_id ORDER BY count(*) DESC LIMIT 1`).Scan(&ws, &want); err != nil {
		t.Skipf("no active entries on this database: %v", err)
	}
	if want <= 100 {
		t.Skipf("need > 100 active entries to prove multi-page paging, have %d", want)
	}
	t.Logf("workspace %s: %d active entries (SQL oracle)", ws, want)
	ctx := identity.WithRequestIdentity(txCtx, &identity.RequestIdentity{UserID: "fix3-it", WorkspaceID: ws})

	repo := fix3RegisteredRepo[*PostgresRatingDescriptionSetEntryRepository](t, db, entityid.RatingDescriptionSetEntry)
	const limit = 50
	seen := map[string]bool{}
	pages := 0
	for page := int32(1); page <= 20; page++ {
		req := &entrypb.ListRatingDescriptionSetEntriesRequest{Sort: fix3IDSort(), Pagination: fix3Page(limit, page)}
		items, pg, err := repo.ListRatingDescriptionSetEntriesPage(ctx, req)
		if err != nil {
			t.Fatalf("page %d: %v", page, err)
		}
		if pg == nil || int(pg.GetTotalItems()) != want {
			t.Fatalf("page %d: total_items=%v want %d", page, pg.GetTotalItems(), want)
		}
		plain, err := repo.ListRatingDescriptionSetEntries(ctx, req)
		if err != nil || len(plain.GetData()) != len(items) {
			t.Fatalf("page %d: List returned %d rows (err %v), Page returned %d", page, len(plain.GetData()), err, len(items))
		}
		pages++
		for i, e := range items {
			if seen[e.GetId()] {
				t.Fatalf("page %d: duplicate entry %s", page, e.GetId())
			}
			seen[e.GetId()] = true
			if plain.GetData()[i].GetId() != e.GetId() {
				t.Fatalf("page %d row %d: List/Page order differ", page, i)
			}
		}
		wantHasNext := len(seen) < want
		if pg.GetHasNext() != wantHasNext || pg.GetHasPrev() != (page > 1) {
			t.Fatalf("page %d: has_next=%v has_prev=%v (seen %d/%d)", page, pg.GetHasNext(), pg.GetHasPrev(), len(seen), want)
		}
		if len(items) < limit {
			break
		}
	}
	if len(seen) != want || pages != (want+limit-1)/limit {
		t.Fatalf("paged %d unique entries over %d pages, want %d over %d", len(seen), pages, want, (want+limit-1)/limit)
	}

	// Set + link page-data: total from SQL COUNT, not an in-memory slice.
	setRepo := fix3RegisteredRepo[*PostgresRatingDescriptionSetRepository](t, db, entityid.RatingDescriptionSet)
	var wantSets int
	if err := tx.QueryRowContext(txCtx, `SELECT count(*) FROM `+ratingDescriptionSetTable+` WHERE active AND workspace_id = $1`, ws).Scan(&wantSets); err != nil {
		t.Fatal(err)
	}
	setSeen := map[string]bool{}
	for page := int32(1); page <= 50; page++ {
		resp, err := setRepo.GetRatingDescriptionSetListPageData(ctx, &setpb.GetRatingDescriptionSetListPageDataRequest{Sort: fix3IDSort(), Pagination: fix3Page(2, page)})
		if err != nil {
			t.Fatalf("set page data %d: %v", page, err)
		}
		if int(resp.GetPagination().GetTotalItems()) != wantSets {
			t.Fatalf("set page data: total=%d want %d", resp.GetPagination().GetTotalItems(), wantSets)
		}
		for _, s := range resp.GetRatingDescriptionSetList() {
			if setSeen[s.GetId()] {
				t.Fatalf("set page data: duplicate %s", s.GetId())
			}
			setSeen[s.GetId()] = true
		}
		if !resp.GetPagination().GetHasNext() {
			break
		}
	}
	if len(setSeen) != wantSets {
		t.Fatalf("set page data paged %d sets, want %d", len(setSeen), wantSets)
	}

	var schedule string
	var wantLinks int
	if err := tx.QueryRowContext(txCtx, `SELECT price_schedule_id, count(*) FROM `+entityid.RatingDescriptionSetProductPlan+`
		WHERE active AND workspace_id = $1 GROUP BY 1 ORDER BY 2 DESC LIMIT 1`, ws).Scan(&schedule, &wantLinks); err == nil {
		linkRepo := fix3RegisteredRepo[*PostgresRatingDescriptionSetProductPlanRepository](t, db, entityid.RatingDescriptionSetProductPlan)
		resp, err := linkRepo.GetRatingDescriptionSetProductPlanListPageData(ctx, &linkpb.GetRatingDescriptionSetProductPlanListPageDataRequest{
			PriceScheduleId: schedule, Sort: fix3IDSort(), Pagination: fix3Page(1, 1)})
		if err != nil {
			t.Fatalf("link page data: %v", err)
		}
		if int(resp.GetPagination().GetTotalItems()) != wantLinks || len(resp.GetRatingDescriptionSetProductPlanList()) != 1 || resp.GetPagination().GetHasNext() != (wantLinks > 1) {
			t.Fatalf("link page data: total=%d rows=%d has_next=%v want total %d", resp.GetPagination().GetTotalItems(), len(resp.GetRatingDescriptionSetProductPlanList()), resp.GetPagination().GetHasNext(), wantLinks)
		}
		// OR logic would OR away the mandatory scope → refused.
		orFilters := fix3EqualsFilter("rating_description_set_id", "x")
		orFilters.Logic = commonpb.FilterLogic_OR
		if _, err := linkRepo.GetRatingDescriptionSetProductPlanListPageData(ctx, &linkpb.GetRatingDescriptionSetProductPlanListPageDataRequest{PriceScheduleId: schedule, Filters: orFilters}); err == nil {
			t.Fatal("OR filter logic must be refused (tenant/AY scope would be OR-ed)")
		}
	}
}

// TestIntegrationFix3_SingleSetOver100EntriesFullyReturned: a DRAFT set with
// 110 entries created inside a rolled-back tx is returned in full by the
// parent-filtered entry list at both 50/page and 100/page (the pre-fix
// adapter capped every call at the first 100 rows).
func TestIntegrationFix3_SingleSetOver100EntriesFullyReturned(t *testing.T) {
	db := openW3LifecycleIntegrationDB(t)
	txCtx, tx := fix3StartTx(t, db, false)
	ws := fix3UUID(1)
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	scale, set := "fix3-it-scale-"+suffix, "fix3-it-set-"+suffix
	bands, criteria := fix3Seed(t, txCtx, tx, ws, scale, set, 11, 10)
	n := 0
	for _, c := range criteria {
		for _, b := range bands {
			n++
			if _, err := tx.ExecContext(txCtx, `INSERT INTO `+ratingDescriptionSetEntryTable+`
				(id, workspace_id, rating_description_set_id, outcome_criteria_id, score_scale_band_id, description, sequence_order, active)
				VALUES ($1, $2, $3, $4, $5, 'd', $6, true)`, fmt.Sprintf("%s-e-%04d", set, n), ws, set, c, b, n); err != nil {
				t.Fatalf("seed entry: %v", err)
			}
		}
	}
	ctx := identity.WithRequestIdentity(txCtx, &identity.RequestIdentity{UserID: "fix3-it", WorkspaceID: ws})
	repo := fix3RegisteredRepo[*PostgresRatingDescriptionSetEntryRepository](t, db, entityid.RatingDescriptionSetEntry)
	for _, limit := range []int32{50, 100} {
		seen := map[string]bool{}
		for page := int32(1); page <= 10; page++ {
			resp, err := repo.ListRatingDescriptionSetEntries(ctx, &entrypb.ListRatingDescriptionSetEntriesRequest{
				Filters: fix3EqualsFilter("rating_description_set_id", set), Sort: fix3IDSort(), Pagination: fix3Page(limit, page)})
			if err != nil {
				t.Fatalf("limit %d page %d: %v", limit, page, err)
			}
			for _, e := range resp.GetData() {
				if e.GetRatingDescriptionSetId() != set || seen[e.GetId()] {
					t.Fatalf("limit %d page %d: foreign or duplicate entry %s", limit, page, e.GetId())
				}
				seen[e.GetId()] = true
			}
			if int32(len(resp.GetData())) < limit {
				break
			}
		}
		if len(seen) != n {
			t.Fatalf("limit %d: returned %d of %d entries", limit, len(seen), n)
		}
	}
	// Band picker source: ListScoreScaleBands now honours pagination too.
	bandRepo := fix3RegisteredRepo[bandpb.ScoreScaleBandDomainServiceServer](t, db, entityid.ScoreScaleBand)
	bandSeen := map[string]bool{}
	for page := int32(1); page <= 10; page++ {
		resp, err := bandRepo.ListScoreScaleBands(ctx, &bandpb.ListScoreScaleBandsRequest{
			Filters: fix3EqualsFilter("score_scale_id", scale), Sort: fix3IDSort(), Pagination: fix3Page(5, page)})
		if err != nil {
			t.Fatalf("bands page %d: %v", page, err)
		}
		for _, b := range resp.GetData() {
			if bandSeen[b.GetId()] {
				t.Fatalf("bands page %d: duplicate %s", page, b.GetId())
			}
			bandSeen[b.GetId()] = true
		}
		if len(resp.GetData()) < 5 {
			break
		}
	}
	if len(bandSeen) != len(bands) {
		t.Fatalf("paged %d of %d bands", len(bandSeen), len(bands))
	}

	_, pg, err := repo.ListRatingDescriptionSetEntriesPage(ctx, &entrypb.ListRatingDescriptionSetEntriesRequest{
		Filters: fix3EqualsFilter("rating_description_set_id", set), Sort: fix3IDSort(), Pagination: fix3Page(100, 2)})
	if err != nil || int(pg.GetTotalItems()) != n || pg.GetHasNext() || !pg.GetHasPrev() {
		t.Fatalf("page 2/100: total=%d has_next=%v has_prev=%v err=%v", pg.GetTotalItems(), pg.GetHasNext(), pg.GetHasPrev(), err)
	}
}

// --- #4 audit -------------------------------------------------------------

// TestIntegrationFix3_CreateSetThroughUseCaseAndRegisteredFactoryAuditsWorkspace
// runs the REAL CreateRatingDescriptionSet use case over the REAL registered
// (audited) factory and the real transactor (joined to an outer rolled-back
// tx): the generic PostgresOperations.Create diff event carries the trusted
// workspace id and lands in the same transaction as the insert.
func TestIntegrationFix3_CreateSetThroughUseCaseAndRegisteredFactoryAuditsWorkspace(t *testing.T) {
	db := openW3LifecycleIntegrationDB(t)
	txCtx, tx := fix3StartTx(t, db, false)
	ws := fix3UUID(2)
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	scale := "fix3-it-audit-scale-" + suffix
	fix3Seed(t, txCtx, tx, ws, scale, "fix3-it-audit-seedset-"+suffix, 1, 1)

	repo := fix3RegisteredRepo[setpb.RatingDescriptionSetDomainServiceServer](t, db, entityid.RatingDescriptionSet)
	uc := setuc.NewCreateRatingDescriptionSetUseCase(setuc.CreateRatingDescriptionSetRepositories{RatingDescriptionSet: repo}, setuc.CreateRatingDescriptionSetServices{
		Authorizer: ports.NewNoOpAuthorizer(), Transactor: fix3Transactor(db), Translator: ports.NewNoOpTranslator(), ActionGatekeeper: fix3Gatekeeper(),
	})
	ctx := identity.WithRequestIdentity(txCtx, &identity.RequestIdentity{UserID: "fix3-it-actor", WorkspaceID: ws})
	setID := "fix3-it-audit-set-" + suffix
	resp, err := uc.Execute(ctx, &setpb.CreateRatingDescriptionSetRequest{Data: &setpb.RatingDescriptionSet{Id: setID, Name: "fix3 audited", ScoreScaleId: scale}})
	if err != nil || len(resp.GetData()) != 1 {
		t.Fatalf("create: resp=%v err=%v", resp, err)
	}
	var auditWS sql.NullString
	var fields int
	if err := tx.QueryRowContext(txCtx, `SELECT ae.workspace_id::text, ae.field_count FROM audit_trail.audit_entry ae
		WHERE ae.entity_type = $1 AND ae.entity_id = $2 AND ae.method_name = 'PostgresOperations.Create'`,
		entityid.RatingDescriptionSet, setID).Scan(&auditWS, &fields); err != nil {
		t.Fatalf("generic create audit row missing (registered factory not audited?): %v", err)
	}
	if !auditWS.Valid || auditWS.String != ws {
		t.Fatalf("audit workspace_id = %v, want trusted %s", auditWS, ws)
	}
	if fields == 0 {
		t.Fatal("generic create audit row has no field changes")
	}
}

// TestIntegrationFix3_CreateSetAuditFailureLeavesNoRow: top-level use-case
// call (its OWN transaction) against existing clone rows (workspace + an
// active owned score scale); the audit backend fails, so the insert must roll
// back — a fresh read afterwards finds no set.
func TestIntegrationFix3_CreateSetAuditFailureLeavesNoRow(t *testing.T) {
	db := openW3LifecycleIntegrationDB(t)
	var ws, scale string
	if err := db.QueryRow(`SELECT s.workspace_id, s.id FROM `+entityid.ScoreScale+` s
		WHERE s.active AND s.workspace_id ~ '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$'
		ORDER BY s.id LIMIT 1`).Scan(&ws, &scale); err != nil {
		t.Skipf("no owned active score scale on this database: %v", err)
	}
	failing := &failingAuditService{}
	repo := NewPostgresRatingDescriptionSetRepository(postgresCore.NewAuditedWorkspaceAwareOperations(db, failing), "")
	uc := setuc.NewCreateRatingDescriptionSetUseCase(setuc.CreateRatingDescriptionSetRepositories{RatingDescriptionSet: repo}, setuc.CreateRatingDescriptionSetServices{
		Authorizer: ports.NewNoOpAuthorizer(), Transactor: fix3Transactor(db), Translator: ports.NewNoOpTranslator(), ActionGatekeeper: fix3Gatekeeper(),
	})
	setID := fmt.Sprintf("fix3-it-auditfail-%d", time.Now().UnixNano())
	t.Cleanup(func() {
		var residue int
		_ = db.QueryRow(`SELECT count(*) FROM `+ratingDescriptionSetTable+` WHERE id = $1`, setID).Scan(&residue)
		if residue != 0 {
			t.Errorf("RESIDUE: set %s persisted on the clone — remove it manually", setID)
		}
	})
	ctx := identity.WithRequestIdentity(context.Background(), &identity.RequestIdentity{UserID: "fix3-it-actor", WorkspaceID: ws})
	_, err := uc.Execute(ctx, &setpb.CreateRatingDescriptionSetRequest{Data: &setpb.RatingDescriptionSet{Id: setID, Name: "fix3 audit fail", ScoreScaleId: scale}})
	if err == nil || !strings.Contains(err.Error(), "simulated audit backend failure") || failing.calls == 0 {
		t.Fatalf("want the audit failure to surface, got %v (calls %d)", err, failing.calls)
	}
	var n int
	if err := db.QueryRow(`SELECT count(*) FROM `+ratingDescriptionSetTable+` WHERE id = $1`, setID).Scan(&n); err != nil || n != 0 {
		t.Fatalf("set must not persist after an audit failure: count=%d err=%v", n, err)
	}
}

// --- #6 publish verification ---------------------------------------------

// TestIntegrationFix3_PublishRejectsEntryWithInactiveOrUnusableBand: DRAFT
// entry → its band deleted through the real band repository (allowed: only
// a DRAFT set references it) → the real Publish use case is rejected with
// INVALID_CONFIG and the set stays DRAFT; same for a band re-roled to
// no_description. Positive control: the untouched set verifies.
func TestIntegrationFix3_PublishRejectsEntryWithInactiveOrUnusableBand(t *testing.T) {
	db := openW3LifecycleIntegrationDB(t)
	txCtx, tx := fix3StartTx(t, db, false)
	ws := fix3UUID(3)
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	scale, set, set2 := "fix3-it-pub-scale-"+suffix, "fix3-it-pub-set-"+suffix, "fix3-it-pub-set2-"+suffix
	bands, criteria := fix3Seed(t, txCtx, tx, ws, scale, set, 2, 1)
	if _, err := tx.ExecContext(txCtx, `INSERT INTO `+ratingDescriptionSetTable+`
		(id, workspace_id, version, version_status, name, score_scale_id, active)
		VALUES ($1, $2, 1, 'VERSION_STATUS_DRAFT', 'fix3 set2', $3, true)`, set2, ws, scale); err != nil {
		t.Fatal(err)
	}
	for i, pair := range [][2]string{{set, bands[0]}, {set2, bands[1]}} {
		if _, err := tx.ExecContext(txCtx, `INSERT INTO `+ratingDescriptionSetEntryTable+`
			(id, workspace_id, rating_description_set_id, outcome_criteria_id, score_scale_band_id, description, active)
			VALUES ($1, $2, $3, $4, $5, 'desc', true)`, fmt.Sprintf("fix3-it-pub-e%d-%s", i, suffix), ws, pair[0], criteria[0], pair[1]); err != nil {
			t.Fatal(err)
		}
	}
	ctx := identity.WithRequestIdentity(txCtx, &identity.RequestIdentity{UserID: "fix3-it-actor", WorkspaceID: ws})

	setRepo := fix3RegisteredRepo[*PostgresRatingDescriptionSetRepository](t, db, entityid.RatingDescriptionSet)
	entryRepo := fix3RegisteredRepo[entrypb.RatingDescriptionSetEntryDomainServiceServer](t, db, entityid.RatingDescriptionSetEntry)
	bandRepo := NewPostgresScoreScaleBandRepository(postgresCore.NewAuditedWorkspaceAwareOperations(db, auditadapter.New(db)), "").(*PostgresScoreScaleBandRepository)
	publish := setuc.NewPublishRatingDescriptionSetUseCase(setRepo, entryRepo, setuc.PublishRatingDescriptionSetServices{
		Authorizer: ports.NewNoOpAuthorizer(), Transactor: fix3Transactor(db), Translator: ports.NewNoOpTranslator(), ActionGatekeeper: fix3Gatekeeper(),
	})

	// Positive control (adapter verify only — no publish yet).
	if err := setRepo.VerifyRatingDescriptionSetPublishScale(ctx, set, scale); err != nil {
		t.Fatalf("control: clean draft set must verify: %v", err)
	}
	// Delete the band while its only referencing set is DRAFT → allowed.
	if _, err := bandRepo.DeleteScoreScaleBand(ctx, &bandpb.DeleteScoreScaleBandRequest{Data: &bandpb.ScoreScaleBand{Id: bands[0]}}); err != nil {
		t.Fatalf("delete band referenced only by a DRAFT set: %v", err)
	}
	if _, err := publish.Execute(ctx, &setpb.PublishRatingDescriptionSetRequest{Id: set}); err == nil || !strings.Contains(err.Error(), "INVALID_CONFIG") {
		t.Fatalf("publish with an inactive-band entry: want INVALID_CONFIG, got %v", err)
	}
	var status string
	if err := tx.QueryRowContext(txCtx, `SELECT version_status FROM `+ratingDescriptionSetTable+` WHERE id = $1`, set).Scan(&status); err != nil || status != ratingDescriptionSetVersionStatusDraft {
		t.Fatalf("rejected publish must leave the set DRAFT: %q err %v", status, err)
	}

	// Unusable band: re-roled to no_description after the DRAFT entry was written.
	if _, err := tx.ExecContext(txCtx, `UPDATE `+entityid.ScoreScaleBand+` SET band_role = $2 WHERE id = $1`, bands[1], noDescriptionBandRole); err != nil {
		t.Fatal(err)
	}
	if _, err := publish.Execute(ctx, &setpb.PublishRatingDescriptionSetRequest{Id: set2}); err == nil || !strings.Contains(err.Error(), "INVALID_CONFIG") {
		t.Fatalf("publish with a no_description-band entry: want INVALID_CONFIG, got %v", err)
	}
	// Restore → the same set publishes (proves the rejection was the band).
	if _, err := tx.ExecContext(txCtx, `UPDATE `+entityid.ScoreScaleBand+` SET band_role = NULL WHERE id = $1`, bands[1]); err != nil {
		t.Fatal(err)
	}
	if _, err := publish.Execute(ctx, &setpb.PublishRatingDescriptionSetRequest{Id: set2}); err != nil {
		t.Fatalf("publish with a usable active band: %v", err)
	}
}
