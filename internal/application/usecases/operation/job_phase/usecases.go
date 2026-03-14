package job_phase

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_phase"
)

// JobPhaseRepositories groups all repository dependencies
type JobPhaseRepositories struct {
	JobPhase pb.JobPhaseDomainServiceServer
}

// JobPhaseServices groups all business service dependencies
type JobPhaseServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// UseCases contains all job-phase-related use cases
type UseCases struct {
	CreateJobPhase          *CreateJobPhaseUseCase
	ReadJobPhase            *ReadJobPhaseUseCase
	UpdateJobPhase          *UpdateJobPhaseUseCase
	DeleteJobPhase          *DeleteJobPhaseUseCase
	ListJobPhases           *ListJobPhasesUseCase
	GetJobPhaseListPageData *GetJobPhaseListPageDataUseCase
	GetJobPhaseItemPageData *GetJobPhaseItemPageDataUseCase
	ListByJob               *ListByJobUseCase
}

// NewUseCases creates a new collection of job phase use cases
func NewUseCases(
	repositories JobPhaseRepositories,
	services JobPhaseServices,
) *UseCases {
	return &UseCases{
		CreateJobPhase: &CreateJobPhaseUseCase{
			repositories: CreateJobPhaseRepositories{JobPhase: repositories.JobPhase},
			services: CreateJobPhaseServices{
				AuthorizationService: services.AuthorizationService,
				TransactionService:   services.TransactionService,
				TranslationService:   services.TranslationService,
				IDService:            services.IDService,
			},
		},
		ReadJobPhase: &ReadJobPhaseUseCase{
			repositories: ReadJobPhaseRepositories{JobPhase: repositories.JobPhase},
			services: ReadJobPhaseServices{
				AuthorizationService: services.AuthorizationService,
				TransactionService:   services.TransactionService,
				TranslationService:   services.TranslationService,
			},
		},
		UpdateJobPhase: &UpdateJobPhaseUseCase{
			repositories: UpdateJobPhaseRepositories{JobPhase: repositories.JobPhase},
			services: UpdateJobPhaseServices{
				AuthorizationService: services.AuthorizationService,
				TransactionService:   services.TransactionService,
				TranslationService:   services.TranslationService,
			},
		},
		DeleteJobPhase: &DeleteJobPhaseUseCase{
			repositories: DeleteJobPhaseRepositories{JobPhase: repositories.JobPhase},
			services: DeleteJobPhaseServices{
				AuthorizationService: services.AuthorizationService,
				TransactionService:   services.TransactionService,
				TranslationService:   services.TranslationService,
			},
		},
		ListJobPhases: &ListJobPhasesUseCase{
			repositories: ListJobPhasesRepositories{JobPhase: repositories.JobPhase},
			services: ListJobPhasesServices{
				AuthorizationService: services.AuthorizationService,
				TransactionService:   services.TransactionService,
				TranslationService:   services.TranslationService,
			},
		},
		GetJobPhaseListPageData: &GetJobPhaseListPageDataUseCase{
			repositories: GetJobPhaseListPageDataRepositories{JobPhase: repositories.JobPhase},
			services: GetJobPhaseListPageDataServices{
				AuthorizationService: services.AuthorizationService,
				TransactionService:   services.TransactionService,
				TranslationService:   services.TranslationService,
			},
		},
		GetJobPhaseItemPageData: &GetJobPhaseItemPageDataUseCase{
			repositories: GetJobPhaseItemPageDataRepositories{JobPhase: repositories.JobPhase},
			services: GetJobPhaseItemPageDataServices{
				AuthorizationService: services.AuthorizationService,
				TransactionService:   services.TransactionService,
				TranslationService:   services.TranslationService,
			},
		},
		ListByJob: &ListByJobUseCase{
			repositories: ListByJobRepositories{JobPhase: repositories.JobPhase},
			services: ListByJobServices{
				AuthorizationService: services.AuthorizationService,
				TranslationService:   services.TranslationService,
			},
		},
	}
}

