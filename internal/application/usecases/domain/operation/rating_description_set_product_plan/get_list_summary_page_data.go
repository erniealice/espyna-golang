package rating_description_set_product_plan

import (
	"context"
	"log"
	"sort"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	domainports "github.com/erniealice/espyna-golang/internal/application/ports/domain"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	setpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/rating_description_set"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/rating_description_set_product_plan"
	productplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_plan"
	priceschedulepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_schedule"
)

// listSummaryPageLimit/listSummaryMaxPages bound every offset-pagination
// loop below (mirrors the sibling rating_description_set package's list
// summary use case).
const (
	linkListSummaryPageLimit = 100
	linkListSummaryMaxPages  = 50
)

func linkListSummaryPagination(page int32) *commonpb.PaginationRequest {
	return &commonpb.PaginationRequest{
		Limit:  linkListSummaryPageLimit,
		Method: &commonpb.PaginationRequest_Offset{Offset: &commonpb.OffsetPagination{Page: page}},
	}
}

func linkListSummaryIDSort() *commonpb.SortRequest {
	return &commonpb.SortRequest{Fields: []*commonpb.SortField{{Field: "id"}}}
}

// ListSummaryScheduleOption is one academic-year (price_schedule) selector
// entry.
type ListSummaryScheduleOption struct {
	ID            string
	Name          string
	Active        bool
	DateTimeStart int64
}

// ListSummarySetInfo is the display name/version of a rating_description_set
// at ANY status (not just PUBLISHED — a link may point at a DEPRECATED set,
// which the assignment list must still label correctly).
type ListSummarySetInfo struct {
	Name    string
	Version int32
}

// ListSummaryLinkRow is one enriched assignment-list row.
type ListSummaryLinkRow struct {
	Link            *pb.RatingDescriptionSetProductPlan
	ProductPlanName string
	Set             ListSummarySetInfo
}

// GetRatingDescriptionSetProductPlanListSummaryPageDataRequest scopes the
// Links slice to one academic year. Leave PriceScheduleId empty to let the
// use case pick the default (most recent ACTIVE schedule, else most recent
// by start) — the same rule list/page.go's loadSchedules used to run
// client-side.
type GetRatingDescriptionSetProductPlanListSummaryPageDataRequest struct {
	PriceScheduleId string
}

// GetRatingDescriptionSetProductPlanListSummaryPageDataResponse carries
// everything the assignment LIST page needs: every AY option (for the
// selector), which one is selected/default, and that AY's links already
// enriched with the offering name and target set name/version
// (codex-review-impl3.out.md finding #1 — the assignment list must not
// separately call ordinary ListProductPlans / ListRatingDescriptionSets).
type GetRatingDescriptionSetProductPlanListSummaryPageDataResponse struct {
	Schedules          []ListSummaryScheduleOption
	SelectedScheduleId string
	Links              []ListSummaryLinkRow
}

// GetRatingDescriptionSetProductPlanListSummaryPageDataRepositories groups
// the primary repository plus cross-domain reads. Each is read DIRECTLY
// here (never through its own entity's gated use case) so this page data is
// authorized exactly ONCE, under rating_description_set_product_plan:list.
//
// ProductPlan is typed as the GENERIC generated interface at the struct
// boundary (so composition wiring stays uniform with the sibling FormPageData
// use case), but the constructor type-asserts it against
// domainports.WorkspaceScopedProductPlanReader — product_plan carries no
// workspace_id column of its own, so the generic
// ProductPlanDomainServiceServer.ListProductPlans has no ownership predicate
// for it (codex-review-impl3.out.md finding #3). A concrete adapter that
// does not implement the scoped port degrades to blank offering names
// (never a raw, unscoped cross-workspace list).
type GetRatingDescriptionSetProductPlanListSummaryPageDataRepositories struct {
	RatingDescriptionSetProductPlan pb.RatingDescriptionSetProductPlanDomainServiceServer
	PriceSchedule                   priceschedulepb.PriceScheduleDomainServiceServer
	ProductPlan                     productplanpb.ProductPlanDomainServiceServer
	RatingDescriptionSet            setpb.RatingDescriptionSetDomainServiceServer
}

