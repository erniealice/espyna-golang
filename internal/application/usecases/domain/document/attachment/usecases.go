package attachment

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
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
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
}

// UseCases contains all attachment-related use cases
type UseCases struct {
	CreateAttachment         *CreateAttachmentUseCase
	ReadAttachment           *ReadAttachmentUseCase
	ReadAttachmentByEntity   *ReadAttachmentByEntityUseCase
	UpdateAttachment         *UpdateAttachmentUseCase
	DeleteAttachment         *DeleteAttachmentUseCase
	DeleteAttachmentByEntity *DeleteAttachmentByEntityUseCase
	ListAttachments          *ListAttachmentsUseCase
	ListAttachmentsByEntity  *ListAttachmentsByEntityUseCase
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
		ActionGatekeeper: services.ActionGatekeeper,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadAttachmentRepositories(repositories)
	readServices := ReadAttachmentServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	readByEntityRepos := ReadAttachmentByEntityRepositories(repositories)
	readByEntityServices := ReadAttachmentByEntityServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	updateRepos := UpdateAttachmentRepositories(repositories)
	updateServices := UpdateAttachmentServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	deleteRepos := DeleteAttachmentRepositories(repositories)
	deleteServices := DeleteAttachmentServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	deleteByEntityRepos := DeleteAttachmentByEntityRepositories(repositories)
	deleteByEntityServices := DeleteAttachmentByEntityServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	listRepos := ListAttachmentsRepositories(repositories)
	listServices := ListAttachmentsServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	listByEntityRepos := ListAttachmentsByEntityRepositories(repositories)
	listByEntityServices := ListAttachmentsByEntityServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	return &UseCases{
		CreateAttachment:         NewCreateAttachmentUseCase(createRepos, createServices),
		ReadAttachment:           NewReadAttachmentUseCase(readRepos, readServices),
		ReadAttachmentByEntity:   NewReadAttachmentByEntityUseCase(readByEntityRepos, readByEntityServices),
		UpdateAttachment:         NewUpdateAttachmentUseCase(updateRepos, updateServices),
		DeleteAttachment:         NewDeleteAttachmentUseCase(deleteRepos, deleteServices),
		DeleteAttachmentByEntity: NewDeleteAttachmentByEntityUseCase(deleteByEntityRepos, deleteByEntityServices),
		ListAttachments:          NewListAttachmentsUseCase(listRepos, listServices),
		ListAttachmentsByEntity:  NewListAttachmentsByEntityUseCase(listByEntityRepos, listByEntityServices),
	}
}
