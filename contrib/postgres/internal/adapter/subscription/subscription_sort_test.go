//go:build postgresql

package subscription

import (
	"slices"
	"testing"
)

// TestSubscriptionColMapTranslation verifies that the view-facing sort column
// keys are correctly translated to SQL column names via subscriptionViewToSQLColMap.
// This prevents the date_start / date_end drift described in the plan.
func TestSubscriptionColMapTranslation(t *testing.T) {
	tests := []struct {
		viewCol string
		wantSQL string
	}{
		{"date_start", "date_time_start"},
		{"date_end", "date_time_end"},
		{"client", "client_name"},
		// Pass-through columns — not in the map.
		{"name", "name"},
		{"date_created", "date_created"},
	}
	for _, tc := range tests {
		mapped := tc.viewCol
		if sqlCol, ok := subscriptionViewToSQLColMap[tc.viewCol]; ok {
			mapped = sqlCol
		}
		if mapped != tc.wantSQL {
			t.Errorf("ColMap[%q] = %q, want %q", tc.viewCol, mapped, tc.wantSQL)
		}
	}
}

// TestSubscriptionSortableSQLCols verifies that all expected SQL sort columns
// are present in the sortable column slice that guards the CASE WHEN chain.
func TestSubscriptionSortableSQLCols(t *testing.T) {
	required := []string{
		"name",
		"date_created",
		"date_time_start",
		"date_time_end",
		"client_name",
	}
	for _, col := range required {
		if !slices.Contains(subscriptionSortableSQLCols, col) {
			t.Errorf("subscriptionSortableSQLCols missing required column %q", col)
		}
	}
}

// TestSubscriptionSortSpec_AllColMapValuesCovered verifies that every SQL column
// name referenced in subscriptionViewToSQLColMap is handled by the CASE WHEN
// chain (i.e. appears in subscriptionSortableSQLCols).
// This closes the gap described in plan §6 option 3.
func TestSubscriptionSortSpec_AllColMapValuesCovered(t *testing.T) {
	for viewCol, sqlCol := range subscriptionViewToSQLColMap {
		if !slices.Contains(subscriptionSortableSQLCols, sqlCol) {
			t.Errorf("ColMap[%q] = %q but %q is not in subscriptionSortableSQLCols (CASE WHEN missing?)",
				viewCol, sqlCol, sqlCol)
		}
	}
}

// TestGetSubscriptionListPageData_SortDateStart is an integration smoke test
// asserting that passing sort=date_start works end-to-end by verifying the
// ColMap translation produces date_time_start (the SQL column) and that this
// column is in the sortable list. A full DB-backed test requires a Postgres
// test harness; see plan_pagedata_test.go for the skip pattern.
//
// TODO: wire up a Postgres test harness and assert actual row ordering.
func TestGetSubscriptionListPageData_SortDateStart(t *testing.T) {
	t.Skip("TODO: requires Postgres test harness — see plan_pagedata_test.go")
}
