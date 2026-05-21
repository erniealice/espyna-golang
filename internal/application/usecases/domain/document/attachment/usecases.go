package attachment

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	attachmentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/document/attachment"
)

// AttachmentRepositories groups all repository dependencies for attachment use cases
type AttachmentRepositories struct {
	Attachment attachmentpb.AttachmentDomainServiceServer
}

// AttachmentServices groups all business service dependencies for attachment use cases
type AttachmentServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	IDGenerator ports.IDGenerator
}

// UseCases contains all attachment-related use cases
type UseCases struct {
	CreateAttachment        *CreateAttachmentUseCase
	ReadAttachment          *ReadAttachmentUseCase
	UpdateAttachment        *UpdateAttachmentUseCase
	DeleteAttachment        *DeleteAttachmentUseCase
	ListAttachments         *ListAttachmentsUseCase
	ListAttachmentsByEntity *ListAttachmentsByEntityUseCase
}

// NewUseCases creates a new collection of attachment use cases
func NewUseCases(
	repositories AttachmentRepositories,
	services AttachmentServices,
) *UseCases {
	createRepos := CreateAttachmentRepositories(repositories)
	createServices := CreateAttachmentServices{
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadAttachmentRepositories(repositories)
	readServices := ReadAttachmentServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	updateRepos := UpdateAttachmentRepositories(repositories)
	updateServices := UpdateAttachmentServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	deleteRepos := DeleteAttachmentRepositories(repositories)
	deleteServices := DeleteAttachmentServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listRepos := ListAttachmentsRepositories(repositories)
	listServices := ListAttachmentsServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listByEntityRepos := ListAttachmentsByEntityRepositories(repositories)
	listByEntityServices := ListAttachmentsByEntityServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	return &UseCases{
		CreateAttachment:        NewCreateAttachmentUseCase(createRepos, createServices),
		ReadAttachment:          NewReadAttachmentUseCase(readRepos, readServices),
		UpdateAttachment:        NewUpdateAttachmentUseCase(updateRepos, updateServices),
		DeleteAttachment:        NewDeleteAttachmentUseCase(deleteRepos, deleteServices),
		ListAttachments:         NewListAttachmentsUseCase(listRepos, listServices),
		ListAttachmentsByEntity: NewListAttachmentsByEntityUseCase(listByEntityRepos, listByEntityServices),
	}
}
