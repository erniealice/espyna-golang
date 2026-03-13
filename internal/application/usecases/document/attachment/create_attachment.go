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

const entityAttachment = "attachment"

// CreateAttachmentRepositories groups all repository dependencies
type CreateAttachmentRepositories struct {
	Attachment attachmentpb.AttachmentDomainServiceServer
}

// CreateAttachmentServices groups all business service dependencies
type CreateAttachmentServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// CreateAttachmentUseCase handles the business logic for creating attachments
type CreateAttachmentUseCase struct {
	repositories CreateAttachmentRepositories
	services     CreateAttachmentServices
}

// NewCreateAttachmentUseCase creates use case with grouped dependencies
func NewCreateAttachmentUseCase(
	repositories CreateAttachmentRepositories,
	services CreateAttachmentServices,
) *CreateAttachmentUseCase {
	return &CreateAttachmentUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the create attachment operation
func (uc *CreateAttachmentUseCase) Execute(ctx context.Context, req *attachmentpb.CreateAttachmentRequest) (*attachmentpb.CreateAttachmentResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityAttachment, ports.ActionCreate); err != nil {
		return nil, err
	}

	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		var result *attachmentpb.CreateAttachmentResponse
		err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
			res, err := uc.executeCore(txCtx, req)
			if err != nil {
				return fmt.Errorf("attachment creation failed: %w", err)
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

func (uc *CreateAttachmentUseCase) executeCore(ctx context.Context, req *attachmentpb.CreateAttachmentRequest) (*attachmentpb.CreateAttachmentResponse, error) {
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "attachment.validation.data_required", "Attachment data is required [DEFAULT]"))
	}

	// Enrich with ID and audit fields
	now := time.Now()
	if req.Data.Id == "" {
		req.Data.Id = uc.services.IDService.GenerateID()
	}
	req.Data.DateCreated = &[]int64{now.UnixMilli()}[0]
	req.Data.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	req.Data.DateModified = &[]int64{now.UnixMilli()}[0]
	req.Data.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	req.Data.Active = true

	return uc.repositories.Attachment.CreateAttachment(ctx, req)
}
