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

type GetScoreScaleBandItemPageDataRepositories struct {
	ScoreScaleBand pb.ScoreScaleBandDomainServiceServer
}

type GetScoreScaleBandItemPageDataServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

type GetScoreScaleBandItemPageDataUseCase struct {
	repositories GetScoreScaleBandItemPageDataRepositories
	services     GetScoreScaleBandItemPageDataServices
}

func NewGetScoreScaleBandItemPageDataUseCase(r GetScoreScaleBandItemPageDataRepositories, s GetScoreScaleBandItemPageDataServices) *GetScoreScaleBandItemPageDataUseCase {
	return &GetScoreScaleBandItemPageDataUseCase{repositories: r, services: s}
}

func (uc *GetScoreScaleBandItemPageDataUseCase) Execute(ctx context.Context, req *pb.GetScoreScaleBandItemPageDataRequest) (*pb.GetScoreScaleBandItemPageDataResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.ScoreScaleBand, Action: entityid.ActionRead}); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "score_scale_band.validation.request_required", "Request is required [DEFAULT]"))
	}
	return uc.repositories.ScoreScaleBand.GetScoreScaleBandItemPageData(ctx, req)
}
