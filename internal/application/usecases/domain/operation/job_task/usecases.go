package job_task

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_task"
)

// JobTaskRepositories groups all repository dependencies
type JobTaskRepositories struct {
	JobTask pb.JobTaskDomainServiceServer
}

// JobTaskServices groups all business service dependencies
type JobTaskServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
}

// UseCases contains all job-task-related use cases
type UseCases struct {
	CreateJobTask          *CreateJobTaskUseCase
	ReadJobTask            *ReadJobTaskUseCase
	UpdateJobTask          *UpdateJobTaskUseCase
	DeleteJobTask          *DeleteJobTaskUseCase
	ListJobTasks           *ListJobTasksUseCase
	GetJobTaskListPageData *GetJobTaskListPageDataUseCase
	GetJobTaskItemPageData *GetJobTaskItemPageDataUseCase
	ListByPhase            *ListByPhaseUseCase
	ListByAssignee         *ListByAssigneeUseCase
}

// NewUseCases creates a new collection of job task use cases
func NewUseCases(
	repositories JobTaskRepositories,
	services JobTaskServices,
) *UseCases {
	return &UseCases{
		CreateJobTask: &CreateJobTaskUseCase{
			repositories: CreateJobTaskRepositories{JobTask: repositories.JobTask},
			services: CreateJobTaskServices{
				ActionGatekeeper: services.ActionGatekeeper,
				Authorizer:  services.Authorizer,
				Transactor:  services.Transactor,
				Translator:  services.Translator,
				IDGenerator: services.IDGenerator,
			},
		},
		ReadJobTask: &ReadJobTaskUseCase{
			repositories: ReadJobTaskRepositories{JobTask: repositories.JobTask},
			services: ReadJobTaskServices{
				ActionGatekeeper: services.ActionGatekeeper,
				Authorizer: services.Authorizer,
				Translator: services.Translator,
			},
		},
		UpdateJobTask: &UpdateJobTaskUseCase{
			repositories: UpdateJobTaskRepositories{JobTask: repositories.JobTask},
			services: UpdateJobTaskServices{
				ActionGatekeeper: services.ActionGatekeeper,
				Authorizer: services.Authorizer,
				Translator: services.Translator,
			},
		},
		DeleteJobTask: &DeleteJobTaskUseCase{
			repositories: DeleteJobTaskRepositories{JobTask: repositories.JobTask},
			services: DeleteJobTaskServices{
				ActionGatekeeper: services.ActionGatekeeper,
				Authorizer: services.Authorizer,
				Translator: services.Translator,
			},
		},
		ListJobTasks: &ListJobTasksUseCase{
			repositories: ListJobTasksRepositories{JobTask: repositories.JobTask},
			services: ListJobTasksServices{
				ActionGatekeeper: services.ActionGatekeeper,
				Authorizer: services.Authorizer,
				Translator: services.Translator,
			},
		},
		GetJobTaskListPageData: &GetJobTaskListPageDataUseCase{
			repositories: GetJobTaskListPageDataRepositories{JobTask: repositories.JobTask},
			services: GetJobTaskListPageDataServices{
				ActionGatekeeper: services.ActionGatekeeper,
				Authorizer: services.Authorizer,
				Translator: services.Translator,
			},
		},
		GetJobTaskItemPageData: &GetJobTaskItemPageDataUseCase{
			repositories: GetJobTaskItemPageDataRepositories{JobTask: repositories.JobTask},
			services: GetJobTaskItemPageDataServices{
				ActionGatekeeper: services.ActionGatekeeper,
				Authorizer: services.Authorizer,
				Translator: services.Translator,
			},
		},
		ListByPhase: &ListByPhaseUseCase{
			repositories: ListByPhaseRepositories{JobTask: repositories.JobTask},
			services: ListByPhaseServices{
				ActionGatekeeper: services.ActionGatekeeper,
				Authorizer: services.Authorizer,
				Translator: services.Translator,
			},
		},
		ListByAssignee: &ListByAssigneeUseCase{
			repositories: ListByAssigneeRepositories{JobTask: repositories.JobTask},
			services: ListByAssigneeServices{
				ActionGatekeeper: services.ActionGatekeeper,
				Authorizer: services.Authorizer,
				Translator: services.Translator,
			},
		},
	}
}

// ---- CreateJobTask ----

