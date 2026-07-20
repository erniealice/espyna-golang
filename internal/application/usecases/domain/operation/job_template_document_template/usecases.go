// Package job_template_document_template holds the sheet-family template-binding
// use cases: standard CRUD/List, the applicability resolver (FindApplicable), and
// the controlled publish transaction. It is the JOSDT sibling (20260720): it binds
// the job_template rendering surface (the outcome-matrix "grade sheet") to a
// document_template, scoped by workspace + (optional) price_schedule + (optional)
// job_category. Each use case action-gates then delegates to the repository;
// tenant isolation for the resolver + publish is enforced in the postgres adapter
// from trusted context.
//
// One file per operation (create.go / read.go / update.go / delete.go /
// list.go / find_applicable.go / publish.go). This file carries only the shared
// wiring (Repositories / Services / UseCases / NewUseCases) + the requireRequest
// helper (S-5).
package job_template_document_template

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template_document_template"
)

// Repositories groups the primary repository dependency.
type Repositories struct {
	JobTemplateDocumentTemplate pb.JobTemplateDocumentTemplateDomainServiceServer
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
	CreateJobTemplateDocumentTemplate         *CreateUseCase
	ReadJobTemplateDocumentTemplate           *ReadUseCase
	UpdateJobTemplateDocumentTemplate         *UpdateUseCase
	DeleteJobTemplateDocumentTemplate         *DeleteUseCase
	ListJobTemplateDocumentTemplates          *ListUseCase
	FindApplicableJobTemplateDocumentTemplate *FindApplicableUseCase
	PublishJobTemplateDocumentTemplate        *PublishUseCase
}

// NewUseCases wires the binding use cases.
func NewUseCases(r Repositories, s Services) *UseCases {
	return &UseCases{
		CreateJobTemplateDocumentTemplate:         &CreateUseCase{repo: r.JobTemplateDocumentTemplate, svc: s},
		ReadJobTemplateDocumentTemplate:           &ReadUseCase{repo: r.JobTemplateDocumentTemplate, svc: s},
		UpdateJobTemplateDocumentTemplate:         &UpdateUseCase{repo: r.JobTemplateDocumentTemplate, svc: s},
		DeleteJobTemplateDocumentTemplate:         &DeleteUseCase{repo: r.JobTemplateDocumentTemplate, svc: s},
		ListJobTemplateDocumentTemplates:          &ListUseCase{repo: r.JobTemplateDocumentTemplate, svc: s},
		FindApplicableJobTemplateDocumentTemplate: &FindApplicableUseCase{repo: r.JobTemplateDocumentTemplate, svc: s},
		PublishJobTemplateDocumentTemplate:        &PublishUseCase{repo: r.JobTemplateDocumentTemplate, svc: s},
	}
}

func (s Services) requireRequest(ctx context.Context, ok bool) error {
	if ok {
		return nil
	}
	return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, s.Translator,
		"job_template_document_template.validation.request_required", "Request is required [DEFAULT]"))
}
