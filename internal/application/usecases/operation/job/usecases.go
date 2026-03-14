package job

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	enumspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/enums"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job"
)

// JobRepositories groups all repository dependencies
type JobRepositories struct {
	Job pb.JobDomainServiceServer
}

// JobServices groups all business service dependencies
type JobServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// UseCases contains all job-related use cases
type UseCases struct {
	CreateJob          *CreateJobUseCase
	ReadJob            *ReadJobUseCase
	UpdateJob          *UpdateJobUseCase
	DeleteJob          *DeleteJobUseCase
	ListJobs           *ListJobsUseCase
	GetJobListPageData *GetJobListPageDataUseCase
	GetJobItemPageData *GetJobItemPageDataUseCase
	GetJobsByClient    *GetJobsByClientUseCase
	GetJobsByOrigin    *GetJobsByOriginUseCase
	UpdateJobStatus    *UpdateJobStatusUseCase
}

// NewUseCases creates a new collection of job use cases
func NewUseCases(
	repositories JobRepositories,
	services JobServices,
) *UseCases {
	return &UseCases{
		CreateJob: &CreateJobUseCase{
			repositories: CreateJobRepositories{Job: repositories.Job},
			services: CreateJobServices{
				AuthorizationService: services.AuthorizationService,
				TransactionService:   services.TransactionService,
				TranslationService:   services.TranslationService,
				IDService:            services.IDService,
			},
		},
		ReadJob: &ReadJobUseCase{
			repositories: ReadJobRepositories{Job: repositories.Job},
			services: ReadJobServices{
				AuthorizationService: services.AuthorizationService,
				TranslationService:   services.TranslationService,
			},
		},
		UpdateJob: &UpdateJobUseCase{
			repositories: UpdateJobRepositories{Job: repositories.Job},
			services: UpdateJobServices{
				AuthorizationService: services.AuthorizationService,
				TranslationService:   services.TranslationService,
			},
		},
		DeleteJob: &DeleteJobUseCase{
			repositories: DeleteJobRepositories{Job: repositories.Job},
			services: DeleteJobServices{
				AuthorizationService: services.AuthorizationService,
				TranslationService:   services.TranslationService,
			},
		},
		ListJobs: &ListJobsUseCase{
			repositories: ListJobsRepositories{Job: repositories.Job},
			services: ListJobsServices{
				AuthorizationService: services.AuthorizationService,
				TranslationService:   services.TranslationService,
			},
		},
		GetJobListPageData: &GetJobListPageDataUseCase{
			repositories: GetJobListPageDataRepositories{Job: repositories.Job},
			services: GetJobListPageDataServices{
				AuthorizationService: services.AuthorizationService,
				TranslationService:   services.TranslationService,
			},
		},
		GetJobItemPageData: &GetJobItemPageDataUseCase{
			repositories: GetJobItemPageDataRepositories{Job: repositories.Job},
			services: GetJobItemPageDataServices{
				AuthorizationService: services.AuthorizationService,
				TranslationService:   services.TranslationService,
			},
		},
		GetJobsByClient: &GetJobsByClientUseCase{
			repositories: GetJobsByClientRepositories{Job: repositories.Job},
			services: GetJobsByClientServices{
				AuthorizationService: services.AuthorizationService,
				TranslationService:   services.TranslationService,
			},
		},
		GetJobsByOrigin: &GetJobsByOriginUseCase{
			repositories: GetJobsByOriginRepositories{Job: repositories.Job},
			services: GetJobsByOriginServices{
				AuthorizationService: services.AuthorizationService,
				TranslationService:   services.TranslationService,
			},
		},
		UpdateJobStatus: &UpdateJobStatusUseCase{
			repositories: UpdateJobStatusRepositories{Job: repositories.Job},
			services: UpdateJobStatusServices{
				AuthorizationService: services.AuthorizationService,
				TransactionService:   services.TransactionService,
				TranslationService:   services.TranslationService,
			},
		},
	}
}

