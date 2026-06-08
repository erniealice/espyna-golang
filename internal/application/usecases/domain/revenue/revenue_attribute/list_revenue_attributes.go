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

// ListRevenueAttributesRepositories groups all repository dependencies
type ListRevenueAttributesRepositories struct {
	RevenueAttribute pb.RevenueAttributeDomainServiceServer
}

// ListRevenueAttributesServices groups all business service dependencies
type ListRevenueAttributesServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// ListRevenueAttributesUseCase handles the business logic for listing revenue attributes
type ListRevenueAttributesUseCase struct {
	repositories ListRevenueAttributesRepositories
	services     ListRevenueAttributesServices
}

// NewListRevenueAttributesUseCase creates a new ListRevenueAttributesUseCase
func NewListRevenueAttributesUseCase(
	repositories ListRevenueAttributesRepositories,
	services ListRevenueAttributesServices,
) *ListRevenueAttributesUseCase {
	return &ListRevenueAttributesUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list revenue attributes operation
func (uc *ListRevenueAttributesUseCase) Execute(ctx context.Context, req *pb.ListRevenueAttributesRequest) (*pb.ListRevenueAttributesResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityRevenueAttribute, entityid.ActionList); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "revenue_attribute.validation.request_required", "Request is required [DEFAULT]"))
	}

	return uc.repositories.RevenueAttribute.ListRevenueAttributes(ctx, req)
}
