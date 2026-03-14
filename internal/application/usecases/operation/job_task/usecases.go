package job_task

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_task"
)

// JobTaskRepositories groups all repository dependencies
type JobTaskRepositories struct {
	JobTask pb.JobTaskDomainServiceServer
}

// JobTaskServices groups all business service dependencies
type JobTaskServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
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
				AuthorizationService: services.AuthorizationService,
				TransactionService:   services.TransactionService,
				TranslationService:   services.TranslationService,
				IDService:            services.IDService,
			},
		},
		ReadJobTask: &ReadJobTaskUseCase{
			repositories: ReadJobTaskRepositories{JobTask: repositories.JobTask},
			services: ReadJobTaskServices{
				AuthorizationService: services.AuthorizationService,
				TranslationService:   services.TranslationService,
			},
		},
		UpdateJobTask: &UpdateJobTaskUseCase{
			repositories: UpdateJobTaskRepositories{JobTask: repositories.JobTask},
			services: UpdateJobTaskServices{
				AuthorizationService: services.AuthorizationService,
				TranslationService:   services.TranslationService,
			},
		},
		DeleteJobTask: &DeleteJobTaskUseCase{
			repositories: DeleteJobTaskRepositories{JobTask: repositories.JobTask},
			services: DeleteJobTaskServices{
				AuthorizationService: services.AuthorizationService,
				TranslationService:   services.TranslationService,
			},
		},
		ListJobTasks: &ListJobTasksUseCase{
			repositories: ListJobTasksRepositories{JobTask: repositories.JobTask},
			services: ListJobTasksServices{
				AuthorizationService: services.AuthorizationService,
				TranslationService:   services.TranslationService,
			},
		},
		GetJobTaskListPageData: &GetJobTaskListPageDataUseCase{
			repositories: GetJobTaskListPageDataRepositories{JobTask: repositories.JobTask},
			services: GetJobTaskListPageDataServices{
				AuthorizationService: services.AuthorizationService,
				TranslationService:   services.TranslationService,
			},
		},
		GetJobTaskItemPageData: &GetJobTaskItemPageDataUseCase{
			repositories: GetJobTaskItemPageDataRepositories{JobTask: repositories.JobTask},
			services: GetJobTaskItemPageDataServices{
				AuthorizationService: services.AuthorizationService,
				TranslationService:   services.TranslationService,
			},
		},
		ListByPhase: &ListByPhaseUseCase{
			repositories: ListByPhaseRepositories{JobTask: repositories.JobTask},
			services: ListByPhaseServices{
				AuthorizationService: services.AuthorizationService,
				TranslationService:   services.TranslationService,
			},
		},
		ListByAssignee: &ListByAssigneeUseCase{
			repositories: ListByAssigneeRepositories{JobTask: repositories.JobTask},
			services: ListByAssigneeServices{
				AuthorizationService: services.AuthorizationService,
				TranslationService:   services.TranslationService,
			},
		},
	}
}

// ---- CreateJobTask ----

type CreateJobTaskRepositories struct{ JobTask pb.JobTaskDomainServiceServer }
type CreateJobTaskServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}
type CreateJobTaskUseCase struct {
	repositories CreateJobTaskRepositories
	services     CreateJobTaskServices
}

func (uc *CreateJobTaskUseCase) Execute(ctx context.Context, req *pb.CreateJobTaskRequest) (*pb.CreateJobTaskResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService, "job_task", ports.ActionCreate); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_task.validation.data_required", "job task data is required [DEFAULT]"))
	}
	if strings.TrimSpace(req.Data.Name) == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_task.validation.name_required", "job task name is required [DEFAULT]"))
	}
	if req.Data.JobPhaseId == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_task.validation.phase_id_required", "job phase ID is required [DEFAULT]"))
	}

	now := time.Now()
	if uc.services.IDService != nil {
		req.Data.Id = uc.services.IDService.GenerateID()
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

	if uc.services.TransactionService != nil {
		var result *pb.CreateJobTaskResponse
		err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
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
	AuthorizationService ports.AuthorizationService
	TranslationService   ports.TranslationService
}
type ReadJobTaskUseCase struct {
	repositories ReadJobTaskRepositories
	services     ReadJobTaskServices
}

func (uc *ReadJobTaskUseCase) Execute(ctx context.Context, req *pb.ReadJobTaskRequest) (*pb.ReadJobTaskResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService, "job_task", ports.ActionRead); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_task.validation.id_required", "job task ID is required"))
	}
	result, err := uc.repositories.JobTask.ReadJobTask(ctx, req)
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_task.errors.not_found", "job task not found [DEFAULT]"))
	}
	return result, nil
}

// ---- UpdateJobTask ----

type UpdateJobTaskRepositories struct{ JobTask pb.JobTaskDomainServiceServer }
type UpdateJobTaskServices struct {
	AuthorizationService ports.AuthorizationService
	TranslationService   ports.TranslationService
}
type UpdateJobTaskUseCase struct {
	repositories UpdateJobTaskRepositories
	services     UpdateJobTaskServices
}

func (uc *UpdateJobTaskUseCase) Execute(ctx context.Context, req *pb.UpdateJobTaskRequest) (*pb.UpdateJobTaskResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService, "job_task", ports.ActionUpdate); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_task.validation.id_required", "job task ID is required"))
	}
	if strings.TrimSpace(req.Data.Name) == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_task.validation.name_required", "job task name is required"))
	}
	now := time.Now()
	dm := now.UnixMilli()
	dms := now.Format(time.RFC3339)
	req.Data.DateModified = &dm
	req.Data.DateModifiedString = &dms

	_, err := uc.repositories.JobTask.UpdateJobTask(ctx, req)
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_task.errors.update_failed", "job task update failed [DEFAULT]"))
	}
	return &pb.UpdateJobTaskResponse{Success: true, Data: []*pb.JobTask{req.Data}}, nil
}

