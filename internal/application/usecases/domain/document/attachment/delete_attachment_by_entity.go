package attachment

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	attachmentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/document/attachment"
)

// DeleteAttachmentByEntityRepositories groups all repository dependencies
type DeleteAttachmentByEntityRepositories struct {
	Attachment attachmentpb.AttachmentDomainServiceServer
}

// DeleteAttachmentByEntityServices groups all business service dependencies
type DeleteAttachmentByEntityServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// DeleteAttachmentByEntityUseCase deletes an attachment by ID but only after
// asserting that the row belongs to the (moduleKey, foreignKey) parent entity AND
// to the caller's workspace. This closes the destruction angle of the
// same-workspace BOLA (ST-H1): a caller cannot delete an arbitrary attachment in
// their workspace by guessing IDs from a detail route they happen to hold
// attachment:delete on — the parent {id} path value is now load-bearing. On
// mismatch it returns a not-found-shaped error without leaking existence and
// without touching the row.
type DeleteAttachmentByEntityUseCase struct {
	repositories DeleteAttachmentByEntityRepositories
	services     DeleteAttachmentByEntityServices
}

// NewDeleteAttachmentByEntityUseCase creates a new DeleteAttachmentByEntityUseCase
func NewDeleteAttachmentByEntityUseCase(
	repositories DeleteAttachmentByEntityRepositories,
	services DeleteAttachmentByEntityServices,
) *DeleteAttachmentByEntityUseCase {
	return &DeleteAttachmentByEntityUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute asserts (moduleKey, foreignKey) + workspace ownership on the metadata
// row, then deletes. Any mismatch short-circuits to a not-found response without
// performing the delete.
func (uc *DeleteAttachmentByEntityUseCase) Execute(ctx context.Context, id, moduleKey, foreignKey string) (*attachmentpb.DeleteAttachmentResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityAttachment, ports.ActionDelete); err != nil {
		return nil, err
	}

	if id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "attachment.validation.id_required", "Attachment ID is required [DEFAULT]"))
	}
	if moduleKey == "" || foreignKey == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "attachment.validation.entity_required", "Module key and entity ID are required [DEFAULT]"))
	}

	// Fetch-by-id first, then assert the (moduleKey, foreignKey) + workspace match
	// BEFORE any destructive call. The read also rides the WorkspaceAwareOperations
	// decorator, which 404s cross-workspace rows.
	readResp, err := uc.repositories.Attachment.ReadAttachment(ctx, &attachmentpb.ReadAttachmentRequest{
		Data: &attachmentpb.Attachment{Id: id},
	})
	if err != nil {
		return nil, err
	}
	if !attachmentMatchesEntity(readResp, moduleKey, foreignKey) || !attachmentMatchesWorkspace(ctx, readResp) {
		// Not-found-shaped: do not leak existence, do not delete.
		return &attachmentpb.DeleteAttachmentResponse{Success: false}, nil
	}

	return uc.repositories.Attachment.DeleteAttachment(ctx, &attachmentpb.DeleteAttachmentRequest{
		Data: &attachmentpb.Attachment{Id: id},
	})
}
