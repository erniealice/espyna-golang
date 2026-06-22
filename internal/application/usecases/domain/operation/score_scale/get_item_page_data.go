package score_scale

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/score_scale"
)

type GetScoreScaleItemPageDataRepositories struct {
	ScoreScale pb.ScoreScaleDomainServiceServer
}

type GetScoreScaleItemPageDataServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

type GetScoreScaleItemPageDataUseCase struct {
	repositories GetScoreScaleItemPageDataRepositories
	services     GetScoreScaleItemPageDataServices
}

func NewGetScoreScaleItemPageDataUseCase(r GetScoreScaleItemPageDataRepositories, s GetScoreScaleItemPageDataServices) *GetScoreScaleItemPageDataUseCase {
	return &GetScoreScaleItemPageDataUseCase{repositories: r, services: s}
}

func (uc *GetScoreScaleItemPageDataUseCase) Execute(ctx context.Context, req *pb.GetScoreScaleItemPageDataRequest) (*pb.GetScoreScaleItemPageDataResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.ScoreScale, Action: entityid.ActionRead}); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "score_scale.validation.request_required", "Request is required [DEFAULT]"))
	}
	return uc.repositories.ScoreScale.GetScoreScaleItemPageData(ctx, req)
}
