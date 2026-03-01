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

const entityExpenditure = "expenditure"

// CreateExpenditureRepositories groups all repository dependencies
type CreateExpenditureRepositories struct {
	Expenditure expenditurepb.ExpenditureDomainServiceServer
}

// CreateExpenditureServices groups all business service dependencies
type CreateExpenditureServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// CreateExpenditureUseCase handles the business logic for creating expenditures
type CreateExpenditureUseCase struct {
	repositories CreateExpenditureRepositories
	services     CreateExpenditureServices
}

// NewCreateExpenditureUseCase creates use case with grouped dependencies
func NewCreateExpenditureUseCase(
	repositories CreateExpenditureRepositories,
	services CreateExpenditureServices,
) *CreateExpenditureUseCase {
	return &CreateExpenditureUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the create expenditure operation
func (uc *CreateExpenditureUseCase) Execute(ctx context.Context, req *expenditurepb.CreateExpenditureRequest) (*expenditurepb.CreateExpenditureResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityExpenditure, ports.ActionCreate); err != nil {
		return nil, err
	}

	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		var result *expenditurepb.CreateExpenditureResponse
		err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
			res, err := uc.executeCore(txCtx, req)
			if err != nil {
				return fmt.Errorf("expenditure creation failed: %w", err)
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

func (uc *CreateExpenditureUseCase) executeCore(ctx context.Context, req *expenditurepb.CreateExpenditureRequest) (*expenditurepb.CreateExpenditureResponse, error) {
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "expenditure.validation.data_required", "Expenditure data is required [DEFAULT]"))
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

	return uc.repositories.Expenditure.CreateExpenditure(ctx, req)
}
