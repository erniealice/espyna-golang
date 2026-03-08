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

const entityDisbursement = "disbursement"

// CreateDisbursementRepositories groups all repository dependencies
type CreateDisbursementRepositories struct {
	Disbursement disbursementpb.DisbursementDomainServiceServer
}

// CreateDisbursementServices groups all business service dependencies
type CreateDisbursementServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// CreateDisbursementUseCase handles the business logic for creating disbursements
type CreateDisbursementUseCase struct {
	repositories CreateDisbursementRepositories
	services     CreateDisbursementServices
}

// NewCreateDisbursementUseCase creates use case with grouped dependencies
func NewCreateDisbursementUseCase(
	repositories CreateDisbursementRepositories,
	services CreateDisbursementServices,
) *CreateDisbursementUseCase {
	return &CreateDisbursementUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the create disbursement operation
func (uc *CreateDisbursementUseCase) Execute(ctx context.Context, req *disbursementpb.CreateDisbursementRequest) (*disbursementpb.CreateDisbursementResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityDisbursement, ports.ActionCreate); err != nil {
		return nil, err
	}

	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		var result *disbursementpb.CreateDisbursementResponse
		err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
			res, err := uc.executeCore(txCtx, req)
			if err != nil {
				return fmt.Errorf("disbursement creation failed: %w", err)
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

func (uc *CreateDisbursementUseCase) executeCore(ctx context.Context, req *disbursementpb.CreateDisbursementRequest) (*disbursementpb.CreateDisbursementResponse, error) {
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "disbursement.validation.data_required", "Disbursement data is required [DEFAULT]"))
	}

	// Enrich with ID and audit fields
	now := time.Now()
	if req.Data.Id == "" {
		req.Data.Id = uc.services.IDService.GenerateID()
	}
	req.Data.DateCreated = &[]int64{now.UnixMilli()}[0]
	req.Data.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	req.Data.DateModified = &[]int64{now.UnixMilli()}[0]
	req.Data.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	req.Data.Active = true

	return uc.repositories.Disbursement.CreateDisbursement(ctx, req)
}
