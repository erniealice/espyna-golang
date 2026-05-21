package disbursement

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"

	advancekindpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common/advance_kind"
	disbursementpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/disbursement"
)

// RefundUnscheduledAdvanceRepositories — disbursement repo only.
type RefundUnscheduledAdvanceRepositories struct {
	TreasuryDisbursement disbursementpb.DisbursementDomainServiceServer
}

// RefundUnscheduledAdvanceServices groups infra services.
type RefundUnscheduledAdvanceServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// RefundUnscheduledAdvanceUseCase — buying-side mirror.
type RefundUnscheduledAdvanceUseCase struct {
	repositories RefundUnscheduledAdvanceRepositories
	services     RefundUnscheduledAdvanceServices
	update       *UpdateDisbursementUseCase // Q1-B routing
}

// NewRefundUnscheduledAdvanceUseCase wires the use case.
func NewRefundUnscheduledAdvanceUseCase(
	repos RefundUnscheduledAdvanceRepositories,
	svcs RefundUnscheduledAdvanceServices,
	update *UpdateDisbursementUseCase,
) *RefundUnscheduledAdvanceUseCase {
	return &RefundUnscheduledAdvanceUseCase{repositories: repos, services: svcs, update: update}
}

// Execute records a refund and flips status to REFUNDED.
func (uc *RefundUnscheduledAdvanceUseCase) Execute(
	ctx context.Context,
	req *disbursementpb.RefundUnscheduledAdvanceDisbursementRequest,
) (*disbursementpb.RefundUnscheduledAdvanceDisbursementResponse, error) {
	if req == nil {
		req = &disbursementpb.RefundUnscheduledAdvanceDisbursementRequest{}
	}
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityTreasuryDisbursement, ports.ActionUpdate); err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.GetTreasuryDisbursementId()) == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.TranslationService,
			"treasury_disbursement.validation.id_required",
			"treasury_disbursement_id is required [DEFAULT]",
		))
	}
	if req.GetAmount() <= 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.TranslationService,
			"treasury_disbursement.validation.refund_amount_required",
			"refund amount must be > 0 [DEFAULT]",
		))
	}

	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		var out *disbursementpb.RefundUnscheduledAdvanceDisbursementResponse
		err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
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
	req *disbursementpb.RefundUnscheduledAdvanceDisbursementRequest,
) (*disbursementpb.RefundUnscheduledAdvanceDisbursementResponse, error) {
	readResp, err := uc.repositories.TreasuryDisbursement.ReadDisbursement(ctx, &disbursementpb.ReadDisbursementRequest{
		Data: &disbursementpb.Disbursement{Id: req.GetTreasuryDisbursementId()},
	})
	if err != nil {
		return nil, err
	}
	if readResp == nil || len(readResp.GetData()) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.TranslationService,
			"treasury_disbursement.errors.not_found",
			"treasury_disbursement not found [DEFAULT]",
		))
	}
	adv := readResp.GetData()[0]

	if adv.GetAdvanceKind() != advancekindpb.AdvanceKind_ADVANCE_KIND_UNSCHEDULED {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.TranslationService,
			"treasury_disbursement.errors.refund_requires_unscheduled",
			"refund is only valid for advance_kind=UNSCHEDULED [DEFAULT]",
		))
	}
	switch adv.GetAdvanceStatus() {
	case advancekindpb.AdvanceStatus_ADVANCE_STATUS_ACTIVE,
		advancekindpb.AdvanceStatus_ADVANCE_STATUS_PARTIALLY_SETTLED:
		// proceed
	default:
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.TranslationService,
			"treasury_disbursement.errors.refund_requires_open_advance",
			"refund requires advance_status=ACTIVE or PARTIALLY_SETTLED [DEFAULT]",
		))
	}

	remaining := adv.GetAdvanceRemainingAmount()
	if req.GetAmount() > remaining {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.TranslationService,
			"treasury_disbursement.errors.refund_amount_exceeds_remaining",
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

	if _, err := uc.update.Execute(ctx, &disbursementpb.UpdateDisbursementRequest{
		Data: adv,
	}); err != nil {
		return nil, err
	}

	return &disbursementpb.RefundUnscheduledAdvanceDisbursementResponse{
		NewRemainingAmount: newRemaining,
		NewStatus:          newStatus,
	}, nil
}
