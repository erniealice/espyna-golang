package attribute

import (
	"context"

	"leapfor.xyz/espyna/internal/application/ports"
	attributepb "leapfor.xyz/esqyma/golang/v1/domain/common"
)

// ListAttributesUseCase handles the business logic for listing attributes
// ListAttributesRepositories groups all repository dependencies
type ListAttributesRepositories struct {
	Attribute attributepb.AttributeDomainServiceServer // Primary entity repository
}

// ListAttributesServices groups all business service dependencies
type ListAttributesServices struct {
	TransactionService ports.TransactionService
	TranslationService ports.TranslationService
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
	// Initialize request if nil
	if req == nil {
		req = &attributepb.ListAttributesRequest{}
	}

	// Call repository with filters (repository handles filter processing)
	return uc.repositories.Attribute.ListAttributes(ctx, req)
}
