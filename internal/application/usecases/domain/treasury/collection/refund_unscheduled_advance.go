package collection

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"

	advancekindpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common/advance_kind"
	collectionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection"
)

// RefundUnscheduledAdvanceRepositories — collection repo only.
type RefundUnscheduledAdvanceRepositories struct {
	TreasuryCollection collectionpb.CollectionDomainServiceServer
}

// RefundUnscheduledAdvanceServices groups infra services.
type RefundUnscheduledAdvanceServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// RefundUnscheduledAdvanceUseCase records the refund and flips status.
type RefundUnscheduledAdvanceUseCase struct {
	repositories RefundUnscheduledAdvanceRepositories
	services     RefundUnscheduledAdvanceServices
	update       *UpdateCollectionUseCase // Q1-B routing
}

// NewRefundUnscheduledAdvanceUseCase wires the use case.
func NewRefundUnscheduledAdvanceUseCase(
	repos RefundUnscheduledAdvanceRepositories,
	svcs RefundUnscheduledAdvanceServices,
	update *UpdateCollectionUseCase,
) *RefundUnscheduledAdvanceUseCase {
	return &RefundUnscheduledAdvanceUseCase{repositories: repos, services: svcs, update: update}
}

// Execute records a refund and updates the advance counters/status to REFUNDED.
//
// Behavior:
//   - Validate advance_kind = UNSCHEDULED + advance_status ∈ {ACTIVE, PARTIALLY_SETTLED}.
//   - Decrement advance_remaining_amount by Amount.
//   - Flip status to REFUNDED (single status for both full and partial refunds
//     per plan §"Settle / Refund drawers"; partial refunds may continue to be
//     followed by additional refunds while remaining > 0).
//   - DOES NOT emit a Revenue row.
func (uc *RefundUnscheduledAdvanceUseCase) Execute(
	ctx context.Context,
	req *collectionpb.RefundUnscheduledAdvanceCollectionRequest,
) (*collectionpb.RefundUnscheduledAdvanceCollectionResponse, error) {
	if req == nil {
		req = &collectionpb.RefundUnscheduledAdvanceCollectionRequest{}
	}
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityTreasuryCollection, entityid.ActionUpdate); err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.GetTreasuryCollectionId()) == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"treasury_collection.validation.id_required",
			"treasury_collection_id is required [DEFAULT]",
		))
	}
	if req.GetAmount() <= 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"treasury_collection.validation.refund_amount_required",
			"refund amount must be > 0 [DEFAULT]",
		))
	}

	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		var out *collectionpb.RefundUnscheduledAdvanceCollectionResponse
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

func (uc *RefundUnscheduledAdvanceUseCase) executeCore(
	ctx context.Context,
	req *collectionpb.RefundUnscheduledAdvanceCollectionRequest,
) (*collectionpb.RefundUnscheduledAdvanceCollectionResponse, error) {
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

	if adv.GetAdvanceKind() != advancekindpb.AdvanceKind_ADVANCE_KIND_UNSCHEDULED {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"treasury_collection.errors.refund_requires_unscheduled",
			"refund is only valid for advance_kind=UNSCHEDULED [DEFAULT]",
		))
	}
	switch adv.GetAdvanceStatus() {
	case advancekindpb.AdvanceStatus_ADVANCE_STATUS_ACTIVE,
		advancekindpb.AdvanceStatus_ADVANCE_STATUS_PARTIALLY_SETTLED:
		// proceed
	default:
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"treasury_collection.errors.refund_requires_open_advance",
			"refund requires advance_status=ACTIVE or PARTIALLY_SETTLED [DEFAULT]",
		))
	}

	remaining := adv.GetAdvanceRemainingAmount()
	if req.GetAmount() > remaining {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"treasury_collection.errors.refund_amount_exceeds_remaining",
			"refund amount exceeds advance_remaining_amount [DEFAULT]",
		))
	}

	newRemaining := remaining - req.GetAmount()
	newStatus := advancekindpb.AdvanceStatus_ADVANCE_STATUS_REFUNDED

	adv.AdvanceRemainingAmount = &newRemaining
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

	return &collectionpb.RefundUnscheduledAdvanceCollectionResponse{
		NewRemainingAmount: newRemaining,
		NewStatus:          newStatus,
	}, nil
}
