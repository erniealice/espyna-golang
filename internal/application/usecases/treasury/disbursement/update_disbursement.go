package disbursement

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	disbursementpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/disbursement"
)

// UpdateDisbursementRepositories groups all repository dependencies
type UpdateDisbursementRepositories struct {
	Disbursement disbursementpb.DisbursementDomainServiceServer
}

// UpdateDisbursementServices groups all business service dependencies
type UpdateDisbursementServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
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

// Execute performs the update disbursement operation
func (uc *UpdateDisbursementUseCase) Execute(ctx context.Context, req *disbursementpb.UpdateDisbursementRequest) (*disbursementpb.UpdateDisbursementResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityDisbursement, ports.ActionUpdate); err != nil {
		return nil, err
	}

	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		var result *disbursementpb.UpdateDisbursementResponse
		err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
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
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "disbursement.validation.id_required", "Disbursement ID is required [DEFAULT]"))
	}

	// Set date_modified
	now := time.Now()
	req.Data.DateModified = &[]int64{now.UnixMilli()}[0]
	req.Data.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return uc.repositories.Disbursement.UpdateDisbursement(ctx, req)
}
