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

	// The BAND_LOCKED guard (contrib/postgres score_scale_band.go,
	// schema-proposal.md §9.3) requires an ambient transaction. Mirrors
	// UpdateScoreScaleBandUseCase / update_rating_description_set_entry.go.
	if uc.services.Transactor == nil || !uc.services.Transactor.SupportsTransactions() {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "score_scale_band.errors.transactor_unavailable", "[ERR-DEFAULT] Delete requires transaction support"))
	}
	var resp *pb.DeleteScoreScaleBandResponse
	err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		r, txErr := uc.repositories.ScoreScaleBand.DeleteScoreScaleBand(txCtx, req)
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
