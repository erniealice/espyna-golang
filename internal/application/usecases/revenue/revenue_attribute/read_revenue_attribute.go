package revenueattribute

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue_attribute"
)

// ReadRevenueAttributeRepositories groups all repository dependencies
type ReadRevenueAttributeRepositories struct {
	RevenueAttribute pb.RevenueAttributeDomainServiceServer
}

// ReadRevenueAttributeServices groups all business service dependencies
type ReadRevenueAttributeServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ReadRevenueAttributeUseCase handles the business logic for reading a revenue attribute
type ReadRevenueAttributeUseCase struct {
	repositories ReadRevenueAttributeRepositories
	services     ReadRevenueAttributeServices
}

// NewReadRevenueAttributeUseCase creates use case with grouped dependencies
func NewReadRevenueAttributeUseCase(
	repositories ReadRevenueAttributeRepositories,
	services ReadRevenueAttributeServices,
) *ReadRevenueAttributeUseCase {
	return &ReadRevenueAttributeUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the read revenue attribute operation
func (uc *ReadRevenueAttributeUseCase) Execute(ctx context.Context, req *pb.ReadRevenueAttributeRequest) (*pb.ReadRevenueAttributeResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityRevenueAttribute, ports.ActionRead); err != nil {
		return nil, err
	}

	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "revenue_attribute.validation.id_required", "Revenue attribute ID is required [DEFAULT]"))
	}

	return uc.repositories.RevenueAttribute.ReadRevenueAttribute(ctx, req)
}
