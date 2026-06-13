package disbursement

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	disbursementpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/disbursement"
)

// UpdateDisbursementRepositories groups all repository dependencies
type UpdateDisbursementRepositories struct {
	Disbursement disbursementpb.DisbursementDomainServiceServer
}

// UpdateDisbursementServices groups all business service dependencies
type UpdateDisbursementServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// UpdateDisbursementUseCase handles the business logic for updating disbursements
type UpdateDisbursementUseCase struct {
	repositories UpdateDisbursementRepositories
	services     UpdateDisbursementServices
}

// NewUpdateDisbursementUseCase creates use case with grouped dependencies
func NewUpdateDisbursementUseCase(
	repositories UpdateDisbursementRepositories,
	services UpdateDisbursementServices,
) *UpdateDisbursementUseCase {
	return &UpdateDisbursementUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the update disbursement operation.
//
// 20260518-hexagonal-strict-adherence Phase 1.C-iv-pre — BURN_DOWN guard +
// transaction-aware behavior. See the matching collection.UpdateCollection
// implementation for the rationale.
func (uc *UpdateDisbursementUseCase) Execute(ctx context.Context, req *disbursementpb.UpdateDisbursementRequest) (*disbursementpb.UpdateDisbursementResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityDisbursement,
		Action: entityid.ActionUpdate,
	}); err != nil {
		return nil, err
	}

	// Plan B Phase 0 hard rule: ADVANCE_KIND_BURN_DOWN is reserved for v2.
	if req != nil && req.Data != nil {
		if err := validateAdvanceKindNotBurnDown(req.Data.GetAdvanceKind()); err != nil {
			return nil, err
		}
	}

	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		if uc.services.Transactor.IsTransactionActive(ctx) {
			return uc.executeCore(ctx, req)
		}
		var result *disbursementpb.UpdateDisbursementResponse
		err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
			res, err := uc.executeCore(txCtx, req)
			if err != nil {
				return fmt.Errorf("disbursement update failed: %w", err)
			}
			result = res
			return nil
		})
		if err != nil {
			return nil, err
		}
		return result, nil
	}

	return uc.executeCore(ctx, req)
}

func (uc *UpdateDisbursementUseCase) executeCore(ctx context.Context, req *disbursementpb.UpdateDisbursementRequest) (*disbursementpb.UpdateDisbursementResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "disbursement.validation.id_required", "Disbursement ID is required [DEFAULT]"))
	}

	// Set date_modified
	now := time.Now()
	req.Data.DateModified = &[]int64{now.UnixMilli()}[0]
	req.Data.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return uc.repositories.Disbursement.UpdateDisbursement(ctx, req)
}