// ---- DeleteJobTask ----

type DeleteJobTaskRepositories struct{ JobTask pb.JobTaskDomainServiceServer }
type DeleteJobTaskServices struct {
	AuthorizationService ports.AuthorizationService
	TranslationService   ports.TranslationService
}
type DeleteJobTaskUseCase struct {
	repositories DeleteJobTaskRepositories
	services     DeleteJobTaskServices
}

func (uc *DeleteJobTaskUseCase) Execute(ctx context.Context, req *pb.DeleteJobTaskRequest) (*pb.DeleteJobTaskResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService, "job_task", ports.ActionDelete); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_task.validation.id_required", "job task ID is required"))
	}
	result, err := uc.repositories.JobTask.DeleteJobTask(ctx, req)
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_task.errors.deletion_failed", "job task deletion failed [DEFAULT]"))
	}
	return result, nil
}

// ---- ListJobTasks ----

type ListJobTasksRepositories struct{ JobTask pb.JobTaskDomainServiceServer }
type ListJobTasksServices struct {
	AuthorizationService ports.AuthorizationService
	TranslationService   ports.TranslationService
}
type ListJobTasksUseCase struct {
	repositories ListJobTasksRepositories
	services     ListJobTasksServices
}

func (uc *ListJobTasksUseCase) Execute(ctx context.Context, req *pb.ListJobTasksRequest) (*pb.ListJobTasksResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService, "job_task", ports.ActionList); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_task.validation.request_required", "request is required"))
	}
	result, err := uc.repositories.JobTask.ListJobTasks(ctx, req)
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_task.errors.list_failed", "job task listing failed [DEFAULT]"))
	}
	return result, nil
}

// ---- GetJobTaskListPageData ----

type GetJobTaskListPageDataRepositories struct{ JobTask pb.JobTaskDomainServiceServer }
type GetJobTaskListPageDataServices struct {
	AuthorizationService ports.AuthorizationService
	TranslationService   ports.TranslationService
}
type GetJobTaskListPageDataUseCase struct {
	repositories GetJobTaskListPageDataRepositories
	services     GetJobTaskListPageDataServices
}

func (uc *GetJobTaskListPageDataUseCase) Execute(ctx context.Context, req *pb.GetJobTaskListPageDataRequest) (*pb.GetJobTaskListPageDataResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService, "job_task", ports.ActionList); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_task.validation.request_required", "request is required"))
	}
	return uc.repositories.JobTask.GetJobTaskListPageData(ctx, req)
}

// ---- GetJobTaskItemPageData ----

type GetJobTaskItemPageDataRepositories struct{ JobTask pb.JobTaskDomainServiceServer }
type GetJobTaskItemPageDataServices struct {
	AuthorizationService ports.AuthorizationService
	TranslationService   ports.TranslationService
}
type GetJobTaskItemPageDataUseCase struct {
	repositories GetJobTaskItemPageDataRepositories
	services     GetJobTaskItemPageDataServices
}

func (uc *GetJobTaskItemPageDataUseCase) Execute(ctx context.Context, req *pb.GetJobTaskItemPageDataRequest) (*pb.GetJobTaskItemPageDataResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService, "job_task", ports.ActionRead); err != nil {
		return nil, err
	}
	if req == nil || req.JobTaskId == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_task.validation.id_required", "job task ID is required"))
	}
	return uc.repositories.JobTask.GetJobTaskItemPageData(ctx, req)
}

// ---- ListByPhase (extra RPC) ----

type ListByPhaseRepositories struct{ JobTask pb.JobTaskDomainServiceServer }
type ListByPhaseServices struct {
	AuthorizationService ports.AuthorizationService
	TranslationService   ports.TranslationService
}
type ListByPhaseUseCase struct {
	repositories ListByPhaseRepositories
	services     ListByPhaseServices
}

func (uc *ListByPhaseUseCase) Execute(ctx context.Context, req *pb.ListJobTasksByPhaseRequest) (*pb.ListJobTasksByPhaseResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService, "job_task", ports.ActionList); err != nil {
		return nil, err
	}
	if req == nil || req.JobPhaseId == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_task.validation.phase_id_required", "job phase ID is required"))
	}
	return uc.repositories.JobTask.ListByPhase(ctx, req)
}

// ---- ListByAssignee (extra RPC) ----

type ListByAssigneeRepositories struct{ JobTask pb.JobTaskDomainServiceServer }
type ListByAssigneeServices struct {
	AuthorizationService ports.AuthorizationService
	TranslationService   ports.TranslationService
}
type ListByAssigneeUseCase struct {
	repositories ListByAssigneeRepositories
	services     ListByAssigneeServices
}

func (uc *ListByAssigneeUseCase) Execute(ctx context.Context, req *pb.ListJobTasksByAssigneeRequest) (*pb.ListJobTasksByAssigneeResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService, "job_task", ports.ActionList); err != nil {
		return nil, err
	}
	if req == nil || req.AssignedTo == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_task.validation.assignee_required", "assignee ID is required"))
	}
	return uc.repositories.JobTask.ListByAssignee(ctx, req)
}
