package expenditure

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	expenditurepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expenditure"
)

// UpdateExpenditureRepositories groups all repository dependencies
type UpdateExpenditureRepositories struct {
	Expenditure expenditurepb.ExpenditureDomainServiceServer
}

// UpdateExpenditureServices groups all business service dependencies
type UpdateExpenditureServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// UpdateExpenditureUseCase handles the business logic for updating expenditures
type UpdateExpenditureUseCase struct {
	repositories UpdateExpenditureRepositories
	services     UpdateExpenditureServices
}

// NewUpdateExpenditureUseCase creates use case with grouped dependencies
func NewUpdateExpenditureUseCase(
	repositories UpdateExpenditureRepositories,
	services UpdateExpenditureServices,
) *UpdateExpenditureUseCase {
	return &UpdateExpenditureUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the update expenditure operation
func (uc *UpdateExpenditureUseCase) Execute(ctx context.Context, req *expenditurepb.UpdateExpenditureRequest) (*expenditurepb.UpdateExpenditureResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityExpenditure, ports.ActionUpdate); err != nil {
		return nil, err
	}

	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		var result *expenditurepb.UpdateExpenditureResponse
		err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
			res, err := uc.executeCore(txCtx, req)
			if err != nil {
				return fmt.Errorf("expenditure update failed: %w", err)
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

func (uc *UpdateExpenditureUseCase) executeCore(ctx context.Context, req *expenditurepb.UpdateExpenditureRequest) (*expenditurepb.UpdateExpenditureResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "expenditure.validation.id_required", "Expenditure ID is required [DEFAULT]"))
	}

	// Set date_modified
	now := time.Now()
	req.Data.DateModified = &[]int64{now.UnixMilli()}[0]
	req.Data.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return uc.repositories.Expenditure.UpdateExpenditure(ctx, req)
}
