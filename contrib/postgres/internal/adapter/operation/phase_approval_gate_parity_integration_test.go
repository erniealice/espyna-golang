//go:build postgresql

package operation

import (
	"context"
	"database/sql"
	"os"
	"reflect"
	"sort"
	"testing"

	"github.com/lib/pq"
)

// TestGateHasDataParity compares the frozen joined-row oracle with the
// template-phase EXISTS form on real group-scoped sheets. All reads use the
// caller-provided read-only TEST_DATABASE_URL.
func TestGateHasDataParity(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	rows, err := db.QueryContext(ctx, `
SELECT j.workspace_id, m.subscription_group_id, jp.template_phase_id
FROM task_outcome t
JOIN job_task jt ON jt.id = t.job_task_id AND jt.active = true
JOIN job_phase jp ON jp.id = jt.job_phase_id AND jp.active = true
JOIN job j ON j.id = jp.job_id
JOIN subscription_group_member m ON m.subscription_id = j.origin_id
  AND m.client_id = j.client_id AND m.workspace_id = j.workspace_id AND m.active = true
WHERE t.active = true AND jp.template_phase_id IS NOT NULL
GROUP BY j.workspace_id, m.subscription_group_id, jp.template_phase_id
ORDER BY count(*) DESC
LIMIT 3`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	type sample struct{ ws, group, phase string }
	var samples []sample
	for rows.Next() {
		var s sample
		if err := rows.Scan(&s.ws, &s.group, &s.phase); err != nil {
			t.Fatal(err)
		}
		samples = append(samples, s)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if len(samples) == 0 {
		t.Skip("no data-bearing template phase on this clone")
	}
	read := func(query string, s sample) []string {
		t.Helper()
		narrow, groupArgs := groupNarrowPredicate(s.group, 3, 2)
		args := append([]any{pq.Array([]string{s.phase}), s.ws}, groupArgs...)
		r, err := db.QueryContext(ctx, queryWithNarrow(query, narrow), args...)
		if err != nil {
			t.Fatal(err)
		}
		defer r.Close()
		var out []string
		for r.Next() {
			var phase string
			if err := r.Scan(&phase); err != nil {
				t.Fatal(err)
			}
			out = append(out, phase)
		}
		if err := r.Err(); err != nil {
			t.Fatal(err)
		}
		sort.Strings(out)
		return out
	}
	for i, s := range samples {
		oldRows := read("legacy", s)
		newRows := read("current", s)
		if !reflect.DeepEqual(oldRows, newRows) {
			t.Fatalf("has-data phase sets differ: old=%d new=%d", len(oldRows), len(newRows))
		}
		if len(oldRows) == 0 {
			t.Fatal("sample with an outcome did not reach either gate query")
		}
		if i == 0 {
			narrow, groupArgs := groupNarrowPredicate(s.group, 3, 2)
			args := append([]any{pq.Array([]string{s.phase}), s.ws}, groupArgs...)
			before := exportExplainBuffers(t, db, legacyGateRollupHasDataSQL(narrow), args)
			after := exportExplainBuffers(t, db, gateRollupHasDataSQL(narrow), args)
			t.Logf("gate has-data EXPLAIN buffers legacy=%d exists=%d", before, after)
		}
	}
	t.Logf("old/new has-data parity passed for %d data-bearing group/template-phase samples", len(samples))
}

func queryWithNarrow(which, narrow string) string {
	if which == "legacy" {
		return legacyGateRollupHasDataSQL(narrow)
	}
	return gateRollupHasDataSQL(narrow)
}
