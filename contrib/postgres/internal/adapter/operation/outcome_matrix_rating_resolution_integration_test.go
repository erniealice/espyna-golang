//go:build postgresql

package operation

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"
	"testing"
	"time"

	_ "github.com/lib/pq"

	enumspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/enums"
	matrixpb "github.com/erniealice/esqyma/pkg/schema/v1/service/operation/outcome_matrix"

	"github.com/erniealice/espyna-golang/registry/entityid"
	"github.com/erniealice/espyna-golang/shared/identity"
)

// openResolveCellRatingDescriptionsDB is TEST_DATABASE_URL-gated, same
// convention as the sibling outcome_matrix_rating_description_integration_test.go.
func openResolveCellRatingDescriptionsDB(t *testing.T) *sql.DB {
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
		"SELECT to_regclass('public.rating_description_set_product_plan')::text").Scan(&table); err != nil || !table.Valid {
		db.Close()
		t.Skip("rating_description_set_product_plan not present; run against the expanded schema (PD 20260925 migration)")
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// -----------------------------------------------------------------------
// Live probe: real loaded data (PD 20260925-criterion-descriptors-by-
// program-year, sequence/DATA-LOADED.done). Read-only — proves the actual
// production SQL end to end against the "Design" (job_template
// 019f6dfe-62e6-799d-a035-9a77f1b2df64 / section subscription_group
// 019f82fb-4b0c-7d2b-a254-60f7e53d4b1a) rating set, matching the resolution
// the w1-schema-data agent's sql-b4/31_resolve_probe.sql independently found
// (every job on that template resolves to set code "ib-myp-design-y5", 8
// bands, no level-0 entry — sequence/w1-schema-data-*.md STEP 5).
// -----------------------------------------------------------------------

const (
	liveDesignWorkspaceID = "019ecb8e-d83f-74ab-aa13-5a6c27afd112" // MMIS
	liveDesignTemplateID  = "019f6dfe-62e6-799d-a035-9a77f1b2df64" // Design x Grade 10
	liveDesignSectionID   = "019f82fb-4b0c-7d2b-a254-60f7e53d4b1a" // subscription_group
)

// TestIntegrationResolveCellRatingDescriptions_LiveDesignOffering discovers a
// couple of real description-mode cells on the live loaded Design template
// and resolves them through the actual adapter method (public entrypoint,
// including the identity.FromContext workspace gate) — not a synthetic
// fixture. It is read-only (SELECT only, no fixture rows written).
func TestIntegrationResolveCellRatingDescriptions_LiveDesignOffering(t *testing.T) {
	db := openResolveCellRatingDescriptionsDB(t)
	ctx := context.Background()

	var wsFound sql.NullString
	if err := db.QueryRowContext(ctx, "SELECT id FROM workspace WHERE id = $1", liveDesignWorkspaceID).Scan(&wsFound); err != nil {
		t.Skipf("live MMIS workspace fixture (%s) not present in this DB: %v", liveDesignWorkspaceID, err)
	}

	rows, err := db.QueryContext(ctx, `
		SELECT j.id, jt.id, ttc.outcome_criteria_id
		FROM `+entityid.Job+` j
		JOIN `+entityid.JobPhase+` jp ON jp.job_id = j.id AND jp.active
		JOIN `+entityid.JobTask+` jt ON jt.job_phase_id = jp.id AND jt.active
		JOIN `+entityid.JobTemplateTask+` jtt ON jtt.id = jt.template_task_id AND jtt.active
		JOIN `+entityid.TemplateTaskCriteria+` ttc
			ON ttc.job_template_task_id = jtt.id AND ttc.active
			AND ttc.rating_mode = 'RATING_MODE_NUMERIC_WITH_DESCRIPTION'
		JOIN `+entityid.SubscriptionGroupMember+` sgm
			ON sgm.subscription_id = j.origin_id AND sgm.subscription_group_id = $2 AND sgm.active
		WHERE j.job_template_id = $1 AND j.active
		LIMIT 3`, liveDesignTemplateID, liveDesignSectionID)
	if err != nil {
		t.Fatalf("discovery query: %v", err)
	}
	var cells []*matrixpb.CellRatingRef
	for rows.Next() {
		var jobID, jobTaskID, criteriaID string
		if err := rows.Scan(&jobID, &jobTaskID, &criteriaID); err != nil {
			rows.Close()
			t.Fatalf("discovery scan: %v", err)
		}
		cells = append(cells, &matrixpb.CellRatingRef{JobId: jobID, JobTaskId: jobTaskID, OutcomeCriteriaId: criteriaID})
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		t.Fatalf("discovery rows: %v", err)
	}
	if len(cells) == 0 {
		t.Skip("no NUMERIC_WITH_DESCRIPTION cells found on the live Design template/section — data load may not have run")
	}

	a := NewPostgresOutcomeMatrixQuery(db).(*PostgresOutcomeMatrixQuery)
	ctx = identity.WithRequestIdentity(ctx, &identity.RequestIdentity{WorkspaceID: liveDesignWorkspaceID})

	resp, err := a.ResolveCellRatingDescriptions(ctx, &matrixpb.ResolveCellRatingDescriptionsRequest{Cells: cells})
	if err != nil {
		t.Fatalf("ResolveCellRatingDescriptions: %v", err)
	}
	if !resp.GetSuccess() {
		t.Fatalf("expected success, got %+v", resp.GetError())
	}
	if len(resp.GetResults()) != len(cells) {
		t.Fatalf("want exactly one result per requested cell (RD-66), got %d for %d cells", len(resp.GetResults()), len(cells))
	}

	var setID string
	for i, r := range resp.GetResults() {
		if r.GetStatus() != enumspb.RatingDescriptionResolutionStatus_RATING_DESCRIPTION_RESOLUTION_STATUS_RESOLVED {
			t.Errorf("cell %d status = %v, want RESOLVED (reason=%q)", i, r.GetStatus(), r.GetReason())
			continue
		}
		if r.GetRatingDescriptionSetId() == "" {
			t.Errorf("cell %d: RESOLVED but no rating_description_set_id", i)
		}
		if setID == "" {
			setID = r.GetRatingDescriptionSetId()
		} else if r.GetRatingDescriptionSetId() != setID {
			t.Errorf("cell %d resolved to a different set (%s) than cell 0 (%s); every job on this template/section shares one product_plan x AY link per w1-schema-data STEP 5", i, r.GetRatingDescriptionSetId(), setID)
		}
		if len(r.GetDescriptions()) != 8 {
			t.Errorf("cell %d: got %d descriptions, want 8 (levels 1-8, no level-0 entry per Q20)", i, len(r.GetDescriptions()))
		}
	}

	if setID != "" {
		var code sql.NullString
		if err := db.QueryRowContext(ctx, `SELECT code FROM `+ratingDescriptionSetTable+` WHERE id = $1`, setID).Scan(&code); err != nil {
			t.Fatalf("verify resolved set code: %v", err)
		}
		if code.String != "ib-myp-design-y5" {
			t.Errorf("resolved set code = %q, want %q (Design x Grade 10 -> program_year Year 5)", code.String, "ib-myp-design-y5")
		}
	}
}

// -----------------------------------------------------------------------
// Rollback-only fixture: every server status this resolver can produce.
//
// AMBIGUOUS is deliberately NOT exercised here: both real unique indexes it
// would require violating are enforced by this schema —
// uq_product_plan_plan_product_variant (plan_id, product_id,
// COALESCE(product_variant_id,'')) and
// uq_rating_description_set_product_plan_active_pair (product_plan_id,
// price_schedule_id) WHERE active — so ">1 active match" is a genuine
// integrity-failure-only path, unreachable through normal inserts. The
// AMBIGUOUS branches are still exercised at the Go level by construction
// (resolveProductPlans/resolveRatingLinks count every matching row, not just
// assume uniqueness).
// -----------------------------------------------------------------------

func TestIntegrationResolveCellRatingDescriptions_FixtureStatuses(t *testing.T) {
	db := openResolveCellRatingDescriptionsDB(t)
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatalf("begin rollback-only fixture transaction: %v", err)
	}
	t.Cleanup(func() { _ = tx.Rollback() })
	ctx := context.Background()

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	id := func(name string) string { return "rdrq-it-" + name + "-" + suffix }

	wsID := id("ws")
	foreignWsID := id("ws-foreign")
	planID := id("plan")
	scheduleID := id("schedule")
	subID := id("sub")
	pricePlanID := id("price-plan")
	templateID := id("template")
	templatePhaseID := id("template-phase")
	templateTaskID := id("template-task")
	criteriaID := id("criteria")
	criteriaStdID := id("criteria-std")
	ttcID := id("ttc")
	ttcStdID := id("ttc-std")
	scaleGroupID := id("scale-group")
	scaleID := id("scale")
	bandID := id("band")

	// A second scale + band, used only by the "entry band on a different
	// scale" fixture below (codex round 1 item 3) — never linked to any
	// offering itself, so it cannot resolve on its own.
	scaleOtherGroupID := id("scale-group-other")
	scaleOtherID := id("scale-other")
	bandOtherScaleID := id("band-other-scale")

	productCivics := id("product-civics")
	productCLF := id("product-clf")
	productVariantBase := id("product-variant-base")
	productNoLink := id("product-nolink")
	productGhost := id("product-ghost")
	productDraft := id("product-draft")
	productForeign := id("product-foreign")
	productDeprecated := id("product-deprecated")
	productEntryForeignWs := id("product-entry-foreign-ws")
	productEntryWrongScale := id("product-entry-wrong-scale")
	variantAID := id("variant-a")
	variantBID := id("variant-b")

	ppCivics := id("pp-civics")
	ppCLF := id("pp-clf")
	ppVariantA := id("pp-variant-a")
	ppVariantB := id("pp-variant-b")
	ppNoLink := id("pp-nolink")
	ppDraft := id("pp-draft")
	ppForeign := id("pp-foreign")
	ppDeprecated := id("pp-deprecated")
	ppEntryForeignWs := id("pp-entry-foreign-ws")
	ppEntryWrongScale := id("pp-entry-wrong-scale")

	setPublished := id("set-published")
	setPublishedVariantA := id("set-published-variant-a")
	setPublishedVariantB := id("set-published-variant-b")
	setDraft := id("set-draft")
	setForeign := id("set-foreign")
	setDeprecated := id("set-deprecated")
	setEntryForeignWs := id("set-entry-foreign-ws")
	setEntryWrongScale := id("set-entry-wrong-scale")

	job1 := id("job1-civics")
	job2 := id("job2-clf")
	job3 := id("job3-variant-a")
	job4 := id("job4-variant-b")
	job5 := id("job5-nolink")
	job6 := id("job6-ghost")
	job7 := id("job7-draft")
	job8 := id("job8-foreign")
	job9 := id("job9-deprecated")
	job10 := id("job10-entry-foreign-ws")
	job11 := id("job11-entry-wrong-scale")

	exec := func(query string, args ...any) {
		t.Helper()
		if _, err := tx.ExecContext(ctx, query, args...); err != nil {
			t.Fatalf("fixture statement %q: %v", query, err)
		}
	}

	exec(`INSERT INTO workspace (id, name, active) VALUES ($1, 'rdrq resolver it', true)`, wsID)
	exec(`INSERT INTO workspace (id, name, active) VALUES ($1, 'rdrq resolver it (foreign)', true)`, foreignWsID)
	exec(`INSERT INTO plan (id, active) VALUES ($1, true)`, planID)
	exec(`INSERT INTO `+entityid.PriceSchedule+` (id, active) VALUES ($1, true)`, scheduleID)
	exec(`INSERT INTO price_plan (id, plan_id, price_schedule_id, active) VALUES ($1, $2, $3, true)`, pricePlanID, planID, scheduleID)
	exec(`INSERT INTO subscription (id, price_plan_id, workspace_id, active) VALUES ($1, $2, $3, true)`, subID, pricePlanID, wsID)

	exec(`INSERT INTO `+entityid.JobTemplate+` (id, name, active, workspace_id) VALUES ($1, 'rdrq template', true, $2)`, templateID, wsID)
	exec(`INSERT INTO `+entityid.JobTemplatePhase+` (id, job_template_id, name, phase_order, active, workspace_id) VALUES ($1, $2, 'Term 1', 1, true, $3)`, templatePhaseID, templateID, wsID)
	exec(`INSERT INTO `+entityid.JobTemplateTask+` (id, job_template_phase_id, name, step_order, active, workspace_id) VALUES ($1, $2, 'Task A', 1, true, $3)`, templateTaskID, templatePhaseID, wsID)

	exec(`INSERT INTO `+entityid.OutcomeCriteria+`
		(id, name, criteria_type, min_score, max_score, score_increment, required, active, workspace_id)
		VALUES ($1, 'Criterion (description mode)', 'CRITERIA_TYPE_NUMERIC_SCORE', 0, 8, 1, true, true, $2),
		       ($3, 'Criterion (standard mode)',    'CRITERIA_TYPE_NUMERIC_SCORE', 0, 8, 1, true, true, $2)`,
		criteriaID, wsID, criteriaStdID)

	exec(`INSERT INTO `+entityid.ScoreScale+`
		(id, scale_group_id, version, version_status, name, scale_kind, input_unit, output_unit, workspace_id, active, created_by)
		VALUES ($1, $2, 1, 'VERSION_STATUS_PUBLISHED', 'rdrq resolver scale', 'SCALE_KIND_EXACT_MAP', 'score', 'label', $3, true, 'rdrq-it')`,
		scaleID, scaleGroupID, wsID)
	exec(`INSERT INTO `+entityid.ScoreScaleBand+`
		(id, workspace_id, active, score_scale_id, sequence_order, input_match, output_value, output_label)
		VALUES ($1, $2, true, $3, 1, 'shared-band', 1, 'Shared Band')`,
		bandID, wsID, scaleID)

	// A second, unrelated scale + band (item 3, "entry band on a different
	// scale"): setEntryWrongScale below correctly declares score_scale_id =
	// scaleID (so it passes resolveRatingSets' slot-scale check), but its
	// entry row references bandOtherScaleID, which belongs to THIS scale —
	// an integrity mismatch resolveRatingEntries must now catch.
	exec(`INSERT INTO `+entityid.ScoreScale+`
		(id, scale_group_id, version, version_status, name, scale_kind, input_unit, output_unit, workspace_id, active, created_by)
		VALUES ($1, $2, 1, 'VERSION_STATUS_PUBLISHED', 'rdrq resolver scale (other)', 'SCALE_KIND_EXACT_MAP', 'score', 'label', $3, true, 'rdrq-it')`,
		scaleOtherID, scaleOtherGroupID, wsID)
	exec(`INSERT INTO `+entityid.ScoreScaleBand+`
		(id, workspace_id, active, score_scale_id, sequence_order, input_match, output_value, output_label)
		VALUES ($1, $2, true, $3, 1, 'other-band', 1, 'Other Scale Band')`,
		bandOtherScaleID, wsID, scaleOtherID)

	exec(`INSERT INTO `+entityid.TemplateTaskCriteria+`
		(id, job_template_task_id, outcome_criteria_id, sequence_order, active, workspace_id, rating_mode, rating_scale_id)
		VALUES ($1, $2, $3, 1, true, $4, 'RATING_MODE_NUMERIC_WITH_DESCRIPTION', $5),
		       ($6, $2, $7, 2, true, $4, 'RATING_MODE_STANDARD', NULL)`,
		ttcID, templateTaskID, criteriaID, wsID, scaleID, ttcStdID, criteriaStdID)

	// Products (+2 variants of the shared "variant base" product).
	for _, p := range []string{productCivics, productCLF, productVariantBase, productNoLink, productGhost, productDraft, productForeign, productDeprecated, productEntryForeignWs, productEntryWrongScale} {
		exec(`INSERT INTO product (id, active) VALUES ($1, true)`, p)
	}
	exec(`INSERT INTO product_variant (id, product_id, active) VALUES ($1, $2, true), ($3, $2, true)`, variantAID, productVariantBase, variantBID)

	// product_plan offerings. productGhost gets NO row (proves the
	// "0 product_plans -> UNRESOLVED_IDENTITY" branch).
	exec(`INSERT INTO `+entityid.ProductPlan+` (id, active, product_id, plan_id, product_variant_id) VALUES
		($1, true, $2, $3, NULL),
		($4, true, $5, $3, NULL),
		($6, true, $7, $3, $8),
		($9, true, $7, $3, $10),
		($11, true, $12, $3, NULL),
		($13, true, $14, $3, NULL),
		($15, true, $16, $3, NULL),
		($17, true, $18, $3, NULL),
		($19, true, $20, $3, NULL),
		($21, true, $22, $3, NULL)`,
		ppCivics, productCivics, planID,
		ppCLF, productCLF,
		ppVariantA, productVariantBase, variantAID,
		ppVariantB, variantBID,
		ppNoLink, productNoLink,
		ppDraft, productDraft,
		ppForeign, productForeign,
		ppDeprecated, productDeprecated,
		ppEntryForeignWs, productEntryForeignWs,
		ppEntryWrongScale, productEntryWrongScale)

	// rating_description_set rows: PUBLISHED (shared by 2 offerings, proving
	// Civics/CLF sharing), 2 more PUBLISHED sets distinguishing the 2 variant
	// offerings (variant isolation), DRAFT (never resolves — Q19),
	// foreign-workspace PUBLISHED (INVALID_CONFIG regardless of status), and
	// DEPRECATED (still resolves — Q19).
	exec(`INSERT INTO `+ratingDescriptionSetTable+`
		(id, workspace_id, version, version_status, name, score_scale_id, active) VALUES
		($1, $2, 1, 'VERSION_STATUS_PUBLISHED', 'Shared Civics/CLF set', $3, true),
		($4, $2, 1, 'VERSION_STATUS_PUBLISHED', 'Variant A set', $3, true),
		($5, $2, 1, 'VERSION_STATUS_PUBLISHED', 'Variant B set', $3, true),
		($6, $2, 1, 'VERSION_STATUS_DRAFT', 'Draft set (never resolves)', $3, true),
		($7, $8, 1, 'VERSION_STATUS_PUBLISHED', 'Foreign-workspace set', $3, true),
		($9, $2, 1, 'VERSION_STATUS_DEPRECATED', 'Deprecated set (still resolves)', $3, true),
		($10, $2, 1, 'VERSION_STATUS_PUBLISHED', 'Entry foreign-workspace set', $3, true),
		($11, $2, 1, 'VERSION_STATUS_PUBLISHED', 'Entry wrong-scale set', $3, true)`,
		setPublished, wsID, scaleID,
		setPublishedVariantA,
		setPublishedVariantB,
		setDraft,
		setForeign, foreignWsID,
		setDeprecated,
		setEntryForeignWs,
		setEntryWrongScale)

	exec(`INSERT INTO `+ratingDescriptionSetEntryTable+`
		(id, workspace_id, rating_description_set_id, outcome_criteria_id, score_scale_band_id, description, active) VALUES
		($1, $2, $3, $4, $5, 'Shared Civics/CLF text', true),
		($6, $2, $7, $4, $5, 'Variant A text', true),
		($8, $2, $9, $4, $5, 'Variant B text', true),
		($10, $2, $11, $4, $5, 'Deprecated but resolves text', true),
		($12, $13, $14, $4, $5, 'Entry belongs to a foreign workspace', true),
		($15, $2, $16, $4, $17, 'Entry band belongs to a different scale', true)`,
		id("entry-shared"), wsID, setPublished, criteriaID, bandID,
		id("entry-variant-a"), setPublishedVariantA,
		id("entry-variant-b"), setPublishedVariantB,
		id("entry-deprecated"), setDeprecated,
		// Item 3 case A: the ENTRY row's own workspace_id is foreignWsID even
		// though its parent set (setEntryForeignWs) is correctly wsID —
		// simulates a cross-tenant import/backfill bug on the entry itself.
		id("entry-foreign-ws"), foreignWsID, setEntryForeignWs,
		// Item 3 case B: the ENTRY row's workspace_id is correct (wsID), and
		// its parent set correctly declares score_scale_id = scaleID, but the
		// entry's score_scale_band_id (bandOtherScaleID) belongs to the
		// UNRELATED scaleOtherID — an entry->band->scale drift no FK forbids.
		id("entry-wrong-scale"), setEntryWrongScale, bandOtherScaleID)

	// Links (rating_description_set_product_plan). ppNoLink deliberately gets
	// NO link row (proves NO_LINK). The foreign-set link is itself correctly
	// workspace-scoped to wsID — only the SET it points at is foreign, which
	// is exactly the integrity failure INVALID_CONFIG exists to catch. The
	// two new item-3 links are themselves entirely valid (correct workspace,
	// correct scale) — only their SET's ENTRY row is corrupt, proving the
	// integrity check lives in resolveRatingEntries, not resolveRatingSets.
	exec(`INSERT INTO `+ratingDescriptionSetProductPlanTable+`
		(id, workspace_id, rating_description_set_id, product_plan_id, price_schedule_id, active) VALUES
		($1, $2, $3, $4, $5, true),
		($6, $2, $3, $7, $5, true),
		($8, $2, $9, $10, $5, true),
		($11, $2, $12, $13, $5, true),
		($14, $2, $15, $16, $5, true),
		($17, $2, $18, $19, $5, true),
		($20, $2, $21, $22, $5, true),
		($23, $2, $24, $25, $5, true),
		($26, $2, $27, $28, $5, true)`,
		id("link-civics"), wsID, setPublished, ppCivics, scheduleID,
		id("link-clf"), ppCLF,
		id("link-variant-a"), setPublishedVariantA, ppVariantA,
		id("link-variant-b"), setPublishedVariantB, ppVariantB,
		id("link-draft"), setDraft, ppDraft,
		id("link-foreign"), setForeign, ppForeign,
		id("link-deprecated"), setDeprecated, ppDeprecated,
		id("link-entry-foreign-ws"), setEntryForeignWs, ppEntryForeignWs,
		id("link-entry-wrong-scale"), setEntryWrongScale, ppEntryWrongScale)

	// Job instances — one per scenario, all sharing templateID/subID so only
	// output_product_id/output_product_variant_id vary the resolution.
	jobIDs := []string{job1, job2, job3, job4, job5, job6, job7, job8, job9, job10, job11}
	outputs := [][2]string{
		{productCivics, ""}, {productCLF, ""},
		{productVariantBase, variantAID}, {productVariantBase, variantBID},
		{productNoLink, ""}, {productGhost, ""},
		{productDraft, ""}, {productForeign, ""}, {productDeprecated, ""},
		{productEntryForeignWs, ""}, {productEntryWrongScale, ""},
	}
	jobTaskByJob := make(map[string]string, len(jobIDs))
	for i, jobID := range jobIDs {
		phaseRowID := id(fmt.Sprintf("phase-%d", i))
		taskRowID := id(fmt.Sprintf("task-%d", i))
		variant := outputs[i][1]
		var variantArg any
		if variant == "" {
			variantArg = nil
		} else {
			variantArg = variant
		}
		exec(`INSERT INTO `+entityid.Job+`
			(id, job_template_id, origin_id, workspace_id, output_product_id, output_product_variant_id, active)
			VALUES ($1, $2, $3, $4, $5, $6, true)`,
			jobID, templateID, subID, wsID, outputs[i][0], variantArg)
		// workspace_id is set on job_phase/job_task too (codex round 1 item 3:
		// the ancestry query now also requires these to equal the caller's
		// workspace, not just the job itself).
		exec(`INSERT INTO `+entityid.JobPhase+` (id, job_id, template_phase_id, active, workspace_id) VALUES ($1, $2, $3, true, $4)`,
			phaseRowID, jobID, templatePhaseID, wsID)
		exec(`INSERT INTO `+entityid.JobTask+` (id, job_phase_id, template_task_id, active, workspace_id) VALUES ($1, $2, $3, true, $4)`,
			taskRowID, phaseRowID, templateTaskID, wsID)
		jobTaskByJob[jobID] = taskRowID
	}

	cell := func(jobID, taskJobID, criteriaID string) *matrixpb.CellRatingRef {
		return &matrixpb.CellRatingRef{JobId: jobID, JobTaskId: jobTaskByJob[taskJobID], OutcomeCriteriaId: criteriaID}
	}
	cells := []*matrixpb.CellRatingRef{
		cell(job1, job1, criteriaID),    // 0: RESOLVED (Civics)
		cell(job2, job2, criteriaID),    // 1: RESOLVED (CLF) -- same set as 0
		cell(job3, job3, criteriaID),    // 2: RESOLVED (variant A)
		cell(job4, job4, criteriaID),    // 3: RESOLVED (variant B) -- different set than 2
		cell(job5, job5, criteriaID),    // 4: NO_LINK
		cell(job6, job6, criteriaID),    // 5: UNRESOLVED_IDENTITY (0 product_plans)
		cell(job2, job1, criteriaID),    // 6: UNRESOLVED_IDENTITY (job_task belongs to job1, not job2)
		cell(job7, job7, criteriaID),    // 7: INVALID_CONFIG (DRAFT set)
		cell(job8, job8, criteriaID),    // 8: INVALID_CONFIG (foreign-workspace set)
		cell(job9, job9, criteriaID),    // 9: RESOLVED (DEPRECATED set still resolves)
		cell(job1, job1, criteriaStdID), // 10: UNSPECIFIED (not NUMERIC_WITH_DESCRIPTION)
		cell(job10, job10, criteriaID),  // 11: INVALID_CONFIG (entry belongs to a foreign workspace)
		cell(job11, job11, criteriaID),  // 12: INVALID_CONFIG (entry band on a different scale)
	}

	a := &PostgresOutcomeMatrixQuery{}
	counter := &countingQueryer{q: tx}
	resp, err := a.resolveCellRatingDescriptionsFrom(ctx, counter, cells, wsID)
	if err != nil {
		t.Fatalf("resolveCellRatingDescriptionsFrom: %v", err)
	}
	if len(resp.GetResults()) != len(cells) {
		t.Fatalf("want exactly one result per requested cell (RD-66), got %d for %d cells", len(resp.GetResults()), len(cells))
	}
	// Bounded query count (rule 6 / RD-41): exactly 5 batch queries — jobs+
	// ancestry+slot-mode, product_plans, links, sets, entries — REGARDLESS of
	// how many cells (11) were requested.
	if counter.n != 5 {
		t.Errorf("query count = %d, want exactly 5 (bounded, one per resolution stage)", counter.n)
	}

	results := resp.GetResults()
	assertStatus := func(i int, want enumspb.RatingDescriptionResolutionStatus, label string) *matrixpb.CellRatingResolution {
		t.Helper()
		r := results[i]
		if r.GetStatus() != want {
			t.Errorf("%s: status = %v, want %v (reason=%q)", label, r.GetStatus(), want, r.GetReason())
		}
		return r
	}

	r0 := assertStatus(0, enumspb.RatingDescriptionResolutionStatus_RATING_DESCRIPTION_RESOLUTION_STATUS_RESOLVED, "civics")
	r1 := assertStatus(1, enumspb.RatingDescriptionResolutionStatus_RATING_DESCRIPTION_RESOLUTION_STATUS_RESOLVED, "clf")
	if r0.GetRatingDescriptionSetId() != setPublished || r1.GetRatingDescriptionSetId() != setPublished {
		t.Errorf("civics/CLF must share ONE set: civics=%s clf=%s want %s", r0.GetRatingDescriptionSetId(), r1.GetRatingDescriptionSetId(), setPublished)
	}
	if len(r0.GetDescriptions()) != 1 || r0.GetDescriptions()[0].GetDescription() != "Shared Civics/CLF text" {
		t.Errorf("civics descriptions = %+v, want [Shared Civics/CLF text]", r0.GetDescriptions())
	}

	r2 := assertStatus(2, enumspb.RatingDescriptionResolutionStatus_RATING_DESCRIPTION_RESOLUTION_STATUS_RESOLVED, "variant A")
	r3 := assertStatus(3, enumspb.RatingDescriptionResolutionStatus_RATING_DESCRIPTION_RESOLUTION_STATUS_RESOLVED, "variant B")
	if r2.GetRatingDescriptionSetId() == r3.GetRatingDescriptionSetId() {
		t.Errorf("variant A and variant B must resolve to DIFFERENT sets (variant isolation), both got %s", r2.GetRatingDescriptionSetId())
	}
	if len(r2.GetDescriptions()) != 1 || r2.GetDescriptions()[0].GetDescription() != "Variant A text" {
		t.Errorf("variant A descriptions = %+v, want [Variant A text]", r2.GetDescriptions())
	}
	if len(r3.GetDescriptions()) != 1 || r3.GetDescriptions()[0].GetDescription() != "Variant B text" {
		t.Errorf("variant B descriptions = %+v, want [Variant B text]", r3.GetDescriptions())
	}

	assertStatus(4, enumspb.RatingDescriptionResolutionStatus_RATING_DESCRIPTION_RESOLUTION_STATUS_NO_LINK, "no link")
	assertStatus(5, enumspb.RatingDescriptionResolutionStatus_RATING_DESCRIPTION_RESOLUTION_STATUS_UNRESOLVED_IDENTITY, "ghost product (0 product_plans)")
	assertStatus(6, enumspb.RatingDescriptionResolutionStatus_RATING_DESCRIPTION_RESOLUTION_STATUS_UNRESOLVED_IDENTITY, "bad ancestry (wrong job_id)")
	assertStatus(7, enumspb.RatingDescriptionResolutionStatus_RATING_DESCRIPTION_RESOLUTION_STATUS_INVALID_CONFIG, "DRAFT set")
	assertStatus(8, enumspb.RatingDescriptionResolutionStatus_RATING_DESCRIPTION_RESOLUTION_STATUS_INVALID_CONFIG, "foreign-workspace set")

	r9 := assertStatus(9, enumspb.RatingDescriptionResolutionStatus_RATING_DESCRIPTION_RESOLUTION_STATUS_RESOLVED, "DEPRECATED set still resolves (Q19)")
	if r9.GetRatingDescriptionSetId() != setDeprecated {
		t.Errorf("DEPRECATED cell set id = %s, want %s", r9.GetRatingDescriptionSetId(), setDeprecated)
	}
	if len(r9.GetDescriptions()) != 1 || r9.GetDescriptions()[0].GetDescription() != "Deprecated but resolves text" {
		t.Errorf("DEPRECATED cell descriptions = %+v, want [Deprecated but resolves text]", r9.GetDescriptions())
	}

	r10 := assertStatus(10, enumspb.RatingDescriptionResolutionStatus_RATING_DESCRIPTION_RESOLUTION_STATUS_UNSPECIFIED, "standard-mode slot")
	if r10.GetRatingDescriptionSetId() != "" || len(r10.GetDescriptions()) != 0 {
		t.Errorf("UNSPECIFIED cell must carry no set/descriptions, got set=%s descriptions=%+v", r10.GetRatingDescriptionSetId(), r10.GetDescriptions())
	}

	// Codex review round 1 item 3 regression cases: the link/set are entirely
	// valid on both of these; only the linked set's ENTRY row is corrupt
	// (foreign workspace_id / wrong-scale band). Both must be rejected as
	// INVALID_CONFIG for the cell, not silently resolved with a dropped or
	// missing entry (RESOLVED with 0 descriptions would be indistinguishable
	// from a legitimate "no entry at this level" — exactly the failure mode
	// the review called out).
	r11 := assertStatus(11, enumspb.RatingDescriptionResolutionStatus_RATING_DESCRIPTION_RESOLUTION_STATUS_INVALID_CONFIG, "entry belongs to a foreign workspace")
	if len(r11.GetDescriptions()) != 0 {
		t.Errorf("foreign-workspace-entry cell must carry no descriptions, got %+v", r11.GetDescriptions())
	}
	r12 := assertStatus(12, enumspb.RatingDescriptionResolutionStatus_RATING_DESCRIPTION_RESOLUTION_STATUS_INVALID_CONFIG, "entry band on a different scale")
	if len(r12.GetDescriptions()) != 0 {
		t.Errorf("wrong-scale-band-entry cell must carry no descriptions, got %+v", r12.GetDescriptions())
	}

	// Every cell echoes its OWN request ref (never mismatched/reordered).
	for i, r := range results {
		if r.GetCell() != cells[i] {
			t.Errorf("result %d must echo the requested cell by identity", i)
		}
	}
}

