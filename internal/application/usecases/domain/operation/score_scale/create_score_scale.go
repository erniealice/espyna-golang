package score_scale

import (
	"context"
	"errors"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/score_scale"
)

type CreateScoreScaleRepositories struct {
	ScoreScale pb.ScoreScaleDomainServiceServer
}

type CreateScoreScaleServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

type CreateScoreScaleUseCase struct {
	repositories CreateScoreScaleRepositories
	services     CreateScoreScaleServices
}

func NewCreateScoreScaleUseCase(r CreateScoreScaleRepositories, s CreateScoreScaleServices) *CreateScoreScaleUseCase {
	return &CreateScoreScaleUseCase{repositories: r, services: s}
}

func (uc *CreateScoreScaleUseCase) Execute(ctx context.Context, req *pb.CreateScoreScaleRequest) (*pb.CreateScoreScaleResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.ScoreScale, Action: entityid.ActionCreate}); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "score_scale.validation.data_required", "Data is required [DEFAULT]"))
	}
	uc.enrich(req.Data)
	return uc.repositories.ScoreScale.CreateScoreScale(ctx, req)
}

func (uc *CreateScoreScaleUseCase) enrich(data *pb.ScoreScale) {
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
