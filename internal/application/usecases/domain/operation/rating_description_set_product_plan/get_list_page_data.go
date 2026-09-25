package rating_description_set_product_plan

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/rating_description_set_product_plan"
)

type GetRatingDescriptionSetProductPlanListPageDataRepositories struct {
	RatingDescriptionSetProductPlan pb.RatingDescriptionSetProductPlanDomainServiceServer
}

type GetRatingDescriptionSetProductPlanListPageDataServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// GetRatingDescriptionSetProductPlanListPageDataUseCase backs the AY setup
// list view (interfaces.md §4 rating_description_set_product_plan.list). See
// the adapter's doc comment (rating_description_set_product_plan.go) for the
// frozen-proto caveat: the response carries link rows only, not a merged
// "every offering including unlinked" grid.
type GetRatingDescriptionSetProductPlanListPageDataUseCase struct {
	repositories GetRatingDescriptionSetProductPlanListPageDataRepositories
	services     GetRatingDescriptionSetProductPlanListPageDataServices
}

func NewGetRatingDescriptionSetProductPlanListPageDataUseCase(r GetRatingDescriptionSetProductPlanListPageDataRepositories, s GetRatingDescriptionSetProductPlanListPageDataServices) *GetRatingDescriptionSetProductPlanListPageDataUseCase {
	return &GetRatingDescriptionSetProductPlanListPageDataUseCase{repositories: r, services: s}
}

func (uc *GetRatingDescriptionSetProductPlanListPageDataUseCase) Execute(ctx context.Context, req *pb.GetRatingDescriptionSetProductPlanListPageDataRequest) (*pb.GetRatingDescriptionSetProductPlanListPageDataResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.RatingDescriptionSetProductPlan, Action: entityid.ActionList}); err != nil {
		return nil, err
	}
	if req == nil || req.PriceScheduleId == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "rating_description_set_product_plan.validation.price_schedule_required", "[ERR-DEFAULT] price_schedule_id is required"))
	}
	return uc.repositories.RatingDescriptionSetProductPlan.GetRatingDescriptionSetProductPlanListPageData(ctx, req)
}
