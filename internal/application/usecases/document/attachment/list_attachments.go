package attachment

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	attachmentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/document/attachment"
)

// ListAttachmentsRepositories groups all repository dependencies
type ListAttachmentsRepositories struct {
	Attachment attachmentpb.AttachmentDomainServiceServer
}

// ListAttachmentsServices groups all business service dependencies
type ListAttachmentsServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ListAttachmentsUseCase handles the business logic for listing attachments
type ListAttachmentsUseCase struct {
	repositories ListAttachmentsRepositories
	services     ListAttachmentsServices
}

// NewListAttachmentsUseCase creates a new ListAttachmentsUseCase
func NewListAttachmentsUseCase(
	repositories ListAttachmentsRepositories,
	services ListAttachmentsServices,
) *ListAttachmentsUseCase {
	return &ListAttachmentsUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list attachments operation
func (uc *ListAttachmentsUseCase) Execute(ctx context.Context, req *attachmentpb.ListAttachmentsRequest) (*attachmentpb.ListAttachmentsResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityAttachment, ports.ActionList); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "attachment.validation.request_required", "Request is required [DEFAULT]"))
	}

	return uc.repositories.Attachment.ListAttachments(ctx, req)
}