type GetRatingDescriptionSetProductPlanListSummaryPageDataServices struct {
	Authorizer       ports.Authorizer
	Translator       ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

type GetRatingDescriptionSetProductPlanListSummaryPageDataUseCase struct {
	repositories      GetRatingDescriptionSetProductPlanListSummaryPageDataRepositories
	productPlanReader domainports.WorkspaceScopedProductPlanReader
	services          GetRatingDescriptionSetProductPlanListSummaryPageDataServices
}

func NewGetRatingDescriptionSetProductPlanListSummaryPageDataUseCase(
	r GetRatingDescriptionSetProductPlanListSummaryPageDataRepositories,
	s GetRatingDescriptionSetProductPlanListSummaryPageDataServices,
) *GetRatingDescriptionSetProductPlanListSummaryPageDataUseCase {
	var reader domainports.WorkspaceScopedProductPlanReader
	if r.ProductPlan != nil {
		reader, _ = r.ProductPlan.(domainports.WorkspaceScopedProductPlanReader)
	}
	return &GetRatingDescriptionSetProductPlanListSummaryPageDataUseCase{repositories: r, productPlanReader: reader, services: s}
}

// Execute authorizes ONCE on rating_description_set_product_plan:list — the
// permission every prescribed role that can reach the assignment list holds
// (copya.md's locked grant matrix: Superadmin/Education Admin hold all 12;
// AY Coordinator holds link list/read/create/update/delete) — then reads
// AY options, the selected AY's links, offering names, and set info
// directly via cross-domain repository interfaces.
// ExecuteForScheduleID is a plain-string-parameter variant of Execute, for
// callers on the OTHER side of a Go module boundary that cannot name this
// package's request type (fayna-golang cannot import espyna-golang's
// internal/ packages — see block/engineblock.go's "op's static type is
// inferred, never named" pattern for the parameterless page-data use
// cases). Identical behavior to Execute(ctx, &GetRatingDescriptionSet
// ProductPlanListSummaryPageDataRequest{PriceScheduleId: priceScheduleID}).
func (uc *GetRatingDescriptionSetProductPlanListSummaryPageDataUseCase) ExecuteForScheduleID(ctx context.Context, priceScheduleID string) (*GetRatingDescriptionSetProductPlanListSummaryPageDataResponse, error) {
	return uc.Execute(ctx, &GetRatingDescriptionSetProductPlanListSummaryPageDataRequest{PriceScheduleId: priceScheduleID})
}

func (uc *GetRatingDescriptionSetProductPlanListSummaryPageDataUseCase) Execute(ctx context.Context, req *GetRatingDescriptionSetProductPlanListSummaryPageDataRequest) (*GetRatingDescriptionSetProductPlanListSummaryPageDataResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.RatingDescriptionSetProductPlan, Action: entityid.ActionList}); err != nil {
		return nil, err
	}

	resp := &GetRatingDescriptionSetProductPlanListSummaryPageDataResponse{}

	if uc.repositories.PriceSchedule != nil {
		schedules, err := listAllPriceSchedulesDirect(ctx, uc.repositories.PriceSchedule)
		if err != nil {
			return nil, err
		}
		resp.Schedules = schedules
	}

	scheduleID := ""
	if req != nil {
		scheduleID = req.PriceScheduleId
	}
	if scheduleID == "" {
		scheduleID = defaultScheduleID(resp.Schedules)
	}
	resp.SelectedScheduleId = scheduleID

	if scheduleID == "" || uc.repositories.RatingDescriptionSetProductPlan == nil {
		return resp, nil
	}

	links, err := listAllLinksForScheduleDirect(ctx, uc.repositories.RatingDescriptionSetProductPlan, scheduleID)
	if err != nil {
		return nil, err
	}

	// Offering names — an OPTIONAL LABEL sourced through the workspace-scoped
	// port. A missing port (non-postgres build) or a failing lookup degrades
	// to blank names (the fayna view already falls back to the raw
	// product_plan_id), it never fails the page.
	productPlanNames := map[string]string{}
	if uc.productPlanReader != nil {
		names, ppErr := listWorkspaceScopedProductPlanNames(ctx, uc.productPlanReader)
		if ppErr != nil {
			log.Printf("rating_description_set_product_plan list summary: offering name lookup failed (degrading to blank names): %v", ppErr)
		} else {
			productPlanNames = names
		}
	}

	setInfo := map[string]ListSummarySetInfo{}
	if uc.repositories.RatingDescriptionSet != nil {
		info, sErr := listAllSetInfoDirect(ctx, uc.repositories.RatingDescriptionSet)
		if sErr != nil {
			return nil, sErr
		}
		setInfo = info
	}

	for _, link := range links {
		if link == nil || !link.GetActive() {
			continue
		}
		resp.Links = append(resp.Links, ListSummaryLinkRow{
			Link:            link,
			ProductPlanName: productPlanNames[link.GetProductPlanId()],
			Set:             setInfo[link.GetRatingDescriptionSetId()],
		})
	}
	return resp, nil
}

