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

type UpdateRatingDescriptionSetRepositories struct {
	RatingDescriptionSet pb.RatingDescriptionSetDomainServiceServer
}

type UpdateRatingDescriptionSetServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// UpdateRatingDescriptionSetUseCase updates a rating_description_set's
// header fields. It requires the injected repository to satisfy
// domainports.RatingDescriptionSetLifecycleRepository (LockForUpdate +
// UpdateIfDraft) via a constructor-time type assertion, mirroring
// PublishRatingDescriptionSetUseCase.
type UpdateRatingDescriptionSetUseCase struct {
	repo            domainports.RatingDescriptionSetLifecycleRepository
	repositoryError error
	services        UpdateRatingDescriptionSetServices
}

func NewUpdateRatingDescriptionSetUseCase(r UpdateRatingDescriptionSetRepositories, s UpdateRatingDescriptionSetServices) *UpdateRatingDescriptionSetUseCase {
	lifecycle, ok := r.RatingDescriptionSet.(domainports.RatingDescriptionSetLifecycleRepository)
	uc := &UpdateRatingDescriptionSetUseCase{repo: lifecycle, services: s}
	if !ok {
		uc.repositoryError = fmt.Errorf("rating description set repository %T does not support the conditional lock/update transition", r.RatingDescriptionSet)
	}
	return uc
}

// Execute updates a rating_description_set's header fields. Only while DRAFT
// (schema-proposal.md §9.2, interfaces.md §4 "rating_description_set.edit ...
// UpdateRatingDescriptionSet (DRAFT only)") — PUBLISHED/DEPRECATED sets are
// immutable and reject with SET_NOT_DRAFT.
//
// W3 follow-up finding (2026-09-25, codex-review-impl1.out.md): the DRAFT
// check and the write now happen under the SAME row lock inside ONE
// transaction (no non-transactional fallback) — LockRatingDescriptionSetForUpdate
// takes `FOR UPDATE` first, then (still inside the transaction)
// UpdateRatingDescriptionSetIfDraft writes the row, having unconditionally
// stripped version_status/version/supersedes_id from the payload (those are
// lifecycle-owned, changed only through Publish/Deprecate/CreateVersion —
// never copied back from a pre-read snapshot, which a concurrent Publish
// could otherwise have raced past between the read and this write).
func (uc *UpdateRatingDescriptionSetUseCase) Execute(ctx context.Context, req *pb.UpdateRatingDescriptionSetRequest) (*pb.UpdateRatingDescriptionSetResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.RatingDescriptionSet, Action: entityid.ActionUpdate}); err != nil {
		return nil, err
	}
	if uc.repositoryError != nil {
		return nil, uc.repositoryError
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "rating_description_set.validation.data_required", "Data is required [DEFAULT]"))
	}
	if uc.services.Transactor == nil || !uc.services.Transactor.SupportsTransactions() {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "rating_description_set.errors.transactor_unavailable", "[ERR-DEFAULT] Update requires transaction support"))
	}

	var updated *pb.RatingDescriptionSet
	err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		status, lockErr := uc.repo.LockRatingDescriptionSetForUpdate(txCtx, req.Data.Id)
		if lockErr != nil {
			return lockErr
		}
		if status != enumspb.VersionStatus_VERSION_STATUS_DRAFT {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "rating_description_set.errors.set_not_draft", "SET_NOT_DRAFT: rating description set can only be edited while DRAFT"))
		}

		// fix2-backend (codex impl2 #4): a changed score_scale must be owned
		// by the caller's workspace (or global); fail closed without a validator.
		if req.Data.ScoreScaleId != "" {
			validator, ok := uc.repo.(domainports.RatingDescriptionSetScaleValidator)
			if !ok {
				return fmt.Errorf("rating description set repository %T does not support score scale ownership validation", uc.repo)
			}
			if vErr := validator.ValidateRatingDescriptionSetScoreScale(txCtx, req.Data.ScoreScaleId); vErr != nil {
				return vErr
			}
		}

		// Lifecycle-owned fields are never written by this generic update,
		// whatever the caller supplied — the adapter also strips them
		// unconditionally (belt and suspenders), but clearing them here keeps
		// the request object honest for any caller that inspects it after.
		req.Data.VersionStatus = enumspb.VersionStatus_VERSION_STATUS_UNSPECIFIED
		req.Data.Version = 0
		req.Data.SupersedesId = nil

		now := time.Now()
		ms := now.UnixMilli()
		s := now.Format(time.RFC3339)
		req.Data.DateModified = &ms
		req.Data.DateModifiedString = &s

		item, writeErr := uc.repo.UpdateRatingDescriptionSetIfDraft(txCtx, req)
		if writeErr != nil {
			return writeErr
		}
		updated = item
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &pb.UpdateRatingDescriptionSetResponse{Data: []*pb.RatingDescriptionSet{updated}, Success: true}, nil
}
