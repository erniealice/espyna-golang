package attachment

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	attachmentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/document/attachment"
)

// ListAttachmentsByEntityRepositories groups all repository dependencies
type ListAttachmentsByEntityRepositories struct {
	Attachment attachmentpb.AttachmentDomainServiceServer
}

// ListAttachmentsByEntityServices groups all business service dependencies
type ListAttachmentsByEntityServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// ListAttachmentsByEntityUseCase handles listing attachments filtered by module_key + foreign_key
type ListAttachmentsByEntityUseCase struct {
	repositories ListAttachmentsByEntityRepositories
	services     ListAttachmentsByEntityServices
}

// NewListAttachmentsByEntityUseCase creates a new ListAttachmentsByEntityUseCase
func NewListAttachmentsByEntityUseCase(
	repositories ListAttachmentsByEntityRepositories,
	services ListAttachmentsByEntityServices,
) *ListAttachmentsByEntityUseCase {
	return &ListAttachmentsByEntityUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute lists attachments belonging to a specific entity (identified by moduleKey + entityID)
func (uc *ListAttachmentsByEntityUseCase) Execute(ctx context.Context, moduleKey, entityID string) (*attachmentpb.ListAttachmentsResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityAttachment, ports.ActionList); err != nil {
		return nil, err
	}

	if moduleKey == "" || entityID == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "attachment.validation.entity_required", "Module key and entity ID are required [DEFAULT]"))
	}

	return uc.repositories.Attachment.ListAttachments(ctx, &attachmentpb.ListAttachmentsRequest{
		Filters: &commonpb.FilterRequest{
			Logic: commonpb.FilterLogic_AND,
			Filters: []*commonpb.TypedFilter{
				{
					Field: "module_key",
					FilterType: &commonpb.TypedFilter_StringFilter{
						StringFilter: &commonpb.StringFilter{
							Operator: commonpb.StringOperator_STRING_EQUALS,
							Value:    moduleKey,
						},
					},
				},
				{
					Field: "foreign_key",
					FilterType: &commonpb.TypedFilter_StringFilter{
						StringFilter: &commonpb.StringFilter{
							Operator: commonpb.StringOperator_STRING_EQUALS,
							Value:    entityID,
						},
					},
				},
			},
		},
	})
}
