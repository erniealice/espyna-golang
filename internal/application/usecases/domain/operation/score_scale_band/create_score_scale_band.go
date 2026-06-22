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

type CreateScoreScaleBandRepositories struct {
	ScoreScaleBand pb.ScoreScaleBandDomainServiceServer
}

type CreateScoreScaleBandServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

type CreateScoreScaleBandUseCase struct {
	repositories CreateScoreScaleBandRepositories
	services     CreateScoreScaleBandServices
}

func NewCreateScoreScaleBandUseCase(r CreateScoreScaleBandRepositories, s CreateScoreScaleBandServices) *CreateScoreScaleBandUseCase {
	return &CreateScoreScaleBandUseCase{repositories: r, services: s}
}

func (uc *CreateScoreScaleBandUseCase) Execute(ctx context.Context, req *pb.CreateScoreScaleBandRequest) (*pb.CreateScoreScaleBandResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.ScoreScaleBand, Action: entityid.ActionCreate}); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "score_scale_band.validation.data_required", "Data is required [DEFAULT]"))
	}
	uc.enrich(req.Data)
	return uc.repositories.ScoreScaleBand.CreateScoreScaleBand(ctx, req)
}

func (uc *CreateScoreScaleBandUseCase) enrich(data *pb.ScoreScaleBand) {
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
