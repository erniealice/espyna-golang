package attribute

import (
	"context"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	attributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
)

// ListAttributesUseCase handles the business logic for listing attributes
// ListAttributesRepositories groups all repository dependencies
type ListAttributesRepositories struct {
	Attribute attributepb.AttributeDomainServiceServer // Primary entity repository
}

// ListAttributesServices groups all business service dependencies
type ListAttributesServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
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
		Transactor: ports.NewNoOpTransactor(),
		Translator:       ports.NewNoOpTranslator(),
		ActionGatekeeper: actiongate.NewActionGatekeeper(nil, ports.NewNoOpTranslator()),
	}

	return NewListAttributesUseCase(repositories, services)
}

// Execute performs the list attributes operation
func (uc *ListAttributesUseCase) Execute(ctx context.Context, req *attributepb.ListAttributesRequest) (*attributepb.ListAttributesResponse, error) {
	// Authorization check
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: "attribute",
		Action: entityid.ActionList,
	}); err != nil {
		return nil, err
	}

	// Initialize request if nil
	if req == nil {
		req = &attributepb.ListAttributesRequest{}
	}

	// Call repository with filters (repository handles filter processing)
	return uc.repositories.Attribute.ListAttributes(ctx, req)
}
