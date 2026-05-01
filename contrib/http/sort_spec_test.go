package espynahttp

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
)

var testSpec = SortSpec{
	AllowedCols: []string{"name", "date_created", "date_start", "date_end"},
	DefaultCol:  "date_created",
	DefaultDir:  "desc",
	ColMap: map[string]string{
		"date_start": "date_time_start",
		"date_end":   "date_time_end",
	},
}

// --- SortSpec.SQLCol ---

func TestSortSpec_SQLCol_mapping(t *testing.T) {
	tests := []struct {
		viewCol string
		want    string
	}{
		{"date_start", "date_time_start"},
		{"date_end", "date_time_end"},
		{"name", "name"},         // passthrough
		{"date_created", "date_created"}, // passthrough
		{"unknown", "unknown"},   // passthrough for unlisted keys
	}
	for _, tc := range tests {
		got := testSpec.SQLCol(tc.viewCol)
		if got != tc.want {
			t.Errorf("SQLCol(%q) = %q, want %q", tc.viewCol, got, tc.want)
		}
	}
}

func TestSortSpec_SQLCol_nilColMap(t *testing.T) {
	s := SortSpec{AllowedCols: []string{"name"}, DefaultCol: "name", DefaultDir: "asc"}
	if got := s.SQLCol("name"); got != "name" {
		t.Errorf("SQLCol with nil ColMap: got %q, want %q", got, "name")
	}
}

// --- ParseTableParamsFromSpec ---

func TestParseTableParamsFromSpec_GET_noQuery_returnsDefaults(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/list", nil)
	p, err := ParseTableParamsFromSpec(r, testSpec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.SortColumn != testSpec.DefaultCol {
		t.Errorf("SortColumn: got %q, want %q", p.SortColumn, testSpec.DefaultCol)
	}
	if p.SortDir != testSpec.DefaultDir {
		t.Errorf("SortDir: got %q, want %q", p.SortDir, testSpec.DefaultDir)
	}
}

func TestParseTableParamsFromSpec_POST_allowedCol(t *testing.T) {
	body := strings.NewReader("sort=date_start&dir=asc&page=1&size=25")
	r := httptest.NewRequest(http.MethodPost, "/list", body)
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	p, err := ParseTableParamsFromSpec(r, testSpec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.SortColumn != "date_start" {
		t.Errorf("SortColumn: got %q, want %q", p.SortColumn, "date_start")
	}
	if p.SortDir != "asc" {
		t.Errorf("SortDir: got %q, want %q", p.SortDir, "asc")
	}
}

func TestParseTableParamsFromSpec_POST_disallowedCol_fallsToDefault(t *testing.T) {
	body := strings.NewReader("sort=customer&dir=asc&page=1&size=25")
	r := httptest.NewRequest(http.MethodPost, "/list", body)
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	p, err := ParseTableParamsFromSpec(r, testSpec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// "customer" is not in AllowedCols → should fall back to DefaultCol
	if p.SortColumn != testSpec.DefaultCol {
		t.Errorf("SortColumn: got %q, want default %q", p.SortColumn, testSpec.DefaultCol)
	}
}

// --- ValidateSortColumns ---

func TestValidateSortColumns_nilRequest(t *testing.T) {
	if err := ValidateSortColumns(testSpec, nil, "subscription"); err != nil {
		t.Errorf("expected nil error for nil request, got %v", err)
	}
}

func TestValidateSortColumns_emptyRequest(t *testing.T) {
	req := &commonpb.SortRequest{}
	if err := ValidateSortColumns(testSpec, req, "subscription"); err != nil {
		t.Errorf("expected nil error for empty request, got %v", err)
	}
}

func TestValidateSortColumns_allowedCols(t *testing.T) {
	for _, col := range testSpec.AllowedCols {
		req := &commonpb.SortRequest{
			Fields: []*commonpb.SortField{{Field: col, Direction: commonpb.SortDirection_ASC}},
		}
		if err := ValidateSortColumns(testSpec, req, "subscription"); err != nil {
			t.Errorf("ValidateSortColumns with allowed col %q returned error: %v", col, err)
		}
	}
}

func TestValidateSortColumns_disallowedCol_returnsError(t *testing.T) {
	req := &commonpb.SortRequest{
		Fields: []*commonpb.SortField{{Field: "customer", Direction: commonpb.SortDirection_ASC}},
	}
	err := ValidateSortColumns(testSpec, req, "subscription")
	if err == nil {
		t.Error("expected error for disallowed column, got nil")
	}
	wantSubstr := `"customer"`
	if !strings.Contains(err.Error(), wantSubstr) {
		t.Errorf("error message %q should contain %q", err.Error(), wantSubstr)
	}
	if !strings.Contains(err.Error(), "subscription") {
		t.Errorf("error message %q should contain entity name %q", err.Error(), "subscription")
	}
}

func TestValidateSortColumns_emptyFieldSkipped(t *testing.T) {
	// Empty field strings (stable tie-breaker "id" is sometimes ""‐valued) must
	// not trip the validator.
	req := &commonpb.SortRequest{
		Fields: []*commonpb.SortField{{Field: "", Direction: commonpb.SortDirection_ASC}},
	}
	if err := ValidateSortColumns(testSpec, req, "subscription"); err != nil {
		t.Errorf("expected nil error for empty field, got %v", err)
	}
}
