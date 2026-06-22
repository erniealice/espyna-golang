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

type DeleteScoreScaleRepositories struct {
	ScoreScale pb.ScoreScaleDomainServiceServer
}

type DeleteScoreScaleServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

type DeleteScoreScaleUseCase struct {
	repositories DeleteScoreScaleRepositories
	services     DeleteScoreScaleServices
}

func NewDeleteScoreScaleUseCase(r DeleteScoreScaleRepositories, s DeleteScoreScaleServices) *DeleteScoreScaleUseCase {
	return &DeleteScoreScaleUseCase{repositories: r, services: s}
}

func (uc *DeleteScoreScaleUseCase) Execute(ctx context.Context, req *pb.DeleteScoreScaleRequest) (*pb.DeleteScoreScaleResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.ScoreScale, Action: entityid.ActionDelete}); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "score_scale.validation.request_required", "Request is required [DEFAULT]"))
	}
	return uc.repositories.ScoreScale.DeleteScoreScale(ctx, req)
}
