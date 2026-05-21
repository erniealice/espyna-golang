package template

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	documenttemplatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/document/template"
)

// DocumentTemplateRepositories groups all repository dependencies for document template use cases
type DocumentTemplateRepositories struct {
	DocumentTemplate documenttemplatepb.DocumentTemplateDomainServiceServer
}

// DocumentTemplateServices groups all business service dependencies for document template use cases
type DocumentTemplateServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	IDGenerator ports.IDGenerator
}

// UseCases contains all document template-related use cases
type UseCases struct {
	CreateDocumentTemplate        *CreateDocumentTemplateUseCase
	ReadDocumentTemplate          *ReadDocumentTemplateUseCase
	UpdateDocumentTemplate        *UpdateDocumentTemplateUseCase
	DeleteDocumentTemplate        *DeleteDocumentTemplateUseCase
	ListDocumentTemplates         *ListDocumentTemplatesUseCase
	ListDocumentTemplatesByModule *ListDocumentTemplatesByModuleUseCase
}

// NewUseCases creates a new collection of document template use cases
func NewUseCases(
	repositories DocumentTemplateRepositories,
	services DocumentTemplateServices,
) *UseCases {
	createRepos := CreateDocumentTemplateRepositories(repositories)
	createServices := CreateDocumentTemplateServices{
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadDocumentTemplateRepositories(repositories)
	readServices := ReadDocumentTemplateServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	updateRepos := UpdateDocumentTemplateRepositories(repositories)
	updateServices := UpdateDocumentTemplateServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	deleteRepos := DeleteDocumentTemplateRepositories(repositories)
	deleteServices := DeleteDocumentTemplateServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listRepos := ListDocumentTemplatesRepositories(repositories)
	listServices := ListDocumentTemplatesServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listByModuleRepos := ListDocumentTemplatesByModuleRepositories(repositories)
	listByModuleServices := ListDocumentTemplatesByModuleServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	return &UseCases{
		CreateDocumentTemplate:        NewCreateDocumentTemplateUseCase(createRepos, createServices),
		ReadDocumentTemplate:          NewReadDocumentTemplateUseCase(readRepos, readServices),
		UpdateDocumentTemplate:        NewUpdateDocumentTemplateUseCase(updateRepos, updateServices),
		DeleteDocumentTemplate:        NewDeleteDocumentTemplateUseCase(deleteRepos, deleteServices),
		ListDocumentTemplates:         NewListDocumentTemplatesUseCase(listRepos, listServices),
		ListDocumentTemplatesByModule: NewListDocumentTemplatesByModuleUseCase(listByModuleRepos, listByModuleServices),
	}
}
