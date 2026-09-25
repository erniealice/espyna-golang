package rating_description_set

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	domainports "github.com/erniealice/espyna-golang/internal/application/ports/domain"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	enumspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/enums"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/rating_description_set"
)

type CreateRatingDescriptionSetRepositories struct {
	RatingDescriptionSet pb.RatingDescriptionSetDomainServiceServer
}

type CreateRatingDescriptionSetServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

type CreateRatingDescriptionSetUseCase struct {
	repositories CreateRatingDescriptionSetRepositories
	services     CreateRatingDescriptionSetServices
}

func NewCreateRatingDescriptionSetUseCase(r CreateRatingDescriptionSetRepositories, s CreateRatingDescriptionSetServices) *CreateRatingDescriptionSetUseCase {
	return &CreateRatingDescriptionSetUseCase{repositories: r, services: s}
}

// Execute creates a new rating_description_set. Every new set starts DRAFT,
// version 1 (schema-proposal.md §9.2 "Create set | DRAFT, version 1") —
// callers cannot self-select a different starting status or version; new
// versions of an existing set are created via CreateRatingDescriptionSetVersion
// (deferred this wave, see W3-ESPYNA.done), never by posting version > 1 here.
func (uc *CreateRatingDescriptionSetUseCase) Execute(ctx context.Context, req *pb.CreateRatingDescriptionSetRequest) (*pb.CreateRatingDescriptionSetResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.RatingDescriptionSet, Action: entityid.ActionCreate}); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "rating_description_set.validation.data_required", "Data is required [DEFAULT]"))
	}
	if req.Data.Name == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "rating_description_set.validation.name_required", "Name is required [DEFAULT]"))
	}
	if req.Data.ScoreScaleId == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "rating_description_set.validation.score_scale_required", "Score scale is required [DEFAULT]"))
	}
	// fix2-backend (codex impl2 #4): the referenced score_scale must be owned
	// by the caller's workspace (or global). Fail closed when the repository
	// cannot validate.
	validator, ok := uc.repositories.RatingDescriptionSet.(domainports.RatingDescriptionSetScaleValidator)
	if !ok {
		return nil, fmt.Errorf("rating description set repository %T does not support score scale ownership validation", uc.repositories.RatingDescriptionSet)
	}
	// fix3-backend (codex-review-impl3 #4): the insert and its generic diff
	// audit event (audited dbOps, rating_description_set.go init()) must be
	// atomic — an audit failure rolls the insert back instead of leaving a set
	// the UI reported as failed. No non-transactional fallback (fail closed),
	// mirroring UpdateRatingDescriptionSetUseCase.
	if uc.services.Transactor == nil || !uc.services.Transactor.SupportsTransactions() {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "rating_description_set.errors.transactor_unavailable", "[ERR-DEFAULT] Create requires transaction support"))
	}
	uc.enrich(req.Data)
	var resp *pb.CreateRatingDescriptionSetResponse
	err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		if vErr := validator.ValidateRatingDescriptionSetScoreScale(txCtx, req.Data.ScoreScaleId); vErr != nil {
			return vErr
		}
		created, cErr := uc.repositories.RatingDescriptionSet.CreateRatingDescriptionSet(txCtx, req)
		if cErr != nil {
			return cErr
		}
		resp = created
		return nil
	})
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (uc *CreateRatingDescriptionSetUseCase) enrich(data *pb.RatingDescriptionSet) {
	now := time.Now()
	if data.Id == "" && uc.services.IDGenerator != nil {
		data.Id = uc.services.IDGenerator.GenerateID()
	}
	data.Active = true
	data.Version = 1
	data.VersionStatus = enumspb.VersionStatus_VERSION_STATUS_DRAFT
	data.SupersedesId = nil
	ms := now.UnixMilli()
	s := now.Format(time.RFC3339)
	data.DateCreated = &ms
	data.DateCreatedString = &s
	data.DateModified = &ms
	data.DateModifiedString = &s
}
