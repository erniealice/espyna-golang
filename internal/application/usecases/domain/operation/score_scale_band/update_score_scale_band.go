package score_scale_band

import (
	"context"
	"errors"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/score_scale_band"
)

type UpdateScoreScaleBandRepositories struct {
	ScoreScaleBand pb.ScoreScaleBandDomainServiceServer
}

type UpdateScoreScaleBandServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

type UpdateScoreScaleBandUseCase struct {
	repositories UpdateScoreScaleBandRepositories
	services     UpdateScoreScaleBandServices
}

func NewUpdateScoreScaleBandUseCase(r UpdateScoreScaleBandRepositories, s UpdateScoreScaleBandServices) *UpdateScoreScaleBandUseCase {
	return &UpdateScoreScaleBandUseCase{repositories: r, services: s}
}

func (uc *UpdateScoreScaleBandUseCase) Execute(ctx context.Context, req *pb.UpdateScoreScaleBandRequest) (*pb.UpdateScoreScaleBandResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.ScoreScaleBand, Action: entityid.ActionUpdate}); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "score_scale_band.validation.data_required", "Data is required [DEFAULT]"))
	}
	now := time.Now()
	ms := now.UnixMilli()
	s := now.Format(time.RFC3339)
	req.Data.DateModified = &ms
	req.Data.DateModifiedString = &s
	return uc.repositories.ScoreScaleBand.UpdateScoreScaleBand(ctx, req)
}
