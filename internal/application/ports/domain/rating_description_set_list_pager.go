package domain

import (
	"context"

	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	entrypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/rating_description_set_entry"
)

// RatingDescriptionSetEntryPager is the hand-written port (fix3-backend,
// codex-review-impl3 #2) for callers that need accurate pagination metadata
// on rating_description_set_entry lists. The generated
// ListRatingDescriptionSetEntriesResponse (frozen proto) carries no
// pagination block; this returns the same page ListRatingDescriptionSetEntries
// returns (search/filters/sort/pagination applied in SQL) plus total_items
// (COUNT(*) over the same predicate) and has_next/has_prev. Callers
// type-assert it on the entry repository, mirroring the other
// rating_description_set* hand-written ports.
type RatingDescriptionSetEntryPager interface {
	ListRatingDescriptionSetEntriesPage(ctx context.Context, req *entrypb.ListRatingDescriptionSetEntriesRequest) ([]*entrypb.RatingDescriptionSetEntry, *commonpb.PaginationResponse, error)
}
