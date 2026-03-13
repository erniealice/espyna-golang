package attachment

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	attachmentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/document/attachment"
)

// DeleteAttachmentRepositories groups all repository dependencies
type DeleteAttachmentRepositories struct {
	Attachment attachmentpb.AttachmentDomainServiceServer
}

// DeleteAttachmentServices groups all business service dependencies
type DeleteAttachmentServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// DeleteAttachmentUseCase handles the business logic for deleting attachments
type DeleteAttachmentUseCase struct {
	repositories DeleteAttachmentRepositories
	services     DeleteAttachmentServices
}

// NewDeleteAttachmentUseCase creates a new DeleteAttachmentUseCase
func NewDeleteAttachmentUseCase(
	repositories DeleteAttachmentRepositories,
	services DeleteAttachmentServices,
) *DeleteAttachmentUseCase {
	return &DeleteAttachmentUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the delete attachment operation
func (uc *DeleteAttachmentUseCase) Execute(ctx context.Context, req *attachmentpb.DeleteAttachmentRequest) (*attachmentpb.DeleteAttachmentResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityAttachment, ports.ActionDelete); err != nil {
		return nil, err
	}

	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "attachment.validation.id_required", "Attachment ID is required [DEFAULT]"))
	}

	return uc.repositories.Attachment.DeleteAttachment(ctx, req)
}
