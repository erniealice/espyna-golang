package disbursement

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
	disbursementpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/disbursement"
)

// SettleUnscheduledAdvanceRepositories — only the disbursement repo.
type SettleUnscheduledAdvanceRepositories struct {
	TreasuryDisbursement disbursementpb.DisbursementDomainServiceServer
}

// SettleUnscheduledAdvanceServices groups infra services.
type SettleUnscheduledAdvanceServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// SettleUnscheduledAdvanceUseCase — buying-side mirror.
type SettleUnscheduledAdvanceUseCase struct {
	repositories SettleUnscheduledAdvanceRepositories
	services     SettleUnscheduledAdvanceServices
	update       *UpdateDisbursementUseCase // Q1-B routing
}

// NewSettleUnscheduledAdvanceUseCase wires the use case.
func NewSettleUnscheduledAdvanceUseCase(
	repos SettleUnscheduledAdvanceRepositories,
	svcs SettleUnscheduledAdvanceServices,
	update *UpdateDisbursementUseCase,
) *SettleUnscheduledAdvanceUseCase {
	return &SettleUnscheduledAdvanceUseCase{repositories: repos, services: svcs, update: update}
}

// Execute records a settlement and updates the advance counters.
func (uc *SettleUnscheduledAdvanceUseCase) Execute(
	ctx context.Context,
	req *disbursementpb.SettleUnscheduledAdvanceDisbursementRequest,
) (*disbursementpb.SettleUnscheduledAdvanceDisbursementResponse, error) {
	if req == nil {
		req = &disbursementpb.SettleUnscheduledAdvanceDisbursementRequest{}
	}
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityTreasuryDisbursement,
		Action: entityid.ActionUpdate,
	}); err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.GetTreasuryDisbursementId()) == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"treasury_disbursement.validation.id_required",
			"treasury_disbursement_id is required [DEFAULT]",
		))
	}
	if req.GetAmount() <= 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"treasury_disbursement.validation.settle_amount_required",
			"settle amount must be > 0 [DEFAULT]",
		))
	}

	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		var out *disbursementpb.SettleUnscheduledAdvanceDisbursementResponse
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
	req *disbursementpb.SettleUnscheduledAdvanceDisbursementRequest,
) (*disbursementpb.SettleUnscheduledAdvanceDisbursementResponse, error) {
	readResp, err := uc.repositories.TreasuryDisbursement.ReadDisbursement(ctx, &disbursementpb.ReadDisbursementRequest{
		Data: &disbursementpb.Disbursement{Id: req.GetTreasuryDisbursementId()},
	})
	if err != nil {
		return nil, err
	}
	if readResp == nil || len(readResp.GetData()) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"treasury_disbursement.errors.not_found",
			"treasury_disbursement not found [DEFAULT]",
		))
	}
	adv := readResp.GetData()[0]

	if adv.GetAdvanceKind() != advancekindpb.AdvanceKind_ADVANCE_KIND_UNSCHEDULED {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"treasury_disbursement.errors.settle_requires_unscheduled",
			"settle is only valid for advance_kind=UNSCHEDULED [DEFAULT]",
		))
	}
	if adv.GetAdvanceStatus() != advancekindpb.AdvanceStatus_ADVANCE_STATUS_ACTIVE &&
		adv.GetAdvanceStatus() != advancekindpb.AdvanceStatus_ADVANCE_STATUS_PARTIALLY_SETTLED {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"treasury_disbursement.errors.settle_requires_active_or_partial",
			"settle requires advance_status=ACTIVE or PARTIALLY_SETTLED [DEFAULT]",
		))
	}

	remaining := adv.GetAdvanceRemainingAmount()
	if req.GetAmount() > remaining {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"treasury_disbursement.errors.settle_amount_exceeds_remaining",
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

	if _, err := uc.update.Execute(ctx, &disbursementpb.UpdateDisbursementRequest{
		Data: adv,
	}); err != nil {
		return nil, err
	}

	return &disbursementpb.SettleUnscheduledAdvanceDisbursementResponse{
		NewRemainingAmount:  newRemaining,
		NewRecognizedAmount: newRecognized,
		NewStatus:           newStatus,
	}, nil
}
