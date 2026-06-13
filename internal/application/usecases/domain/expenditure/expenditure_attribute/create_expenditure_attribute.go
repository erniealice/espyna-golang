package expenditureattribute

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expenditure_attribute"
)

const entityExpenditureAttribute = "expenditure_attribute"

// CreateExpenditureAttributeRepositories groups all repository dependencies
type CreateExpenditureAttributeRepositories struct {
	ExpenditureAttribute pb.ExpenditureAttributeDomainServiceServer
}

// CreateExpenditureAttributeServices groups all business service dependencies
type CreateExpenditureAttributeServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
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
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityExpenditureAttribute,
		Action: entityid.ActionCreate,
	}); err != nil {
		return nil, err
	}

	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		var result *pb.CreateExpenditureAttributeResponse
		err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
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
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "expenditure_attribute.validation.data_required", "Expenditure attribute data is required [DEFAULT]"))
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

	return uc.repositories.ExpenditureAttribute.CreateExpenditureAttribute(ctx, req)
}
