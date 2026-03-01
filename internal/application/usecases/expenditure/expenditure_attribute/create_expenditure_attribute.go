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

const entityExpenditureAttribute = "expenditure_attribute"

// CreateExpenditureAttributeRepositories groups all repository dependencies
type CreateExpenditureAttributeRepositories struct {
	ExpenditureAttribute pb.ExpenditureAttributeDomainServiceServer
}

// CreateExpenditureAttributeServices groups all business service dependencies
type CreateExpenditureAttributeServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// CreateExpenditureAttributeUseCase handles the business logic for creating expenditure attributes
type CreateExpenditureAttributeUseCase struct {
	repositories CreateExpenditureAttributeRepositories
	services     CreateExpenditureAttributeServices
}

// NewCreateExpenditureAttributeUseCase creates use case with grouped dependencies
func NewCreateExpenditureAttributeUseCase(
	repositories CreateExpenditureAttributeRepositories,
	services CreateExpenditureAttributeServices,
) *CreateExpenditureAttributeUseCase {
	return &CreateExpenditureAttributeUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the create expenditure attribute operation
func (uc *CreateExpenditureAttributeUseCase) Execute(ctx context.Context, req *pb.CreateExpenditureAttributeRequest) (*pb.CreateExpenditureAttributeResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityExpenditureAttribute, ports.ActionCreate); err != nil {
		return nil, err
	}

	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		var result *pb.CreateExpenditureAttributeResponse
		err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
			res, err := uc.executeCore(txCtx, req)
			if err != nil {
				return fmt.Errorf("expenditure attribute creation failed: %w", err)
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

func (uc *CreateExpenditureAttributeUseCase) executeCore(ctx context.Context, req *pb.CreateExpenditureAttributeRequest) (*pb.CreateExpenditureAttributeResponse, error) {
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "expenditure_attribute.validation.data_required", "Expenditure attribute data is required [DEFAULT]"))
	}

	now := time.Now()
	if req.Data.Id == "" {
		req.Data.Id = uc.services.IDService.GenerateID()
	}
	req.Data.DateCreated = &[]int64{now.UnixMilli()}[0]
	req.Data.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	req.Data.DateModified = &[]int64{now.UnixMilli()}[0]
	req.Data.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	req.Data.Active = true

	return uc.repositories.ExpenditureAttribute.CreateExpenditureAttribute(ctx, req)
}
