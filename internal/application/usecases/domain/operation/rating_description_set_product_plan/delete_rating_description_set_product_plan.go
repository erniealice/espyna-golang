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

type DeleteRatingDescriptionSetProductPlanRepositories struct {
	RatingDescriptionSetProductPlan pb.RatingDescriptionSetProductPlanDomainServiceServer
}

type DeleteRatingDescriptionSetProductPlanServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// DeleteRatingDescriptionSetProductPlanUseCase is the plain generic-CRUD hard
// delete. NOT wired to an HTTP route — the routed write is
// UnlinkRatingDescriptionSetProductPlanUseCase, which deactivates (keeps
// history) rather than deleting. Kept for API-shape parity / internal cleanup
// tooling only.
type DeleteRatingDescriptionSetProductPlanUseCase struct {
	repositories DeleteRatingDescriptionSetProductPlanRepositories
	services     DeleteRatingDescriptionSetProductPlanServices
}

func NewDeleteRatingDescriptionSetProductPlanUseCase(r DeleteRatingDescriptionSetProductPlanRepositories, s DeleteRatingDescriptionSetProductPlanServices) *DeleteRatingDescriptionSetProductPlanUseCase {
	return &DeleteRatingDescriptionSetProductPlanUseCase{repositories: r, services: s}
}

func (uc *DeleteRatingDescriptionSetProductPlanUseCase) Execute(ctx context.Context, req *pb.DeleteRatingDescriptionSetProductPlanRequest) (*pb.DeleteRatingDescriptionSetProductPlanResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.RatingDescriptionSetProductPlan, Action: entityid.ActionDelete}); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "rating_description_set_product_plan.validation.request_required", "Request is required [DEFAULT]"))
	}
	return uc.repositories.RatingDescriptionSetProductPlan.DeleteRatingDescriptionSetProductPlan(ctx, req)
}