// ---- CreateJob ----

type CreateJobRepositories struct{ Job pb.JobDomainServiceServer }
type CreateJobServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}
type CreateJobUseCase struct {
	repositories CreateJobRepositories
	services     CreateJobServices
}

func (uc *CreateJobUseCase) Execute(ctx context.Context, req *pb.CreateJobRequest) (*pb.CreateJobResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService, "job", ports.ActionCreate); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job.validation.data_required", "job data is required [DEFAULT]"))
	}
	if strings.TrimSpace(req.Data.Name) == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job.validation.name_required", "job name is required [DEFAULT]"))
	}

	now := time.Now()
	if uc.services.IDService != nil {
		req.Data.Id = uc.services.IDService.GenerateID()
	} else {
		req.Data.Id = fmt.Sprintf("job-%d", now.UnixNano())
	}
	dc := now.UnixMilli()
	dcs := now.Format(time.RFC3339)
	req.Data.DateCreated = &dc
	req.Data.DateCreatedString = &dcs
	req.Data.DateModified = &dc
	req.Data.DateModifiedString = &dcs
	req.Data.Active = true

	// Default status to DRAFT for new jobs
	if req.Data.Status == enumspb.JobStatus_JOB_STATUS_UNSPECIFIED {
		req.Data.Status = enumspb.JobStatus_JOB_STATUS_DRAFT
	}

	if uc.services.TransactionService != nil {
		var result *pb.CreateJobResponse
		err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
			res, err := uc.repositories.Job.CreateJob(txCtx, req)
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
	return uc.repositories.Job.CreateJob(ctx, req)
}

// ---- ReadJob ----

type ReadJobRepositories struct{ Job pb.JobDomainServiceServer }
type ReadJobServices struct {
	AuthorizationService ports.AuthorizationService
	TranslationService   ports.TranslationService
}
type ReadJobUseCase struct {
	repositories ReadJobRepositories
	services     ReadJobServices
}

func (uc *ReadJobUseCase) Execute(ctx context.Context, req *pb.ReadJobRequest) (*pb.ReadJobResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService, "job", ports.ActionRead); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job.validation.id_required", "job ID is required"))
	}
	result, err := uc.repositories.Job.ReadJob(ctx, req)
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job.errors.not_found", "job not found [DEFAULT]"))
	}
	return result, nil
}

// ---- UpdateJob ----

type UpdateJobRepositories struct{ Job pb.JobDomainServiceServer }
type UpdateJobServices struct {
	AuthorizationService ports.AuthorizationService
	TranslationService   ports.TranslationService
}
type UpdateJobUseCase struct {
	repositories UpdateJobRepositories
	services     UpdateJobServices
}

func (uc *UpdateJobUseCase) Execute(ctx context.Context, req *pb.UpdateJobRequest) (*pb.UpdateJobResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService, "job", ports.ActionUpdate); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job.validation.id_required", "job ID is required"))
	}
	if strings.TrimSpace(req.Data.Name) == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job.validation.name_required", "job name is required"))
	}
	now := time.Now()
	dm := now.UnixMilli()
	dms := now.Format(time.RFC3339)
	req.Data.DateModified = &dm
	req.Data.DateModifiedString = &dms

	_, err := uc.repositories.Job.UpdateJob(ctx, req)
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job.errors.update_failed", "job update failed [DEFAULT]"))
	}
	return &pb.UpdateJobResponse{Success: true, Data: []*pb.Job{req.Data}}, nil
}

// ---- DeleteJob ----

type DeleteJobRepositories struct{ Job pb.JobDomainServiceServer }
type DeleteJobServices struct {
	AuthorizationService ports.AuthorizationService
	TranslationService   ports.TranslationService
}
type DeleteJobUseCase struct {
	repositories DeleteJobRepositories
	services     DeleteJobServices
}