// ---- CreateJobPhase ----

type CreateJobPhaseRepositories struct{ JobPhase pb.JobPhaseDomainServiceServer }
type CreateJobPhaseServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}
type CreateJobPhaseUseCase struct {
	repositories CreateJobPhaseRepositories
	services     CreateJobPhaseServices
}

func (uc *CreateJobPhaseUseCase) Execute(ctx context.Context, req *pb.CreateJobPhaseRequest) (*pb.CreateJobPhaseResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService, "job_phase", ports.ActionCreate); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_phase.validation.data_required", "job phase data is required [DEFAULT]"))
	}
	if strings.TrimSpace(req.Data.Name) == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_phase.validation.name_required", "job phase name is required [DEFAULT]"))
	}
	if req.Data.JobId == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_phase.validation.job_id_required", "job ID is required [DEFAULT]"))
	}

	now := time.Now()
	if uc.services.IDService != nil {
		req.Data.Id = uc.services.IDService.GenerateID()
	} else {
		req.Data.Id = fmt.Sprintf("job_phase-%d", now.UnixNano())
	}
	dc := now.UnixMilli()
	dcs := now.Format(time.RFC3339)
	req.Data.DateCreated = &dc
	req.Data.DateCreatedString = &dcs
	req.Data.DateModified = &dc
	req.Data.DateModifiedString = &dcs
	req.Data.Active = true

	if uc.services.TransactionService != nil {
		var result *pb.CreateJobPhaseResponse
		err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
			res, err := uc.repositories.JobPhase.CreateJobPhase(txCtx, req)
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
	return uc.repositories.JobPhase.CreateJobPhase(ctx, req)
}

// ---- ReadJobPhase ----

type ReadJobPhaseRepositories struct{ JobPhase pb.JobPhaseDomainServiceServer }
type ReadJobPhaseServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}
type ReadJobPhaseUseCase struct {
	repositories ReadJobPhaseRepositories
	services     ReadJobPhaseServices
}

func (uc *ReadJobPhaseUseCase) Execute(ctx context.Context, req *pb.ReadJobPhaseRequest) (*pb.ReadJobPhaseResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService, "job_phase", ports.ActionRead); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_phase.validation.id_required", "job phase ID is required"))
	}
	result, err := uc.repositories.JobPhase.ReadJobPhase(ctx, req)
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_phase.errors.not_found", "job phase not found [DEFAULT]"))
	}
	return result, nil
}

// ---- UpdateJobPhase ----

type UpdateJobPhaseRepositories struct{ JobPhase pb.JobPhaseDomainServiceServer }
type UpdateJobPhaseServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}
type UpdateJobPhaseUseCase struct {
	repositories UpdateJobPhaseRepositories
	services     UpdateJobPhaseServices
}

func (uc *UpdateJobPhaseUseCase) Execute(ctx context.Context, req *pb.UpdateJobPhaseRequest) (*pb.UpdateJobPhaseResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService, "job_phase", ports.ActionUpdate); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_phase.validation.id_required", "job phase ID is required"))
	}
	if strings.TrimSpace(req.Data.Name) == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_phase.validation.name_required", "job phase name is required"))
	}
	now := time.Now()
	dm := now.UnixMilli()
	dms := now.Format(time.RFC3339)
	req.Data.DateModified = &dm
	req.Data.DateModifiedString = &dms

	_, err := uc.repositories.JobPhase.UpdateJobPhase(ctx, req)
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_phase.errors.update_failed", "job phase update failed [DEFAULT]"))
	}
	return &pb.UpdateJobPhaseResponse{Success: true, Data: []*pb.JobPhase{req.Data}}, nil
}

// ---- DeleteJobPhase ----

