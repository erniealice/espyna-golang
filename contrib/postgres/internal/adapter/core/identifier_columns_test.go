//go:build postgresql

package core

import (
	"context"
	"database/sql"
	"os"
	"strings"
	"testing"

	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
)

// Plan 20260927-db-query-performance AC-01 (audit DB-04).

func TestIsIdentifierName(t *testing.T) {
	cases := map[string]bool{
		"id":           true,
		"origin_id":    true, // polymorphic, no FK: the measured slow case
		"workspace_id": true,
		"client_id":    true,
		"internal_id":  false, // human-entered student number
		"tax_id":       false,
		"code":         false,
		"name":         false,
		"recorded_by":  false, // key column, but only the catalog half knows
		"idea":         false,
	}
	for column, want := range cases {
		if got := isIdentifierName(column); got != want {
			t.Errorf("isIdentifierName(%q) = %v, want %v", column, got, want)
		}
	}
}

func TestIsIdentifierColumn_NoCatalogFallsBackToNaming(t *testing.T) {
	p := &PostgresOperations{}
	ctx := context.Background()
	if !p.isIdentifierColumn(ctx, "job", "origin_id") {
		t.Error("origin_id should be an identifier without a catalog")
	}
	if p.isIdentifierColumn(ctx, "task_outcome", "recorded_by") {
		t.Error("recorded_by needs the catalog; without one it must keep LOWER()")
	}
	if p.isIdentifierColumn(ctx, "client", "internal_id") {
		t.Error("internal_id is a human code and must stay case-insensitive")
	}
}

func TestBuildStringFilter_IdentifierExact(t *testing.T) {
	p := &PostgresOperations{}
	cases := []struct {
		name      string
		op        commonpb.StringOperator
		exact     bool
		caseSens  bool
		wantSQL   string
		wantValue string
	}{
		{"id equals", commonpb.StringOperator_STRING_EQUALS, true, false, "origin_id = $1", "abc"},
		{"id not equals", commonpb.StringOperator_STRING_NOT_EQUALS, true, false, "origin_id != $1", "abc"},
		{"id contains keeps LOWER", commonpb.StringOperator_STRING_CONTAINS, true, false, "LOWER(origin_id) LIKE $1 ESCAPE '\\'", "%abc%"},
		{"text equals keeps LOWER", commonpb.StringOperator_STRING_EQUALS, false, false, "LOWER(origin_id) = $1", "abc"},
		{"case-sensitive untouched", commonpb.StringOperator_STRING_EQUALS, true, true, "origin_id = $1", "AbC"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sqlText, values, next := p.buildStringFilter("origin_id",
				&commonpb.StringFilter{Value: "AbC", Operator: tc.op, CaseSensitive: tc.caseSens}, tc.exact, 1)
			if sqlText != tc.wantSQL {
				t.Fatalf("sql = %q, want %q", sqlText, tc.wantSQL)
			}
			if len(values) != 1 || values[0] != tc.wantValue {
				t.Fatalf("values = %v, want [%q]", values, tc.wantValue)
			}
			if next != 2 {
				t.Fatalf("next = %d, want 2", next)
			}
		})
	}
}

func TestBuildFilterConditions_IdentifierVsText(t *testing.T) {
	p := &PostgresOperations{}
	filters := &commonpb.FilterRequest{Filters: []*commonpb.TypedFilter{
		{Field: "origin_id", FilterType: &commonpb.TypedFilter_StringFilter{StringFilter: &commonpb.StringFilter{
			Value: "X", Operator: commonpb.StringOperator_STRING_EQUALS}}},
		{Field: "internal_id", FilterType: &commonpb.TypedFilter_StringFilter{StringFilter: &commonpb.StringFilter{
			Value: "X", Operator: commonpb.StringOperator_STRING_EQUALS}}},
	}}
	conditions, values, _, err := p.buildFilterConditions(context.Background(), "job", filters, 1)
	if err != nil {
		t.Fatalf("buildFilterConditions: %v", err)
	}
	want := []string{"origin_id = $1", "LOWER(internal_id) = $2"}
	if strings.Join(conditions, "|") != strings.Join(want, "|") {
		t.Fatalf("conditions = %v, want %v", conditions, want)
	}
	if values[0] != "x" || values[1] != "x" {
		t.Fatalf("values = %v, want lowercased", values)
	}
}

// TestIsIdentifierColumn_CatalogKeys reads PK/FK columns from a live clone.
// Skips without TEST_DATABASE_URL.
func TestIsIdentifierColumn_CatalogKeys(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		t.Skipf("cannot reach TEST_DATABASE_URL: %v", err)
	}
	p := &PostgresOperations{db: db}
	ctx := context.Background()
	if !p.isIdentifierColumn(ctx, "task_outcome", "recorded_by") {
		t.Error("task_outcome.recorded_by is an FK column and should compare exactly")
	}
	if p.isIdentifierColumn(ctx, "outcome_criteria", "code") {
		t.Error("code is a human-code key and must stay case-insensitive")
	}
	if p.isIdentifierColumn(ctx, "job", "name") {
		t.Error("job.name is not an identifier")
	}
}
