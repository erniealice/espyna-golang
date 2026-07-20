package job_template_document_template

import (
	"context"
	"errors"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	enums "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/enums"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template_document_template"
)

type CreateUseCase struct {
	repo pb.JobTemplateDocumentTemplateDomainServiceServer
	svc  Services
}

func (uc *CreateUseCase) Execute(ctx context.Context, req *pb.CreateJobTemplateDocumentTemplateRequest) (*pb.CreateJobTemplateDocumentTemplateResponse, error) {
	if err := uc.svc.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.JobTemplateDocumentTemplate, Action: entityid.ActionCreate}); err != nil {
		return nil, err
	}
	if err := uc.svc.requireRequest(ctx, req != nil && req.Data != nil); err != nil {
		return nil, err
	}
	// S-6 — minimal required-field validation: a binding must reference the
	// document_template artifact it selects.
	if req.Data.DocumentTemplateId == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.svc.Translator,
			"job_template_document_template.validation.document_template_required",
			"A document template is required [DEFAULT]"))
	}
	// Tenancy is assigned from the trusted request context by the persistence
	// layer; never honor a client-supplied workspace on write (gate H1).
	req.Data.WorkspaceId = ""
	// RA2 P1 — server-owned lifecycle. A binding is ALWAYS born DRAFT and
	// unversioned/provisional (version=0); the publish audit is stamped only by
	// the Publish transaction. Never honor a client-supplied version_status,
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
	return uc.repo.CreateJobTemplateDocumentTemplate(ctx, req)
}
