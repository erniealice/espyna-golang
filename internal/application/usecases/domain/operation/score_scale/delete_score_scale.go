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
	// fix2-backend (codex impl2 #1): DeleteScoreScale deactivates the scale
	// (generic Delete sets active=false), which would silently drop every
	// PUBLISHED/DEPRECATED descriptor entry of its bands from resolution. The
	// postgres adapter guards it (lock scale FOR UPDATE → BAND_LOCKED when a
	// band is referenced by a published/deprecated set); that guard requires an
	// ambient transaction, so the delete runs inside one (no non-tx fallback).
	if uc.services.Transactor == nil || !uc.services.Transactor.SupportsTransactions() {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "score_scale.errors.transactor_unavailable", "[ERR-DEFAULT] Delete requires transaction support"))
	}
	var resp *pb.DeleteScoreScaleResponse
	err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		r, txErr := uc.repositories.ScoreScale.DeleteScoreScale(txCtx, req)
		if txErr != nil {
			return txErr
		}
		resp = r
		return nil
	})
	if err != nil {
		return nil, err
	}
	return resp, nil
}