type CreateJobTaskRepositories struct{ JobTask pb.JobTaskDomainServiceServer }
type CreateJobTaskServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
}
type CreateJobTaskUseCase struct {
	repositories CreateJobTaskRepositories
	services     CreateJobTaskServices
}

func (uc *CreateJobTaskUseCase) Execute(ctx context.Context, req *pb.CreateJobTaskRequest) (*pb.CreateJobTaskResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: "job_task", Action: entityid.ActionCreate}); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "job_task.validation.data_required", "job task data is required [DEFAULT]"))
	}
	if strings.TrimSpace(req.Data.Name) == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "job_task.validation.name_required", "job task name is required [DEFAULT]"))
	}
	if req.Data.JobPhaseId == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "job_task.validation.phase_id_required", "job phase ID is required [DEFAULT]"))
	}

	now := time.Now()
	if uc.services.IDGenerator != nil {
		req.Data.Id = uc.services.IDGenerator.GenerateID()
	} else {
		req.Data.Id = fmt.Sprintf("job_task-%d", now.UnixNano())
	}
	dc := now.UnixMilli()
	dcs := now.Format(time.RFC3339)
	req.Data.DateCreated = &dc
	req.Data.DateCreatedString = &dcs
	req.Data.DateModified = &dc
	req.Data.DateModifiedString = &dcs
	req.Data.Active = true

	if uc.services.Transactor != nil {
		var result *pb.CreateJobTaskResponse
		err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
			res, err := uc.repositories.JobTask.CreateJobTask(txCtx, req)
			if err != nil {
				return err
			}
			result = res
			return nil
		})
		if err != nil {
			return nil, err
		}
		return result, nil
	}
	return uc.repositories.JobTask.CreateJobTask(ctx, req)
}

// ---- ReadJobTask ----

type ReadJobTaskRepositories struct{ JobTask pb.JobTaskDomainServiceServer }
type ReadJobTaskServices struct {
	Authorizer ports.Authorizer
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}
type ReadJobTaskUseCase struct {
	repositories ReadJobTaskRepositories
	services     ReadJobTaskServices
}

func (uc *ReadJobTaskUseCase) Execute(ctx context.Context, req *pb.ReadJobTaskRequest) (*pb.ReadJobTaskResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: "job_task", Action: entityid.ActionRead}); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "job_task.validation.id_required", "job task ID is required"))
	}
	result, err := uc.repositories.JobTask.ReadJobTask(ctx, req)
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "job_task.errors.not_found", "job task not found [DEFAULT]"))
	}
	return result, nil
}

// ---- UpdateJobTask ----

type UpdateJobTaskRepositories struct{ JobTask pb.JobTaskDomainServiceServer }
type UpdateJobTaskServices struct {
	Authorizer ports.Authorizer
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}
type UpdateJobTaskUseCase struct {
	repositories UpdateJobTaskRepositories
	services     UpdateJobTaskServices
}

func (uc *UpdateJobTaskUseCase) Execute(ctx context.Context, req *pb.UpdateJobTaskRequest) (*pb.UpdateJobTaskResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: "job_task", Action: entityid.ActionUpdate}); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "job_task.validation.id_required", "job task ID is required"))
	}
	if strings.TrimSpace(req.Data.Name) == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "job_task.validation.name_required", "job task name is required"))
	}
	now := time.Now()
	dm := now.UnixMilli()
	dms := now.Format(time.RFC3339)
	req.Data.DateModified = &dm
	req.Data.DateModifiedString = &dms

	_, err := uc.repositories.JobTask.UpdateJobTask(ctx, req)
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "job_task.errors.update_failed", "job task update failed [DEFAULT]"))
	}
	return &pb.UpdateJobTaskResponse{Success: true, Data: []*pb.JobTask{req.Data}}, nil
}

// ---- DeleteJobTask ----

type DeleteJobTaskRepositories struct{ JobTask pb.JobTaskDomainServiceServer }
type DeleteJobTaskServices struct {
	Authorizer ports.Authorizer
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}
type DeleteJobTaskUseCase struct {
	repositories DeleteJobTaskRepositories
	services     DeleteJobTaskServices
}

