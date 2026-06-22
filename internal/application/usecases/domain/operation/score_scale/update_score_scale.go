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

type UpdateScoreScaleRepositories struct {
	ScoreScale pb.ScoreScaleDomainServiceServer
}

type UpdateScoreScaleServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

type UpdateScoreScaleUseCase struct {
	repositories UpdateScoreScaleRepositories
	services     UpdateScoreScaleServices
}

func NewUpdateScoreScaleUseCase(r UpdateScoreScaleRepositories, s UpdateScoreScaleServices) *UpdateScoreScaleUseCase {
	return &UpdateScoreScaleUseCase{repositories: r, services: s}
}

func (uc *UpdateScoreScaleUseCase) Execute(ctx context.Context, req *pb.UpdateScoreScaleRequest) (*pb.UpdateScoreScaleResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.ScoreScale, Action: entityid.ActionUpdate}); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "score_scale.validation.data_required", "Data is required [DEFAULT]"))
	}
	now := time.Now()
	ms := now.UnixMilli()
	s := now.Format(time.RFC3339)
	req.Data.DateModified = &ms
	req.Data.DateModifiedString = &s
	return uc.repositories.ScoreScale.UpdateScoreScale(ctx, req)
}
