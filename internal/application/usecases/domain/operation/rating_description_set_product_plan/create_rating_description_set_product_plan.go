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

type CreateRatingDescriptionSetProductPlanRepositories struct {
	RatingDescriptionSetProductPlan pb.RatingDescriptionSetProductPlanDomainServiceServer
}

type CreateRatingDescriptionSetProductPlanServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// CreateRatingDescriptionSetProductPlanUseCase is the plain generic-CRUD create
// path. NOT wired to an HTTP route (interfaces.md §4 has no
// rating_description_set_product_plan.add — first link and replacement both go
// through RelinkRatingDescriptionSetProductPlanUseCase, which enforces the
// lock/compare/deactivate/insert transaction). Kept for API-shape parity with
// every other entity (and for tests / internal callers), same reasoning as
// score_scale.
type CreateRatingDescriptionSetProductPlanUseCase struct {
	repositories CreateRatingDescriptionSetProductPlanRepositories
	services     CreateRatingDescriptionSetProductPlanServices
}

func NewCreateRatingDescriptionSetProductPlanUseCase(r CreateRatingDescriptionSetProductPlanRepositories, s CreateRatingDescriptionSetProductPlanServices) *CreateRatingDescriptionSetProductPlanUseCase {
	return &CreateRatingDescriptionSetProductPlanUseCase{repositories: r, services: s}
}

func (uc *CreateRatingDescriptionSetProductPlanUseCase) Execute(ctx context.Context, req *pb.CreateRatingDescriptionSetProductPlanRequest) (*pb.CreateRatingDescriptionSetProductPlanResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.RatingDescriptionSetProductPlan, Action: entityid.ActionCreate}); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "rating_description_set_product_plan.validation.data_required", "Data is required [DEFAULT]"))
	}
	uc.enrich(req.Data)
	return uc.repositories.RatingDescriptionSetProductPlan.CreateRatingDescriptionSetProductPlan(ctx, req)
}

func (uc *CreateRatingDescriptionSetProductPlanUseCase) enrich(data *pb.RatingDescriptionSetProductPlan) {
	now := time.Now()
	if data.Id == "" && uc.services.IDGenerator != nil {
		data.Id = uc.services.IDGenerator.GenerateID()
	}
	data.Active = true
	ms := now.UnixMilli()
	s := now.Format(time.RFC3339)
	data.DateCreated = &ms
	data.DateCreatedString = &s
	data.DateModified = &ms
	data.DateModifiedString = &s
}