func (uc *DeleteJobUseCase) Execute(ctx context.Context, req *pb.DeleteJobRequest) (*pb.DeleteJobResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService, "job", ports.ActionDelete); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job.validation.id_required", "job ID is required"))
	}
	result, err := uc.repositories.Job.DeleteJob(ctx, req)
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job.errors.deletion_failed", "job deletion failed [DEFAULT]"))
	}
	return result, nil
}

// ---- ListJobs ----

type ListJobsRepositories struct{ Job pb.JobDomainServiceServer }
type ListJobsServices struct {
	AuthorizationService ports.AuthorizationService
	TranslationService   ports.TranslationService
}
type ListJobsUseCase struct {
	repositories ListJobsRepositories
	services     ListJobsServices
}

func (uc *ListJobsUseCase) Execute(ctx context.Context, req *pb.ListJobsRequest) (*pb.ListJobsResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService, "job", ports.ActionList); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job.validation.request_required", "request is required"))
	}
	result, err := uc.repositories.Job.ListJobs(ctx, req)
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job.errors.list_failed", "job listing failed [DEFAULT]"))
	}
	return result, nil
}

// ---- GetJobListPageData ----

type GetJobListPageDataRepositories struct{ Job pb.JobDomainServiceServer }
type GetJobListPageDataServices struct {
	AuthorizationService ports.AuthorizationService
	TranslationService   ports.TranslationService
}
type GetJobListPageDataUseCase struct {
	repositories GetJobListPageDataRepositories
	services     GetJobListPageDataServices
}

func (uc *GetJobListPageDataUseCase) Execute(ctx context.Context, req *pb.GetJobListPageDataRequest) (*pb.GetJobListPageDataResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService, "job", ports.ActionList); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job.validation.request_required", "request is required"))
	}
	return uc.repositories.Job.GetJobListPageData(ctx, req)
}

// ---- GetJobItemPageData ----

type GetJobItemPageDataRepositories struct{ Job pb.JobDomainServiceServer }
type GetJobItemPageDataServices struct {
	AuthorizationService ports.AuthorizationService
	TranslationService   ports.TranslationService
}
type GetJobItemPageDataUseCase struct {
	repositories GetJobItemPageDataRepositories
	services     GetJobItemPageDataServices
}

func (uc *GetJobItemPageDataUseCase) Execute(ctx context.Context, req *pb.GetJobItemPageDataRequest) (*pb.GetJobItemPageDataResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService, "job", ports.ActionRead); err != nil {
		return nil, err
	}
	if req == nil || req.JobId == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job.validation.id_required", "job ID is required"))
	}
	return uc.repositories.Job.GetJobItemPageData(ctx, req)
}

// ---- GetJobsByClient (extra RPC) ----

type GetJobsByClientRepositories struct{ Job pb.JobDomainServiceServer }
type GetJobsByClientServices struct {
	AuthorizationService ports.AuthorizationService
	TranslationService   ports.TranslationService
}
type GetJobsByClientUseCase struct {
	repositories GetJobsByClientRepositories
	services     GetJobsByClientServices
}

func (uc *GetJobsByClientUseCase) Execute(ctx context.Context, req *pb.GetJobsByClientRequest) (*pb.GetJobsByClientResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService, "job", ports.ActionList); err != nil {
		return nil, err
	}
	if req == nil || req.ClientId == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job.validation.client_id_required", "client ID is required"))
	}
	return uc.repositories.Job.GetJobsByClient(ctx, req)
}

// ---- GetJobsByOrigin (extra RPC) ----

type GetJobsByOriginRepositories struct{ Job pb.JobDomainServiceServer }
type GetJobsByOriginServices struct {
	AuthorizationService ports.AuthorizationService
	TranslationService   ports.TranslationService
}
type GetJobsByOriginUseCase struct {
	repositories GetJobsByOriginRepositories
	services     GetJobsByOriginServices
}