func defaultScheduleID(schedules []ListSummaryScheduleOption) string {
	for _, s := range schedules {
		if s.Active {
			return s.ID
		}
	}
	if len(schedules) > 0 {
		return schedules[0].ID
	}
	return ""
}

func listAllPriceSchedulesDirect(ctx context.Context, repo priceschedulepb.PriceScheduleDomainServiceServer) ([]ListSummaryScheduleOption, error) {
	var out []ListSummaryScheduleOption
	for page := int32(1); page <= linkListSummaryMaxPages; page++ {
		resp, err := repo.ListPriceSchedules(ctx, &priceschedulepb.ListPriceSchedulesRequest{Sort: linkListSummaryIDSort(), Pagination: linkListSummaryPagination(page)})
		if err != nil {
			return nil, err
		}
		data := resp.GetData()
		for _, s := range data {
			if s == nil || !s.GetActive() {
				continue
			}
			start := int64(0)
			if ts := s.GetDateTimeStart(); ts != nil {
				start = ts.GetSeconds()
			}
			out = append(out, ListSummaryScheduleOption{ID: s.GetId(), Name: s.GetName(), Active: s.GetActive(), DateTimeStart: start})
		}
		if len(data) < linkListSummaryPageLimit {
			break
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].DateTimeStart > out[j].DateTimeStart })
	return out, nil
}

func listAllLinksForScheduleDirect(ctx context.Context, repo pb.RatingDescriptionSetProductPlanDomainServiceServer, scheduleID string) ([]*pb.RatingDescriptionSetProductPlan, error) {
	var all []*pb.RatingDescriptionSetProductPlan
	for page := int32(1); page <= linkListSummaryMaxPages; page++ {
		resp, err := repo.GetRatingDescriptionSetProductPlanListPageData(ctx, &pb.GetRatingDescriptionSetProductPlanListPageDataRequest{
			PriceScheduleId: scheduleID,
			Sort:            linkListSummaryIDSort(),
			Pagination:      linkListSummaryPagination(page),
		})
		if err != nil {
			return nil, err
		}
		data := resp.GetRatingDescriptionSetProductPlanList()
		all = append(all, data...)
		if len(data) < linkListSummaryPageLimit {
			break
		}
	}
	return all, nil
}

func listWorkspaceScopedProductPlanNames(ctx context.Context, reader domainports.WorkspaceScopedProductPlanReader) (map[string]string, error) {
	items, err := reader.ListWorkspaceScopedProductPlans(ctx)
	if err != nil {
		return nil, err
	}
	names := map[string]string{}
	for _, pp := range items {
		if pp == nil {
			continue
		}
		name := pp.GetName()
		if name == "" {
			name = pp.GetId()
		}
		names[pp.GetId()] = name
	}
	return names, nil
}

func listAllSetInfoDirect(ctx context.Context, repo setpb.RatingDescriptionSetDomainServiceServer) (map[string]ListSummarySetInfo, error) {
	info := map[string]ListSummarySetInfo{}
	for page := int32(1); page <= linkListSummaryMaxPages; page++ {
		resp, err := repo.ListRatingDescriptionSets(ctx, &setpb.ListRatingDescriptionSetsRequest{Sort: linkListSummaryIDSort(), Pagination: linkListSummaryPagination(page)})
		if err != nil {
			return nil, err
		}
		data := resp.GetData()
		for _, s := range data {
			if s == nil {
				continue
			}
			info[s.GetId()] = ListSummarySetInfo{Name: s.GetName(), Version: s.GetVersion()}
		}
		if len(data) < linkListSummaryPageLimit {
			break
		}
	}
	return info, nil
}
