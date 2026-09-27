//go:build postgresql

package core

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"slices"
	"strings"
	"testing"

	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
)

// Plan 20260927-db-query-performance: live parity for generic List changes.
// SELECT-only; skips without TEST_DATABASE_URL (use a read-only session).

func openParityDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := db.Ping(); err != nil {
		db.Close()
		t.Skipf("cannot reach TEST_DATABASE_URL: %v", err)
	}
	return db
}

// TestListParity_IdentifierFilter (AC-01, AC-05): a case-insensitive
// STRING_EQUALS on an identifier column now emits col = lower($n); it must
// return exactly the rows LOWER(col) = lower($n) returned, including for an
// upper-cased caller value.
func TestListParity_IdentifierFilter(t *testing.T) {
	db := openParityDB(t)
	defer db.Close()
	ctx := context.Background()

	const table, column = "job", "origin_id"
	var value string
	err := db.QueryRowContext(ctx, fmt.Sprintf(
		`SELECT %s FROM %s WHERE active = true AND %s IS NOT NULL AND %s <> ''
		 GROUP BY %s HAVING count(*) <= 100 ORDER BY count(*) DESC, %s LIMIT 1`,
		column, table, column, column, column, column)).Scan(&value)
	if err != nil {
		t.Skipf("no %s.%s sample: %v", table, column, err)
	}

	rows, err := db.QueryContext(ctx, fmt.Sprintf(
		`SELECT id FROM %s WHERE active = true AND LOWER(%s) = $1`, table, column), strings.ToLower(value))
	if err != nil {
		t.Fatalf("legacy query: %v", err)
	}
	var want []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			t.Fatalf("scan: %v", err)
		}
		want = append(want, id)
	}
	rows.Close()
	slices.Sort(want)

	p := &PostgresOperations{db: db}
	result, err := p.List(ctx, table, &interfaces.ListParams{
		Filters: &commonpb.FilterRequest{Filters: []*commonpb.TypedFilter{{
			Field: column,
			FilterType: &commonpb.TypedFilter_StringFilter{StringFilter: &commonpb.StringFilter{
				Value: strings.ToUpper(value), Operator: commonpb.StringOperator_STRING_EQUALS,
			}},
		}}},
		Pagination: &commonpb.PaginationRequest{Limit: 100},
	})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	var got []string
	for _, row := range result.Data {
		got = append(got, fmt.Sprint(row["id"]))
	}
	slices.Sort(got)

	if len(want) == 0 {
		t.Fatal("sample produced no legacy rows")
	}
	if !slices.Equal(got, want) {
		t.Fatalf("List returned %d ids, legacy LOWER() returned %d; sets differ", len(got), len(want))
	}
}

// TestListParity_JobPhasesTabFilters (AC-02 live half): the Phases tab now
// sends job_id = X to job_phase and job_phase_id IN (...) to job_task. Both must
// return exactly that job's active rows through the real generic List.
func TestListParity_JobPhasesTabFilters(t *testing.T) {
	db := openParityDB(t)
	defer db.Close()
	ctx := context.Background()

	var jobID string
	var wantPhases, wantTasks int
	err := db.QueryRowContext(ctx, `
		SELECT jp.job_id, count(DISTINCT jp.id), count(jt.id)
		FROM job_phase jp
		LEFT JOIN job_task jt ON jt.job_phase_id = jp.id AND jt.active = true
		WHERE jp.active = true
		GROUP BY jp.job_id
		ORDER BY count(jt.id) DESC, jp.job_id
		LIMIT 1`).Scan(&jobID, &wantPhases, &wantTasks)
	if err != nil {
		t.Skipf("no job_phase sample: %v", err)
	}

	p := &PostgresOperations{db: db}
	phases, err := p.List(ctx, "job_phase", &interfaces.ListParams{
		Filters: &commonpb.FilterRequest{Filters: []*commonpb.TypedFilter{{
			Field: "job_id",
			FilterType: &commonpb.TypedFilter_StringFilter{StringFilter: &commonpb.StringFilter{
				Value: jobID, Operator: commonpb.StringOperator_STRING_EQUALS, CaseSensitive: true,
			}},
		}}},
		Pagination: &commonpb.PaginationRequest{Limit: 100},
	})
	if err != nil {
		t.Fatalf("List job_phase: %v", err)
	}
	if len(phases.Data) != wantPhases {
		t.Fatalf("job_phase rows = %d, want %d", len(phases.Data), wantPhases)
	}
	phaseIDs := make([]string, 0, len(phases.Data))
	for _, row := range phases.Data {
		phaseIDs = append(phaseIDs, fmt.Sprint(row["id"]))
	}

	tasks, err := p.List(ctx, "job_task", &interfaces.ListParams{
		Filters: &commonpb.FilterRequest{Filters: []*commonpb.TypedFilter{{
			Field: "job_phase_id",
			FilterType: &commonpb.TypedFilter_ListFilter{ListFilter: &commonpb.ListFilter{
				Values: phaseIDs, Operator: commonpb.ListOperator_LIST_IN,
			}},
		}}},
		Pagination: &commonpb.PaginationRequest{Limit: 100},
	})
	if err != nil {
		t.Fatalf("List job_task: %v", err)
	}
	if len(tasks.Data) != wantTasks {
		t.Fatalf("job_task rows = %d, want %d", len(tasks.Data), wantTasks)
	}
}
