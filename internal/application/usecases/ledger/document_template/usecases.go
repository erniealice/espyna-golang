package document_template

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	documenttemplatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/document_template"
)

// DocumentTemplateRepositories groups all repository dependencies for document template use cases
type DocumentTemplateRepositories struct {
	DocumentTemplate documenttemplatepb.DocumentTemplateDomainServiceServer
}

// DocumentTemplateServices groups all business service dependencies for document template use cases
type DocumentTemplateServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// UseCases contains all document template-related use cases
type UseCases struct {
	CreateDocumentTemplate *CreateDocumentTemplateUseCase
	ReadDocumentTemplate   *ReadDocumentTemplateUseCase
	UpdateDocumentTemplate *UpdateDocumentTemplateUseCase
	DeleteDocumentTemplate *DeleteDocumentTemplateUseCase
	ListDocumentTemplates  *ListDocumentTemplatesUseCase
}

// NewUseCases creates a new collection of document template use cases
func NewUseCases(
	repositories DocumentTemplateRepositories,
	services DocumentTemplateServices,
) *UseCases {
	createRepos := CreateDocumentTemplateRepositories(repositories)
	createServices := CreateDocumentTemplateServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	readRepos := ReadDocumentTemplateRepositories(repositories)
	readServices := ReadDocumentTemplateServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	updateRepos := UpdateDocumentTemplateRepositories(repositories)
	updateServices := UpdateDocumentTemplateServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	deleteRepos := DeleteDocumentTemplateRepositories(repositories)
	deleteServices := DeleteDocumentTemplateServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listRepos := ListDocumentTemplatesRepositories(repositories)
	listServices := ListDocumentTemplatesServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	return &UseCases{
		CreateDocumentTemplate: NewCreateDocumentTemplateUseCase(createRepos, createServices),
		ReadDocumentTemplate:   NewReadDocumentTemplateUseCase(readRepos, readServices),
		UpdateDocumentTemplate: NewUpdateDocumentTemplateUseCase(updateRepos, updateServices),
		DeleteDocumentTemplate: NewDeleteDocumentTemplateUseCase(deleteRepos, deleteServices),
		ListDocumentTemplates:  NewListDocumentTemplatesUseCase(listRepos, listServices),
	}
}