// countingQueryer wraps a real *sql.Tx and counts QueryContext calls, so the
// bounded-query-count assertion above is a real measurement, not a guess.
type countingQueryer struct {
	q outcomeMatrixRowsQueryer
	n int
}

func (c *countingQueryer) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	c.n++
	return c.q.QueryContext(ctx, query, args...)
}

// -----------------------------------------------------------------------
// Placeholder rendering (schema-proposal.md §10) — rollback-only fixture.
// Two clients' cells in ONE batch resolve the same tagged set, and each
// cell's text is rendered with ITS OWN client's user.first_name; a client
// whose user has a blank first name fails that cell closed
// (PLACEHOLDER_UNRESOLVED, no descriptions); a no-tag set in the same batch
// is returned verbatim; and a batch containing only no-tag sets runs no
// extra query (5, not 6).
// -----------------------------------------------------------------------

func TestIntegrationResolveCellRatingDescriptions_PlaceholderRendering(t *testing.T) {
	db := openResolveCellRatingDescriptionsDB(t)
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatalf("begin rollback-only fixture transaction: %v", err)
	}
	t.Cleanup(func() { _ = tx.Rollback() })
	ctx := context.Background()

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	id := func(name string) string { return "rdph-it-" + name + "-" + suffix }
	exec := func(query string, args ...any) {
		t.Helper()
		if _, err := tx.ExecContext(ctx, query, args...); err != nil {
			t.Fatalf("fixture statement %q: %v", query, err)
		}
	}

	wsID, planID, scheduleID, subID, pricePlanID := id("ws"), id("plan"), id("schedule"), id("sub"), id("price-plan")
	templateID, templatePhaseID, templateTaskID := id("template"), id("template-phase"), id("template-task")
	criteriaID, ttcID := id("criteria"), id("ttc")
	scaleGroupID, scaleID, band1, band2 := id("scale-group"), id("scale"), id("band-1"), id("band-2")
	productTag, productPlain := id("product-tag"), id("product-plain")
	ppTag, ppPlain := id("pp-tag"), id("pp-plain")
	setTag, setPlain := id("set-tag"), id("set-plain")

	exec(`INSERT INTO workspace (id, name, active) VALUES ($1, 'rdph resolver it', true)`, wsID)
	exec(`INSERT INTO plan (id, active) VALUES ($1, true)`, planID)
	exec(`INSERT INTO `+entityid.PriceSchedule+` (id, active) VALUES ($1, true)`, scheduleID)
	exec(`INSERT INTO price_plan (id, plan_id, price_schedule_id, active) VALUES ($1, $2, $3, true)`, pricePlanID, planID, scheduleID)
	exec(`INSERT INTO subscription (id, price_plan_id, workspace_id, active) VALUES ($1, $2, $3, true)`, subID, pricePlanID, wsID)
	exec(`INSERT INTO `+entityid.JobTemplate+` (id, name, active, workspace_id) VALUES ($1, 'rdph template', true, $2)`, templateID, wsID)
	exec(`INSERT INTO `+entityid.JobTemplatePhase+` (id, job_template_id, name, phase_order, active, workspace_id) VALUES ($1, $2, 'Term 1', 1, true, $3)`, templatePhaseID, templateID, wsID)
	exec(`INSERT INTO `+entityid.JobTemplateTask+` (id, job_template_phase_id, name, step_order, active, workspace_id) VALUES ($1, $2, 'Task A', 1, true, $3)`, templateTaskID, templatePhaseID, wsID)
	exec(`INSERT INTO `+entityid.OutcomeCriteria+`
		(id, name, criteria_type, min_score, max_score, score_increment, required, active, workspace_id)
		VALUES ($1, 'Criterion (description mode)', 'CRITERIA_TYPE_NUMERIC_SCORE', 0, 8, 1, true, true, $2)`, criteriaID, wsID)
	exec(`INSERT INTO `+entityid.ScoreScale+`
		(id, scale_group_id, version, version_status, name, scale_kind, input_unit, output_unit, workspace_id, active, created_by)
		VALUES ($1, $2, 1, 'VERSION_STATUS_PUBLISHED', 'rdph scale', 'SCALE_KIND_EXACT_MAP', 'score', 'label', $3, true, 'rdph-it')`,
		scaleID, scaleGroupID, wsID)
	exec(`INSERT INTO `+entityid.ScoreScaleBand+`
		(id, workspace_id, active, score_scale_id, sequence_order, input_match, output_value, output_label)
		VALUES ($1, $2, true, $3, 1, '1', 1, 'One'), ($4, $2, true, $3, 2, '2', 2, 'Two')`,
		band1, wsID, scaleID, band2)
	exec(`INSERT INTO `+entityid.TemplateTaskCriteria+`
		(id, job_template_task_id, outcome_criteria_id, sequence_order, active, workspace_id, rating_mode, rating_scale_id)
		VALUES ($1, $2, $3, 1, true, $4, 'RATING_MODE_NUMERIC_WITH_DESCRIPTION', $5)`,
		ttcID, templateTaskID, criteriaID, wsID, scaleID)
	exec(`INSERT INTO product (id, active) VALUES ($1, true), ($2, true)`, productTag, productPlain)
	exec(`INSERT INTO `+entityid.ProductPlan+` (id, active, product_id, plan_id, product_variant_id) VALUES
		($1, true, $2, $3, NULL), ($4, true, $5, $3, NULL)`, ppTag, productTag, planID, ppPlain, productPlain)
	exec(`INSERT INTO `+ratingDescriptionSetTable+`
		(id, workspace_id, version, version_status, name, score_scale_id, active) VALUES
		($1, $2, 1, 'VERSION_STATUS_PUBLISHED', 'Tagged set', $3, true),
		($4, $2, 1, 'VERSION_STATUS_PUBLISHED', 'Plain set', $3, true)`, setTag, wsID, scaleID, setPlain)
	exec(`INSERT INTO `+ratingDescriptionSetEntryTable+`
		(id, workspace_id, rating_description_set_id, outcome_criteria_id, score_scale_band_id, description, sequence_order, active) VALUES
		($1, $2, $3, $4, $5, '{client.user.first_name} rarely states the problem.', 1, true),
		($6, $2, $3, $4, $7, '{client.user.first_name} states the problem; {client.user.first_name} is {not.a_tag} {literal}.', 2, true),
		($8, $2, $9, $4, $5, 'Rarely states the problem.', 1, true)`,
		id("entry-tag-1"), wsID, setTag, criteriaID, band1,
		id("entry-tag-2"), band2,
		id("entry-plain-1"), setPlain)
	exec(`INSERT INTO `+ratingDescriptionSetProductPlanTable+`
		(id, workspace_id, rating_description_set_id, product_plan_id, price_schedule_id, active) VALUES
		($1, $2, $3, $4, $5, true), ($6, $2, $7, $8, $5, true)`,
		id("link-tag"), wsID, setTag, ppTag, scheduleID, id("link-plain"), setPlain, ppPlain)

	// Users + clients: Ana, Ben (with a brace-bearing surname that must stay
	// irrelevant), and one whose first name is blank.
	type person struct{ user, client, first string }
	ana := person{id("user-ana"), id("client-ana"), "Ana"}
	ben := person{id("user-ben"), id("client-ben"), "Ben"}
	blank := person{id("user-blank"), id("client-blank"), "  "}
	for _, p := range []person{ana, ben, blank} {
		exec(`INSERT INTO "user" (id, first_name, last_name, active) VALUES ($1, $2, '{client.user.first_name}', true)`, p.user, p.first)
		exec(`INSERT INTO `+entityid.Client+` (id, user_id, workspace_id, active) VALUES ($1, $2, $3, true)`, p.client, p.user, wsID)
	}

	type jobSpec struct{ key, product, client string }
	specs := []jobSpec{
		{"ana", productTag, ana.client},
		{"ben", productTag, ben.client},
		{"blank", productTag, blank.client},
		{"plain", productPlain, ana.client},
		{"noclient", productTag, ""},
	}
	cellByKey := map[string]*matrixpb.CellRatingRef{}
	for i, sp := range specs {
		jobID, phaseID, taskID := id("job-"+sp.key), id(fmt.Sprintf("phase-%d", i)), id(fmt.Sprintf("task-%d", i))
		var clientArg any
		if sp.client != "" {
			clientArg = sp.client
		}
		exec(`INSERT INTO `+entityid.Job+`
			(id, job_template_id, origin_id, workspace_id, output_product_id, client_id, active)
			VALUES ($1, $2, $3, $4, $5, $6, true)`, jobID, templateID, subID, wsID, sp.product, clientArg)
		exec(`INSERT INTO `+entityid.JobPhase+` (id, job_id, template_phase_id, active, workspace_id) VALUES ($1, $2, $3, true, $4)`, phaseID, jobID, templatePhaseID, wsID)
		exec(`INSERT INTO `+entityid.JobTask+` (id, job_phase_id, template_task_id, active, workspace_id) VALUES ($1, $2, $3, true, $4)`, taskID, phaseID, templateTaskID, wsID)
		cellByKey[sp.key] = &matrixpb.CellRatingRef{JobId: jobID, JobTaskId: taskID, OutcomeCriteriaId: criteriaID}
	}

	a := &PostgresOutcomeMatrixQuery{}
	descTexts := func(r *matrixpb.CellRatingResolution) []string {
		var out []string
		for _, d := range r.GetDescriptions() {
			out = append(out, d.GetDescription())
		}
		return out
	}
	eq := func(a, b []string) bool { return fmt.Sprint(a) == fmt.Sprint(b) }

	// Batch 1: tagged + plain + failing cells together → exactly 6 queries.
	cells := []*matrixpb.CellRatingRef{cellByKey["ana"], cellByKey["ben"], cellByKey["blank"], cellByKey["plain"], cellByKey["noclient"]}
	counter := &countingQueryer{q: tx}
	resp, err := a.resolveCellRatingDescriptionsFrom(ctx, counter, cells, wsID)
	if err != nil {
		t.Fatalf("resolveCellRatingDescriptionsFrom: %v", err)
	}
	if counter.n != 6 {
		t.Errorf("query count = %d, want 6 (5 resolution stages + 1 batched placeholder-value query)", counter.n)
	}
	res := resp.GetResults()
	if len(res) != len(cells) {
		t.Fatalf("want %d results, got %d", len(cells), len(res))
	}
	resolved := enumspb.RatingDescriptionResolutionStatus_RATING_DESCRIPTION_RESOLUTION_STATUS_RESOLVED
	unresolved := enumspb.RatingDescriptionResolutionStatus_RATING_DESCRIPTION_RESOLUTION_STATUS_PLACEHOLDER_UNRESOLVED

	// {not.a_tag} is grammar-matching but NOT allowlisted: the authoring
	// guard blocks it, and at render time it fails closed too.
	if res[0].GetStatus() != unresolved || len(res[0].GetDescriptions()) != 0 {
		t.Errorf("ana (set with an unknown tag) = %v %v, want PLACEHOLDER_UNRESOLVED with no descriptions", res[0].GetStatus(), descTexts(res[0]))
	}
	if !strings.Contains(res[0].GetReason(), "UNKNOWN_PLACEHOLDER") || strings.Contains(res[0].GetReason(), "Ana") {
		t.Errorf("ana reason = %q, want bounded UNKNOWN_PLACEHOLDER without values", res[0].GetReason())
	}

	// Fix the fixture's second entry to allowlisted tags only, then re-run.
	exec(`UPDATE `+ratingDescriptionSetEntryTable+` SET description = $1 WHERE id = $2`,
		"{client.user.first_name} states the problem; {client.user.first_name} uses {literal} braces.", id("entry-tag-2"))
	counter = &countingQueryer{q: tx}
	resp, err = a.resolveCellRatingDescriptionsFrom(ctx, counter, cells, wsID)
	if err != nil {
		t.Fatalf("resolveCellRatingDescriptionsFrom (2): %v", err)
	}
	if counter.n != 6 {
		t.Errorf("query count = %d, want 6", counter.n)
	}
	res = resp.GetResults()
	if r := res[0]; r.GetStatus() != resolved || !eq(descTexts(r), []string{"Ana rarely states the problem.", "Ana states the problem; Ana uses {literal} braces."}) {
		t.Errorf("ana = %v %q", r.GetStatus(), descTexts(r))
	}
	if r := res[1]; r.GetStatus() != resolved || !eq(descTexts(r), []string{"Ben rarely states the problem.", "Ben states the problem; Ben uses {literal} braces."}) {
		t.Errorf("ben = %v %q (must render with ITS OWN client's name)", r.GetStatus(), descTexts(r))
	}
	if r := res[2]; r.GetStatus() != unresolved || len(r.GetDescriptions()) != 0 || !strings.Contains(r.GetReason(), "client.user.first_name") {
		t.Errorf("blank first name = %v %q reason=%q, want PLACEHOLDER_UNRESOLVED, no descriptions", r.GetStatus(), descTexts(r), r.GetReason())
	}
	if r := res[3]; r.GetStatus() != resolved || !eq(descTexts(r), []string{"Rarely states the problem."}) {
		t.Errorf("plain set = %v %q, want verbatim", r.GetStatus(), descTexts(r))
	}
	if r := res[4]; r.GetStatus() != unresolved || len(r.GetDescriptions()) != 0 {
		t.Errorf("job without client = %v %q, want PLACEHOLDER_UNRESOLVED", r.GetStatus(), descTexts(r))
	}
	// Shared entry messages must not have been mutated in place.
	if res[0].GetDescriptions()[0] == res[1].GetDescriptions()[0] {
		t.Errorf("rendered descriptions must be per-cell messages, not shared pointers")
	}

	// Batch 2: only a no-tag set → no extra query, text unchanged.
	counter = &countingQueryer{q: tx}
	resp, err = a.resolveCellRatingDescriptionsFrom(ctx, counter, []*matrixpb.CellRatingRef{cellByKey["plain"]}, wsID)
	if err != nil {
		t.Fatalf("resolveCellRatingDescriptionsFrom (plain): %v", err)
	}
	if counter.n != 5 {
		t.Errorf("no-tag batch query count = %d, want 5 (no placeholder query)", counter.n)
	}
	if r := resp.GetResults()[0]; r.GetStatus() != resolved || !eq(descTexts(r), []string{"Rarely states the problem."}) {
		t.Errorf("plain-only = %v %q", r.GetStatus(), descTexts(r))
	}

	// Foreign workspace caller: the client join is workspace-scoped, so even
	// a (hypothetically) resolved tagged cell could never read another
	// tenant's name — here the ancestry already fails closed.
	counter = &countingQueryer{q: tx}
	resp, err = a.resolveCellRatingDescriptionsFrom(ctx, counter, []*matrixpb.CellRatingRef{cellByKey["ana"]}, id("other-ws"))
	if err != nil {
		t.Fatalf("resolveCellRatingDescriptionsFrom (foreign ws): %v", err)
	}
	if r := resp.GetResults()[0]; len(r.GetDescriptions()) != 0 {
		t.Errorf("foreign-workspace caller must not receive rendered text, got %q", descTexts(r))
	}
}

