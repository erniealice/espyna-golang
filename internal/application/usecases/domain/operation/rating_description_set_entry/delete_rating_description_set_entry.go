package rating_description_set_entry

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/rating_description_set_entry"
)

type DeleteRatingDescriptionSetEntryRepositories struct {
	RatingDescriptionSetEntry pb.RatingDescriptionSetEntryDomainServiceServer
}

type DeleteRatingDescriptionSetEntryServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

type DeleteRatingDescriptionSetEntryUseCase struct {
	repositories DeleteRatingDescriptionSetEntryRepositories
	services     DeleteRatingDescriptionSetEntryServices
}

func NewDeleteRatingDescriptionSetEntryUseCase(r DeleteRatingDescriptionSetEntryRepositories, s DeleteRatingDescriptionSetEntryServices) *DeleteRatingDescriptionSetEntryUseCase {
	return &DeleteRatingDescriptionSetEntryUseCase{repositories: r, services: s}
}

// Authorization (W3 follow-up finding, 2026-09-25): checked against the
// PARENT set (rating_description_set:update) — entries carry no permission
// codes of their own. See create_rating_description_set_entry.go's doc
// comment.
func (uc *DeleteRatingDescriptionSetEntryUseCase) Execute(ctx context.Context, req *pb.DeleteRatingDescriptionSetEntryRequest) (*pb.DeleteRatingDescriptionSetEntryResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.RatingDescriptionSet, Action: entityid.ActionUpdate}); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "rating_description_set_entry.validation.request_required", "Request is required [DEFAULT]"))
	}
	if uc.services.Transactor == nil || !uc.services.Transactor.SupportsTransactions() {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "rating_description_set_entry.errors.transactor_unavailable", "[ERR-DEFAULT] Delete requires transaction support"))
	}
	var resp *pb.DeleteRatingDescriptionSetEntryResponse
	err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		r, txErr := uc.repositories.RatingDescriptionSetEntry.DeleteRatingDescriptionSetEntry(txCtx, req)
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
