package revenueattribute

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue_attribute"
)

// DeleteRevenueAttributeRepositories groups all repository dependencies
type DeleteRevenueAttributeRepositories struct {
	RevenueAttribute pb.RevenueAttributeDomainServiceServer
}

// DeleteRevenueAttributeServices groups all business service dependencies
type DeleteRevenueAttributeServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// DeleteRevenueAttributeUseCase handles the business logic for deleting revenue attributes
type DeleteRevenueAttributeUseCase struct {
	repositories DeleteRevenueAttributeRepositories
	services     DeleteRevenueAttributeServices
}

// NewDeleteRevenueAttributeUseCase creates a new DeleteRevenueAttributeUseCase
func NewDeleteRevenueAttributeUseCase(
	repositories DeleteRevenueAttributeRepositories,
	services DeleteRevenueAttributeServices,
) *DeleteRevenueAttributeUseCase {
	return &DeleteRevenueAttributeUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the delete revenue attribute operation
func (uc *DeleteRevenueAttributeUseCase) Execute(ctx context.Context, req *pb.DeleteRevenueAttributeRequest) (*pb.DeleteRevenueAttributeResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityRevenueAttribute, entityid.ActionDelete); err != nil {
		return nil, err
	}

	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "revenue_attribute.validation.id_required", "Revenue attribute ID is required [DEFAULT]"))
	}

	return uc.repositories.RevenueAttribute.DeleteRevenueAttribute(ctx, req)
}