func (uc *GetJobsByOriginUseCase) Execute(ctx context.Context, req *pb.GetJobsByOriginRequest) (*pb.GetJobsByOriginResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService, "job", ports.ActionList); err != nil {
		return nil, err
	}
	if req == nil || req.OriginId == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job.validation.origin_id_required", "origin ID is required"))
	}
	return uc.repositories.Job.GetJobsByOrigin(ctx, req)
}

// ---- UpdateJobStatus (extra RPC — state machine) ----

type UpdateJobStatusRepositories struct{ Job pb.JobDomainServiceServer }
type UpdateJobStatusServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}
type UpdateJobStatusUseCase struct {
	repositories UpdateJobStatusRepositories
	services     UpdateJobStatusServices
}

// validTransitions defines the allowed state machine transitions for jobs:
//
//	DRAFT     -> PENDING, CLOSED
//	PENDING   -> ACTIVE, CLOSED
//	ACTIVE    -> PAUSED, COMPLETED
//	PAUSED    -> ACTIVE, CLOSED
//	COMPLETED -> CLOSED
//	CLOSED    -> (terminal)
var validTransitions = map[enumspb.JobStatus][]enumspb.JobStatus{
	enumspb.JobStatus_JOB_STATUS_DRAFT:     {enumspb.JobStatus_JOB_STATUS_PENDING, enumspb.JobStatus_JOB_STATUS_CLOSED},
	enumspb.JobStatus_JOB_STATUS_PENDING:   {enumspb.JobStatus_JOB_STATUS_ACTIVE, enumspb.JobStatus_JOB_STATUS_CLOSED},
	enumspb.JobStatus_JOB_STATUS_ACTIVE:    {enumspb.JobStatus_JOB_STATUS_PAUSED, enumspb.JobStatus_JOB_STATUS_COMPLETED},
	enumspb.JobStatus_JOB_STATUS_PAUSED:    {enumspb.JobStatus_JOB_STATUS_ACTIVE, enumspb.JobStatus_JOB_STATUS_CLOSED},
	enumspb.JobStatus_JOB_STATUS_COMPLETED: {enumspb.JobStatus_JOB_STATUS_CLOSED},
	// CLOSED is terminal — no valid transitions
}

func (uc *UpdateJobStatusUseCase) Execute(ctx context.Context, req *pb.UpdateJobStatusRequest) (*pb.UpdateJobStatusResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService, "job", ports.ActionUpdate); err != nil {
		return nil, err
	}
	if req == nil || req.JobId == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job.validation.id_required", "job ID is required"))
	}
	if req.Status == enumspb.JobStatus_JOB_STATUS_UNSPECIFIED {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job.validation.status_required", "target status is required"))
	}

	// Read current job to validate transition
	currentJob, err := uc.repositories.Job.ReadJob(ctx, &pb.ReadJobRequest{Data: &pb.Job{Id: req.JobId}})
	if err != nil || currentJob == nil || len(currentJob.Data) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job.errors.not_found", "job not found"))
	}

	currentStatus := currentJob.Data[0].Status

	// Validate state machine transition
	allowed, ok := validTransitions[currentStatus]
	if !ok {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"job.errors.terminal_status", "cannot transition from terminal status %s"), currentStatus.String())
	}

	transitionValid := false
	for _, s := range allowed {
		if s == req.Status {
			transitionValid = true
			break
		}
	}
	if !transitionValid {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"job.errors.invalid_transition", "cannot transition from %s to %s"), currentStatus.String(), req.Status.String())
	}

	// Execute status update (within transaction if available)
	if uc.services.TransactionService != nil {
		var result *pb.UpdateJobStatusResponse
		txErr := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
			res, err := uc.repositories.Job.UpdateJobStatus(txCtx, req)
			if err != nil {
				return err
			}
			result = res
			return nil
		})
		if txErr != nil {
			return nil, txErr
		}
		return result, nil
	}
	return uc.repositories.Job.UpdateJobStatus(ctx, req)
}
