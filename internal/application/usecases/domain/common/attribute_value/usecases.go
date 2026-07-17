package attribute_value

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	attributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
)

// AttributeValueRepositories groups all repository dependencies for attribute_value use cases
type AttributeValueRepositories struct {
	AttributeValue attributepb.AttributeValueDomainServiceServer // Primary entity repository
}

// AttributeValueServices groups all business service dependencies for attribute_value use cases
type AttributeValueServices struct {
	Transactor       ports.Transactor
	Translator       ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator      ports.IDGenerator
}

// UseCases contains all attribute_value-related use cases.
//
// Only the read (list) path is surfaced today: the client-attribute drawer needs
// the enum option rows (value + label) for select controls. Create/Update/Delete
// of attribute_value rows is not a client-facing operation and is intentionally
// omitted (attribute_value rows are seeded/administered, not edited from the drawer).
type UseCases struct {
	ListAttributeValues *ListAttributeValuesUseCase
}

// NewUseCases creates a new collection of attribute_value use cases
func NewUseCases(
	repositories AttributeValueRepositories,
	services AttributeValueServices,
) *UseCases {
	listRepos := ListAttributeValuesRepositories(repositories)
	listServices := ListAttributeValuesServices{
		Transactor:       services.Transactor,
		Translator:       services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	return &UseCases{
		ListAttributeValues: NewListAttributeValuesUseCase(listRepos, listServices),
	}
}

// NewUseCasesUngrouped creates a new collection of attribute_value use cases with individual parameters
// Deprecated: Use NewUseCases with grouped parameters instead
func NewUseCasesUngrouped(attributeValueRepo attributepb.AttributeValueDomainServiceServer) *UseCases {
	repositories := AttributeValueRepositories{
		AttributeValue: attributeValueRepo,
	}

	services := AttributeValueServices{
		Transactor:       ports.NewNoOpTransactor(),
		Translator:       ports.NewNoOpTranslator(),
		ActionGatekeeper: actiongate.NewActionGatekeeper(nil, ports.NewNoOpTranslator()),
	}

	return NewUseCases(repositories, services)
}
