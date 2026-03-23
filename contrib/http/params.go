package espynahttp

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	"google.golang.org/protobuf/encoding/protojson"
)

// TableQueryParams holds sanitized values parsed from standard pyeza table POST params.
type TableQueryParams struct {
	Page       int
	PageSize   int
	Search     string
	SortColumn string
	SortDir    string // "asc" or "desc"
	Timezone   string
	Filters    []*commonpb.TypedFilter
	FiltersRaw string // raw JSON string for round-tripping back to template
}

// ParseTableParams reads standard pyeza table parameters from the request.
// For POST requests, reads from form values (r.FormValue).
// For GET requests (CRUD refresh), returns zero-value defaults.
// allowedSortColumns prevents ORDER BY injection — caller provides the list.
// Defaults: page=1, size=25, sort=date_created, dir=desc.
func ParseTableParams(r *http.Request, allowedSortColumns []string) (TableQueryParams, error) {
	// GET requests return defaults (CRUD refresh path)
	if r.Method == http.MethodGet {
		return TableQueryParams{
			Page:       1,
			PageSize:   25,
			SortColumn: "date_created",
			SortDir:    "desc",
			Timezone:   "UTC",
		}, nil
	}

	// POST path — parse form values
	if err := r.ParseForm(); err != nil {
		return TableQueryParams{}, fmt.Errorf("failed to parse form: %w", err)
	}

	page, _ := strconv.Atoi(r.FormValue("page"))
	if page < 1 {
		page = 1
	}

	size, _ := strconv.Atoi(r.FormValue("size"))
	if size < 1 || size > 250 {
		size = 25
	}

	sortCol := r.FormValue("sort")
	if !isAllowed(sortCol, allowedSortColumns) {
		sortCol = "date_created"
	}

	sortDir := r.FormValue("dir")
	if sortDir != "asc" && sortDir != "desc" {
		sortDir = "desc"
	}

	tz := r.FormValue("tz")
	if tz != "" {
		if _, err := time.LoadLocation(tz); err != nil {
			tz = "UTC"
		}
	} else {
		tz = "UTC"
	}

	var filters []*commonpb.TypedFilter
	filtersRaw := r.FormValue("filters")
	if filtersRaw != "" {
		var wrapper commonpb.FilterRequest
		if err := protojson.Unmarshal([]byte(filtersRaw), &wrapper); err != nil {
			return TableQueryParams{}, fmt.Errorf("invalid filter JSON: %w", err)
		}
		filters = wrapper.Filters
	}

	return TableQueryParams{
		Page:       page,
		PageSize:   size,
		Search:     r.FormValue("search"),
		SortColumn: sortCol,
		SortDir:    sortDir,
		Timezone:   tz,
		Filters:    filters,
		FiltersRaw: filtersRaw,
	}, nil
}

func isAllowed(val string, list []string) bool {
	for _, v := range list {
		if v == val {
			return true
		}
	}
	return false
}
