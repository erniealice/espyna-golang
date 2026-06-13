package collection

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"

	advancekindpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common/advance_kind"
	collectionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection"
)

// SettleUnscheduledAdvanceRepositories — only needs the collection repo since
// settlement is a counter-only operation (no recognition emission).
type SettleUnscheduledAdvanceRepositories struct {
	TreasuryCollection collectionpb.CollectionDomainServiceServer
}

// SettleUnscheduledAdvanceServices groups infra services.
type SettleUnscheduledAdvanceServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// SettleUnscheduledAdvanceUseCase records the settlement and flips status.
type SettleUnscheduledAdvanceUseCase struct {
	repositories SettleUnscheduledAdvanceRepositories
	services     SettleUnscheduledAdvanceServices
	update       *UpdateCollectionUseCase // Q1-B routing
}

// NewSettleUnscheduledAdvanceUseCase wires the use case.
func NewSettleUnscheduledAdvanceUseCase(
	repos SettleUnscheduledAdvanceRepositories,
	svcs SettleUnscheduledAdvanceServices,
	update *UpdateCollectionUseCase,
) *SettleUnscheduledAdvanceUseCase {
	return &SettleUnscheduledAdvanceUseCase{repositories: repos, services: svcs, update: update}
}

// Execute records a settlement and updates the advance counters.
//
// Behavior:
//   - Validate advance_kind = UNSCHEDULED + advance_status = ACTIVE.
//   - Decrement advance_remaining_amount by Amount.
//   - When remaining hits 0 → status=SETTLED; otherwise → status=PARTIALLY_SETTLED.
//   - DOES NOT emit a Revenue row. Settlement is a cash event; recognition
//     (if any) is a separate operator action.
func (uc *SettleUnscheduledAdvanceUseCase) Execute(
	ctx context.Context,
	req *collectionpb.SettleUnscheduledAdvanceCollectionRequest,
) (*collectionpb.SettleUnscheduledAdvanceCollectionResponse, error) {
	if req == nil {
		req = &collectionpb.SettleUnscheduledAdvanceCollectionRequest{}
	}
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityTreasuryCollection,
		Action: entityid.ActionUpdate,
	}); err != nil {
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
			"treasury_collection.validation.settle_amount_required",
			"settle amount must be > 0 [DEFAULT]",
		))
	}

	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		var out *collectionpb.SettleUnscheduledAdvanceCollectionResponse
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

func (uc *SettleUnscheduledAdvanceUseCase) executeCore(
	ctx context.Context,
	req *collectionpb.SettleUnscheduledAdvanceCollectionRequest,
) (*collectionpb.SettleUnscheduledAdvanceCollectionResponse, error) {
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
			"treasury_collection.errors.settle_requires_unscheduled",
			"settle is only valid for advance_kind=UNSCHEDULED [DEFAULT]",
		))
	}
	if adv.GetAdvanceStatus() != advancekindpb.AdvanceStatus_ADVANCE_STATUS_ACTIVE &&
		adv.GetAdvanceStatus() != advancekindpb.AdvanceStatus_ADVANCE_STATUS_PARTIALLY_SETTLED {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"treasury_collection.errors.settle_requires_active_or_partial",
			"settle requires advance_status=ACTIVE or PARTIALLY_SETTLED [DEFAULT]",
		))
	}

	remaining := adv.GetAdvanceRemainingAmount()
	if req.GetAmount() > remaining {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"treasury_collection.errors.settle_amount_exceeds_remaining",
			"settle amount exceeds advance_remaining_amount [DEFAULT]",
		))
	}

	newRemaining := remaining - req.GetAmount()
	newRecognized := adv.GetAdvanceRecognizedAmount() + req.GetAmount()
	var newStatus advancekindpb.AdvanceStatus
	if newRemaining == 0 {
		newStatus = advancekindpb.AdvanceStatus_ADVANCE_STATUS_SETTLED
	} else {
		newStatus = advancekindpb.AdvanceStatus_ADVANCE_STATUS_PARTIALLY_SETTLED
	}

	adv.AdvanceRemainingAmount = &newRemaining
	adv.AdvanceRecognizedAmount = &newRecognized
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

	return &collectionpb.SettleUnscheduledAdvanceCollectionResponse{
		NewRemainingAmount:  newRemaining,
		NewRecognizedAmount: newRecognized,
		NewStatus:           newStatus,
	}, nil
}
