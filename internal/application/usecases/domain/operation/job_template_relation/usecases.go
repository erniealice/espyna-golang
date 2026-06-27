// Package job_template_relation contains Layer-7 use case wrappers for the
// JobTemplateRelation proto domain service. 20260518-hexagonal-strict-adherence
// Phase 3 F7 closure — replaces the raw
// jobtemplaterelationpb.JobTemplateRelationDomainServiceServer leak that was
// previously exposed as a flat field on OperationUseCases.
package job_template_relation

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	jobtemplaterelationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template_relation"
)

// JobTemplateRelationRepositories groups all repository dependencies.
type JobTemplateRelationRepositories struct {
	JobTemplateRelation jobtemplaterelationpb.JobTemplateRelationDomainServiceServer
}

// JobTemplateRelationServices groups all business service dependencies.
type JobTemplateRelationServices struct {
	Authorizer ports.Authorizer
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// UseCases contains all job-template-relation use cases.
type UseCases struct {
	ListByParent *ListByParentUseCase
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
		ListByParent: NewListByParentUseCase(
			ListByParentRepositories{JobTemplateRelation: repositories.JobTemplateRelation},
			ListByParentServices{
				ActionGatekeeper: services.ActionGatekeeper,
				Authorizer: services.Authorizer,
				Translator: services.Translator,
			},
		),
	}
}
