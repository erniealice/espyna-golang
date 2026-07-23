package template_task_criteria

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	jobtemplatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template"
	jobtemplatephasepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template_phase"
	jobtemplatetaskpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template_task"
	outcomecriteriapb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/outcome_criteria"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/template_task_criteria"
)

// TemplateTaskCriteriaRepositories groups all repository dependencies. The
// job_template_task/phase/template repos anchor the fail-closed
// cross-workspace guard (template_task_criteria carries no workspace_id), and
// OutcomeCriteria anchors the pinned-criterion workspace check.
type TemplateTaskCriteriaRepositories struct {
	TemplateTaskCriteria pb.TemplateTaskCriteriaDomainServiceServer
	JobTemplateTask      jobtemplatetaskpb.JobTemplateTaskDomainServiceServer
	JobTemplatePhase     jobtemplatephasepb.JobTemplatePhaseDomainServiceServer
	JobTemplate          jobtemplatepb.JobTemplateDomainServiceServer
	OutcomeCriteria      outcomecriteriapb.OutcomeCriteriaDomainServiceServer
}

// TemplateTaskCriteriaServices groups all business service dependencies
type TemplateTaskCriteriaServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
}

// UseCases contains all template_task_criteria-related use cases
type UseCases struct {
	CreateTemplateTaskCriteria          *CreateTemplateTaskCriteriaUseCase
	ReadTemplateTaskCriteria            *ReadTemplateTaskCriteriaUseCase
	UpdateTemplateTaskCriteria          *UpdateTemplateTaskCriteriaUseCase
	DeleteTemplateTaskCriteria          *DeleteTemplateTaskCriteriaUseCase
	ListTemplateTaskCriteria            *ListTemplateTaskCriteriaUseCase
	GetTemplateTaskCriteriaListPageData *GetTemplateTaskCriteriaListPageDataUseCase
	GetTemplateTaskCriteriaItemPageData *GetTemplateTaskCriteriaItemPageDataUseCase
	ListByTemplateTask                  *ListByTemplateTaskUseCase
	ListByCriteria                      *ListByCriteriaUseCase
}

// NewUseCases creates a new collection of template_task_criteria use cases
func NewUseCases(
	repositories TemplateTaskCriteriaRepositories,
	services TemplateTaskCriteriaServices,
) *UseCases {
	createRepos := CreateTemplateTaskCriteriaRepositories{
		TemplateTaskCriteria: repositories.TemplateTaskCriteria,
		JobTemplateTask:      repositories.JobTemplateTask,
		JobTemplatePhase:     repositories.JobTemplatePhase,
		JobTemplate:          repositories.JobTemplate,
		OutcomeCriteria:      repositories.OutcomeCriteria,
	}
	createServices := CreateTemplateTaskCriteriaServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadTemplateTaskCriteriaRepositories{
		TemplateTaskCriteria: repositories.TemplateTaskCriteria,
		JobTemplateTask:      repositories.JobTemplateTask,
		JobTemplatePhase:     repositories.JobTemplatePhase,
		JobTemplate:          repositories.JobTemplate,
	}
	readServices := ReadTemplateTaskCriteriaServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	updateRepos := UpdateTemplateTaskCriteriaRepositories{
		TemplateTaskCriteria: repositories.TemplateTaskCriteria,
		JobTemplateTask:      repositories.JobTemplateTask,
		JobTemplatePhase:     repositories.JobTemplatePhase,
		JobTemplate:          repositories.JobTemplate,
		OutcomeCriteria:      repositories.OutcomeCriteria,
	}
	updateServices := UpdateTemplateTaskCriteriaServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	deleteRepos := DeleteTemplateTaskCriteriaRepositories{
		TemplateTaskCriteria: repositories.TemplateTaskCriteria,
		JobTemplateTask:      repositories.JobTemplateTask,
		JobTemplatePhase:     repositories.JobTemplatePhase,
		JobTemplate:          repositories.JobTemplate,
	}
	deleteServices := DeleteTemplateTaskCriteriaServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listRepos := ListTemplateTaskCriteriaRepositories{
		TemplateTaskCriteria: repositories.TemplateTaskCriteria,
	}
	listServices := ListTemplateTaskCriteriaServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listPageDataRepos := GetTemplateTaskCriteriaListPageDataRepositories{
		TemplateTaskCriteria: repositories.TemplateTaskCriteria,
	}
	listPageDataServices := GetTemplateTaskCriteriaListPageDataServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	itemPageDataRepos := GetTemplateTaskCriteriaItemPageDataRepositories{
		TemplateTaskCriteria: repositories.TemplateTaskCriteria,
	}
	itemPageDataServices := GetTemplateTaskCriteriaItemPageDataServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listByTemplateTaskRepos := ListByTemplateTaskRepositories{
		TemplateTaskCriteria: repositories.TemplateTaskCriteria,
		JobTemplateTask:      repositories.JobTemplateTask,
		JobTemplatePhase:     repositories.JobTemplatePhase,
		JobTemplate:          repositories.JobTemplate,
	}
	listByTemplateTaskServices := ListByTemplateTaskServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listByCriteriaRepos := ListByCriteriaRepositories{
		TemplateTaskCriteria: repositories.TemplateTaskCriteria,
	}
	listByCriteriaServices := ListByCriteriaServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	return &UseCases{
		CreateTemplateTaskCriteria:          NewCreateTemplateTaskCriteriaUseCase(createRepos, createServices),
		ReadTemplateTaskCriteria:            NewReadTemplateTaskCriteriaUseCase(readRepos, readServices),
		UpdateTemplateTaskCriteria:          NewUpdateTemplateTaskCriteriaUseCase(updateRepos, updateServices),
		DeleteTemplateTaskCriteria:          NewDeleteTemplateTaskCriteriaUseCase(deleteRepos, deleteServices),
		ListTemplateTaskCriteria:            NewListTemplateTaskCriteriaUseCase(listRepos, listServices),
		GetTemplateTaskCriteriaListPageData: NewGetTemplateTaskCriteriaListPageDataUseCase(listPageDataRepos, listPageDataServices),
		GetTemplateTaskCriteriaItemPageData: NewGetTemplateTaskCriteriaItemPageDataUseCase(itemPageDataRepos, itemPageDataServices),
		ListByTemplateTask:                  NewListByTemplateTaskUseCase(listByTemplateTaskRepos, listByTemplateTaskServices),
		ListByCriteria:                      NewListByCriteriaUseCase(listByCriteriaRepos, listByCriteriaServices),
	}
}