// -----------------------------------------------------------------------
// Selection options (rating_description_set_selectable) — rollback-only.
// -----------------------------------------------------------------------

type selectionFixture struct {
	tx                     *sql.Tx
	ctx                    context.Context
	id                     func(string) string
	exec                   func(string, ...any)
	ws, foreignWS          string
	schedule, productPlan  string
	setDefault, setB, setC string
	cell                   *matrixpb.CellRatingRef
	cellOtherClient        *matrixpb.CellRatingRef
	defaultTexts           []string
	addSelectable          func(setID, label string, seq int)
}

func newSelectionFixture(t *testing.T) *selectionFixture {
	t.Helper()
	db := openResolveCellRatingDescriptionsDB(t)
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatalf("begin rollback-only fixture transaction: %v", err)
	}
	t.Cleanup(func() { _ = tx.Rollback() })
	ctx := context.Background()
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	id := func(name string) string { return "rdsel-it-" + name + "-" + suffix }
	exec := func(query string, args ...any) {
		t.Helper()
		if _, err := tx.ExecContext(ctx, query, args...); err != nil {
			t.Fatalf("fixture statement %q: %v", query, err)
		}
	}

	f := &selectionFixture{tx: tx, ctx: ctx, id: id, exec: exec, ws: id("ws"), foreignWS: id("ws-foreign"), schedule: id("schedule"), productPlan: id("pp"),
		setDefault: id("set-default"), setB: id("set-b"), setC: id("set-c")}
	planID, subID, pricePlanID := id("plan"), id("sub"), id("price-plan")
	templateID, templatePhaseID, templateTaskID := id("template"), id("template-phase"), id("template-task")
	criteriaID, ttcID, productID := id("criteria"), id("ttc"), id("product")
	scaleID, band1, band2 := id("scale"), id("band-1"), id("band-2")

	exec(`INSERT INTO workspace (id, name, active) VALUES ($1, 'rdsel it', true), ($2, 'rdsel it foreign', true)`, f.ws, f.foreignWS)
	exec(`INSERT INTO plan (id, active) VALUES ($1, true)`, planID)
	exec(`INSERT INTO `+entityid.PriceSchedule+` (id, active) VALUES ($1, true)`, f.schedule)
	exec(`INSERT INTO price_plan (id, plan_id, price_schedule_id, active) VALUES ($1, $2, $3, true)`, pricePlanID, planID, f.schedule)
	exec(`INSERT INTO subscription (id, price_plan_id, workspace_id, active) VALUES ($1, $2, $3, true)`, subID, pricePlanID, f.ws)
	exec(`INSERT INTO `+entityid.JobTemplate+` (id, name, active, workspace_id) VALUES ($1, 'rdsel template', true, $2)`, templateID, f.ws)
	exec(`INSERT INTO `+entityid.JobTemplatePhase+` (id, job_template_id, name, phase_order, active, workspace_id) VALUES ($1, $2, 'Term 1', 1, true, $3)`, templatePhaseID, templateID, f.ws)
	exec(`INSERT INTO `+entityid.JobTemplateTask+` (id, job_template_phase_id, name, step_order, active, workspace_id) VALUES ($1, $2, 'Task A', 1, true, $3)`, templateTaskID, templatePhaseID, f.ws)
	exec(`INSERT INTO `+entityid.OutcomeCriteria+`
		(id, name, criteria_type, min_score, max_score, score_increment, required, active, workspace_id)
		VALUES ($1, 'Criterion', 'CRITERIA_TYPE_NUMERIC_SCORE', 0, 8, 1, true, true, $2)`, criteriaID, f.ws)
	exec(`INSERT INTO `+entityid.ScoreScale+`
		(id, scale_group_id, version, version_status, name, scale_kind, input_unit, output_unit, workspace_id, active, created_by)
		VALUES ($1, $2, 1, 'VERSION_STATUS_PUBLISHED', 'rdsel scale', 'SCALE_KIND_EXACT_MAP', 'score', 'label', $3, true, 'rdsel-it'),
		       ($4, $5, 1, 'VERSION_STATUS_PUBLISHED', 'rdsel scale other', 'SCALE_KIND_EXACT_MAP', 'score', 'label', $3, true, 'rdsel-it')`,
		scaleID, id("scale-group"), f.ws, id("scale-other"), id("scale-group-other"))
	exec(`INSERT INTO `+entityid.ScoreScaleBand+`
		(id, workspace_id, active, score_scale_id, sequence_order, input_match, output_value, output_label)
		VALUES ($1, $2, true, $3, 1, '1', 1, 'One'), ($4, $2, true, $3, 2, '2', 2, 'Two')`, band1, f.ws, scaleID, band2)
	exec(`INSERT INTO `+entityid.TemplateTaskCriteria+`
		(id, job_template_task_id, outcome_criteria_id, sequence_order, active, workspace_id, rating_mode, rating_scale_id)
		VALUES ($1, $2, $3, 1, true, $4, 'RATING_MODE_NUMERIC_WITH_DESCRIPTION', $5)`, ttcID, templateTaskID, criteriaID, f.ws, scaleID)
	exec(`INSERT INTO product (id, active) VALUES ($1, true)`, productID)
	exec(`INSERT INTO `+entityid.ProductPlan+` (id, active, product_id, plan_id, product_variant_id) VALUES ($1, true, $2, $3, NULL)`, f.productPlan, productID, planID)

	for _, s := range []struct{ id, word string }{{f.setDefault, "default"}, {f.setB, "second"}, {f.setC, "third"}} {
		exec(`INSERT INTO `+ratingDescriptionSetTable+`
			(id, workspace_id, version, version_status, name, score_scale_id, active)
			VALUES ($1, $2, 1, 'VERSION_STATUS_PUBLISHED', $3, $4, true)`, s.id, f.ws, s.word, scaleID)
		for i, band := range []string{band1, band2} {
			exec(`INSERT INTO `+ratingDescriptionSetEntryTable+`
				(id, workspace_id, rating_description_set_id, outcome_criteria_id, score_scale_band_id, description, sequence_order, active)
				VALUES ($1, $2, $3, $4, $5, $6, $7, true)`,
				id(fmt.Sprintf("entry-%s-%d", s.word, i)), f.ws, s.id, criteriaID, band,
				fmt.Sprintf("{client.user.first_name} %s wording band %d.", s.word, i+1), i+1)
		}
	}
	exec(`INSERT INTO `+ratingDescriptionSetProductPlanTable+`
		(id, workspace_id, rating_description_set_id, product_plan_id, price_schedule_id, active) VALUES ($1, $2, $3, $4, $5, true)`,
		id("link"), f.ws, f.setDefault, f.productPlan, f.schedule)

	cellFor := func(key, first string) *matrixpb.CellRatingRef {
		userID, clientID := id("user-"+key), id("client-"+key)
		jobID, phaseID, taskID := id("job-"+key), id("phase-"+key), id("task-"+key)
		exec(`INSERT INTO "user" (id, first_name, last_name, active) VALUES ($1, $2, 'x', true)`, userID, first)
		exec(`INSERT INTO `+entityid.Client+` (id, user_id, workspace_id, active) VALUES ($1, $2, $3, true)`, clientID, userID, f.ws)
		exec(`INSERT INTO `+entityid.Job+`
			(id, job_template_id, origin_id, workspace_id, output_product_id, client_id, active)
			VALUES ($1, $2, $3, $4, $5, $6, true)`, jobID, templateID, subID, f.ws, productID, clientID)
		exec(`INSERT INTO `+entityid.JobPhase+` (id, job_id, template_phase_id, active, workspace_id) VALUES ($1, $2, $3, true, $4)`, phaseID, jobID, templatePhaseID, f.ws)
		exec(`INSERT INTO `+entityid.JobTask+` (id, job_phase_id, template_task_id, active, workspace_id) VALUES ($1, $2, $3, true, $4)`, taskID, phaseID, templateTaskID, f.ws)
		return &matrixpb.CellRatingRef{JobId: jobID, JobTaskId: taskID, OutcomeCriteriaId: criteriaID}
	}
	f.cell = cellFor("a", "Ana")
	f.cellOtherClient = cellFor("b", "Ben")
	f.defaultTexts = []string{"Ana default wording band 1.", "Ana default wording band 2."}

	n := 0
	f.addSelectable = func(setID, label string, seq int) {
		n++
		exec(`INSERT INTO `+ratingDescriptionSetSelectableTable+`
			(id, workspace_id, product_plan_id, price_schedule_id, rating_description_set_id, option_label, sequence_order, active)
			VALUES ($1, $2, $3, $4, $5, $6, $7, true)`, id(fmt.Sprintf("sel-%d", n)), f.ws, f.productPlan, f.schedule, setID, label, seq)
	}
	return f
}

