package collection

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"

	advancekindpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common/advance_kind"
	collectionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection"
)

// CancelAdvanceRepositories — only needs the collection repo since cancel is
// a counter-only operation (no recognition emission).
type CancelAdvanceRepositories struct {
	TreasuryCollection collectionpb.CollectionDomainServiceServer
}

// CancelAdvanceServices groups infra services.
type CancelAdvanceServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// CancelAdvanceUseCase flips advance_status to CANCELLED.
type CancelAdvanceUseCase struct {
	repositories CancelAdvanceRepositories
	services     CancelAdvanceServices
	// 20260518-hexagonal-strict-adherence Q1-B (LOCKED) — route the terminal
	// row update through the wrapping UpdateCollection use case so the
	// BURN_DOWN guard + authcheck on the entity update path always fire,
	// even when the workflow reaches the adapter via this advance use case.
	// The wrapping use case is transaction-aware (IsTransactionActive check)
	// so this does not start a nested independent transaction.
	update *UpdateCollectionUseCase
}

// NewCancelAdvanceUseCase wires the use case.
func NewCancelAdvanceUseCase(
	repos CancelAdvanceRepositories,
	svcs CancelAdvanceServices,
	update *UpdateCollectionUseCase,
) *CancelAdvanceUseCase {
	return &CancelAdvanceUseCase{repositories: repos, services: svcs, update: update}
}

// Execute cancels an active advance Collection.
//
// Behavior:
//   - Validate advance_kind != NONE and advance_status ∈ {ACTIVE, PARTIALLY_SETTLED}.
//   - Reason is REQUIRED — empty reason rejects.
//   - Flip status to CANCELLED.
//   - DOES NOT emit a Revenue row.
func (uc *CancelAdvanceUseCase) Execute(
	ctx context.Context,
	req *collectionpb.CancelAdvanceCollectionRequest,
) (*collectionpb.CancelAdvanceCollectionResponse, error) {
	if req == nil {
		req = &collectionpb.CancelAdvanceCollectionRequest{}
	}
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityTreasuryCollection, ports.ActionUpdate); err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.GetTreasuryCollectionId()) == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"treasury_collection.validation.id_required",
			"treasury_collection_id is required [DEFAULT]",
		))
	}
	if strings.TrimSpace(req.GetReason()) == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"treasury_collection.validation.cancel_reason_required",
			"cancel reason is required [DEFAULT]",
		))
	}

	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		var out *collectionpb.CancelAdvanceCollectionResponse
		err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
			res, execErr := uc.executeCore(txCtx, req)
			if execErr != nil {
				return execErr
			}
			out = res
			return nil
		})
		if err != nil {
			return nil, err
		}
		return out, nil
	}
	return uc.executeCore(ctx, req)
}

func (uc *CancelAdvanceUseCase) executeCore(
	ctx context.Context,
	req *collectionpb.CancelAdvanceCollectionRequest,
) (*collectionpb.CancelAdvanceCollectionResponse, error) {
	readResp, err := uc.repositories.TreasuryCollection.ReadCollection(ctx, &collectionpb.ReadCollectionRequest{
		Data: &collectionpb.Collection{Id: req.GetTreasuryCollectionId()},
	})
	if err != nil {
		return nil, err
	}
	if readResp == nil || len(readResp.GetData()) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"treasury_collection.errors.not_found",
			"treasury_collection not found [DEFAULT]",
		))
	}
	adv := readResp.GetData()[0]

	if adv.GetAdvanceKind() == advancekindpb.AdvanceKind_ADVANCE_KIND_NONE {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"treasury_collection.errors.cancel_requires_advance",
			"cancel is only valid when advance_kind != NONE [DEFAULT]",
		))
	}
	switch adv.GetAdvanceStatus() {
	case advancekindpb.AdvanceStatus_ADVANCE_STATUS_ACTIVE,
		advancekindpb.AdvanceStatus_ADVANCE_STATUS_PARTIALLY_SETTLED:
		// proceed
	default:
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"treasury_collection.errors.cancel_requires_open_advance",
			"cancel requires advance_status=ACTIVE or PARTIALLY_SETTLED [DEFAULT]",
		))
	}

	newStatus := advancekindpb.AdvanceStatus_ADVANCE_STATUS_CANCELLED
	adv.AdvanceStatus = &newStatus
	now := time.Now()
	dm := now.UnixMilli()
	dmStr := now.Format(time.RFC3339)
	adv.DateModified = &dm
	adv.DateModifiedString = &dmStr

	if _, err := uc.update.Execute(ctx, &collectionpb.UpdateCollectionRequest{
		Data: adv,
	}); err != nil {
		return nil, err
	}

	return &collectionpb.CancelAdvanceCollectionResponse{NewStatus: newStatus}, nil
}
