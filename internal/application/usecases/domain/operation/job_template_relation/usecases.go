// Package job_template_relation contains Layer-7 use case wrappers for the
// JobTemplateRelation proto domain service. 20260518-hexagonal-strict-adherence
// Phase 3 F7 closure — replaces the raw
// jobtemplaterelationpb.JobTemplateRelationDomainServiceServer leak that was
// previously exposed as a flat field on OperationUseCases.
//
// 20260721 — CRUD use cases (Create/Read/Update/Delete) added with a
// fail-closed cross-workspace guard baked in (red-team HIGH #4): the relation
// join table carries no workspace_id, so scope is resolved from the owning
// parent/child job_template rows.
package job_template_relation

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	jobtemplatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template"
	jobtemplaterelationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template_relation"
)

// JobTemplateRelationRepositories groups all repository dependencies.
type JobTemplateRelationRepositories struct {
	JobTemplateRelation jobtemplaterelationpb.JobTemplateRelationDomainServiceServer
	// JobTemplate is the workspace anchor for the (column-less) relation rows.
	JobTemplate jobtemplatepb.JobTemplateDomainServiceServer
}

// JobTemplateRelationServices groups all business service dependencies.
type JobTemplateRelationServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// UseCases contains all job-template-relation use cases.
type UseCases struct {
	CreateJobTemplateRelation *CreateJobTemplateRelationUseCase
	ReadJobTemplateRelation   *ReadJobTemplateRelationUseCase
	UpdateJobTemplateRelation *UpdateJobTemplateRelationUseCase
	DeleteJobTemplateRelation *DeleteJobTemplateRelationUseCase
	ListByParent              *ListByParentUseCase
}

// NewUseCases creates the job-template-relation use case sub-aggregate.
func NewUseCases(
	repositories JobTemplateRelationRepositories,
	services JobTemplateRelationServices,
) *UseCases {
	if repositories.JobTemplateRelation == nil {
		return &UseCases{}
	}
	return &UseCases{
		CreateJobTemplateRelation: NewCreateJobTemplateRelationUseCase(
			CreateJobTemplateRelationRepositories{
				JobTemplateRelation: repositories.JobTemplateRelation,
				JobTemplate:         repositories.JobTemplate,
			},
			CreateJobTemplateRelationServices{
				Authorizer:       services.Authorizer,
				Transactor:       services.Transactor,
				Translator:       services.Translator,
				IDGenerator:      services.IDGenerator,
				ActionGatekeeper: services.ActionGatekeeper,
			},
		),
		ReadJobTemplateRelation: NewReadJobTemplateRelationUseCase(
			ReadJobTemplateRelationRepositories{
				JobTemplateRelation: repositories.JobTemplateRelation,
				JobTemplate:         repositories.JobTemplate,
			},
			ReadJobTemplateRelationServices{
				Authorizer:       services.Authorizer,
				Translator:       services.Translator,
				ActionGatekeeper: services.ActionGatekeeper,
			},
		),
		UpdateJobTemplateRelation: NewUpdateJobTemplateRelationUseCase(
			UpdateJobTemplateRelationRepositories{
				JobTemplateRelation: repositories.JobTemplateRelation,
				JobTemplate:         repositories.JobTemplate,
			},
			UpdateJobTemplateRelationServices{
				Authorizer:       services.Authorizer,
				Transactor:       services.Transactor,
				Translator:       services.Translator,
				ActionGatekeeper: services.ActionGatekeeper,
			},
		),
		DeleteJobTemplateRelation: NewDeleteJobTemplateRelationUseCase(
			DeleteJobTemplateRelationRepositories{
				JobTemplateRelation: repositories.JobTemplateRelation,
				JobTemplate:         repositories.JobTemplate,
			},
			DeleteJobTemplateRelationServices{
				Authorizer:       services.Authorizer,
				Transactor:       services.Transactor,
				Translator:       services.Translator,
				ActionGatekeeper: services.ActionGatekeeper,
			},
		),
		ListByParent: NewListByParentUseCase(
			ListByParentRepositories{
				JobTemplateRelation: repositories.JobTemplateRelation,
				JobTemplate:         repositories.JobTemplate,
			},
			ListByParentServices{
				ActionGatekeeper: services.ActionGatekeeper,
				Authorizer:       services.Authorizer,
				Translator:       services.Translator,
			},
		),
	}
}
