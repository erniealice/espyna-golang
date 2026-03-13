package attachment

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	attachmentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/document/attachment"
)

// UpdateAttachmentRepositories groups all repository dependencies
type UpdateAttachmentRepositories struct {
	Attachment attachmentpb.AttachmentDomainServiceServer
}

// UpdateAttachmentServices groups all business service dependencies
type UpdateAttachmentServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// UpdateAttachmentUseCase handles the business logic for updating attachments
type UpdateAttachmentUseCase struct {
	repositories UpdateAttachmentRepositories
	services     UpdateAttachmentServices
}

// NewUpdateAttachmentUseCase creates use case with grouped dependencies
func NewUpdateAttachmentUseCase(
	repositories UpdateAttachmentRepositories,
	services UpdateAttachmentServices,
) *UpdateAttachmentUseCase {
	return &UpdateAttachmentUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the update attachment operation
func (uc *UpdateAttachmentUseCase) Execute(ctx context.Context, req *attachmentpb.UpdateAttachmentRequest) (*attachmentpb.UpdateAttachmentResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityAttachment, ports.ActionUpdate); err != nil {
		return nil, err
	}

	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		var result *attachmentpb.UpdateAttachmentResponse
		err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
			res, err := uc.executeCore(txCtx, req)
			if err != nil {
				return fmt.Errorf("attachment update failed: %w", err)
			}
			result = res
			return nil
		})
		if err != nil {
			return nil, err
		}
		return result, nil
	}

	return uc.executeCore(ctx, req)
}

func (uc *UpdateAttachmentUseCase) executeCore(ctx context.Context, req *attachmentpb.UpdateAttachmentRequest) (*attachmentpb.UpdateAttachmentResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "attachment.validation.id_required", "Attachment ID is required [DEFAULT]"))
	}

	// Set date_modified
	now := time.Now()
	req.Data.DateModified = &[]int64{now.UnixMilli()}[0]
	req.Data.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return uc.repositories.Attachment.UpdateAttachment(ctx, req)
}
