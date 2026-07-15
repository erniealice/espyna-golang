// Package job_outcome_summary_document_template holds the report-card
// template-binding use cases: standard CRUD/List, the applicability resolver
// (FindApplicable), and the controlled publish transaction. Each use case
// action-gates then delegates to the repository; tenant isolation for the
// resolver + publish is enforced in the postgres adapter from trusted context.
package job_outcome_summary_document_template

import (
	"context"
	"errors"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_outcome_summary_document_template"
)

// Repositories groups the primary repository dependency.
type Repositories struct {
	JobOutcomeSummaryDocumentTemplate pb.JobOutcomeSummaryDocumentTemplateDomainServiceServer
}

// Services groups the shared business-service dependencies.
type Services struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// UseCases aggregates the binding CRUD + resolver + publish use cases.
type UseCases struct {
	CreateJobOutcomeSummaryDocumentTemplate         *CreateUseCase
	ReadJobOutcomeSummaryDocumentTemplate           *ReadUseCase
	UpdateJobOutcomeSummaryDocumentTemplate         *UpdateUseCase
	DeleteJobOutcomeSummaryDocumentTemplate         *DeleteUseCase
	ListJobOutcomeSummaryDocumentTemplates          *ListUseCase
	FindApplicableJobOutcomeSummaryDocumentTemplate *FindApplicableUseCase
	PublishJobOutcomeSummaryDocumentTemplate        *PublishUseCase
}

// NewUseCases wires the binding use cases.
func NewUseCases(r Repositories, s Services) *UseCases {
	return &UseCases{
		CreateJobOutcomeSummaryDocumentTemplate:         &CreateUseCase{repo: r.JobOutcomeSummaryDocumentTemplate, svc: s},
		ReadJobOutcomeSummaryDocumentTemplate:           &ReadUseCase{repo: r.JobOutcomeSummaryDocumentTemplate, svc: s},
		UpdateJobOutcomeSummaryDocumentTemplate:         &UpdateUseCase{repo: r.JobOutcomeSummaryDocumentTemplate, svc: s},
		DeleteJobOutcomeSummaryDocumentTemplate:         &DeleteUseCase{repo: r.JobOutcomeSummaryDocumentTemplate, svc: s},
		ListJobOutcomeSummaryDocumentTemplates:          &ListUseCase{repo: r.JobOutcomeSummaryDocumentTemplate, svc: s},
		FindApplicableJobOutcomeSummaryDocumentTemplate: &FindApplicableUseCase{repo: r.JobOutcomeSummaryDocumentTemplate, svc: s},
		PublishJobOutcomeSummaryDocumentTemplate:        &PublishUseCase{repo: r.JobOutcomeSummaryDocumentTemplate, svc: s},
	}
}

func (s Services) requireRequest(ctx context.Context, ok bool) error {
	if ok {
		return nil
	}
	return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, s.Translator,
		"job_outcome_summary_document_template.validation.request_required", "Request is required [DEFAULT]"))
}

// --- Create ---------------------------------------------------------------

type CreateUseCase struct {
	repo pb.JobOutcomeSummaryDocumentTemplateDomainServiceServer
	svc  Services
}

func (uc *CreateUseCase) Execute(ctx context.Context, req *pb.CreateJobOutcomeSummaryDocumentTemplateRequest) (*pb.CreateJobOutcomeSummaryDocumentTemplateResponse, error) {
	if err := uc.svc.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.JobOutcomeSummaryDocumentTemplate, Action: entityid.ActionCreate}); err != nil {
		return nil, err
	}
	if err := uc.svc.requireRequest(ctx, req != nil && req.Data != nil); err != nil {
		return nil, err
	}
	// Tenancy is assigned from the trusted request context by the persistence
	// layer; never honor a client-supplied workspace on write (gate H1).
	req.Data.WorkspaceId = ""
	now := time.Now()
	if req.Data.Id == "" && uc.svc.IDGenerator != nil {
		req.Data.Id = uc.svc.IDGenerator.GenerateID()
	}
	req.Data.Active = true
	ms := now.UnixMilli()
	req.Data.DateCreated = &ms
	req.Data.DateModified = &ms
	return uc.repo.CreateJobOutcomeSummaryDocumentTemplate(ctx, req)
}

// --- Read -----------------------------------------------------------------

type ReadUseCase struct {
	repo pb.JobOutcomeSummaryDocumentTemplateDomainServiceServer
	svc  Services
}

