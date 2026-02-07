package domain

import (
	"fmt"

	"leapfor.xyz/espyna/internal/composition/contracts"
	"leapfor.xyz/espyna/internal/infrastructure/registry"

	// Protobuf domain services - Workflow domain
	activitypb "leapfor.xyz/esqyma/golang/v1/domain/workflow/activity"
	activitytemplatepb "leapfor.xyz/esqyma/golang/v1/domain/workflow/activity_template"
	stagepb "leapfor.xyz/esqyma/golang/v1/domain/workflow/stage"
	stagetemplatepb "leapfor.xyz/esqyma/golang/v1/domain/workflow/stage_template"
	workflowpb "leapfor.xyz/esqyma/golang/v1/domain/workflow/workflow"
	workflowtemplatepb "leapfor.xyz/esqyma/golang/v1/domain/workflow/workflow_template"
)

// WorkflowRepositories contains all 6 workflow domain repositories
type WorkflowRepositories struct {
	Workflow         workflowpb.WorkflowDomainServiceServer
	WorkflowTemplate workflowtemplatepb.WorkflowTemplateDomainServiceServer
	Stage            stagepb.StageDomainServiceServer
	Activity         activitypb.ActivityDomainServiceServer
	StageTemplate    stagetemplatepb.StageTemplateDomainServiceServer
	ActivityTemplate activitytemplatepb.ActivityTemplateDomainServiceServer
}

// NewWorkflowRepositories creates and returns a new set of WorkflowRepositories
func NewWorkflowRepositories(dbProvider contracts.Provider, dbTableConfig *registry.DatabaseTableConfig) (*WorkflowRepositories, error) {
	if dbProvider == nil {
		return nil, fmt.Errorf("database provider not initialized")
	}

	repoCreator, ok := dbProvider.(contracts.RepositoryProvider)
	if !ok {
		return nil, fmt.Errorf("database provider doesn't implement contracts.RepositoryProvider interface")
	}

	conn := repoCreator.GetConnection()

	// Create each repository individually using configured table names directly from dbTableConfig
	workflowRepo, err := repoCreator.CreateRepository("workflow", conn, dbTableConfig.Workflow)
	if err != nil {
		return nil, fmt.Errorf("failed to create workflow repository: %w", err)
	}

	workflowTemplateRepo, err := repoCreator.CreateRepository("workflow_template", conn, dbTableConfig.WorkflowTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to create workflow_template repository: %w", err)
	}

	stageRepo, err := repoCreator.CreateRepository("stage", conn, dbTableConfig.Stage)
	if err != nil {
		return nil, fmt.Errorf("failed to create stage repository: %w", err)
	}

	stageTemplateRepo, err := repoCreator.CreateRepository("stage_template", conn, dbTableConfig.StageTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to create stage_template repository: %w", err)
	}

	activityRepo, err := repoCreator.CreateRepository("activity", conn, dbTableConfig.Activity)
	if err != nil {
		return nil, fmt.Errorf("failed to create activity repository: %w", err)
	}

	activityTemplateRepo, err := repoCreator.CreateRepository("activity_template", conn, dbTableConfig.ActivityTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to create activity_template repository: %w", err)
	}

	// Type assert each repository to its interface
	return &WorkflowRepositories{
		Workflow:         workflowRepo.(workflowpb.WorkflowDomainServiceServer),
		WorkflowTemplate: workflowTemplateRepo.(workflowtemplatepb.WorkflowTemplateDomainServiceServer),
		Stage:            stageRepo.(stagepb.StageDomainServiceServer),
		StageTemplate:    stageTemplateRepo.(stagetemplatepb.StageTemplateDomainServiceServer),
		Activity:         activityRepo.(activitypb.ActivityDomainServiceServer),
		ActivityTemplate: activityTemplateRepo.(activitytemplatepb.ActivityTemplateDomainServiceServer),
	}, nil
}
