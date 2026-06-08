package disbursement

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
	disbursementpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/disbursement"
)

// CancelAdvanceRepositories — disbursement repo only.
type CancelAdvanceRepositories struct {
	TreasuryDisbursement disbursementpb.DisbursementDomainServiceServer
}

// CancelAdvanceServices groups infra services.
type CancelAdvanceServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// CancelAdvanceUseCase — buying-side mirror.
type CancelAdvanceUseCase struct {
	repositories CancelAdvanceRepositories
	services     CancelAdvanceServices
	update       *UpdateDisbursementUseCase // Q1-B routing
}

// NewCancelAdvanceUseCase wires the use case.
func NewCancelAdvanceUseCase(
	repos CancelAdvanceRepositories,
	svcs CancelAdvanceServices,
	update *UpdateDisbursementUseCase,
) *CancelAdvanceUseCase {
	return &CancelAdvanceUseCase{repositories: repos, services: svcs, update: update}
}

// Execute cancels an active advance Disbursement.
//
// Behavior:
//   - Validate advance_kind != NONE and advance_status ∈ {ACTIVE, PARTIALLY_SETTLED}.
//   - Reason is REQUIRED.
//   - Flip status to CANCELLED.
//   - DOES NOT emit an ExpenseRecognition row.
func (uc *CancelAdvanceUseCase) Execute(
	ctx context.Context,
	req *disbursementpb.CancelAdvanceDisbursementRequest,
) (*disbursementpb.CancelAdvanceDisbursementResponse, error) {
	if req == nil {
		req = &disbursementpb.CancelAdvanceDisbursementRequest{}
	}
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityTreasuryDisbursement, entityid.ActionUpdate); err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.GetTreasuryDisbursementId()) == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"treasury_disbursement.validation.id_required",
			"treasury_disbursement_id is required [DEFAULT]",
		))
	}
	if strings.TrimSpace(req.GetReason()) == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"treasury_disbursement.validation.cancel_reason_required",
			"cancel reason is required [DEFAULT]",
		))
	}

	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		var out *disbursementpb.CancelAdvanceDisbursementResponse
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
	req *disbursementpb.CancelAdvanceDisbursementRequest,
) (*disbursementpb.CancelAdvanceDisbursementResponse, error) {
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

	if adv.GetAdvanceKind() == advancekindpb.AdvanceKind_ADVANCE_KIND_NONE {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"treasury_disbursement.errors.cancel_requires_advance",
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
			"treasury_disbursement.errors.cancel_requires_open_advance",
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

	if _, err := uc.update.Execute(ctx, &disbursementpb.UpdateDisbursementRequest{
		Data: adv,
	}); err != nil {
		return nil, err
	}

	return &disbursementpb.CancelAdvanceDisbursementResponse{NewStatus: newStatus}, nil
}
