package attachment

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	attachmentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/document/attachment"
)

// ReadAttachmentRepositories groups all repository dependencies
type ReadAttachmentRepositories struct {
	Attachment attachmentpb.AttachmentDomainServiceServer
}

// ReadAttachmentServices groups all business service dependencies
type ReadAttachmentServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ReadAttachmentUseCase handles the business logic for reading an attachment
type ReadAttachmentUseCase struct {
	repositories ReadAttachmentRepositories
	services     ReadAttachmentServices
}

// NewReadAttachmentUseCase creates use case with grouped dependencies
func NewReadAttachmentUseCase(
	repositories ReadAttachmentRepositories,
	services ReadAttachmentServices,
) *ReadAttachmentUseCase {
	return &ReadAttachmentUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the read attachment operation
func (uc *ReadAttachmentUseCase) Execute(ctx context.Context, req *attachmentpb.ReadAttachmentRequest) (*attachmentpb.ReadAttachmentResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityAttachment, ports.ActionRead); err != nil {
		return nil, err
	}

	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "attachment.validation.id_required", "Attachment ID is required [DEFAULT]"))
	}

	return uc.repositories.Attachment.ReadAttachment(ctx, req)
}
