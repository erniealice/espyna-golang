package attribute

import (
	"context"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	attributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
)

// ListAttributesUseCase handles the business logic for listing attributes
// ListAttributesRepositories groups all repository dependencies
type ListAttributesRepositories struct {
	Attribute attributepb.AttributeDomainServiceServer // Primary entity repository
}

// ListAttributesServices groups all business service dependencies
type ListAttributesServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ListAttributesUseCase handles the business logic for listing attributes
type ListAttributesUseCase struct {
	repositories ListAttributesRepositories
	services     ListAttributesServices
}

// NewListAttributesUseCase creates use case with grouped dependencies
func NewListAttributesUseCase(
	repositories ListAttributesRepositories,
	services ListAttributesServices,
) *ListAttributesUseCase {
	return &ListAttributesUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewListAttributesUseCaseUngrouped creates a new ListAttributesUseCase
// Deprecated: Use NewListAttributesUseCase with grouped parameters instead
func NewListAttributesUseCaseUngrouped(attributeRepo attributepb.AttributeDomainServiceServer) *ListAttributesUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := ListAttributesRepositories{
		Attribute: attributeRepo,
	}

	services := ListAttributesServices{
		TransactionService: ports.NewNoOpTransactionService(),
		TranslationService: ports.NewNoOpTranslationService(),
	}

	return NewListAttributesUseCase(repositories, services)
}

// Execute performs the list attributes operation
func (uc *ListAttributesUseCase) Execute(ctx context.Context, req *attributepb.ListAttributesRequest) (*attributepb.ListAttributesResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		"attribute", ports.ActionList); err != nil {
		return nil, err
	}

	// Initialize request if nil
	if req == nil {
		req = &attributepb.ListAttributesRequest{}
	}

	// Call repository with filters (repository handles filter processing)
	return uc.repositories.Attribute.ListAttributes(ctx, req)
}