// TestIntegrationResolveCellRatingDescriptions_DefaultUnchangedBySelectableRows
// is AC-12 and the auto-fill gate at the resolver: rows change nothing on the
// save path, which runs the same query count without touching the new table.
func TestIntegrationResolveCellRatingDescriptions_DefaultUnchangedBySelectableRows(t *testing.T) {
	f := newSelectionFixture(t)
	a := &PostgresOutcomeMatrixQuery{}
	resolved := enumspb.RatingDescriptionResolutionStatus_RATING_DESCRIPTION_RESOLUTION_STATUS_RESOLVED

	snapshot := func(flag bool) (string, int) {
		t.Helper()
		c := &countingQueryer{q: f.tx}
		resp, err := a.resolveCellRatingDescriptionsWithOptionsFrom(f.ctx, c, []*matrixpb.CellRatingRef{f.cell}, f.ws, flag)
		if err != nil {
			t.Fatalf("resolve: %v", err)
		}
		r := resp.GetResults()[0]
		if r.GetStatus() != resolved {
			t.Fatalf("status = %v (%s)", r.GetStatus(), r.GetReason())
		}
		s := r.GetRatingDescriptionSetId()
		for _, d := range r.GetDescriptions() {
			s += "|" + d.GetRatingDescriptionSetEntryId() + "=" + d.GetDescription()
		}
		return s, c.n
	}

	before, nBefore := snapshot(false)
	if nBefore != 6 {
		t.Errorf("save path query count = %d, want 6", nBefore)
	}
	if !strings.Contains(before, f.defaultTexts[0]) || !strings.Contains(before, f.setDefault) {
		t.Errorf("default snapshot = %q", before)
	}
	withFlagNoRows, nFlagNoRows := snapshot(true)
	if withFlagNoRows != before {
		t.Errorf("flag with no rows changed the default:\n%s\n%s", before, withFlagNoRows)
	}
	if nFlagNoRows != 8 {
		t.Errorf("flag with no rows query count = %d, want 8 (one selectable read, one RLS check)", nFlagNoRows)
	}

	f.addSelectable(f.setDefault, "1", 1)
	f.addSelectable(f.setB, "3", 2)
	after, nAfter := snapshot(false)
	if after != before || nAfter != nBefore {
		t.Errorf("rows changed the save path: %d queries\n%s\n%s", nAfter, before, after)
	}
	flagged, _ := snapshot(true)
	if flagged != before {
		t.Errorf("flag with rows changed the default:\n%s\n%s", before, flagged)
	}

	// A queryer that refuses any read of the new table proves two things: the
	// save path never issues one, and on the drawer path its failure hides the
	// options but still returns the default.
	refusing := &refusingSelectableQueryer{q: f.tx}
	resp, err := a.resolveCellRatingDescriptionsFrom(f.ctx, refusing, []*matrixpb.CellRatingRef{f.cell}, f.ws)
	if err != nil || refusing.refused != 0 {
		t.Errorf("save path touched the selectable table: err=%v refused=%d", err, refusing.refused)
	}
	if got := resp.GetResults()[0]; got.GetStatus() != resolved || got.GetDescriptions()[0].GetDescription() != f.defaultTexts[0] {
		t.Errorf("save path result = %v", got)
	}
	refusing = &refusingSelectableQueryer{q: f.tx}
	resp, err = a.resolveCellRatingDescriptionsWithOptionsFrom(f.ctx, refusing, []*matrixpb.CellRatingRef{f.cell}, f.ws, true)
	if err != nil {
		t.Fatalf("selectable read failure must not fail the call: %v", err)
	}
	if refusing.refused != 1 {
		t.Errorf("refused = %d, want 1", refusing.refused)
	}
	got := resp.GetResults()[0]
	if got.GetStatus() != resolved || got.GetDescriptions()[0].GetDescription() != f.defaultTexts[0] || len(got.GetSelectionOptions()) != 0 {
		t.Errorf("selectable read failure: status=%v options=%d", got.GetStatus(), len(got.GetSelectionOptions()))
	}
}

