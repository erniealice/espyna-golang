package rating_description_set

import (
	"context"
	"log"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	domainports "github.com/erniealice/espyna-golang/internal/application/ports/domain"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/rating_description_set"
	entrypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/rating_description_set_entry"
	linkpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/rating_description_set_product_plan"
	scalepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/score_scale"
)

// listSummaryPageLimit/listSummaryMaxPages bound every offset-pagination
// loop below (mirrors the loops fayna's list/page.go used to run itself —
// codex-review-impl2.out.md finding #7 — now moved server-side so they run
// under ONE authorization check instead of the fayna view separately
// calling four differently-gated closures).
const (
	listSummaryPageLimit = 100
	listSummaryMaxPages  = 50
)

func listSummaryPagination(page int32) *commonpb.PaginationRequest {
	return &commonpb.PaginationRequest{
		Limit:  listSummaryPageLimit,
		Method: &commonpb.PaginationRequest_Offset{Offset: &commonpb.OffsetPagination{Page: page}},
	}
}

func listSummaryIDSort() *commonpb.SortRequest {
	return &commonpb.SortRequest{Fields: []*commonpb.SortField{{Field: "id"}}}
}

// ListSummaryRow is one enriched rating_description_set list row.
type ListSummaryRow struct {
	Set            *pb.RatingDescriptionSet
	ScoreScaleName string
	EntryCount     int32
	LinkCount      int32
}

// GetRatingDescriptionSetListSummaryPageDataRequest has no fields — the
// list page is workspace-wide; the caller applies its own status-tab
// filter (active/deprecated/all) client-side, same as before.
type GetRatingDescriptionSetListSummaryPageDataRequest struct{}

// GetRatingDescriptionSetListSummaryPageDataResponse carries every set the
// list page needs, already enriched with its scale name and entry/link
// counts — so the fayna view never has to call ListScoreScales /
// ListRatingDescriptionSetEntries / ListRatingDescriptionSetProductPlans
// itself (codex-review-impl3.out.md finding #1: those calls require
// score_scale:list / rating_description_set_product_plan:list, permissions
// the Section Template Manager role — set list/read ONLY, per copya.md's
// locked grant matrix — does not hold; that made the whole list page 500
// for that role, and for every principal once a missing score_scale:list
// grant became a hard failure).
type GetRatingDescriptionSetListSummaryPageDataResponse struct {
	Rows []ListSummaryRow
}

// GetRatingDescriptionSetListSummaryPageDataRepositories groups the primary
// repository plus three OPTIONAL, nil-safe cross-domain reads. Each is read
// DIRECTLY here — never through its own entity's gated List*UseCase — so
// this page data is authorized exactly ONCE, under rating_description_set:
// list (schema-proposal.md §9.4's picker pattern, extended from drawers to
// this list page).
type GetRatingDescriptionSetListSummaryPageDataRepositories struct {
	RatingDescriptionSet pb.RatingDescriptionSetDomainServiceServer
	// ScoreScale backs the Scale display column — an OPTIONAL LABEL. A
	// missing/denied/failing lookup degrades to a blank name for every row
	// (logged), it never fails the page.
	ScoreScale scalepb.ScoreScaleDomainServiceServer
	// RatingDescriptionSetEntry/RatingDescriptionSetProductPlan back the
	// Entries/Links count columns — core list content, not optional labels.
	// The constructor type-asserts each against its scoped counter port
	// (domainports.RatingDescriptionSetEntryCounter /
	// ...ProductPlanCounter); a genuine aggregation-query failure through
	// that port propagates as a page error, exactly as before.
	RatingDescriptionSetEntry       entrypb.RatingDescriptionSetEntryDomainServiceServer
	RatingDescriptionSetProductPlan linkpb.RatingDescriptionSetProductPlanDomainServiceServer
}

