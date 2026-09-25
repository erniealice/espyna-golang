package rating_description_set

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	domainports "github.com/erniealice/espyna-golang/internal/application/ports/domain"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/rating_description_set"
	entrypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/rating_description_set_entry"
)

type PublishRatingDescriptionSetServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// PublishRatingDescriptionSetUseCase transitions a set DRAFT -> PUBLISHED
// (schema-proposal.md §9.2). It requires the injected repository to satisfy
// domainports.RatingDescriptionSetLifecycleRepository — the postgres adapter
// does (PublishRatingDescriptionSetIfDraft) — via a constructor-time type
// assertion, mirroring subscription_group_document_template.DeleteDraftPairUseCase.
type PublishRatingDescriptionSetUseCase struct {
	repo            domainports.RatingDescriptionSetLifecycleRepository
	scaleLocker     domainports.RatingDescriptionSetPublishLocker // fix2-backend: lock order scale → set
	entryRepo       entrypb.RatingDescriptionSetEntryDomainServiceServer
	repositoryError error
	services        PublishRatingDescriptionSetServices
}

func NewPublishRatingDescriptionSetUseCase(
	repository pb.RatingDescriptionSetDomainServiceServer,
	entryRepository entrypb.RatingDescriptionSetEntryDomainServiceServer,
	services PublishRatingDescriptionSetServices,
) *PublishRatingDescriptionSetUseCase {
	lifecycle, ok := repository.(domainports.RatingDescriptionSetLifecycleRepository)
	uc := &PublishRatingDescriptionSetUseCase{repo: lifecycle, entryRepo: entryRepository, services: services}
	if !ok {
		uc.repositoryError = fmt.Errorf("rating description set repository %T does not support the conditional publish/deprecate transitions", repository)
	}
	// fix2-backend (codex impl2 #1): publish must join the descriptor lock
	// protocol (scale → band → set); a repository without it fails closed.
	locker, lockOK := repository.(domainports.RatingDescriptionSetPublishLocker)
	uc.scaleLocker = locker
	if !lockOK && uc.repositoryError == nil {
		uc.repositoryError = fmt.Errorf("rating description set repository %T does not support the publish scale lock protocol", repository)
	}
	return uc
}

// Execute publishes a DRAFT set: requires >= 1 entry (schema-proposal.md §9.2
// "Publish | ... requires ≥1 entry"), then runs the conditional
// DRAFT->PUBLISHED UPDATE inside one transaction (no non-transactional
// fallback — Q-27/§9.2 "if the transactor is unavailable the use case
// fails").
//
// W3 follow-up finding (2026-09-25, codex-review-impl1.out.md): the entry
// count is now taken AFTER the parent row lock, inside the SAME transaction —
// LockRatingDescriptionSetForUpdate takes `FOR UPDATE` on the set row first;
// only then is the entry count queried (on txCtx, so it observes the locked
// row's consistent state) and the conditional publish performed. A concurrent
// entry delete takes `FOR SHARE` on this same row
// (guardRatingDescriptionSetEntryWrite), so it blocks until this transaction
// ends — the last entry can no longer disappear between the count and the
// publish.
func (uc *PublishRatingDescriptionSetUseCase) Execute(ctx context.Context, req *pb.PublishRatingDescriptionSetRequest) (*pb.PublishRatingDescriptionSetResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.RatingDescriptionSet, Action: entityid.ActionPublish}); err != nil {
		return nil, err
	}
	if uc.repositoryError != nil {
		return nil, uc.repositoryError
	}
	if req == nil || req.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "rating_description_set.validation.id_required", "[ERR-DEFAULT] Rating description set ID is required"))
	}
	if uc.entryRepo == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "rating_description_set.errors.entry_repository_unavailable", "[ERR-DEFAULT] Rating description set entry repository is unavailable"))
	}
	if uc.services.Transactor == nil || !uc.services.Transactor.SupportsTransactions() {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "rating_description_set.errors.transactor_unavailable", "[ERR-DEFAULT] Publish requires transaction support"))
	}
	var published *pb.RatingDescriptionSet
	err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		// Lock order (fix2-backend, codex impl2 #1): score_scale FOR SHARE
		// FIRST, then the set FOR UPDATE, then verify the set still declares
		// that scale and all entry bands belong to it. Scale/band meaning
		// editors lock score_scale FOR UPDATE first, so they serialize with
		// this publish (see postgres score_scale.go "Descriptor lock protocol").
		scaleID, scaleErr := uc.scaleLocker.LockScoreScaleForPublish(txCtx, req.Id)
		if scaleErr != nil {
			return scaleErr
		}
		if _, lockErr := uc.repo.LockRatingDescriptionSetForUpdate(txCtx, req.Id); lockErr != nil {
			return lockErr
		}
		if verifyErr := uc.scaleLocker.VerifyRatingDescriptionSetPublishScale(txCtx, req.Id, scaleID); verifyErr != nil {
			return verifyErr
		}
		entriesResp, listErr := uc.entryRepo.ListRatingDescriptionSetEntries(txCtx, &entrypb.ListRatingDescriptionSetEntriesRequest{
			Filters: equalsFilter("rating_description_set_id", req.Id),
		})
		if listErr != nil {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "rating_description_set.errors.entry_check_failed", "[ERR-DEFAULT] Failed to check rating description set entries"))
		}
		if entriesResp == nil || len(entriesResp.Data) == 0 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "rating_description_set.errors.publish_requires_entry", "[ERR-DEFAULT] Rating description set must have at least one entry before it can be published"))
		}
		item, txErr := uc.repo.PublishRatingDescriptionSetIfDraft(txCtx, req.Id)
		if txErr != nil {
			return txErr
		}
		published = item
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &pb.PublishRatingDescriptionSetResponse{Data: []*pb.RatingDescriptionSet{published}, Success: true}, nil
}
