package activity_labor

import (
	"context"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/activity_labor"
)

// ActivityLaborRepositories groups all repository dependencies
type ActivityLaborRepositories struct {
	ActivityLabor pb.ActivityLaborDomainServiceServer
}

// ActivityLaborServices groups all business service dependencies
type ActivityLaborServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// UseCases contains all activity labor use cases
type UseCases struct {
	CreateActivityLabor          *CreateActivityLaborUseCase
	ReadActivityLabor            *ReadActivityLaborUseCase
	UpdateActivityLabor          *UpdateActivityLaborUseCase
	DeleteActivityLabor          *DeleteActivityLaborUseCase
	ListActivityLabors           *ListActivityLaborsUseCase
	GetActivityLaborListPageData *GetActivityLaborListPageDataUseCase
	GetActivityLaborItemPageData *GetActivityLaborItemPageDataUseCase
	ListByStaff                  *ListByStaffUseCase
	ListByJob                    *ListByJobUseCase
}

// NewUseCases creates a new collection of activity labor use cases
func NewUseCases(
	repositories ActivityLaborRepositories,
	services ActivityLaborServices,
) *UseCases {
	return &UseCases{
		CreateActivityLabor: &CreateActivityLaborUseCase{
			Repo:    repositories.ActivityLabor,
			AuthSvc: services.AuthorizationService,
			TxSvc:   services.TransactionService,
			I18nSvc: services.TranslationService,
			IDSvc:   services.IDService,
		},
		ReadActivityLabor: &ReadActivityLaborUseCase{
			Repo:    repositories.ActivityLabor,
			AuthSvc: services.AuthorizationService,
		},
		UpdateActivityLabor: &UpdateActivityLaborUseCase{
			Repo:    repositories.ActivityLabor,
			AuthSvc: services.AuthorizationService,
			TxSvc:   services.TransactionService,
			I18nSvc: services.TranslationService,
		},
		DeleteActivityLabor: &DeleteActivityLaborUseCase{
			Repo:    repositories.ActivityLabor,
			AuthSvc: services.AuthorizationService,
			TxSvc:   services.TransactionService,
			I18nSvc: services.TranslationService,
		},
		ListActivityLabors: &ListActivityLaborsUseCase{
			Repo:    repositories.ActivityLabor,
			AuthSvc: services.AuthorizationService,
		},
		GetActivityLaborListPageData: &GetActivityLaborListPageDataUseCase{
			Repo:    repositories.ActivityLabor,
			AuthSvc: services.AuthorizationService,
		},
		GetActivityLaborItemPageData: &GetActivityLaborItemPageDataUseCase{
			Repo:    repositories.ActivityLabor,
			AuthSvc: services.AuthorizationService,
		},
		ListByStaff: &ListByStaffUseCase{
			Repo:    repositories.ActivityLabor,
			AuthSvc: services.AuthorizationService,
		},
		ListByJob: &ListByJobUseCase{
			Repo:    repositories.ActivityLabor,
			AuthSvc: services.AuthorizationService,
		},
	}
}

// =============================================================================
// Create
// =============================================================================

// CreateActivityLaborUseCase handles creating a new activity labor record
type CreateActivityLaborUseCase struct {
	Repo    pb.ActivityLaborDomainServiceServer
	AuthSvc ports.AuthorizationService
	TxSvc   ports.TransactionService
	I18nSvc ports.TranslationService
	IDSvc   ports.IDService
}

// Execute creates a new activity labor record
// Note: activity_id is the PK (1:1 with job_activity), not auto-generated
func (uc *CreateActivityLaborUseCase) Execute(ctx context.Context, req *pb.CreateActivityLaborRequest) (*pb.CreateActivityLaborResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("activity labor data is required")
	}
	if req.Data.ActivityId == "" {
		return nil, fmt.Errorf("activity_id is required (must match parent job_activity)")
	}
	return uc.Repo.CreateActivityLabor(ctx, req)
}

// =============================================================================
// Read
// =============================================================================

// ReadActivityLaborUseCase handles reading a single activity labor record
type ReadActivityLaborUseCase struct {
	Repo    pb.ActivityLaborDomainServiceServer
	AuthSvc ports.AuthorizationService
}

// Execute reads an activity labor by activity_id
func (uc *ReadActivityLaborUseCase) Execute(ctx context.Context, req *pb.ReadActivityLaborRequest) (*pb.ReadActivityLaborResponse, error) {
	return uc.Repo.ReadActivityLabor(ctx, req)
}

// =============================================================================
// Update
// =============================================================================

