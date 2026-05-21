package attribute

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	attributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
)

// AttributeRepositories groups all repository dependencies for attribute use cases
type AttributeRepositories struct {
	Attribute attributepb.AttributeDomainServiceServer // Primary entity repository
}

// AttributeServices groups all business service dependencies for attribute use cases
type AttributeServices struct {
	TransactionService ports.TransactionService
	TranslationService ports.TranslationService
	IDService          ports.IDService
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
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
		IDService:          services.IDService,
	}

	readRepos := ReadAttributeRepositories(repositories)
	readServices := ReadAttributeServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
	}

	updateRepos := UpdateAttributeRepositories(repositories)
	updateServices := UpdateAttributeServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
	}

	deleteRepos := DeleteAttributeRepositories(repositories)
	deleteServices := DeleteAttributeServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
	}

	listRepos := ListAttributesRepositories(repositories)
	listServices := ListAttributesServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
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
		TransactionService: ports.NewNoOpTransactionService(),
		TranslationService: ports.NewNoOpTranslationService(),
	}

	return NewUseCases(repositories, services)
}
