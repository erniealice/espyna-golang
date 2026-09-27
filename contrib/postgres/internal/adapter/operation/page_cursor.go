//go:build postgresql

package operation

import (
	"fmt"
	"strings"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
)

// scopedPageSort uses the already validated BuildOrderBy result. Only plain
// projected columns can enter a ScopedPageSet; expressions need a deliberate
// adapter-specific projection instead of a second sort interpretation.
func scopedPageSort(orderBy string) ([]postgresCore.AdapterSortKey, error) {
	body, ok := strings.CutPrefix(orderBy, "ORDER BY ")
	if !ok {
		return nil, fmt.Errorf("scoped page requires ORDER BY")
	}
	parts := strings.Split(body, ",")
	keys := make([]postgresCore.AdapterSortKey, 0, len(parts))
	for _, part := range parts {
		fields := strings.Fields(part)
		if len(fields) != 2 && len(fields) != 4 {
			return nil, fmt.Errorf("unsupported scoped page sort %q", part)
		}
		column := strings.Trim(fields[0], `"`)
		if dot := strings.LastIndexByte(column, '.'); dot >= 0 {
			column = strings.Trim(column[dot+1:], `"`)
		}
		if err := postgresCore.ValidateSQLIdent(column); err != nil {
			return nil, err
		}
		direction := strings.ToUpper(fields[1])
		if direction != "ASC" && direction != "DESC" {
			return nil, fmt.Errorf("unsupported scoped page direction %q", direction)
		}
		key := postgresCore.AdapterSortKey{Column: column, Desc: direction == "DESC", NullsFirst: direction == "DESC"}
		if len(fields) == 4 {
			if strings.ToUpper(fields[2]) != "NULLS" || (strings.ToUpper(fields[3]) != "FIRST" && strings.ToUpper(fields[3]) != "LAST") {
				return nil, fmt.Errorf("unsupported scoped page null order %q", part)
			}
			key.NullsFirst = strings.EqualFold(fields[3], "FIRST")
		}
		keys = append(keys, key)
	}
	return keys, nil
}

func scopedPageMetadata(page postgresCore.Page, total int64, firstID, lastID string) *commonpb.PaginationResponse {
	totalPages := int32((total + int64(page.Limit) - 1) / int64(page.Limit))
	result := &commonpb.PaginationResponse{
		TotalItems: int32(total), CurrentPage: &page.Number, TotalPages: &totalPages,
		HasNext: page.Number < totalPages, HasPrev: page.Number > 1,
	}
	if result.HasNext && lastID != "" {
		cursor := postgresCore.EncodePageCursor(page.Number+1, "next", lastID)
		if cursor != "" {
			result.NextCursor = &cursor
		}
	}
	if result.HasPrev && firstID != "" {
		cursor := postgresCore.EncodePageCursor(page.Number-1, "prev", firstID)
		if cursor != "" {
			result.PrevCursor = &cursor
		}
	}
	return result
}