// TestIntegrationResolveCellRatingDescriptions_HiddenSelectableRowsLogOnce runs as a
// non-owner role with row-level security on the table: zero rows, no error. It needs
// the table to hold committed rows, so that the statistics report them.
func TestIntegrationResolveCellRatingDescriptions_HiddenSelectableRowsLogOnce(t *testing.T) {
	f := newSelectionFixture(t)
	var live int64
	if err := f.tx.QueryRowContext(f.ctx, `SELECT pg_stat_get_live_tuples('`+ratingDescriptionSetSelectableTable+`'::regclass)`).Scan(&live); err != nil {
		t.Fatalf("statistics: %v", err)
	}
	if live == 0 {
		t.Skip("needs a database whose selectable table holds committed rows")
	}
	var buf bytes.Buffer
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })
	ratingSelectableHiddenLogged.Store(false)
	t.Cleanup(func() { ratingSelectableHiddenLogged.Store(false) })

	a := &PostgresOutcomeMatrixQuery{}
	resolve := func() *matrixpb.CellRatingResolution {
		t.Helper()
		resp, err := a.resolveCellRatingDescriptionsWithOptionsFrom(f.ctx, f.tx, []*matrixpb.CellRatingRef{f.cell}, f.ws, true)
		if err != nil {
			t.Fatalf("resolve: %v", err)
		}
		return resp.GetResults()[0]
	}
	const line = "reads none in this workspace"

	// Owner, RLS off: an offering without rows is normal, so no line.
	resolve()
	if strings.Contains(buf.String(), line) {
		t.Errorf("logged for an offering that simply has no rows")
	}

	role := f.id("role")
	f.exec(`CREATE ROLE "` + role + `"`)
	f.exec(`GRANT USAGE ON SCHEMA public TO "` + role + `"`)
	f.exec(`GRANT SELECT ON ALL TABLES IN SCHEMA public TO "` + role + `"`)
	f.exec(`ALTER TABLE ` + ratingDescriptionSetSelectableTable + ` ENABLE ROW LEVEL SECURITY`)
	f.exec(`SET LOCAL ROLE "` + role + `"`)

	// The copy's other tables carry row-level security too, so this role cannot run the
	// whole resolver. The check is called directly; the query-count test above proves
	// the option path reaches it when no rows come back.
	a.logIfSelectableRowsHidden(f.ctx, f.tx, f.ws)
	a.logIfSelectableRowsHidden(f.ctx, f.tx, f.ws)
	if got := strings.Count(buf.String(), line); got != 1 {
		t.Errorf("logged %d times, want once", got)
	}
	f.exec(`RESET ROLE`)

	// The save path never runs the check, and the default is intact.
	ratingSelectableHiddenLogged.Store(false)
	buf.Reset()
	resp, err := a.resolveCellRatingDescriptionsFrom(f.ctx, f.tx, []*matrixpb.CellRatingRef{f.cell}, f.ws)
	if err != nil {
		t.Fatalf("save path: %v", err)
	}
	if r := resp.GetResults()[0]; r.GetRatingDescriptionSetId() != f.setDefault || len(r.GetDescriptions()) != 2 {
		t.Errorf("save path default not intact: status=%v", r.GetStatus())
	}
	if strings.Contains(buf.String(), line) {
		t.Errorf("the save path logged")
	}
}

