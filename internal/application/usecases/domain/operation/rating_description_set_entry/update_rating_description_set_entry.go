package rating_description_set_entry

import (
	"context"
	"errors"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/rating_description_set_entry"
)

type UpdateRatingDescriptionSetEntryRepositories struct {
	RatingDescriptionSetEntry pb.RatingDescriptionSetEntryDomainServiceServer
}

type UpdateRatingDescriptionSetEntryServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

type UpdateRatingDescriptionSetEntryUseCase struct {
	repositories UpdateRatingDescriptionSetEntryRepositories
	services     UpdateRatingDescriptionSetEntryServices
}

func NewUpdateRatingDescriptionSetEntryUseCase(r UpdateRatingDescriptionSetEntryRepositories, s UpdateRatingDescriptionSetEntryServices) *UpdateRatingDescriptionSetEntryUseCase {
	return &UpdateRatingDescriptionSetEntryUseCase{repositories: r, services: s}
}

// Authorization (W3 follow-up finding, 2026-09-25): checked against the
// PARENT set (rating_description_set:update) — entries carry no permission
// codes of their own. See create_rating_description_set_entry.go's doc
// comment.
func (uc *UpdateRatingDescriptionSetEntryUseCase) Execute(ctx context.Context, req *pb.UpdateRatingDescriptionSetEntryRequest) (*pb.UpdateRatingDescriptionSetEntryResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.RatingDescriptionSet, Action: entityid.ActionUpdate}); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "rating_description_set_entry.validation.data_required", "Data is required [DEFAULT]"))
	}
	now := time.Now()
	ms := now.UnixMilli()
	s := now.Format(time.RFC3339)
	req.Data.DateModified = &ms
	req.Data.DateModifiedString = &s

	if uc.services.Transactor == nil || !uc.services.Transactor.SupportsTransactions() {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "rating_description_set_entry.errors.transactor_unavailable", "[ERR-DEFAULT] Update requires transaction support"))
	}
	var resp *pb.UpdateRatingDescriptionSetEntryResponse
	err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		r, txErr := uc.repositories.RatingDescriptionSetEntry.UpdateRatingDescriptionSetEntry(txCtx, req)
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