func (uc *ReadUseCase) Execute(ctx context.Context, req *pb.ReadJobOutcomeSummaryDocumentTemplateRequest) (*pb.ReadJobOutcomeSummaryDocumentTemplateResponse, error) {
	if err := uc.svc.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.JobOutcomeSummaryDocumentTemplate, Action: entityid.ActionRead}); err != nil {
		return nil, err
	}
	if err := uc.svc.requireRequest(ctx, req != nil); err != nil {
		return nil, err
	}
	return uc.repo.ReadJobOutcomeSummaryDocumentTemplate(ctx, req)
}

// --- Update ---------------------------------------------------------------

type UpdateUseCase struct {
	repo pb.JobOutcomeSummaryDocumentTemplateDomainServiceServer
	svc  Services
}

func (uc *UpdateUseCase) Execute(ctx context.Context, req *pb.UpdateJobOutcomeSummaryDocumentTemplateRequest) (*pb.UpdateJobOutcomeSummaryDocumentTemplateResponse, error) {
	if err := uc.svc.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.JobOutcomeSummaryDocumentTemplate, Action: entityid.ActionUpdate}); err != nil {
		return nil, err
	}
	if err := uc.svc.requireRequest(ctx, req != nil && req.Data != nil); err != nil {
		return nil, err
	}
	// workspace_id is the immutable tenant anchor; never honor a client-supplied
	// workspace on update (gate H1).
	req.Data.WorkspaceId = ""
	ms := time.Now().UnixMilli()
	req.Data.DateModified = &ms
	return uc.repo.UpdateJobOutcomeSummaryDocumentTemplate(ctx, req)
}

// --- Delete ---------------------------------------------------------------

type DeleteUseCase struct {
	repo pb.JobOutcomeSummaryDocumentTemplateDomainServiceServer
	svc  Services
}

func (uc *DeleteUseCase) Execute(ctx context.Context, req *pb.DeleteJobOutcomeSummaryDocumentTemplateRequest) (*pb.DeleteJobOutcomeSummaryDocumentTemplateResponse, error) {
	if err := uc.svc.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.JobOutcomeSummaryDocumentTemplate, Action: entityid.ActionDelete}); err != nil {
		return nil, err
	}
	if err := uc.svc.requireRequest(ctx, req != nil); err != nil {
		return nil, err
	}
	return uc.repo.DeleteJobOutcomeSummaryDocumentTemplate(ctx, req)
}

// --- List -----------------------------------------------------------------

type ListUseCase struct {
	repo pb.JobOutcomeSummaryDocumentTemplateDomainServiceServer
	svc  Services
}

func (uc *ListUseCase) Execute(ctx context.Context, req *pb.ListJobOutcomeSummaryDocumentTemplatesRequest) (*pb.ListJobOutcomeSummaryDocumentTemplatesResponse, error) {
	if err := uc.svc.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.JobOutcomeSummaryDocumentTemplate, Action: entityid.ActionList}); err != nil {
		return nil, err
	}
	if err := uc.svc.requireRequest(ctx, req != nil); err != nil {
		return nil, err
	}
	return uc.repo.ListJobOutcomeSummaryDocumentTemplates(ctx, req)
}

// --- FindApplicable (resolver) --------------------------------------------

type FindApplicableUseCase struct {
	repo pb.JobOutcomeSummaryDocumentTemplateDomainServiceServer
	svc  Services
}

func (uc *FindApplicableUseCase) Execute(ctx context.Context, req *pb.FindApplicableJobOutcomeSummaryDocumentTemplateRequest) (*pb.FindApplicableJobOutcomeSummaryDocumentTemplateResponse, error) {
	if err := uc.svc.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.JobOutcomeSummaryDocumentTemplate, Action: entityid.ActionList}); err != nil {
		return nil, err
	}
	if err := uc.svc.requireRequest(ctx, req != nil); err != nil {
		return nil, err
	}
	return uc.repo.FindApplicableJobOutcomeSummaryDocumentTemplate(ctx, req)
}

// --- Publish --------------------------------------------------------------

type PublishUseCase struct {
	repo pb.JobOutcomeSummaryDocumentTemplateDomainServiceServer
	svc  Services
}

func (uc *PublishUseCase) Execute(ctx context.Context, req *pb.PublishJobOutcomeSummaryDocumentTemplateRequest) (*pb.PublishJobOutcomeSummaryDocumentTemplateResponse, error) {
	if err := uc.svc.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.JobOutcomeSummaryDocumentTemplate, Action: entityid.ActionUpdate}); err != nil {
		return nil, err
	}
	if err := uc.svc.requireRequest(ctx, req != nil && req.Id != ""); err != nil {
		return nil, err
	}
	return uc.repo.PublishJobOutcomeSummaryDocumentTemplate(ctx, req)
}
