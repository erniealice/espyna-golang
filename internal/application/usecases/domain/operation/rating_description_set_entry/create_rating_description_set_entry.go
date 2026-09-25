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

type CreateRatingDescriptionSetEntryRepositories struct {
	RatingDescriptionSetEntry pb.RatingDescriptionSetEntryDomainServiceServer
}

type CreateRatingDescriptionSetEntryServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

type CreateRatingDescriptionSetEntryUseCase struct {
	repositories CreateRatingDescriptionSetEntryRepositories
	services     CreateRatingDescriptionSetEntryServices
}

func NewCreateRatingDescriptionSetEntryUseCase(r CreateRatingDescriptionSetEntryRepositories, s CreateRatingDescriptionSetEntryServices) *CreateRatingDescriptionSetEntryUseCase {
	return &CreateRatingDescriptionSetEntryUseCase{repositories: r, services: s}
}

// Execute creates an entry. The adapter's write guard (parent set DRAFT,
// band_role != no_description) requires an ambient transaction and fails
// closed without one — every write is wrapped in
// services.Transactor.ExecuteInTransaction (no non-transactional fallback),
// mirroring the rating_description_set lifecycle use cases.
//
// Authorization (W3 follow-up finding, 2026-09-25): entries have no
// permission codes of their own — the agreed twelve permissions (schema-
// proposal.md §9.2, copya.md) grant PARENT-SET permissions only. An entry is
// content that lives entirely inside a rating_description_set, so authoring
// an entry is authorized as rating_description_set:update, the same
// permission that edits the set's own header fields.
func (uc *CreateRatingDescriptionSetEntryUseCase) Execute(ctx context.Context, req *pb.CreateRatingDescriptionSetEntryRequest) (*pb.CreateRatingDescriptionSetEntryResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.RatingDescriptionSet, Action: entityid.ActionUpdate}); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "rating_description_set_entry.validation.data_required", "Data is required [DEFAULT]"))
	}
	if req.Data.RatingDescriptionSetId == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "rating_description_set_entry.validation.set_required", "Rating description set is required [DEFAULT]"))
	}
	if req.Data.OutcomeCriteriaId == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "rating_description_set_entry.validation.criterion_required", "Outcome criteria is required [DEFAULT]"))
	}
	if req.Data.ScoreScaleBandId == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "rating_description_set_entry.validation.band_required", "Score scale band is required [DEFAULT]"))
	}
	if req.Data.Description == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "rating_description_set_entry.validation.description_required", "Description is required [DEFAULT]"))
	}
	uc.enrich(req.Data)

	if uc.services.Transactor == nil || !uc.services.Transactor.SupportsTransactions() {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "rating_description_set_entry.errors.transactor_unavailable", "[ERR-DEFAULT] Create requires transaction support"))
	}
	var resp *pb.CreateRatingDescriptionSetEntryResponse
	err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		r, txErr := uc.repositories.RatingDescriptionSetEntry.CreateRatingDescriptionSetEntry(txCtx, req)
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

func (uc *CreateRatingDescriptionSetEntryUseCase) enrich(data *pb.RatingDescriptionSetEntry) {
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
