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

type GetScoreScaleBandListPageDataRepositories struct {
	ScoreScaleBand pb.ScoreScaleBandDomainServiceServer
}

type GetScoreScaleBandListPageDataServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

type GetScoreScaleBandListPageDataUseCase struct {
	repositories GetScoreScaleBandListPageDataRepositories
	services     GetScoreScaleBandListPageDataServices
}

func NewGetScoreScaleBandListPageDataUseCase(r GetScoreScaleBandListPageDataRepositories, s GetScoreScaleBandListPageDataServices) *GetScoreScaleBandListPageDataUseCase {
	return &GetScoreScaleBandListPageDataUseCase{repositories: r, services: s}
}

func (uc *GetScoreScaleBandListPageDataUseCase) Execute(ctx context.Context, req *pb.GetScoreScaleBandListPageDataRequest) (*pb.GetScoreScaleBandListPageDataResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.ScoreScaleBand, Action: entityid.ActionList}); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "score_scale_band.validation.request_required", "Request is required [DEFAULT]"))
	}
	return uc.repositories.ScoreScaleBand.GetScoreScaleBandListPageData(ctx, req)
}
