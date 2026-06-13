package template

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
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
	ActionGatekeeper *actiongate.ActionGatekeeper
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
		ActionGatekeeper: services.ActionGatekeeper,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadDocumentTemplateRepositories(repositories)
	readServices := ReadDocumentTemplateServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	updateRepos := UpdateDocumentTemplateRepositories(repositories)
	updateServices := UpdateDocumentTemplateServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	deleteRepos := DeleteDocumentTemplateRepositories(repositories)
	deleteServices := DeleteDocumentTemplateServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	listRepos := ListDocumentTemplatesRepositories(repositories)
	listServices := ListDocumentTemplatesServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	listByModuleRepos := ListDocumentTemplatesByModuleRepositories(repositories)
	listByModuleServices := ListDocumentTemplatesByModuleServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
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
