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
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
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
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	readRepos := ReadAttachmentRepositories(repositories)
	readServices := ReadAttachmentServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	updateRepos := UpdateAttachmentRepositories(repositories)
	updateServices := UpdateAttachmentServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	deleteRepos := DeleteAttachmentRepositories(repositories)
	deleteServices := DeleteAttachmentServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listRepos := ListAttachmentsRepositories(repositories)
	listServices := ListAttachmentsServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listByEntityRepos := ListAttachmentsByEntityRepositories(repositories)
	listByEntityServices := ListAttachmentsByEntityServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
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
