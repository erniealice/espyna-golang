package subscription_group_document_template

import (
	"context"
	"errors"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	enums "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/enums"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/subscription_group_document_template"
)

type CreateUseCase struct {
	repo pb.SubscriptionGroupDocumentTemplateDomainServiceServer
	svc  Services
}

func (uc *CreateUseCase) Execute(ctx context.Context, req *pb.CreateSubscriptionGroupDocumentTemplateRequest) (*pb.CreateSubscriptionGroupDocumentTemplateResponse, error) {
	if err := uc.svc.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.SubscriptionGroupDocumentTemplate, Action: entityid.ActionCreate}); err != nil {
		return nil, err
	}
	if err := uc.svc.requireRequest(ctx, req != nil && req.Data != nil); err != nil {
		return nil, err
	}
	// S-6 — minimal required-field validation: a binding must reference the
	// document_template artifact it selects.
	if req.Data.DocumentTemplateId == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.svc.Translator,
			"subscription_group_document_template.validation.document_template_required",
			"A document template is required [DEFAULT]"))
	}
	// Tenancy is assigned from the trusted request context by the persistence
	// layer; never honor a client-supplied workspace on write (gate H1).
	req.Data.WorkspaceId = ""
	// RA2 P1 — server-owned lifecycle. A binding is ALWAYS born DRAFT and
	// unversioned/provisional (version=0); the publish transaction stamps only the
	// publish audit trail. Never honor a client-supplied version_status,
	// version, or publish audit.
	req.Data.VersionStatus = enums.VersionStatus_VERSION_STATUS_DRAFT
	req.Data.Version = 0
	req.Data.PublishedAt = nil
	req.Data.PublishedBy = nil
	now := time.Now()
	if req.Data.Id == "" && uc.svc.IDGenerator != nil {
		req.Data.Id = uc.svc.IDGenerator.GenerateID()
	}
	req.Data.Active = true
	ms := now.UnixMilli()
	req.Data.DateCreated = &ms
	req.Data.DateModified = &ms
	if uc.svc.Transactor == nil || !uc.svc.Transactor.SupportsTransactions() {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.svc.Translator,
			"subscription_group_document_template.errors.transaction_unavailable",
			"Transaction support is required to create a binding [DEFAULT]"))
	}

	var response *pb.CreateSubscriptionGroupDocumentTemplateResponse
	if err := uc.svc.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		var createErr error
		response, createErr = uc.repo.CreateSubscriptionGroupDocumentTemplate(txCtx, req)
		return createErr
	}); err != nil {
		return nil, err
	}
	return response, nil
}
