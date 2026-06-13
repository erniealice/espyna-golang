package attachment

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	attachmentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/document/attachment"
)

// ReadAttachmentRepositories groups all repository dependencies
type ReadAttachmentRepositories struct {
	Attachment attachmentpb.AttachmentDomainServiceServer
}

// ReadAttachmentServices groups all business service dependencies
type ReadAttachmentServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
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
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityAttachment,
		Action: entityid.ActionRead,
	}); err != nil {
		return nil, err
	}

	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "attachment.validation.id_required", "Attachment ID is required [DEFAULT]"))
	}

	return uc.repositories.Attachment.ReadAttachment(ctx, req)
}
