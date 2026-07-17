package attribute_value

import (
	"context"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	attributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
)

// ListAttributeValuesRepositories groups all repository dependencies
type ListAttributeValuesRepositories struct {
	AttributeValue attributepb.AttributeValueDomainServiceServer // Primary entity repository
}

// ListAttributeValuesServices groups all business service dependencies
type ListAttributeValuesServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// ListAttributeValuesUseCase handles the business logic for listing attribute values.
//
// This is the generic list path that PRESERVES the av.label column (W3-part1 MED#6):
// the drawer's <select> option labels ride this same read. The page/CTE queries that
// drop the label column are a different path and are never used here. The client
// attribute drawer resolves enum options through this use case so the display label
// (label ∥ value) is available client-side.
type ListAttributeValuesUseCase struct {
	repositories ListAttributeValuesRepositories
	services     ListAttributeValuesServices
}

// NewListAttributeValuesUseCase creates use case with grouped dependencies
func NewListAttributeValuesUseCase(
	repositories ListAttributeValuesRepositories,
	services ListAttributeValuesServices,
) *ListAttributeValuesUseCase {
	return &ListAttributeValuesUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewListAttributeValuesUseCaseUngrouped creates a new ListAttributeValuesUseCase
// Deprecated: Use NewListAttributeValuesUseCase with grouped parameters instead
func NewListAttributeValuesUseCaseUngrouped(attributeValueRepo attributepb.AttributeValueDomainServiceServer) *ListAttributeValuesUseCase {
	repositories := ListAttributeValuesRepositories{
		AttributeValue: attributeValueRepo,
	}

	services := ListAttributeValuesServices{
		Transactor:       ports.NewNoOpTransactor(),
		Translator:       ports.NewNoOpTranslator(),
		ActionGatekeeper: actiongate.NewActionGatekeeper(nil, ports.NewNoOpTranslator()),
	}

	return NewListAttributeValuesUseCase(repositories, services)
}

// Execute performs the list attribute values operation
func (uc *ListAttributeValuesUseCase) Execute(ctx context.Context, req *attributepb.ListAttributeValuesRequest) (*attributepb.ListAttributeValuesResponse, error) {
	// Authorization check. Reuse the parent "attribute" permission: attribute_value
	// rows are the option set OF an attribute — anyone who may list attribute
	// definitions may list their allowed values. This avoids introducing a new
	// permission for a read that always accompanies an attribute list.
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.Attribute,
		Action: entityid.ActionList,
	}); err != nil {
		return nil, err
	}

	// Initialize request if nil
	if req == nil {
		req = &attributepb.ListAttributeValuesRequest{}
	}

	// Call repository with filters (repository handles filter processing)
	return uc.repositories.AttributeValue.ListAttributeValues(ctx, req)
}
