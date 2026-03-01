package expenditureattribute

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expenditure_attribute"
)

// UpdateExpenditureAttributeRepositories groups all repository dependencies
type UpdateExpenditureAttributeRepositories struct {
	ExpenditureAttribute pb.ExpenditureAttributeDomainServiceServer
}

// UpdateExpenditureAttributeServices groups all business service dependencies
type UpdateExpenditureAttributeServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// UpdateExpenditureAttributeUseCase handles the business logic for updating expenditure attributes
type UpdateExpenditureAttributeUseCase struct {
	repositories UpdateExpenditureAttributeRepositories
	services     UpdateExpenditureAttributeServices
}

// NewUpdateExpenditureAttributeUseCase creates use case with grouped dependencies
func NewUpdateExpenditureAttributeUseCase(
	repositories UpdateExpenditureAttributeRepositories,
	services UpdateExpenditureAttributeServices,
) *UpdateExpenditureAttributeUseCase {
	return &UpdateExpenditureAttributeUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the update expenditure attribute operation
func (uc *UpdateExpenditureAttributeUseCase) Execute(ctx context.Context, req *pb.UpdateExpenditureAttributeRequest) (*pb.UpdateExpenditureAttributeResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityExpenditureAttribute, ports.ActionUpdate); err != nil {
		return nil, err
	}

	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		var result *pb.UpdateExpenditureAttributeResponse
		err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
			res, err := uc.executeCore(txCtx, req)
			if err != nil {
				return fmt.Errorf("expenditure attribute update failed: %w", err)
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

func (uc *UpdateExpenditureAttributeUseCase) executeCore(ctx context.Context, req *pb.UpdateExpenditureAttributeRequest) (*pb.UpdateExpenditureAttributeResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "expenditure_attribute.validation.id_required", "Expenditure attribute ID is required [DEFAULT]"))
	}

	now := time.Now()
	req.Data.DateModified = &[]int64{now.UnixMilli()}[0]
	req.Data.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return uc.repositories.ExpenditureAttribute.UpdateExpenditureAttribute(ctx, req)
}
