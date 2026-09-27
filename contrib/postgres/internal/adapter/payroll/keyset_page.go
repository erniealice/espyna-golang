//go:build postgresql

package payroll

import (
	"fmt"
	"strings"
	"time"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
)

// pageSortFromOrderBy converts the already-whitelisted BuildOrderBy result to
// the projected column used by the shared scoped keyset builder.
func pageSortFromOrderBy(orderBy string) ([]postgresCore.AdapterSortKey, error) {
	const prefix = "ORDER BY "
	if !strings.HasPrefix(orderBy, prefix) {
		return nil, fmt.Errorf("invalid page order clause %q", orderBy)
	}
	first := strings.TrimSpace(strings.SplitN(strings.TrimPrefix(orderBy, prefix), ",", 2)[0])
	parts := strings.Fields(first)
	if len(parts) != 2 || (parts[1] != "ASC" && parts[1] != "DESC") {
		return nil, fmt.Errorf("invalid page sort key %q", first)
	}
	column := strings.Trim(parts[0], `"`)
	if dot := strings.LastIndex(column, "."); dot >= 0 {
		column = strings.Trim(column[dot+1:], `"`)
	}
	if err := postgresCore.ValidateSQLIdent(column); err != nil {
		return nil, fmt.Errorf("invalid projected page sort column: %w", err)
	}
	desc := parts[1] == "DESC"
	return []postgresCore.AdapterSortKey{{Column: column, Desc: desc, NullsFirst: desc}}, nil
}

func scopedPageMetadata(total int64, page postgresCore.Page, firstID, lastID string) *commonpb.PaginationResponse {
	number := page.Number
	totalPages := int32((total + int64(page.Limit) - 1) / int64(page.Limit))
	result := &commonpb.PaginationResponse{
		TotalItems: int32(total), CurrentPage: &number, TotalPages: &totalPages,
		HasNext: number < totalPages, HasPrev: number > 1,
	}
	if result.HasNext {
		token := postgresCore.EncodePageCursor(number+1, "next", lastID)
		if token == "" { // legacy string IDs cannot be represented by the shared k1 codec
			token = fmt.Sprintf("offset:%d", number*page.Limit)
		}
		result.NextCursor = &token
	}
	if result.HasPrev {
		token := postgresCore.EncodePageCursor(number-1, "prev", firstID)
		if token == "" {
			token = fmt.Sprintf("offset:%d", (number-2)*page.Limit)
		}
		result.PrevCursor = &token
	}
	return result
}

func pageMillis(value *time.Time) *int64 {
	if value == nil {
		return nil
	}
	millis := value.UnixMilli()
	return &millis
}
