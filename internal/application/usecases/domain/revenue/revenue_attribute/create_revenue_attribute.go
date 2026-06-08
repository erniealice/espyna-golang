package revenueattribute

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue_attribute"
)

const entityRevenueAttribute = "revenue_attribute"

// CreateRevenueAttributeRepositories groups all repository dependencies
type CreateRevenueAttributeRepositories struct {
	RevenueAttribute pb.RevenueAttributeDomainServiceServer
}

// CreateRevenueAttributeServices groups all business service dependencies
type CreateRevenueAttributeServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	IDGenerator ports.IDGenerator
}

// CreateRevenueAttributeUseCase handles the business logic for creating revenue attributes
type CreateRevenueAttributeUseCase struct {
	repositories CreateRevenueAttributeRepositories
	services     CreateRevenueAttributeServices
}

// NewCreateRevenueAttributeUseCase creates use case with grouped dependencies
func NewCreateRevenueAttributeUseCase(
	repositories CreateRevenueAttributeRepositories,
	services CreateRevenueAttributeServices,
) *CreateRevenueAttributeUseCase {
	return &CreateRevenueAttributeUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the create revenue attribute operation
func (uc *CreateRevenueAttributeUseCase) Execute(ctx context.Context, req *pb.CreateRevenueAttributeRequest) (*pb.CreateRevenueAttributeResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityRevenueAttribute, entityid.ActionCreate); err != nil {
		return nil, err
	}

	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		var result *pb.CreateRevenueAttributeResponse
		err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
			res, err := uc.executeCore(txCtx, req)
			if err != nil {
				return fmt.Errorf("revenue attribute creation failed: %w", err)
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

func (uc *CreateRevenueAttributeUseCase) executeCore(ctx context.Context, req *pb.CreateRevenueAttributeRequest) (*pb.CreateRevenueAttributeResponse, error) {
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "revenue_attribute.validation.data_required", "Revenue attribute data is required [DEFAULT]"))
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

	return uc.repositories.RevenueAttribute.CreateRevenueAttribute(ctx, req)
}
