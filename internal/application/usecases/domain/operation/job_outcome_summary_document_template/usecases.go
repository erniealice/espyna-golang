// Package job_outcome_summary_document_template holds the report-card
// template-binding use cases: standard CRUD/List, the applicability resolver
// (FindApplicable), and the controlled publish transaction. Each use case
// action-gates then delegates to the repository; tenant isolation for the
// resolver + publish is enforced in the postgres adapter from trusted context.
//
// One file per operation (create.go / read.go / update.go / delete.go /
// list.go / find_applicable.go / publish.go). This file carries only the shared
// wiring (Repositories / Services / UseCases / NewUseCases) + the requireRequest
// helper (S-5).
package job_outcome_summary_document_template

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
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