type DeleteJobPhaseRepositories struct{ JobPhase pb.JobPhaseDomainServiceServer }
type DeleteJobPhaseServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}
type DeleteJobPhaseUseCase struct {
	repositories DeleteJobPhaseRepositories
	services     DeleteJobPhaseServices
}

func (uc *DeleteJobPhaseUseCase) Execute(ctx context.Context, req *pb.DeleteJobPhaseRequest) (*pb.DeleteJobPhaseResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService, "job_phase", ports.ActionDelete); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_phase.validation.id_required", "job phase ID is required"))
	}
	result, err := uc.repositories.JobPhase.DeleteJobPhase(ctx, req)
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_phase.errors.deletion_failed", "job phase deletion failed [DEFAULT]"))
	}
	return result, nil
}

// ---- ListJobPhases ----

type ListJobPhasesRepositories struct{ JobPhase pb.JobPhaseDomainServiceServer }
type ListJobPhasesServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}
type ListJobPhasesUseCase struct {
	repositories ListJobPhasesRepositories
	services     ListJobPhasesServices
}

func (uc *ListJobPhasesUseCase) Execute(ctx context.Context, req *pb.ListJobPhasesRequest) (*pb.ListJobPhasesResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService, "job_phase", ports.ActionList); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_phase.validation.request_required", "request is required"))
	}
	result, err := uc.repositories.JobPhase.ListJobPhases(ctx, req)
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_phase.errors.list_failed", "job phase listing failed [DEFAULT]"))
	}
	return result, nil
}

// ---- GetJobPhaseListPageData ----

type GetJobPhaseListPageDataRepositories struct{ JobPhase pb.JobPhaseDomainServiceServer }
type GetJobPhaseListPageDataServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}
type GetJobPhaseListPageDataUseCase struct {
	repositories GetJobPhaseListPageDataRepositories
	services     GetJobPhaseListPageDataServices
}

func (uc *GetJobPhaseListPageDataUseCase) Execute(ctx context.Context, req *pb.GetJobPhaseListPageDataRequest) (*pb.GetJobPhaseListPageDataResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService, "job_phase", ports.ActionList); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_phase.validation.request_required", "request is required"))
	}
	return uc.repositories.JobPhase.GetJobPhaseListPageData(ctx, req)
}

// ---- GetJobPhaseItemPageData ----

type GetJobPhaseItemPageDataRepositories struct{ JobPhase pb.JobPhaseDomainServiceServer }
type GetJobPhaseItemPageDataServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}
type GetJobPhaseItemPageDataUseCase struct {
	repositories GetJobPhaseItemPageDataRepositories
	services     GetJobPhaseItemPageDataServices
}

func (uc *GetJobPhaseItemPageDataUseCase) Execute(ctx context.Context, req *pb.GetJobPhaseItemPageDataRequest) (*pb.GetJobPhaseItemPageDataResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService, "job_phase", ports.ActionRead); err != nil {
		return nil, err
	}
	if req == nil || req.JobPhaseId == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_phase.validation.id_required", "job phase ID is required"))
	}
	return uc.repositories.JobPhase.GetJobPhaseItemPageData(ctx, req)
}

// ---- ListByJob (extra RPC) ----

type ListByJobRepositories struct{ JobPhase pb.JobPhaseDomainServiceServer }
type ListByJobServices struct {
	AuthorizationService ports.AuthorizationService
	TranslationService   ports.TranslationService
}
type ListByJobUseCase struct {
	repositories ListByJobRepositories
	services     ListByJobServices
}

func (uc *ListByJobUseCase) Execute(ctx context.Context, req *pb.ListJobPhasesByJobRequest) (*pb.ListJobPhasesByJobResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService, "job_phase", ports.ActionList); err != nil {
		return nil, err
	}
	if req == nil || req.JobId == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_phase.validation.job_id_required", "job ID is required"))
	}
	return uc.repositories.JobPhase.ListByJob(ctx, req)
}
