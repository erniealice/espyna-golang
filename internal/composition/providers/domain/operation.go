package domain

import (
	"fmt"

	"github.com/erniealice/espyna-golang/internal/composition/contracts"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"

	jobpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job"
	jobphasepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_phase"
	jobtaskpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_task"
	jobtemplatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template"
	jobtemplatephasepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template_phase"
	jobtemplatetaskpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template_task"
)

// OperationRepositories contains all operation domain repositories.
type OperationRepositories struct {
	Job              jobpb.JobDomainServiceServer
	JobPhase         jobphasepb.JobPhaseDomainServiceServer
	JobTask          jobtaskpb.JobTaskDomainServiceServer
	JobTemplate      jobtemplatepb.JobTemplateDomainServiceServer
	JobTemplatePhase jobtemplatephasepb.JobTemplatePhaseDomainServiceServer
	JobTemplateTask  jobtemplatetaskpb.JobTemplateTaskDomainServiceServer
}

// NewOperationRepositories creates and returns a new set of OperationRepositories.
func NewOperationRepositories(dbProvider contracts.Provider, dbTableConfig *registry.DatabaseTableConfig) (*OperationRepositories, error) {
	if dbProvider == nil {
		return nil, fmt.Errorf("database provider not initialized")
	}

	repoCreator, ok := dbProvider.(contracts.RepositoryProvider)
	if !ok {
		return nil, fmt.Errorf("database provider doesn't implement contracts.RepositoryProvider interface")
	}

	conn := repoCreator.GetConnection()

	jobRepo, err := repoCreator.CreateRepository(entityid.Job, conn, dbTableConfig.Job)
	if err != nil {
		return nil, fmt.Errorf("failed to create job repository: %w", err)
	}

	jobPhaseRepo, err := repoCreator.CreateRepository(entityid.JobPhase, conn, dbTableConfig.JobPhase)
	if err != nil {
		return nil, fmt.Errorf("failed to create job_phase repository: %w", err)
	}

	jobTaskRepo, err := repoCreator.CreateRepository(entityid.JobTask, conn, dbTableConfig.JobTask)
	if err != nil {
		return nil, fmt.Errorf("failed to create job_task repository: %w", err)
	}

	jobTemplateRepo, err := repoCreator.CreateRepository(entityid.JobTemplate, conn, dbTableConfig.JobTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to create job_template repository: %w", err)
	}

	jobTemplatePhaseRepo, err := repoCreator.CreateRepository(entityid.JobTemplatePhase, conn, dbTableConfig.JobTemplatePhase)
	if err != nil {
		return nil, fmt.Errorf("failed to create job_template_phase repository: %w", err)
	}

	jobTemplateTaskRepo, err := repoCreator.CreateRepository(entityid.JobTemplateTask, conn, dbTableConfig.JobTemplateTask)
	if err != nil {
		return nil, fmt.Errorf("failed to create job_template_task repository: %w", err)
	}

	return &OperationRepositories{
		Job:              jobRepo.(jobpb.JobDomainServiceServer),
		JobPhase:         jobPhaseRepo.(jobphasepb.JobPhaseDomainServiceServer),
		JobTask:          jobTaskRepo.(jobtaskpb.JobTaskDomainServiceServer),
		JobTemplate:      jobTemplateRepo.(jobtemplatepb.JobTemplateDomainServiceServer),
		JobTemplatePhase: jobTemplatePhaseRepo.(jobtemplatephasepb.JobTemplatePhaseDomainServiceServer),
		JobTemplateTask:  jobTemplateTaskRepo.(jobtemplatetaskpb.JobTemplateTaskDomainServiceServer),
	}, nil
}
