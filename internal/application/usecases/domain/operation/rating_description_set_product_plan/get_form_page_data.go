package rating_description_set_product_plan

import (
	"context"
	"sort"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	domainports "github.com/erniealice/espyna-golang/internal/application/ports/domain"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	enums "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/enums"
	setpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/rating_description_set"
	productplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_plan"
	priceschedulepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_schedule"
)

// formPickerPageLimit/formPickerMaxPages bound every offset-pagination loop
// below (codex-review-impl2.out.md finding #7 — "same for any list in these
// views ... counts and offering selectors"; espyna-golang query_budget.go
// caps a single page at 100 rows).
const (
	formPickerPageLimit = 100
	formPickerMaxPages  = 50
)

func formPickerPagination(page int32) *commonpb.PaginationRequest {
	return &commonpb.PaginationRequest{
		Limit:  formPickerPageLimit,
		Method: &commonpb.PaginationRequest_Offset{Offset: &commonpb.OffsetPagination{Page: page}},
	}
}

func formPickerIDSort() *commonpb.SortRequest {
	return &commonpb.SortRequest{Fields: []*commonpb.SortField{{Field: "id"}}}
}

// FormPickerOption is one <select> entry for a
// rating_description_set_product_plan drawer or AY selector.
type FormPickerOption struct {
	ID   string
	Name string
}

// SchedulePickerOption is one academic-year (price_schedule) entry.
// Active/DateTimeStart are carried through (not collapsed to a plain
// FormPickerOption) so the caller can replicate the existing "most recent
// active schedule, else most recent by start" default-selection rule
// (list/page.go's loadSchedules) without a second, separately-gated call.
type SchedulePickerOption struct {
	ID            string
	Name          string
	Active        bool
	DateTimeStart int64
}

// GetRatingDescriptionSetProductPlanFormPageDataRequest has no fields today
// — every picker below is workspace-wide (the Relink drawer's offering and
// AY are already fixed by the caller's query string before this page data
// is fetched; only the option LISTS are workspace-scoped).
type GetRatingDescriptionSetProductPlanFormPageDataRequest struct{}

// GetRatingDescriptionSetProductPlanFormPageDataResponse carries every
// picker option the AY setup list (academic-year selector) and Relink
// drawer (offering + PUBLISHED-set selectors) need.
type GetRatingDescriptionSetProductPlanFormPageDataResponse struct {
	ProductPlans   []FormPickerOption
	PriceSchedules []SchedulePickerOption
	PublishedSets  []FormPickerOption
}

type GetRatingDescriptionSetProductPlanFormPageDataRepositories struct {
	// RatingDescriptionSet/ProductPlan/PriceSchedule — picker-only
	// cross-domain reads, read directly here (not through their own gated
	// List*UseCase) so this page data is authorized once, under
	// rating_description_set_product_plan:read — not under separate
	// rating_description_set:list / product_plan:list / price_schedule:list
	// grants (codex-review-impl2.out.md finding #6 — "a user with only the
	// prescribed Education AY Coordinator role opens assignments ...
	// loadSchedules swallows the permission error and presents no academic
	// year"; schema-proposal.md §9.4).
	//
	// ProductPlan is typed as the GENERIC generated interface at the struct
	// boundary (composition wiring stays uniform), but the constructor
	// type-asserts it against domainports.WorkspaceScopedProductPlanReader
	// — product_plan carries no workspace_id column of its own, so the
	// generic ProductPlanDomainServiceServer.ListProductPlans has no
	// ownership predicate and would expose cross-workspace offering
	// IDs/names (codex-review-impl3.out.md finding #3). A concrete adapter
	// that does not implement the scoped port degrades to an empty
	// offering picker, never an unscoped cross-workspace list.
	RatingDescriptionSet setpb.RatingDescriptionSetDomainServiceServer
	ProductPlan          productplanpb.ProductPlanDomainServiceServer
	PriceSchedule        priceschedulepb.PriceScheduleDomainServiceServer
}

