//go:build postgresql

package operation

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"reflect"
	"testing"
	"time"

	_ "github.com/lib/pq"

	enumspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/enums"
)

// openRatingDescriptionProjectionDB is deliberately TEST_DATABASE_URL-gated.
// The fixture below is inserted in one transaction and rolled back, so the
// adapter reads the real PostgreSQL schema without leaving rows in the target.
func openRatingDescriptionProjectionDB(t *testing.T) *sql.DB {
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
		"SELECT to_regclass('public.template_task_criteria_rating_description')::text").Scan(&table); err != nil || !table.Valid {
		db.Close()
		t.Skip("template_task_criteria_rating_description not present; run against the expanded schema")
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// TestIntegrationOutcomeMatrixProjectionRatingDescriptions proves the
// binding-level grain at the real SQL boundary: two template-task bindings
// share one reusable scale and its bands, but their projected descriptions stay
// isolated. The query runs through the transaction seam so the fixture is
// rollback-only and cannot mutate the configured test database.
func TestIntegrationOutcomeMatrixProjectionRatingDescriptions(t *testing.T) {
	db := openRatingDescriptionProjectionDB(t)
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatalf("begin rollback-only fixture transaction: %v", err)
	}
	t.Cleanup(func() { _ = tx.Rollback() })

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	wsID := "ratingdesc-it-ws-" + suffix
	templateID := "ratingdesc-it-template-" + suffix
	phaseID := "ratingdesc-it-phase-" + suffix
	taskAID := "ratingdesc-it-task-a-" + suffix
	taskBID := "ratingdesc-it-task-b-" + suffix
	criteriaAID := "ratingdesc-it-criteria-a-" + suffix
	criteriaBID := "ratingdesc-it-criteria-b-" + suffix
	scaleID := "ratingdesc-it-scale-" + suffix
	bandLowID := "ratingdesc-it-band-low-" + suffix
	bandHighID := "ratingdesc-it-band-high-" + suffix
	ttcAID := "ratingdesc-it-ttc-a-" + suffix
	ttcBID := "ratingdesc-it-ttc-b-" + suffix

	exec := func(query string, args ...any) {
		t.Helper()
		if _, err := tx.ExecContext(context.Background(), query, args...); err != nil {
			t.Fatalf("fixture statement %q: %v", query, err)
		}
	}

	exec(`INSERT INTO workspace (id, name, active) VALUES ($1, $2, true)`, wsID, "rating description integration")
	exec(`INSERT INTO job_template (id, name, active, workspace_id) VALUES ($1, $2, true, $3)`, templateID, "rating description template", wsID)
	exec(`INSERT INTO job_template_phase (id, job_template_id, name, phase_order, active, workspace_id) VALUES ($1, $2, $3, 1, true, $4)`, phaseID, templateID, "Term 1", wsID)
	exec(`INSERT INTO job_template_task (id, job_template_phase_id, name, step_order, active, workspace_id) VALUES
		($1, $2, 'Binding A', 1, true, $3), ($4, $2, 'Binding B', 2, true, $3)`, taskAID, phaseID, wsID, taskBID)
	exec(`INSERT INTO outcome_criteria
		(id, name, criteria_type, min_score, max_score, score_increment, required, active, workspace_id)
		VALUES ($1, 'Criterion A', 'CRITERIA_TYPE_NUMERIC_SCORE', 0, 8, 1, true, true, $2),
		       ($3, 'Criterion B', 'CRITERIA_TYPE_NUMERIC_SCORE', 0, 8, 1, true, true, $2)`, criteriaAID, wsID, criteriaBID)
	exec(`INSERT INTO score_scale
		(id, scale_group_id, version, version_status, name, scale_kind, input_unit, output_unit, workspace_id, active, created_by)
		VALUES ($1, $2, 1, 'published', 'Reusable 0-8 scale', 'SCALE_KIND_EXACT_MAP', 'score', 'label', $3, true, 'ratingdesc-it')`, scaleID, "ratingdesc-it-scale-group-"+suffix, wsID)
	exec(`INSERT INTO score_scale_band
		(id, workspace_id, active, score_scale_id, sequence_order, input_match, output_value, output_label)
		VALUES ($1, $2, true, $3, 1, '1', 1, 'Low'),
		       ($4, $2, true, $3, 2, '8', 8, 'High')`, bandLowID, wsID, scaleID, bandHighID)
	exec(`INSERT INTO template_task_criteria
		(id, job_template_task_id, outcome_criteria_id, sequence_order, active, workspace_id, rating_mode, rating_scale_id)
		VALUES ($1, $2, $3, 1, true, $4, 'RATING_MODE_NUMERIC_WITH_DESCRIPTION', $5),
		       ($6, $7, $8, 1, true, $4, 'RATING_MODE_NUMERIC_WITH_DESCRIPTION', $5)`, ttcAID, taskAID, criteriaAID, wsID, scaleID, ttcBID, taskBID, criteriaBID)
	exec(`INSERT INTO template_task_criteria_rating_description
		(id, template_task_criteria_id, score_scale_band_id, description, sequence_order, active, workspace_id)
		VALUES ($1, $2, $3, 'A: needs support', 1, true, $4),
		       ($5, $2, $6, 'A: exemplary', 2, true, $4),
		       ($7, $8, $3, 'B: developing', 1, true, $4),
		       ($9, $8, $6, 'B: mastered', 2, true, $4)`,
		"ratingdesc-it-desc-a-low-"+suffix, ttcAID, bandLowID, wsID,
		"ratingdesc-it-desc-a-high-"+suffix, bandHighID,
		"ratingdesc-it-desc-b-low-"+suffix, ttcBID,
		"ratingdesc-it-desc-b-high-"+suffix)

	ctx := context.Background()
	a := &PostgresOutcomeMatrixQuery{}
	phases, err := a.loadColumnTreeFrom(ctx, tx, templateID, wsID)
	if err != nil {
		t.Fatalf("load binding-scoped column tree: %v", err)
	}
	if len(phases) != 1 || len(phases[0].GetTasks()) != 2 {
		t.Fatalf("projected tree = %d phases / %d tasks, want 1 / 2", len(phases), len(phases[0].GetTasks()))
	}

	type wantDescription struct {
		input string
		text  string
	}
	want := map[string][]wantDescription{
		"Binding A": {{input: "1", text: "A: needs support"}, {input: "8", text: "A: exemplary"}},
		"Binding B": {{input: "1", text: "B: developing"}, {input: "8", text: "B: mastered"}},
	}
	for _, task := range phases[0].GetTasks() {
		criteria := task.GetCriteria()
		if len(criteria) != 1 {
			t.Fatalf("task %q projected %d criteria, want 1", task.GetLabel(), len(criteria))
		}
		criterion := criteria[0]
		if criterion.GetRatingMode() != enumspb.RatingMode_RATING_MODE_NUMERIC_WITH_DESCRIPTION {
			t.Errorf("task %q rating mode = %v, want numeric-with-description", task.GetLabel(), criterion.GetRatingMode())
		}
		if criterion.GetRatingScaleId() != scaleID {
			t.Errorf("task %q scale = %q, want %q", task.GetLabel(), criterion.GetRatingScaleId(), scaleID)
		}
		got := make([]wantDescription, 0, len(criterion.GetRatingDescriptions()))
		for _, description := range criterion.GetRatingDescriptions() {
			got = append(got, wantDescription{input: description.GetInputMatch(), text: description.GetDescription()})
			if description.GetScaleKind() != enumspb.ScaleKind_SCALE_KIND_EXACT_MAP {
				t.Errorf("task %q description %q scale kind = %v, want exact-map", task.GetLabel(), description.GetDescription(), description.GetScaleKind())
			}
		}
		wantDescriptions, ok := want[task.GetLabel()]
		if !ok {
			t.Fatalf("unexpected projected task %q", task.GetLabel())
		}
		if !reflect.DeepEqual(got, wantDescriptions) {
			t.Errorf("task %q descriptions = %v, want %v", task.GetLabel(), got, wantDescriptions)
		}
	}
}