type refusingSelectableQueryer struct {
	q       outcomeMatrixRowsQueryer
	refused int
}

func (c *refusingSelectableQueryer) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	if strings.Contains(query, ratingDescriptionSetSelectableTable) {
		c.refused++
		return nil, fmt.Errorf(`relation "%s" does not exist`, ratingDescriptionSetSelectableTable)
	}
	return c.q.QueryContext(ctx, query, args...)
}

// TestIntegrationResolveCellRatingDescriptions_SelectionOptions is AC-13 and AC-14.
func TestIntegrationResolveCellRatingDescriptions_SelectionOptions(t *testing.T) {
	resolve := func(t *testing.T, f *selectionFixture, flag bool, cells ...*matrixpb.CellRatingRef) []*matrixpb.CellRatingResolution {
		t.Helper()
		a := &PostgresOutcomeMatrixQuery{}
		resp, err := a.resolveCellRatingDescriptionsWithOptionsFrom(f.ctx, f.tx, cells, f.ws, flag)
		if err != nil {
			t.Fatalf("resolve: %v", err)
		}
		return resp.GetResults()
	}
	assertDefaultIntact := func(t *testing.T, f *selectionFixture, r *matrixpb.CellRatingResolution) {
		t.Helper()
		if r.GetRatingDescriptionSetId() != f.setDefault || len(r.GetDescriptions()) != 2 || r.GetDescriptions()[0].GetDescription() != f.defaultTexts[0] {
			t.Errorf("default not intact: set=%s descriptions=%d status=%v", r.GetRatingDescriptionSetId(), len(r.GetDescriptions()), r.GetStatus())
		}
	}

	t.Run("phase not in progress has no options, default intact", func(t *testing.T) {
		f := newSelectionFixture(t)
		f.addSelectable(f.setDefault, "1", 1)
		f.addSelectable(f.setB, "3", 2)
		if _, err := f.tx.ExecContext(f.ctx, `UPDATE job_phase SET approval_status = 'PHASE_APPROVAL_STATUS_FOR_REVIEW'
			WHERE id = (SELECT job_phase_id FROM job_task WHERE id = $1)`, f.cell.GetJobTaskId()); err != nil {
			t.Fatalf("phase update: %v", err)
		}
		res := resolve(t, f, true, f.cell)
		if len(res[0].GetSelectionOptions()) != 0 {
			t.Errorf("options on a phase sent for review: %d", len(res[0].GetSelectionOptions()))
		}
		assertDefaultIntact(t, f, res[0])
	})

	t.Run("options per set with entry ids after rendering", func(t *testing.T) {
		f := newSelectionFixture(t)
		f.addSelectable(f.setDefault, "1", 1)
		f.addSelectable(f.setB, "3", 2)
		f.addSelectable(f.setC, "5", 3)
		res := resolve(t, f, true, f.cell, f.cellOtherClient)
		assertDefaultIntact(t, f, res[0])
		opts := res[0].GetSelectionOptions()
		if len(opts) != 3 {
			t.Fatalf("options = %d, want 3", len(opts))
		}
		wantLabel, wantWord := []string{"1", "3", "5"}, []string{"default", "second", "third"}
		seen := map[string]bool{}
		for i, o := range opts {
			if o.GetOptionLabel() != wantLabel[i] {
				t.Errorf("option %d label = %q", i, o.GetOptionLabel())
			}
			if o.GetIsDefault() != (i == 0) {
				t.Errorf("option %d is_default = %v", i, o.GetIsDefault())
			}
			if len(o.GetDescriptions()) != 2 {
				t.Fatalf("option %d descriptions = %d", i, len(o.GetDescriptions()))
			}
			for b, d := range o.GetDescriptions() {
				want := fmt.Sprintf("Ana %s wording band %d.", wantWord[i], b+1)
				if d.GetDescription() != want {
					t.Errorf("option %d band %d text = %q, want %q", i, b, d.GetDescription(), want)
				}
				if d.GetRatingDescriptionSetEntryId() == "" || seen[d.GetRatingDescriptionSetEntryId()] {
					t.Errorf("option %d band %d entry id missing or repeated: %q", i, b, d.GetRatingDescriptionSetEntryId())
				}
				seen[d.GetRatingDescriptionSetEntryId()] = true
			}
		}
		if got := res[1].GetSelectionOptions()[1].GetDescriptions()[0].GetDescription(); got != "Ben second wording band 1." {
			t.Errorf("second cell must render its own name, got %q", got)
		}
	})

	t.Run("no flag no options", func(t *testing.T) {
		f := newSelectionFixture(t)
		f.addSelectable(f.setDefault, "1", 1)
		f.addSelectable(f.setB, "3", 2)
		if opts := resolve(t, f, false, f.cell)[0].GetSelectionOptions(); len(opts) != 0 {
			t.Errorf("options without flag = %d", len(opts))
		}
	})

	t.Run("inactive rows are off", func(t *testing.T) {
		f := newSelectionFixture(t)
		f.addSelectable(f.setDefault, "1", 1)
		f.addSelectable(f.setB, "3", 2)
		f.exec(`UPDATE `+ratingDescriptionSetSelectableTable+` SET active = false WHERE workspace_id = $1`, f.ws)
		if opts := resolve(t, f, true, f.cell)[0].GetSelectionOptions(); len(opts) != 0 {
			t.Errorf("options = %d", len(opts))
		}
	})

	t.Run("deprecated or inactive non-default set is not offered", func(t *testing.T) {
		for name, stmt := range map[string]string{
			"deprecated": `UPDATE ` + ratingDescriptionSetTable + ` SET version_status = 'VERSION_STATUS_DEPRECATED' WHERE id = $1`,
			"inactive":   `UPDATE ` + ratingDescriptionSetTable + ` SET active = false WHERE id = $1`,
		} {
			t.Run(name, func(t *testing.T) {
				f := newSelectionFixture(t)
				f.addSelectable(f.setDefault, "1", 1)
				f.addSelectable(f.setB, "3", 2)
				f.addSelectable(f.setC, "5", 3)
				f.exec(stmt, f.setB)
				r := resolve(t, f, true, f.cell)[0]
				assertDefaultIntact(t, f, r)
				opts := r.GetSelectionOptions()
				if len(opts) != 2 || opts[0].GetRatingDescriptionSetId() != f.setDefault || opts[1].GetRatingDescriptionSetId() != f.setC {
					t.Errorf("options = %v, want the default and the third set", opts)
				}
			})
		}
	})

	t.Run("deprecated default keeps resolving and hides the list", func(t *testing.T) {
		f := newSelectionFixture(t)
		f.addSelectable(f.setDefault, "1", 1)
		f.addSelectable(f.setB, "3", 2)
		f.exec(`UPDATE `+ratingDescriptionSetTable+` SET version_status = 'VERSION_STATUS_DEPRECATED' WHERE id = $1`, f.setDefault)
		r := resolve(t, f, true, f.cell)[0]
		assertDefaultIntact(t, f, r)
		if opts := r.GetSelectionOptions(); len(opts) != 0 {
			t.Errorf("options = %d, want none", len(opts))
		}
	})

	t.Run("option set without entries offers no text", func(t *testing.T) {
		f := newSelectionFixture(t)
		f.addSelectable(f.setDefault, "1", 1)
		f.addSelectable(f.setB, "3", 2)
		f.exec(`UPDATE `+ratingDescriptionSetEntryTable+` SET active = false WHERE rating_description_set_id = $1`, f.setB)
		r := resolve(t, f, true, f.cell)[0]
		assertDefaultIntact(t, f, r)
		if opts := r.GetSelectionOptions(); len(opts) != 2 || len(opts[1].GetDescriptions()) != 0 {
			t.Errorf("options = %v, want the empty set to carry no descriptions", opts)
		}
	})

	for _, tc := range []struct {
		name    string
		breakIt func(f *selectionFixture)
	}{
		{"default not among rows", func(f *selectionFixture) {
			f.addSelectable(f.setB, "3", 2)
			f.addSelectable(f.setC, "5", 3)
		}},
		{"option set on another scale", func(f *selectionFixture) {
			f.addSelectable(f.setDefault, "1", 1)
			f.addSelectable(f.setB, "3", 2)
			f.exec(`UPDATE `+ratingDescriptionSetTable+` SET score_scale_id = $1 WHERE id = $2`, f.id("scale-other"), f.setB)
		}},
		{"option set unpublished", func(f *selectionFixture) {
			f.addSelectable(f.setDefault, "1", 1)
			f.addSelectable(f.setB, "3", 2)
			f.exec(`UPDATE `+ratingDescriptionSetTable+` SET version_status = 'VERSION_STATUS_DRAFT' WHERE id = $1`, f.setB)
		}},
		{"option set in another workspace", func(f *selectionFixture) {
			f.addSelectable(f.setDefault, "1", 1)
			f.addSelectable(f.setB, "3", 2)
			f.exec(`UPDATE `+ratingDescriptionSetEntryTable+` SET workspace_id = $1 WHERE rating_description_set_id = $2`, f.foreignWS, f.setB)
			f.exec(`UPDATE `+ratingDescriptionSetTable+` SET workspace_id = $1 WHERE id = $2`, f.foreignWS, f.setB)
		}},
		{"rows in another workspace are ignored", func(f *selectionFixture) {
			f.addSelectable(f.setDefault, "1", 1)
			f.addSelectable(f.setB, "3", 2)
			f.exec(`UPDATE `+ratingDescriptionSetSelectableTable+` SET workspace_id = $1 WHERE workspace_id = $2`, f.foreignWS, f.ws)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newSelectionFixture(t)
			tc.breakIt(f)
			r := resolve(t, f, true, f.cell)[0]
			assertDefaultIntact(t, f, r)
			if len(r.GetSelectionOptions()) != 0 {
				t.Errorf("options = %d, want 0", len(r.GetSelectionOptions()))
			}
		})
	}
}
