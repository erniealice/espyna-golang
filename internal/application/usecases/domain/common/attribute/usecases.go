package attribute

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	attributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
)

// AttributeRepositories groups all repository dependencies for attribute use cases
type AttributeRepositories struct {
	Attribute attributepb.AttributeDomainServiceServer // Primary entity repository
}

// AttributeServices groups all business service dependencies for attribute use cases
type AttributeServices struct {
	Transactor  ports.Transactor
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
}

// UseCases contains all attribute-related use cases
type UseCases struct {
	CreateAttribute *CreateAttributeUseCase
	ReadAttribute   *ReadAttributeUseCase
	UpdateAttribute *UpdateAttributeUseCase
	DeleteAttribute *DeleteAttributeUseCase
	ListAttributes  *ListAttributesUseCase
}

// NewUseCases creates a new collection of attribute use cases
func NewUseCases(
	repositories AttributeRepositories,
	services AttributeServices,
) *UseCases {
	// Build individual grouped parameters for each use case
	createRepos := CreateAttributeRepositories(repositories)
	createServices := CreateAttributeServices{
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadAttributeRepositories(repositories)
	readServices := ReadAttributeServices{
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	updateRepos := UpdateAttributeRepositories(repositories)
	updateServices := UpdateAttributeServices{
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	deleteRepos := DeleteAttributeRepositories(repositories)
	deleteServices := DeleteAttributeServices{
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	listRepos := ListAttributesRepositories(repositories)
	listServices := ListAttributesServices{
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	return &UseCases{
		CreateAttribute: NewCreateAttributeUseCase(createRepos, createServices),
		ReadAttribute:   NewReadAttributeUseCase(readRepos, readServices),
		UpdateAttribute: NewUpdateAttributeUseCase(updateRepos, updateServices),
		DeleteAttribute: NewDeleteAttributeUseCase(deleteRepos, deleteServices),
		ListAttributes:  NewListAttributesUseCase(listRepos, listServices),
	}
}

// NewUseCasesUngrouped creates a new collection of attribute use cases with individual parameters
// Deprecated: Use NewUseCases with grouped parameters instead
func NewUseCasesUngrouped(attributeRepo attributepb.AttributeDomainServiceServer) *UseCases {
	// Build grouped parameters internally for backward compatibility
	repositories := AttributeRepositories{
		Attribute: attributeRepo,
	}

	services := AttributeServices{
		Transactor: ports.NewNoOpTransactor(),
		Translator:       ports.NewNoOpTranslator(),
		ActionGatekeeper: actiongate.NewActionGatekeeper(nil, ports.NewNoOpTranslator()),
	}

	return NewUseCases(repositories, services)
}
