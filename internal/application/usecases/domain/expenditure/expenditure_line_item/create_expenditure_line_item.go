package expenditurelineitem

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expenditure_line_item"
)

const entityExpenditureLineItem = "expenditure_line_item"

// CreateExpenditureLineItemRepositories groups all repository dependencies
type CreateExpenditureLineItemRepositories struct {
	ExpenditureLineItem pb.ExpenditureLineItemDomainServiceServer
}

// CreateExpenditureLineItemServices groups all business service dependencies
type CreateExpenditureLineItemServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
}

// CreateExpenditureLineItemUseCase handles the business logic for creating expenditure line items
type CreateExpenditureLineItemUseCase struct {
	repositories CreateExpenditureLineItemRepositories
	services     CreateExpenditureLineItemServices
}

// NewCreateExpenditureLineItemUseCase creates use case with grouped dependencies
func NewCreateExpenditureLineItemUseCase(
	repositories CreateExpenditureLineItemRepositories,
	services CreateExpenditureLineItemServices,
) *CreateExpenditureLineItemUseCase {
	return &CreateExpenditureLineItemUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the create expenditure line item operation
func (uc *CreateExpenditureLineItemUseCase) Execute(ctx context.Context, req *pb.CreateExpenditureLineItemRequest) (*pb.CreateExpenditureLineItemResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityExpenditureLineItem,
		Action: entityid.ActionCreate,
	}); err != nil {
		return nil, err
	}

	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		var result *pb.CreateExpenditureLineItemResponse
		err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
			res, err := uc.executeCore(txCtx, req)
			if err != nil {
				return fmt.Errorf("expenditure line item creation failed: %w", err)
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

func (uc *CreateExpenditureLineItemUseCase) executeCore(ctx context.Context, req *pb.CreateExpenditureLineItemRequest) (*pb.CreateExpenditureLineItemResponse, error) {
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "expenditure_line_item.validation.data_required", "Expenditure line item data is required [DEFAULT]"))
	}

	now := time.Now()
	if req.Data.Id == "" {
		req.Data.Id = uc.services.IDGenerator.GenerateID()
	}
	req.Data.DateCreated = &[]int64{now.UnixMilli()}[0]
	req.Data.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	req.Data.DateModified = &[]int64{now.UnixMilli()}[0]
	req.Data.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	req.Data.Active = true

	return uc.repositories.ExpenditureLineItem.CreateExpenditureLineItem(ctx, req)
}
