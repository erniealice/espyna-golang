package attachment

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	attachmentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/document/attachment"
)

const entityAttachment = "attachment"

// CreateAttachmentRepositories groups all repository dependencies
type CreateAttachmentRepositories struct {
	Attachment attachmentpb.AttachmentDomainServiceServer
}

// CreateAttachmentServices groups all business service dependencies
type CreateAttachmentServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
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
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityAttachment,
		Action: entityid.ActionCreate,
	}); err != nil {
		return nil, err
	}

	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		var result *attachmentpb.CreateAttachmentResponse
		err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
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
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "attachment.validation.data_required", "Attachment data is required [DEFAULT]"))
	}

	// W4 (Q-ST-POLICY B->C): SERVER-AUTHORITATIVE upload policy. This runs for EVERY
	// caller of CreateAttachment, not just the hybra HTTP upload handler — closing
	// the gap that the file-type/count policy previously lived only in that one
	// handler. The policy is resolved by module_key from the espyna-side default-deny
	// registry (policy.go); an unregistered module_key yields a zero Policy that
	// rejects every content type.
	if err := uc.assertUploadPolicy(ctx, req.Data); err != nil {
		return nil, err
	}

	// Stamp WorkspaceId from context (pairs with the workspace-prefix object keys and
	// the additive workspace_id NOT NULL migration, Q-ST-WSSCOPE). A request that
	// already carries a workspace_id is left untouched (e.g. service-to-service);
	// otherwise the caller's workspace context is the source of truth.
	if req.Data.GetWorkspaceId() == "" {
		if ws := contextutil.ExtractWorkspaceIDFromContext(ctx); ws != "" {
			req.Data.WorkspaceId = &ws
		}
	}

	// Enrich with ID and audit fields
	now := time.Now()
	if req.Data.Id == "" {
		req.Data.Id = uc.services.IDGenerator.GenerateID()
	}
	req.Data.DateCreated = &[]int64{now.UnixMilli()}[0]
	req.Data.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	req.Data.DateModified = &[]int64{now.UnixMilli()}[0]
	req.Data.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	req.Data.Active = true

	return uc.repositories.Attachment.CreateAttachment(ctx, req)
}

// assertUploadPolicy enforces the server-side upload policy before insert:
//
//	(a) content-type allow-list resolved by module_key (DEFAULT-DENY: an
//	    unregistered module_key rejects everything; the persisted content_type — which
//	    the hybra handler derives from a magic-byte sniff, never the raw client
//	    header — must be on the module's allow-list);
//	(b) per-record MaxFileCount: count existing ACTIVE attachments for
//	    (module_key, foreign_key) and reject when the count has reached the cap.
//
// Both checks fail with a translated error. This is the authoritative backstop; the
// hybra upload handler may additionally short-circuit earlier for nicer UX.
func (uc *CreateAttachmentUseCase) assertUploadPolicy(ctx context.Context, data *attachmentpb.Attachment) error {
	moduleKey := data.GetModuleKey()
	policy := policyFor(moduleKey)

	// (a) content-type allow-list (default-deny).
	if !policy.allowsContentType(data.GetContentType()) {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"attachment.validation.content_type_not_allowed",
			"This file type is not permitted for this record [DEFAULT]"))
	}

	// (b) per-record count cap. 0 means "no cap enforced".
	if policy.MaxFileCount > 0 {
		foreignKey := data.GetForeignKey()
		if moduleKey != "" && foreignKey != "" {
			active, err := uc.countActiveAttachments(ctx, moduleKey, foreignKey)
			if err != nil {
				return err
			}
			if active >= policy.MaxFileCount {
				return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
					"attachment.validation.max_file_count_reached",
					"The maximum number of attachments for this record has been reached [DEFAULT]"))
			}
		}
	}

	return nil
}

// countActiveAttachments counts ACTIVE attachment rows for (moduleKey, foreignKey),
// scoped to the caller's workspace when one is present in context (mirrors
// list_attachments_by_entity's workspace backstop). Active-ness is filtered in Go
// so this does not depend on a particular boolean-filter operator shape.
func (uc *CreateAttachmentUseCase) countActiveAttachments(ctx context.Context, moduleKey, foreignKey string) (int, error) {
	filters := []*commonpb.TypedFilter{
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
					Value:    foreignKey,
				},
			},
		},
	}
	if ws := contextutil.ExtractWorkspaceIDFromContext(ctx); ws != "" {
		filters = append(filters, &commonpb.TypedFilter{
			Field: "workspace_id",
			FilterType: &commonpb.TypedFilter_StringFilter{
				StringFilter: &commonpb.StringFilter{
					Operator: commonpb.StringOperator_STRING_EQUALS,
					Value:    ws,
				},
			},
		})
	}

	resp, err := uc.repositories.Attachment.ListAttachments(ctx, &attachmentpb.ListAttachmentsRequest{
		Filters: &commonpb.FilterRequest{
			Logic:   commonpb.FilterLogic_AND,
			Filters: filters,
		},
	})
	if err != nil {
		return 0, err
	}

	count := 0
	for _, att := range resp.GetData() {
		if att.GetActive() {
			count++
		}
	}
	return count, nil
}
