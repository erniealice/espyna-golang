package rating_description_set_product_plan

import (
	"context"
	"errors"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/rating_description_set_product_plan"
)

type UpdateRatingDescriptionSetProductPlanRepositories struct {
	RatingDescriptionSetProductPlan pb.RatingDescriptionSetProductPlanDomainServiceServer
}

type UpdateRatingDescriptionSetProductPlanServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// UpdateRatingDescriptionSetProductPlanUseCase is the plain generic-CRUD update
// path. NOT wired to an HTTP route — a link's rating_description_set_id is
// never changed in place (schema-proposal.md §3 "history kept — never an
// in-place update"); RelinkRatingDescriptionSetProductPlanUseCase is the only
// way to change which set an offering x AY uses. Kept for API-shape parity.
type UpdateRatingDescriptionSetProductPlanUseCase struct {
	repositories UpdateRatingDescriptionSetProductPlanRepositories
	services     UpdateRatingDescriptionSetProductPlanServices
}

func NewUpdateRatingDescriptionSetProductPlanUseCase(r UpdateRatingDescriptionSetProductPlanRepositories, s UpdateRatingDescriptionSetProductPlanServices) *UpdateRatingDescriptionSetProductPlanUseCase {
	return &UpdateRatingDescriptionSetProductPlanUseCase{repositories: r, services: s}
}

func (uc *UpdateRatingDescriptionSetProductPlanUseCase) Execute(ctx context.Context, req *pb.UpdateRatingDescriptionSetProductPlanRequest) (*pb.UpdateRatingDescriptionSetProductPlanResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.RatingDescriptionSetProductPlan, Action: entityid.ActionUpdate}); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "rating_description_set_product_plan.validation.data_required", "Data is required [DEFAULT]"))
	}
	now := time.Now()
	ms := now.UnixMilli()
	s := now.Format(time.RFC3339)
	req.Data.DateModified = &ms
	req.Data.DateModifiedString = &s
	return uc.repositories.RatingDescriptionSetProductPlan.UpdateRatingDescriptionSetProductPlan(ctx, req)
}