// UpdateActivityLaborUseCase handles updating an activity labor record
type UpdateActivityLaborUseCase struct {
	Repo    pb.ActivityLaborDomainServiceServer
	AuthSvc ports.AuthorizationService
	TxSvc   ports.TransactionService
	I18nSvc ports.TranslationService
}

// Execute updates an activity labor record
func (uc *UpdateActivityLaborUseCase) Execute(ctx context.Context, req *pb.UpdateActivityLaborRequest) (*pb.UpdateActivityLaborResponse, error) {
	if req.Data == nil || req.Data.ActivityId == "" {
		return nil, fmt.Errorf("activity_id is required")
	}
	return uc.Repo.UpdateActivityLabor(ctx, req)
}

// =============================================================================
// Delete
// =============================================================================

// DeleteActivityLaborUseCase handles deleting an activity labor record
type DeleteActivityLaborUseCase struct {
	Repo    pb.ActivityLaborDomainServiceServer
	AuthSvc ports.AuthorizationService
	TxSvc   ports.TransactionService
	I18nSvc ports.TranslationService
}

// Execute deletes an activity labor record
func (uc *DeleteActivityLaborUseCase) Execute(ctx context.Context, req *pb.DeleteActivityLaborRequest) (*pb.DeleteActivityLaborResponse, error) {
	if req.Data == nil || req.Data.ActivityId == "" {
		return nil, fmt.Errorf("activity_id is required")
	}
	return uc.Repo.DeleteActivityLabor(ctx, req)
}

// =============================================================================
// List
// =============================================================================

// ListActivityLaborsUseCase handles listing activity labor records
type ListActivityLaborsUseCase struct {
	Repo    pb.ActivityLaborDomainServiceServer
	AuthSvc ports.AuthorizationService
}

// Execute lists activity labor records with optional filters
func (uc *ListActivityLaborsUseCase) Execute(ctx context.Context, req *pb.ListActivityLaborsRequest) (*pb.ListActivityLaborsResponse, error) {
	return uc.Repo.ListActivityLabors(ctx, req)
}

// =============================================================================
// GetActivityLaborListPageData
// =============================================================================

// GetActivityLaborListPageDataUseCase handles paginated list page data
type GetActivityLaborListPageDataUseCase struct {
	Repo    pb.ActivityLaborDomainServiceServer
	AuthSvc ports.AuthorizationService
}

// Execute retrieves paginated activity labor list page data
func (uc *GetActivityLaborListPageDataUseCase) Execute(ctx context.Context, req *pb.GetActivityLaborListPageDataRequest) (*pb.GetActivityLaborListPageDataResponse, error) {
	return uc.Repo.GetActivityLaborListPageData(ctx, req)
}

// =============================================================================
// GetActivityLaborItemPageData
// =============================================================================

// GetActivityLaborItemPageDataUseCase handles single item page data
type GetActivityLaborItemPageDataUseCase struct {
	Repo    pb.ActivityLaborDomainServiceServer
	AuthSvc ports.AuthorizationService
}

// Execute retrieves a single activity labor with all related data
func (uc *GetActivityLaborItemPageDataUseCase) Execute(ctx context.Context, req *pb.GetActivityLaborItemPageDataRequest) (*pb.GetActivityLaborItemPageDataResponse, error) {
	return uc.Repo.GetActivityLaborItemPageData(ctx, req)
}

// =============================================================================
// ListByStaff
// =============================================================================

// ListByStaffUseCase handles listing labor records for a specific staff member
type ListByStaffUseCase struct {
	Repo    pb.ActivityLaborDomainServiceServer
	AuthSvc ports.AuthorizationService
}

// Execute lists all labor records for a given staff member
func (uc *ListByStaffUseCase) Execute(ctx context.Context, req *pb.ListActivityLaborsByStaffRequest) (*pb.ListActivityLaborsByStaffResponse, error) {
	if req.StaffId == "" {
		return nil, fmt.Errorf("staff ID is required")
	}
	return uc.Repo.ListByStaff(ctx, req)
}

// =============================================================================
// ListByJob
// =============================================================================

// ListByJobUseCase handles listing labor records for a specific job (via join)
type ListByJobUseCase struct {
	Repo    pb.ActivityLaborDomainServiceServer
	AuthSvc ports.AuthorizationService
}

// Execute lists all labor records for a given job (joins through job_activity)
func (uc *ListByJobUseCase) Execute(ctx context.Context, req *pb.ListActivityLaborsByJobRequest) (*pb.ListActivityLaborsByJobResponse, error) {
	if req.JobId == "" {
		return nil, fmt.Errorf("job ID is required")
	}
	return uc.Repo.ListByJob(ctx, req)
}
