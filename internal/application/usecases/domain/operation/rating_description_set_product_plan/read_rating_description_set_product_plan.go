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

type ReadRatingDescriptionSetProductPlanRepositories struct {
	RatingDescriptionSetProductPlan pb.RatingDescriptionSetProductPlanDomainServiceServer
}

type ReadRatingDescriptionSetProductPlanServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

type ReadRatingDescriptionSetProductPlanUseCase struct {
	repositories ReadRatingDescriptionSetProductPlanRepositories
	services     ReadRatingDescriptionSetProductPlanServices
}

func NewReadRatingDescriptionSetProductPlanUseCase(r ReadRatingDescriptionSetProductPlanRepositories, s ReadRatingDescriptionSetProductPlanServices) *ReadRatingDescriptionSetProductPlanUseCase {
	return &ReadRatingDescriptionSetProductPlanUseCase{repositories: r, services: s}
}

func (uc *ReadRatingDescriptionSetProductPlanUseCase) Execute(ctx context.Context, req *pb.ReadRatingDescriptionSetProductPlanRequest) (*pb.ReadRatingDescriptionSetProductPlanResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.RatingDescriptionSetProductPlan, Action: entityid.ActionRead}); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "rating_description_set_product_plan.validation.request_required", "Request is required [DEFAULT]"))
	}
	return uc.repositories.RatingDescriptionSetProductPlan.ReadRatingDescriptionSetProductPlan(ctx, req)
}
