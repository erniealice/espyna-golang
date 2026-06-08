package attachment

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	attachmentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/document/attachment"
)

// ReadAttachmentByEntityRepositories groups all repository dependencies
type ReadAttachmentByEntityRepositories struct {
	Attachment attachmentpb.AttachmentDomainServiceServer
}

// ReadAttachmentByEntityServices groups all business service dependencies
type ReadAttachmentByEntityServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// ReadAttachmentByEntityUseCase reads a single attachment by ID but only after
// asserting that the row belongs to the (moduleKey, foreignKey) parent entity
// AND to the caller's workspace. This closes the same-workspace BOLA (ST-H1) and
// the cross-workspace IDOR backstop (ST-H4) at the use-case layer: a caller who
// guesses an attachment ID owned by a different entity (e.g. a different
// disbursement, collection, price_plan) or a different workspace receives a
// not-found error that does NOT leak existence — mirroring how
// list_attachments_by_entity scopes by module_key + foreign_key.
type ReadAttachmentByEntityUseCase struct {
	repositories ReadAttachmentByEntityRepositories
	services     ReadAttachmentByEntityServices
}

// NewReadAttachmentByEntityUseCase creates use case with grouped dependencies
func NewReadAttachmentByEntityUseCase(
	repositories ReadAttachmentByEntityRepositories,
	services ReadAttachmentByEntityServices,
) *ReadAttachmentByEntityUseCase {
	return &ReadAttachmentByEntityUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute reads an attachment by id and asserts (moduleKey, foreignKey) +
// workspace ownership before returning it. A mismatch on any axis returns a
// not-found-shaped error (Success=false, nil Data) so existence is not leaked.
func (uc *ReadAttachmentByEntityUseCase) Execute(ctx context.Context, id, moduleKey, foreignKey string) (*attachmentpb.ReadAttachmentResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityAttachment, entityid.ActionRead); err != nil {
		return nil, err
	}

	if id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "attachment.validation.id_required", "Attachment ID is required [DEFAULT]"))
	}
	if moduleKey == "" || foreignKey == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "attachment.validation.entity_required", "Module key and entity ID are required [DEFAULT]"))
	}

	resp, err := uc.repositories.Attachment.ReadAttachment(ctx, &attachmentpb.ReadAttachmentRequest{
		Data: &attachmentpb.Attachment{Id: id},
	})
	if err != nil {
		return nil, err
	}

	// Entity-scope + workspace assertion on the metadata row BEFORE any byte
	// stream is opened by the caller. Treat any mismatch as not-found.
	if !attachmentMatchesEntity(resp, moduleKey, foreignKey) || !attachmentMatchesWorkspace(ctx, resp) {
		return notFoundResponse(), nil
	}

	return resp, nil
}

// attachmentMatchesEntity returns true only when the response carries exactly one
// row whose module_key + foreign_key equal the requested parent entity.
func attachmentMatchesEntity(resp *attachmentpb.ReadAttachmentResponse, moduleKey, foreignKey string) bool {
	if resp == nil || len(resp.GetData()) == 0 {
		return false
	}
	att := resp.GetData()[0]
	if att == nil {
		return false
	}
	return att.GetModuleKey() == moduleKey && att.GetForeignKey() == foreignKey
}

// attachmentMatchesWorkspace asserts the row's workspace_id matches the caller's
// workspace context. A non-workspaced context (empty ctx workspace) is treated as
// a no-op pass-through here: the postgres WorkspaceAwareOperations decorator owns
// the cross-workspace 404 in that path; this assertion is the use-case backstop
// for when a workspace IS present in context.
func attachmentMatchesWorkspace(ctx context.Context, resp *attachmentpb.ReadAttachmentResponse) bool {
	ctxWS := contextutil.ExtractWorkspaceIDFromContext(ctx)
	if ctxWS == "" {
		return true
	}
	if resp == nil || len(resp.GetData()) == 0 {
		return false
	}
	rowWS := resp.GetData()[0].GetWorkspaceId()
	return rowWS == ctxWS
}

// notFoundResponse returns a successful-but-empty response shape so the caller's
// len(Data)==0 not-found branch fires without leaking whether the ID exists.
func notFoundResponse() *attachmentpb.ReadAttachmentResponse {
	return &attachmentpb.ReadAttachmentResponse{
		Success: true,
		Data:    nil,
	}
}
