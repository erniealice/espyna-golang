package score_scale_band

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/score_scale_band"
)

type DeleteScoreScaleBandRepositories struct {
	ScoreScaleBand pb.ScoreScaleBandDomainServiceServer
}

type DeleteScoreScaleBandServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

type DeleteScoreScaleBandUseCase struct {
	repositories DeleteScoreScaleBandRepositories
	services     DeleteScoreScaleBandServices
}

func NewDeleteScoreScaleBandUseCase(r DeleteScoreScaleBandRepositories, s DeleteScoreScaleBandServices) *DeleteScoreScaleBandUseCase {
	return &DeleteScoreScaleBandUseCase{repositories: r, services: s}
}

func (uc *DeleteScoreScaleBandUseCase) Execute(ctx context.Context, req *pb.DeleteScoreScaleBandRequest) (*pb.DeleteScoreScaleBandResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.ScoreScaleBand, Action: entityid.ActionDelete}); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "score_scale_band.validation.request_required", "Request is required [DEFAULT]"))
	}
	return uc.repositories.ScoreScaleBand.DeleteScoreScaleBand(ctx, req)
}
