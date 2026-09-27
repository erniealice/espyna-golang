//go:build postgresql

package operation

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/shared/identity"
	exportpb "github.com/erniealice/esqyma/pkg/schema/v1/service/operation/subscription_group_outcome_export"
	_ "github.com/lib/pq"
)

// TestSectionExportParity executes the frozen pre-P5 section SQL and current
// SQL with identical trusted binds. Comparing every JSON row covers the entire
// matrix, including any blank cells, order, scope, and selector output.
func TestSectionExportParity(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	rows, err := db.Query(`
SELECT sg.workspace_id, sg.id, jt.job_category_id, min(jtp.code)
FROM subscription_group sg
JOIN subscription_group_member m ON m.subscription_group_id = sg.id
  AND m.workspace_id = sg.workspace_id AND m.active = true
JOIN job j ON j.origin_id = m.subscription_id AND j.client_id = m.client_id
  AND j.workspace_id = sg.workspace_id AND j.active = true
JOIN job_template jt ON jt.id = j.job_template_id
  AND jt.workspace_id = sg.workspace_id AND jt.active = true
JOIN job_template_phase jtp ON jtp.job_template_id = jt.id
  AND jtp.workspace_id = sg.workspace_id AND jtp.active = true AND jtp.code <> ''
WHERE sg.active = true AND jt.job_category_id IS NOT NULL
GROUP BY sg.workspace_id, sg.id, jt.job_category_id
ORDER BY count(*) DESC, sg.id
LIMIT 3`)
	if err != nil {
		t.Fatal(err)
	}
	type sample struct{ ws, group, cat, phase string }
	var samples []sample
	for rows.Next() {
		var s sample
		if err := rows.Scan(&s.ws, &s.group, &s.cat, &s.phase); err != nil {
			t.Fatal(err)
		}
		samples = append(samples, s)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	rows.Close()
	// Seek sections whose clients have different subject-template counts. A
	// selected phase in this cohort can expose the rectangular blank cells.
	uneven, err := db.Query(`
WITH client_template_counts AS (
  SELECT sg.workspace_id, sg.id AS group_id, jt.job_category_id, m.client_id,
         min(jtp.code) AS phase_code, count(DISTINCT jt.id) AS template_count
  FROM subscription_group sg
  JOIN subscription_group_member m ON m.subscription_group_id = sg.id
    AND m.workspace_id = sg.workspace_id AND m.active = true
  JOIN job j ON j.origin_id = m.subscription_id AND j.client_id = m.client_id
    AND j.workspace_id = sg.workspace_id AND j.active = true
  JOIN job_template jt ON jt.id = j.job_template_id
    AND jt.workspace_id = sg.workspace_id AND jt.active = true
  JOIN job_template_phase jtp ON jtp.job_template_id = jt.id
    AND jtp.workspace_id = sg.workspace_id AND jtp.active = true AND jtp.code <> ''
  WHERE sg.active = true AND jt.job_category_id IS NOT NULL
  GROUP BY sg.workspace_id, sg.id, jt.job_category_id, m.client_id
)
SELECT workspace_id, group_id, job_category_id, min(phase_code)
FROM client_template_counts
GROUP BY workspace_id, group_id, job_category_id
HAVING max(template_count) > min(template_count)
ORDER BY max(template_count) - min(template_count) DESC, group_id
LIMIT 3`)
	if err != nil {
		t.Fatal(err)
	}
	for uneven.Next() {
		var s sample
		if err := uneven.Scan(&s.ws, &s.group, &s.cat, &s.phase); err != nil {
			t.Fatal(err)
		}
		samples = append(samples, s)
	}
	if err := uneven.Err(); err != nil {
		t.Fatal(err)
	}
	uneven.Close()
	if len(samples) == 0 {
		t.Skip("no active phase export sample")
	}
	read := func(stmt string, args []any) []string {
		t.Helper()
		r, err := db.Query(stmt, args...)
		if err != nil {
			t.Fatal(err)
		}
		defer r.Close()
		var out []string
		for r.Next() {
			var kind string
			var payload []byte
			if err := r.Scan(&kind, &payload); err != nil {
				t.Fatal(err)
			}
			out = append(out, kind+":"+string(payload))
		}
		if err := r.Err(); err != nil {
			t.Fatal(err)
		}
		return out
	}
	blankCells := 0
	cellRows := 0
	explained := false
	for _, s := range samples {
		ctx := identity.WithRequestIdentity(context.Background(), &identity.RequestIdentity{WorkspaceID: s.ws})
		id, _ := identity.FromContext(ctx)
		req := &exportpb.GetSubscriptionGroupOutcomeExportRequest{SubscriptionGroupId: s.group, JobCategoryId: &s.cat}
		optionsBuilt := buildSubscriptionGroupOutcomeExportSQL(ctx, id, req, ports.SubscriptionGroupOutcomeExportScope{WorkspaceWide: true}, "options", "")
		optionRows := read(optionsBuilt.statement, optionsBuilt.args)
		phaseCode := ""
		for _, row := range optionRows {
			const prefix = "phase:"
			if len(row) < len(prefix) || row[:len(prefix)] != prefix {
				continue
			}
			var option struct {
				CategoryID string `json:"job_category_id"`
				Code       string `json:"code"`
				Ambiguous  bool   `json:"ambiguous"`
			}
			if err := json.Unmarshal([]byte(row[len(prefix):]), &option); err != nil {
				t.Fatal(err)
			}
			if option.CategoryID == s.cat && !option.Ambiguous && phaseCode == "" {
				phaseCode = option.Code
			}
		}
		choices := []struct{ kind, phase string }{{"options", ""}, {"final", ""}}
		if phaseCode != "" {
			choices = append(choices, struct{ kind, phase string }{"phase", phaseCode})
		}
		for _, choice := range choices {
			built := buildSubscriptionGroupOutcomeExportSQL(ctx, id, req, ports.SubscriptionGroupOutcomeExportScope{WorkspaceWide: true}, choice.kind, choice.phase)
			legacy := renderOutcomeExportTables(legacyOutcomeExportCTEs("") + outcomeExportRowsSQL)
			oldRows := read(legacy, built.args)
			newRows := read(built.statement, built.args)
			if !reflect.DeepEqual(oldRows, newRows) {
				t.Fatalf("%s export diverged: old=%d new=%d rows", choice.kind, len(oldRows), len(newRows))
			}
			if !explained && choice.kind == "phase" {
				before := exportExplainBuffers(t, db, legacy, built.args)
				after := exportExplainBuffers(t, db, built.statement, built.args)
				t.Logf("section export phase EXPLAIN buffers legacy=%d keyed=%d", before, after)
				explained = true
			}
			for _, row := range newRows {
				const prefix = "cell:"
				if len(row) < len(prefix) || row[:len(prefix)] != prefix {
					continue
				}
				cellRows++
				var cell struct {
					JobPresent bool `json:"job_present"`
				}
				if err := json.Unmarshal([]byte(row[len(prefix):]), &cell); err != nil {
					t.Fatal(err)
				}
				if !cell.JobPresent {
					blankCells++
				}
			}
		}
		// With no workspace-user assignment and WorkspaceWide=false, both
		// versions must fail closed to the same empty scoped result.
		built := buildSubscriptionGroupOutcomeExportSQL(ctx, id, req, ports.SubscriptionGroupOutcomeExportScope{}, "phase", phaseCode)
		legacy := renderOutcomeExportTables(legacyOutcomeExportCTEs("") + outcomeExportRowsSQL)
		if got, want := read(built.statement, built.args), read(legacy, built.args); !reflect.DeepEqual(got, want) {
			t.Fatal("non-wide scoped export diverged")
		}
	}
	t.Logf("old/new export parity passed for %d sections, cells=%d blank cells=%d", len(samples), cellRows, blankCells)
	if cellRows == 0 {
		t.Fatal("sampled sections had no selected matrix cells")
	}
	if blankCells == 0 {
		t.Log("sampled sections had no blank cells; blank-cell live coverage remains open")
	}
}

// TestSectionExportBlankCellJoinParity uses read-only VALUES when the clone's
// real sections have no missing subject. The joined matrix must still return
// the unselected template as a blank cell.
func TestSectionExportBlankCellJoinParity(t *testing.T) {
	if !strings.Contains(outcomeExportCTEs(""), "FULL OUTER JOIN selected_job_keys sj") ||
		!strings.Contains(outcomeExportCTEs(""), "WHERE sj.client_id IS NOT NULL") {
		t.Fatal("production matrix no longer uses the parity-checked key join")
	}
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	const common = `WITH positions(client_id, template_id) AS (VALUES
  ('client-a'::text, 'template-a'::text),
  ('client-a'::text, 'template-b'::text)),
selected_jobs(client_id, template_id, job_id) AS (VALUES
  ('client-a'::text, 'template-a'::text, 'job-a'::text)),`
	queries := []string{
		common + ` joined AS (
SELECT p.client_id, p.template_id, sj.job_id FROM positions p
LEFT JOIN selected_jobs sj ON sj.client_id = p.client_id AND sj.template_id = p.template_id)
SELECT template_id, job_id FROM joined ORDER BY template_id`,
		common + ` joined AS MATERIALIZED (
SELECT p.client_id, p.template_id, sj.job_id FROM positions p
FULL OUTER JOIN selected_jobs sj ON sj.client_id = p.client_id AND sj.template_id = p.template_id)
SELECT template_id, job_id FROM joined WHERE client_id IS NOT NULL ORDER BY template_id`,
	}
	var resultSets [][]string
	for _, query := range queries {
		rows, err := db.Query(query)
		if err != nil {
			t.Fatal(err)
		}
		var result []string
		for rows.Next() {
			var templateID string
			var jobID sql.NullString
			if err := rows.Scan(&templateID, &jobID); err != nil {
				rows.Close()
				t.Fatal(err)
			}
			result = append(result, templateID+"|"+jobID.String)
		}
		if err := rows.Err(); err != nil {
			t.Fatal(err)
		}
		rows.Close()
		resultSets = append(resultSets, result)
	}
	want := []string{"template-a|job-a", "template-b|"}
	if !reflect.DeepEqual(resultSets[0], want) || !reflect.DeepEqual(resultSets[1], want) {
		t.Fatalf("blank-cell join changed: legacy=%v keyed=%v", resultSets[0], resultSets[1])
	}
}

func exportExplainBuffers(t *testing.T, db *sql.DB, query string, args []any) int64 {
	t.Helper()
	var data []byte
	if err := db.QueryRow("EXPLAIN (ANALYZE, BUFFERS, FORMAT JSON) "+query, args...).Scan(&data); err != nil {
		t.Fatal(err)
	}
	var plans []struct {
		Plan struct {
			Hit  int64 `json:"Shared Hit Blocks"`
			Read int64 `json:"Shared Read Blocks"`
		} `json:"Plan"`
	}
	if err := json.Unmarshal(data, &plans); err != nil || len(plans) != 1 {
		t.Fatalf("decode EXPLAIN: plans=%d error=%v", len(plans), err)
	}
	return plans[0].Plan.Hit + plans[0].Plan.Read
}