type GetRatingDescriptionSetProductPlanFormPageDataServices struct {
	Authorizer       ports.Authorizer
	Translator       ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

type GetRatingDescriptionSetProductPlanFormPageDataUseCase struct {
	repositories      GetRatingDescriptionSetProductPlanFormPageDataRepositories
	productPlanReader domainports.WorkspaceScopedProductPlanReader
	services          GetRatingDescriptionSetProductPlanFormPageDataServices
}

func NewGetRatingDescriptionSetProductPlanFormPageDataUseCase(r GetRatingDescriptionSetProductPlanFormPageDataRepositories, s GetRatingDescriptionSetProductPlanFormPageDataServices) *GetRatingDescriptionSetProductPlanFormPageDataUseCase {
	var reader domainports.WorkspaceScopedProductPlanReader
	if r.ProductPlan != nil {
		reader, _ = r.ProductPlan.(domainports.WorkspaceScopedProductPlanReader)
	}
	return &GetRatingDescriptionSetProductPlanFormPageDataUseCase{repositories: r, productPlanReader: reader, services: s}
}

// Execute authorizes on rating_description_set_product_plan:read — the
// baseline management permission every prescribed role that can open the
// assignments page holds (the page itself gates on :list; the mutating
// actions separately re-check :create/:update before writing) — then reads
// offerings, academic years, and PUBLISHED sets directly via their
// repository interfaces.
func (uc *GetRatingDescriptionSetProductPlanFormPageDataUseCase) Execute(ctx context.Context, _ *GetRatingDescriptionSetProductPlanFormPageDataRequest) (*GetRatingDescriptionSetProductPlanFormPageDataResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.RatingDescriptionSetProductPlan, Action: entityid.ActionRead}); err != nil {
		return nil, err
	}
	resp := &GetRatingDescriptionSetProductPlanFormPageDataResponse{}

	if uc.productPlanReader != nil {
		items, err := uc.productPlanReader.ListWorkspaceScopedProductPlans(ctx)
		if err != nil {
			return nil, err
		}
		for _, pp := range items {
			if pp == nil || !pp.GetActive() {
				continue
			}
			name := pp.GetName()
			if name == "" {
				name = pp.GetId()
			}
			resp.ProductPlans = append(resp.ProductPlans, FormPickerOption{ID: pp.GetId(), Name: name})
		}
		sort.SliceStable(resp.ProductPlans, func(i, j int) bool { return resp.ProductPlans[i].Name < resp.ProductPlans[j].Name })
	}

	if uc.repositories.PriceSchedule != nil {
		for page := int32(1); page <= formPickerMaxPages; page++ {
			psResp, err := uc.repositories.PriceSchedule.ListPriceSchedules(ctx, &priceschedulepb.ListPriceSchedulesRequest{Sort: formPickerIDSort(), Pagination: formPickerPagination(page)})
			if err != nil {
				return nil, err
			}
			data := psResp.GetData()
			for _, ps := range data {
				if ps == nil || !ps.GetActive() {
					continue
				}
				start := int64(0)
				if ts := ps.GetDateTimeStart(); ts != nil {
					start = ts.GetSeconds()
				}
				resp.PriceSchedules = append(resp.PriceSchedules, SchedulePickerOption{
					ID:            ps.GetId(),
					Name:          ps.GetName(),
					Active:        ps.GetActive(),
					DateTimeStart: start,
				})
			}
			if len(data) < formPickerPageLimit {
				break
			}
		}
	}

	if uc.repositories.RatingDescriptionSet != nil {
		for page := int32(1); page <= formPickerMaxPages; page++ {
			setResp, err := uc.repositories.RatingDescriptionSet.ListRatingDescriptionSets(ctx, &setpb.ListRatingDescriptionSetsRequest{Sort: formPickerIDSort(), Pagination: formPickerPagination(page)})
			if err != nil {
				return nil, err
			}
			data := setResp.GetData()
			for _, s := range data {
				if s == nil || s.GetVersionStatus() != enums.VersionStatus_VERSION_STATUS_PUBLISHED {
					continue
				}
				resp.PublishedSets = append(resp.PublishedSets, FormPickerOption{ID: s.GetId(), Name: s.GetName()})
			}
			if len(data) < formPickerPageLimit {
				break
			}
		}
		sort.SliceStable(resp.PublishedSets, func(i, j int) bool { return resp.PublishedSets[i].Name < resp.PublishedSets[j].Name })
	}

	return resp, nil
}
