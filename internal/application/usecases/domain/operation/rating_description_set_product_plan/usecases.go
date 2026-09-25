// Package rating_description_set_product_plan is the entity use-case group for
// RatingDescriptionSetProductPlan — the per-offering x AY link
// (docs/plan/20260925-criterion-descriptors-by-program-year). Plain CRUD + List
// + GetListPageData mirror score_scale; Relink/Unlink are the lock-and-swap
// transaction use cases (schema-proposal.md §9.2).
//
// Deferred (logged, not implemented this wave — see W3-ESPYNA.done):
// BulkRelinkRatingDescriptionSetProductPlans,
// ProposeRatingDescriptionSetProductPlansFromPreviousPriceSchedule.
package rating_description_set_product_plan

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	setpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/rating_description_set"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/rating_description_set_product_plan"
	productplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_plan"
	priceschedulepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_schedule"
)

// UseCases aggregates the rating_description_set_product_plan CRUD + link
// lifecycle + page-data use cases.
type UseCases struct {
	CreateRatingDescriptionSetProductPlan          *CreateRatingDescriptionSetProductPlanUseCase
	ReadRatingDescriptionSetProductPlan            *ReadRatingDescriptionSetProductPlanUseCase
	UpdateRatingDescriptionSetProductPlan          *UpdateRatingDescriptionSetProductPlanUseCase
	DeleteRatingDescriptionSetProductPlan          *DeleteRatingDescriptionSetProductPlanUseCase
	ListRatingDescriptionSetProductPlans           *ListRatingDescriptionSetProductPlansUseCase
	GetRatingDescriptionSetProductPlanListPageData *GetRatingDescriptionSetProductPlanListPageDataUseCase
	RelinkRatingDescriptionSetProductPlan          *RelinkRatingDescriptionSetProductPlanUseCase
	UnlinkRatingDescriptionSetProductPlan          *UnlinkRatingDescriptionSetProductPlanUseCase
	// GetRatingDescriptionSetProductPlanFormPageData — the AY selector +
	// Relink drawer's offering/PUBLISHED-set pickers, authorized under
	// rating_description_set_product_plan:read (no separate
	// product_plan:list / price_schedule:list / rating_description_set:list
	// grant — codex-review-impl2.out.md finding #6, schema-proposal.md
	// §9.4).
	GetRatingDescriptionSetProductPlanFormPageData *GetRatingDescriptionSetProductPlanFormPageDataUseCase
	// GetRatingDescriptionSetProductPlanListSummaryPageData — the
	// assignment LIST page's SOLE data source, authorized ONCE under
	// rating_description_set_product_plan:list (codex-review-impl3.out.md
	// findings #1 and #3).
	GetRatingDescriptionSetProductPlanListSummaryPageData *GetRatingDescriptionSetProductPlanListSummaryPageDataUseCase
}

// Repositories groups the primary repository dependency.
type Repositories struct {
	RatingDescriptionSetProductPlan pb.RatingDescriptionSetProductPlanDomainServiceServer
	// RatingDescriptionSet/ProductPlan/PriceSchedule — picker-only
	// cross-domain reads for GetRatingDescriptionSetProductPlanFormPageData
	// (finding #6). Optional: nil-safe, the picker use case degrades to
	// empty option lists.
	RatingDescriptionSet setpb.RatingDescriptionSetDomainServiceServer
	ProductPlan          productplanpb.ProductPlanDomainServiceServer
	PriceSchedule        priceschedulepb.PriceScheduleDomainServiceServer
}

// Services groups the shared business-service dependencies.
type Services struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// NewUseCases wires the rating_description_set_product_plan use cases.
func NewUseCases(r Repositories, s Services) *UseCases {
	repo := r.RatingDescriptionSetProductPlan
	return &UseCases{
		CreateRatingDescriptionSetProductPlan: NewCreateRatingDescriptionSetProductPlanUseCase(
			CreateRatingDescriptionSetProductPlanRepositories{RatingDescriptionSetProductPlan: repo}, CreateRatingDescriptionSetProductPlanServices(s)),
		ReadRatingDescriptionSetProductPlan: NewReadRatingDescriptionSetProductPlanUseCase(
			ReadRatingDescriptionSetProductPlanRepositories{RatingDescriptionSetProductPlan: repo}, ReadRatingDescriptionSetProductPlanServices(s)),
		UpdateRatingDescriptionSetProductPlan: NewUpdateRatingDescriptionSetProductPlanUseCase(
			UpdateRatingDescriptionSetProductPlanRepositories{RatingDescriptionSetProductPlan: repo}, UpdateRatingDescriptionSetProductPlanServices(s)),
		DeleteRatingDescriptionSetProductPlan: NewDeleteRatingDescriptionSetProductPlanUseCase(
			DeleteRatingDescriptionSetProductPlanRepositories{RatingDescriptionSetProductPlan: repo}, DeleteRatingDescriptionSetProductPlanServices(s)),
		ListRatingDescriptionSetProductPlans: NewListRatingDescriptionSetProductPlansUseCase(
			ListRatingDescriptionSetProductPlansRepositories{RatingDescriptionSetProductPlan: repo}, ListRatingDescriptionSetProductPlansServices(s)),
		GetRatingDescriptionSetProductPlanListPageData: NewGetRatingDescriptionSetProductPlanListPageDataUseCase(
			GetRatingDescriptionSetProductPlanListPageDataRepositories{RatingDescriptionSetProductPlan: repo}, GetRatingDescriptionSetProductPlanListPageDataServices(s)),
		RelinkRatingDescriptionSetProductPlan: NewRelinkRatingDescriptionSetProductPlanUseCase(repo, RelinkRatingDescriptionSetProductPlanServices(s)),
		UnlinkRatingDescriptionSetProductPlan: NewUnlinkRatingDescriptionSetProductPlanUseCase(repo, UnlinkRatingDescriptionSetProductPlanServices(s)),
		GetRatingDescriptionSetProductPlanFormPageData: NewGetRatingDescriptionSetProductPlanFormPageDataUseCase(
			GetRatingDescriptionSetProductPlanFormPageDataRepositories{
				RatingDescriptionSet: r.RatingDescriptionSet,
				ProductPlan:          r.ProductPlan,
				PriceSchedule:        r.PriceSchedule,
			},
			GetRatingDescriptionSetProductPlanFormPageDataServices{Authorizer: s.Authorizer, Translator: s.Translator, ActionGatekeeper: s.ActionGatekeeper}),
		GetRatingDescriptionSetProductPlanListSummaryPageData: NewGetRatingDescriptionSetProductPlanListSummaryPageDataUseCase(
			GetRatingDescriptionSetProductPlanListSummaryPageDataRepositories{
				RatingDescriptionSetProductPlan: repo,
				RatingDescriptionSet:            r.RatingDescriptionSet,
				ProductPlan:                     r.ProductPlan,
				PriceSchedule:                   r.PriceSchedule,
			},
			GetRatingDescriptionSetProductPlanListSummaryPageDataServices{Authorizer: s.Authorizer, Translator: s.Translator, ActionGatekeeper: s.ActionGatekeeper}),
	}
}