type GetRatingDescriptionSetListSummaryPageDataServices struct {
	Authorizer       ports.Authorizer
	Translator       ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

type GetRatingDescriptionSetListSummaryPageDataUseCase struct {
	repositories GetRatingDescriptionSetListSummaryPageDataRepositories
	// entryCounter/linkCounter — scoped SQL-aggregation ports, type-asserted
	// from the generic repositories above at construction (codex-review-
	// impl4.out.md round-3/round-4 disposition #2: the former workspace-wide,
	// 5,000-row-capped page loops undercounted larger workspaces). A
	// non-conforming repository (never true for the postgres-only provider
	// this feature requires — see usecases.go's all-or-nothing gate) degrades
	// that column to 0 rather than falling back to the removed capped scan,
	// mirroring WorkspaceScopedProductPlanReader's nil-safe-to-empty
	// convention.
	entryCounter domainports.RatingDescriptionSetEntryCounter
	linkCounter  domainports.RatingDescriptionSetProductPlanCounter
	services     GetRatingDescriptionSetListSummaryPageDataServices
}

func NewGetRatingDescriptionSetListSummaryPageDataUseCase(r GetRatingDescriptionSetListSummaryPageDataRepositories, s GetRatingDescriptionSetListSummaryPageDataServices) *GetRatingDescriptionSetListSummaryPageDataUseCase {
	var entryCounter domainports.RatingDescriptionSetEntryCounter
	if r.RatingDescriptionSetEntry != nil {
		entryCounter, _ = r.RatingDescriptionSetEntry.(domainports.RatingDescriptionSetEntryCounter)
	}
	var linkCounter domainports.RatingDescriptionSetProductPlanCounter
	if r.RatingDescriptionSetProductPlan != nil {
		linkCounter, _ = r.RatingDescriptionSetProductPlan.(domainports.RatingDescriptionSetProductPlanCounter)
	}
	return &GetRatingDescriptionSetListSummaryPageDataUseCase{repositories: r, entryCounter: entryCounter, linkCounter: linkCounter, services: s}
}

// Execute authorizes ONCE on rating_description_set:list — the permission
// every prescribed role that can reach this page holds (copya.md's locked
// grant matrix: Superadmin/Education Admin hold all 12; AY Coordinator and
// Section Template Manager both hold set list/read) — then reads the set
// rows plus every enrichment field directly via cross-domain repository
// interfaces, bypassing each sibling entity's own ActionGatekeeper check.
func (uc *GetRatingDescriptionSetListSummaryPageDataUseCase) Execute(ctx context.Context, _ *GetRatingDescriptionSetListSummaryPageDataRequest) (*GetRatingDescriptionSetListSummaryPageDataResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.RatingDescriptionSet, Action: entityid.ActionList}); err != nil {
		return nil, err
	}

	resp := &GetRatingDescriptionSetListSummaryPageDataResponse{}
	if uc.repositories.RatingDescriptionSet == nil {
		return resp, nil
	}

	sets, err := listAllRatingDescriptionSetsDirect(ctx, uc.repositories.RatingDescriptionSet)
	if err != nil {
		return nil, err
	}

	scaleNames := map[string]string{}
	if uc.repositories.ScoreScale != nil {
		names, sErr := listScoreScaleNamesDirect(ctx, uc.repositories.ScoreScale)
		if sErr != nil {
			log.Printf("rating_description_set list summary: score scale name lookup failed (degrading to blank names): %v", sErr)
		} else {
			scaleNames = names
		}
	}

	setIDs := make([]string, 0, len(sets))
	for _, s := range sets {
		if s != nil {
			setIDs = append(setIDs, s.GetId())
		}
	}

	entryCounts := map[string]int32{}
	if uc.entryCounter != nil {
		counts, eErr := uc.entryCounter.CountActiveRatingDescriptionSetEntriesBySet(ctx, setIDs)
		if eErr != nil {
			return nil, eErr
		}
		entryCounts = counts
	} else if uc.repositories.RatingDescriptionSetEntry != nil {
		log.Printf("rating_description_set list summary: entry repository does not implement the scoped counter port (degrading Entries to 0)")
	}

	linkCounts := map[string]int32{}
	if uc.linkCounter != nil {
		counts, lErr := uc.linkCounter.CountActiveRatingDescriptionSetProductPlansBySet(ctx, setIDs)
		if lErr != nil {
			return nil, lErr
		}
		linkCounts = counts
	} else if uc.repositories.RatingDescriptionSetProductPlan != nil {
		log.Printf("rating_description_set list summary: link repository does not implement the scoped counter port (degrading Links to 0)")
	}

	for _, s := range sets {
		if s == nil {
			continue
		}
		resp.Rows = append(resp.Rows, ListSummaryRow{
			Set:            s,
			ScoreScaleName: scaleNames[s.GetScoreScaleId()],
			EntryCount:     entryCounts[s.GetId()],
			LinkCount:      linkCounts[s.GetId()],
		})
	}
	return resp, nil
}

func listAllRatingDescriptionSetsDirect(ctx context.Context, repo pb.RatingDescriptionSetDomainServiceServer) ([]*pb.RatingDescriptionSet, error) {
	var all []*pb.RatingDescriptionSet
	for page := int32(1); page <= listSummaryMaxPages; page++ {
		resp, err := repo.ListRatingDescriptionSets(ctx, &pb.ListRatingDescriptionSetsRequest{Sort: listSummaryIDSort(), Pagination: listSummaryPagination(page)})
		if err != nil {
			return nil, err
		}
		data := resp.GetData()
		all = append(all, data...)
		if len(data) < listSummaryPageLimit {
			break
		}
	}
	return all, nil
}

func listScoreScaleNamesDirect(ctx context.Context, repo scalepb.ScoreScaleDomainServiceServer) (map[string]string, error) {
	names := map[string]string{}
	for page := int32(1); page <= listSummaryMaxPages; page++ {
		resp, err := repo.ListScoreScales(ctx, &scalepb.ListScoreScalesRequest{Sort: listSummaryIDSort(), Pagination: listSummaryPagination(page)})
		if err != nil {
			return nil, err
		}
		data := resp.GetData()
		for _, s := range data {
			if s == nil {
				continue
			}
			names[s.GetId()] = s.GetName()
		}
		if len(data) < listSummaryPageLimit {
			break
		}
	}
	return names, nil
}

