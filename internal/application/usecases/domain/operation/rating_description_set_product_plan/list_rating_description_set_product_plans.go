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

type ListRatingDescriptionSetProductPlansRepositories struct {
	RatingDescriptionSetProductPlan pb.RatingDescriptionSetProductPlanDomainServiceServer
}

type ListRatingDescriptionSetProductPlansServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

type ListRatingDescriptionSetProductPlansUseCase struct {
	repositories ListRatingDescriptionSetProductPlansRepositories
	services     ListRatingDescriptionSetProductPlansServices
}

func NewListRatingDescriptionSetProductPlansUseCase(r ListRatingDescriptionSetProductPlansRepositories, s ListRatingDescriptionSetProductPlansServices) *ListRatingDescriptionSetProductPlansUseCase {
	return &ListRatingDescriptionSetProductPlansUseCase{repositories: r, services: s}
}

func (uc *ListRatingDescriptionSetProductPlansUseCase) Execute(ctx context.Context, req *pb.ListRatingDescriptionSetProductPlansRequest) (*pb.ListRatingDescriptionSetProductPlansResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.RatingDescriptionSetProductPlan, Action: entityid.ActionList}); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "rating_description_set_product_plan.validation.request_required", "Request is required [DEFAULT]"))
	}
	return uc.repositories.RatingDescriptionSetProductPlan.ListRatingDescriptionSetProductPlans(ctx, req)
}
