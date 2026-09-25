package rating_description_set

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/rating_description_set"
)

type GetRatingDescriptionSetItemPageDataRepositories struct {
	RatingDescriptionSet pb.RatingDescriptionSetDomainServiceServer
}

type GetRatingDescriptionSetItemPageDataServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

type GetRatingDescriptionSetItemPageDataUseCase struct {
	repositories GetRatingDescriptionSetItemPageDataRepositories
	services     GetRatingDescriptionSetItemPageDataServices
}

func NewGetRatingDescriptionSetItemPageDataUseCase(r GetRatingDescriptionSetItemPageDataRepositories, s GetRatingDescriptionSetItemPageDataServices) *GetRatingDescriptionSetItemPageDataUseCase {
	return &GetRatingDescriptionSetItemPageDataUseCase{repositories: r, services: s}
}

func (uc *GetRatingDescriptionSetItemPageDataUseCase) Execute(ctx context.Context, req *pb.GetRatingDescriptionSetItemPageDataRequest) (*pb.GetRatingDescriptionSetItemPageDataResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.RatingDescriptionSet, Action: entityid.ActionRead}); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "rating_description_set.validation.request_required", "Request is required [DEFAULT]"))
	}
	return uc.repositories.RatingDescriptionSet.GetRatingDescriptionSetItemPageData(ctx, req)
}