func (uc *DeleteJobTaskUseCase) Execute(ctx context.Context, req *pb.DeleteJobTaskRequest) (*pb.DeleteJobTaskResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: "job_task", Action: entityid.ActionDelete}); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "job_task.validation.id_required", "job task ID is required"))
	}
	result, err := uc.repositories.JobTask.DeleteJobTask(ctx, req)
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "job_task.errors.deletion_failed", "job task deletion failed [DEFAULT]"))
	}
	return result, nil
}

// ---- ListJobTasks ----

type ListJobTasksRepositories struct{ JobTask pb.JobTaskDomainServiceServer }
type ListJobTasksServices struct {
	Authorizer ports.Authorizer
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}
type ListJobTasksUseCase struct {
	repositories ListJobTasksRepositories
	services     ListJobTasksServices
}

func (uc *ListJobTasksUseCase) Execute(ctx context.Context, req *pb.ListJobTasksRequest) (*pb.ListJobTasksResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: "job_task", Action: entityid.ActionList}); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "job_task.validation.request_required", "request is required"))
	}
	result, err := uc.repositories.JobTask.ListJobTasks(ctx, req)
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "job_task.errors.list_failed", "job task listing failed [DEFAULT]"))
	}
	return result, nil
}

// ---- GetJobTaskListPageData ----

type GetJobTaskListPageDataRepositories struct{ JobTask pb.JobTaskDomainServiceServer }
type GetJobTaskListPageDataServices struct {
	Authorizer ports.Authorizer
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}
type GetJobTaskListPageDataUseCase struct {
	repositories GetJobTaskListPageDataRepositories
	services     GetJobTaskListPageDataServices
}

func (uc *GetJobTaskListPageDataUseCase) Execute(ctx context.Context, req *pb.GetJobTaskListPageDataRequest) (*pb.GetJobTaskListPageDataResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: "job_task", Action: entityid.ActionList}); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "job_task.validation.request_required", "request is required"))
	}
	return uc.repositories.JobTask.GetJobTaskListPageData(ctx, req)
}

// ---- GetJobTaskItemPageData ----

type GetJobTaskItemPageDataRepositories struct{ JobTask pb.JobTaskDomainServiceServer }
type GetJobTaskItemPageDataServices struct {
	Authorizer ports.Authorizer
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}
type GetJobTaskItemPageDataUseCase struct {
	repositories GetJobTaskItemPageDataRepositories
	services     GetJobTaskItemPageDataServices
}

func (uc *GetJobTaskItemPageDataUseCase) Execute(ctx context.Context, req *pb.GetJobTaskItemPageDataRequest) (*pb.GetJobTaskItemPageDataResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: "job_task", Action: entityid.ActionRead}); err != nil {
		return nil, err
	}
	if req == nil || req.JobTaskId == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "job_task.validation.id_required", "job task ID is required"))
	}
	return uc.repositories.JobTask.GetJobTaskItemPageData(ctx, req)
}

// ---- ListByPhase (extra RPC) ----

type ListByPhaseRepositories struct{ JobTask pb.JobTaskDomainServiceServer }
type ListByPhaseServices struct {
	Authorizer ports.Authorizer
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}
type ListByPhaseUseCase struct {
	repositories ListByPhaseRepositories
	services     ListByPhaseServices
}

func (uc *ListByPhaseUseCase) Execute(ctx context.Context, req *pb.ListJobTasksByPhaseRequest) (*pb.ListJobTasksByPhaseResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: "job_task", Action: entityid.ActionList}); err != nil {
		return nil, err
	}
	if req == nil || req.JobPhaseId == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "job_task.validation.phase_id_required", "job phase ID is required"))
	}
	return uc.repositories.JobTask.ListByPhase(ctx, req)
}

// ---- ListByAssignee (extra RPC) ----

type ListByAssigneeRepositories struct{ JobTask pb.JobTaskDomainServiceServer }
type ListByAssigneeServices struct {
	Authorizer ports.Authorizer
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}
type ListByAssigneeUseCase struct {
	repositories ListByAssigneeRepositories
	services     ListByAssigneeServices
}

func (uc *ListByAssigneeUseCase) Execute(ctx context.Context, req *pb.ListJobTasksByAssigneeRequest) (*pb.ListJobTasksByAssigneeResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: "job_task", Action: entityid.ActionList}); err != nil {
		return nil, err
	}
	if req == nil || req.AssignedTo == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "job_task.validation.assignee_required", "assignee ID is required"))
	}
	return uc.repositories.JobTask.ListByAssignee(ctx, req)
}
