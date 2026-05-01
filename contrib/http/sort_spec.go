package espynahttp

import (
	"fmt"
	"net/http"

	"github.com/erniealice/espyna-golang/tableparams"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
)

// SortSpec declares all sort-related constraints for a single list page.
// It is the single source of truth that drives the view layer (ParseTableParamsFromSpec,
// TableConfig defaults) and the adapter layer (ValidateSortColumns).
//
// AllowedCols are the view-facing column keys (what the browser sends in the
// "sort" param, e.g. "date_start"). ColMap translates view-facing keys to their
// SQL counterparts when the names differ (e.g. "date_start" → "date_time_start").
// Columns not present in ColMap are passed through unchanged.
type SortSpec struct {
	// AllowedCols is the list of view-facing sort column keys the page accepts.
	// The zero value (nil/empty) is treated as "no column whitelisting" and
	// ParseTableParamsFromSpec will reject every non-default sort column.
	AllowedCols []string

	// DefaultCol is the sort column applied when the request supplies no valid
	// sort parameter. It must be present in AllowedCols.
	DefaultCol string

	// DefaultDir is the sort direction applied when the request supplies no valid
	// sort parameter. Must be "asc" or "desc"; any other value is normalised to
	// "desc" at parse time.
	DefaultDir string

	// ColMap translates view-facing column keys to SQL column names. Only
	// entries where the view name differs from the SQL name need to be listed.
	// Example: {"date_start": "date_time_start", "date_end": "date_time_end"}
	ColMap map[string]string
}

// SQLCol returns the SQL column name for the given view-facing column key.
// If no mapping exists the key is returned unchanged.
func (s SortSpec) SQLCol(viewCol string) string {
	if s.ColMap != nil {
		if sqlCol, ok := s.ColMap[viewCol]; ok {
			return sqlCol
		}
	}
	return viewCol
}

// ParseTableParamsFromSpec is an additive overload of ParseTableParams that
// derives allowedSortColumns, defaultSortColumn, and defaultSortDir from a
// SortSpec. The existing ParseTableParams signature is unchanged so the 21
// call sites that use it directly are not broken during incremental migration.
func ParseTableParamsFromSpec(r *http.Request, spec SortSpec) (tableparams.TableQueryParams, error) {
	return ParseTableParams(r, spec.AllowedCols, spec.DefaultCol, spec.DefaultDir)
}

// ValidateSortColumns checks that every sort field in req is in spec.AllowedCols.
// It is intended to be called at the adapter layer as a backstop against direct
// API callers and test harnesses that bypass the HTTP layer.
//
// On failure it returns an error with the message:
//
//	"unknown sort column %q for %s list (allowed: %v)"
//
// The entity parameter is used only for the error message (e.g. "subscription").
// An empty sort request is always valid.
func ValidateSortColumns(spec SortSpec, req *commonpb.SortRequest, entity string) error {
	if req == nil {
		return nil
	}
	for _, f := range req.Fields {
		col := f.GetField()
		if col == "" {
			continue
		}
		if !isAllowed(col, spec.AllowedCols) {
			return fmt.Errorf("unknown sort column %q for %s list (allowed: %v)", col, entity, spec.AllowedCols)
		}
	}
	return nil
}
