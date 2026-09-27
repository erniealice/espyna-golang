//go:build postgresql

package entity

import (
	"context"
	"fmt"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	"github.com/erniealice/espyna-golang/shared/database/sqlexec"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
)

// pageSortKeys mirrors BuildOrderBy's first nonempty field and fixed id tie.
// Call BuildOrderBy with the method's whitelist before invoking this helper.
func pageSortKeys(sort *commonpb.SortRequest, fallback string, fallbackDesc bool) []postgresCore.AdapterSortKey {
	key := postgresCore.AdapterSortKey{Column: fallback, Desc: fallbackDesc, NullsFirst: fallbackDesc}
	for _, field := range sort.GetFields() {
		if field.GetField() != "" {
			key.Column = field.GetField()
			key.Desc = field.GetDirection() == commonpb.SortDirection_DESC
			key.NullsFirst = key.Desc
			break
		}
	}
	return []postgresCore.AdapterSortKey{key}
}

func countScopedPage(ctx context.Context, exec sqlexec.DBExecutor, q postgresCore.ScopedPageQueries) (int64, error) {
	var total int64
	if err := exec.QueryRowContext(ctx, q.CountSQL, q.CountArgs...).Scan(&total); err != nil {
		return 0, fmt.Errorf("count scoped page: %w", err)
	}
	return total, nil
}

func setPageCursors(p *commonpb.PaginationResponse, firstID, lastID string, limit int32) {
	if p.HasNext && lastID != "" {
		token := postgresCore.EncodePageCursor(*p.CurrentPage+1, "next", lastID)
		if token == "" {
			token = fmt.Sprintf("offset:%d", *p.CurrentPage*limit)
		}
		p.NextCursor = &token
	}
	if p.HasPrev && firstID != "" {
		token := postgresCore.EncodePageCursor(*p.CurrentPage-1, "prev", firstID)
		if token == "" {
			token = fmt.Sprintf("offset:%d", (*p.CurrentPage-2)*limit)
		}
		p.PrevCursor = &token
	}
}
