package attachment

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	attachmentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/document/attachment"
)

// ListAttachmentsRepositories groups all repository dependencies
type ListAttachmentsRepositories struct {
	Attachment attachmentpb.AttachmentDomainServiceServer
}

// ListAttachmentsServices groups all business service dependencies
type ListAttachmentsServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
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
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityAttachment, ports.ActionList); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "attachment.validation.request_required", "Request is required [DEFAULT]"))
	}

	return uc.repositories.Attachment.ListAttachments(ctx, req)
}
