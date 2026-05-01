package espynahttp

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/erniealice/espyna-golang/tableparams"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	"google.golang.org/protobuf/encoding/protojson"
)

// ParseTableParams reads standard pyeza table parameters from the request.
// For POST requests, reads from form values (r.FormValue).
// For GET requests without query params (CRUD refresh), returns zero-value defaults.
// For GET requests with query params (server-side pagination/search), parses them.
// allowedSortColumns prevents ORDER BY injection — caller provides the list.
// defaultSortColumn and defaultSortDir are used when the request does not supply
// valid sort values. defaultSortDir must be "asc" or "desc"; any other value falls
// back to "desc".
func ParseTableParams(r *http.Request, allowedSortColumns []string, defaultSortColumn, defaultSortDir string) (tableparams.TableQueryParams, error) {
	if defaultSortDir != "asc" && defaultSortDir != "desc" {
		defaultSortDir = "desc"
	}

	// GET requests without query params return defaults (CRUD refresh path)
	if r.Method == http.MethodGet && len(r.URL.RawQuery) == 0 {
		return tableparams.TableQueryParams{
			Page:       1,
			PageSize:   25,
			SortColumn: defaultSortColumn,
			SortDir:    defaultSortDir,
			Timezone:   "UTC",
		}, nil
	}

	// POST path or GET with query params — parse form values
	// (r.FormValue reads both POST body and URL query params)
	if err := r.ParseForm(); err != nil {
		return tableparams.TableQueryParams{}, fmt.Errorf("failed to parse form: %w", err)
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
		sortCol = defaultSortColumn
	}

	sortDir := r.FormValue("dir")
	if sortDir != "asc" && sortDir != "desc" {
		sortDir = defaultSortDir
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
			return tableparams.TableQueryParams{}, fmt.Errorf("invalid filter JSON: %w", err)
		}
		filters = wrapper.Filters
	}

	return tableparams.TableQueryParams{
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
