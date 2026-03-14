package activity_material

import (
	"context"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/activity_material"
)

// ActivityMaterialRepositories groups all repository dependencies
type ActivityMaterialRepositories struct {
	ActivityMaterial pb.ActivityMaterialDomainServiceServer
}

// ActivityMaterialServices groups all business service dependencies
type ActivityMaterialServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// UseCases contains all activity material use cases
type UseCases struct {
	CreateActivityMaterial          *CreateActivityMaterialUseCase
	ReadActivityMaterial            *ReadActivityMaterialUseCase
	UpdateActivityMaterial          *UpdateActivityMaterialUseCase
	DeleteActivityMaterial          *DeleteActivityMaterialUseCase
	ListActivityMaterials           *ListActivityMaterialsUseCase
	GetActivityMaterialListPageData *GetActivityMaterialListPageDataUseCase
	GetActivityMaterialItemPageData *GetActivityMaterialItemPageDataUseCase
}

// NewUseCases creates a new collection of activity material use cases
func NewUseCases(
	repositories ActivityMaterialRepositories,
	services ActivityMaterialServices,
) *UseCases {
	return &UseCases{
		CreateActivityMaterial: &CreateActivityMaterialUseCase{
			Repo:    repositories.ActivityMaterial,
			AuthSvc: services.AuthorizationService,
			TxSvc:   services.TransactionService,
			I18nSvc: services.TranslationService,
			IDSvc:   services.IDService,
		},
		ReadActivityMaterial: &ReadActivityMaterialUseCase{
			Repo:    repositories.ActivityMaterial,
			AuthSvc: services.AuthorizationService,
		},
		UpdateActivityMaterial: &UpdateActivityMaterialUseCase{
			Repo:    repositories.ActivityMaterial,
			AuthSvc: services.AuthorizationService,
			TxSvc:   services.TransactionService,
			I18nSvc: services.TranslationService,
		},
		DeleteActivityMaterial: &DeleteActivityMaterialUseCase{
			Repo:    repositories.ActivityMaterial,
			AuthSvc: services.AuthorizationService,
			TxSvc:   services.TransactionService,
			I18nSvc: services.TranslationService,
		},
		ListActivityMaterials: &ListActivityMaterialsUseCase{
			Repo:    repositories.ActivityMaterial,
			AuthSvc: services.AuthorizationService,
		},
		GetActivityMaterialListPageData: &GetActivityMaterialListPageDataUseCase{
			Repo:    repositories.ActivityMaterial,
			AuthSvc: services.AuthorizationService,
		},
		GetActivityMaterialItemPageData: &GetActivityMaterialItemPageDataUseCase{
			Repo:    repositories.ActivityMaterial,
			AuthSvc: services.AuthorizationService,
		},
	}
}

// =============================================================================
// Create
// =============================================================================

// CreateActivityMaterialUseCase handles creating a new activity material record
type CreateActivityMaterialUseCase struct {
	Repo    pb.ActivityMaterialDomainServiceServer
	AuthSvc ports.AuthorizationService
	TxSvc   ports.TransactionService
	I18nSvc ports.TranslationService
	IDSvc   ports.IDService
}

// Execute creates a new activity material record
// Note: activity_id is the PK (1:1 with job_activity), not auto-generated
func (uc *CreateActivityMaterialUseCase) Execute(ctx context.Context, req *pb.CreateActivityMaterialRequest) (*pb.CreateActivityMaterialResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("activity material data is required")
	}
	if req.Data.ActivityId == "" {
		return nil, fmt.Errorf("activity_id is required (must match parent job_activity)")
	}
	return uc.Repo.CreateActivityMaterial(ctx, req)
}

// =============================================================================
// Read
// =============================================================================

// ReadActivityMaterialUseCase handles reading a single activity material record
type ReadActivityMaterialUseCase struct {
	Repo    pb.ActivityMaterialDomainServiceServer
	AuthSvc ports.AuthorizationService
}

// Execute reads an activity material by activity_id
func (uc *ReadActivityMaterialUseCase) Execute(ctx context.Context, req *pb.ReadActivityMaterialRequest) (*pb.ReadActivityMaterialResponse, error) {
	return uc.Repo.ReadActivityMaterial(ctx, req)
}

// =============================================================================
// Update
// =============================================================================

// UpdateActivityMaterialUseCase handles updating an activity material record
type UpdateActivityMaterialUseCase struct {
	Repo    pb.ActivityMaterialDomainServiceServer
	AuthSvc ports.AuthorizationService
	TxSvc   ports.TransactionService
	I18nSvc ports.TranslationService
}

// Execute updates an activity material record
func (uc *UpdateActivityMaterialUseCase) Execute(ctx context.Context, req *pb.UpdateActivityMaterialRequest) (*pb.UpdateActivityMaterialResponse, error) {
	if req.Data == nil || req.Data.ActivityId == "" {
		return nil, fmt.Errorf("activity_id is required")
	}
	return uc.Repo.UpdateActivityMaterial(ctx, req)
}

// =============================================================================
// Delete
// =============================================================================

// DeleteActivityMaterialUseCase handles deleting an activity material record
type DeleteActivityMaterialUseCase struct {
	Repo    pb.ActivityMaterialDomainServiceServer
	AuthSvc ports.AuthorizationService
	TxSvc   ports.TransactionService
	I18nSvc ports.TranslationService
}

// Execute deletes an activity material record
func (uc *DeleteActivityMaterialUseCase) Execute(ctx context.Context, req *pb.DeleteActivityMaterialRequest) (*pb.DeleteActivityMaterialResponse, error) {
	if req.Data == nil || req.Data.ActivityId == "" {
		return nil, fmt.Errorf("activity_id is required")
	}
	return uc.Repo.DeleteActivityMaterial(ctx, req)
}

// =============================================================================
// List
// =============================================================================

// ListActivityMaterialsUseCase handles listing activity material records
type ListActivityMaterialsUseCase struct {
	Repo    pb.ActivityMaterialDomainServiceServer
	AuthSvc ports.AuthorizationService
}

// Execute lists activity material records with optional filters
func (uc *ListActivityMaterialsUseCase) Execute(ctx context.Context, req *pb.ListActivityMaterialsRequest) (*pb.ListActivityMaterialsResponse, error) {
	return uc.Repo.ListActivityMaterials(ctx, req)
}

// =============================================================================
// GetActivityMaterialListPageData
// =============================================================================

// GetActivityMaterialListPageDataUseCase handles paginated list page data
type GetActivityMaterialListPageDataUseCase struct {
	Repo    pb.ActivityMaterialDomainServiceServer
	AuthSvc ports.AuthorizationService
}

// Execute retrieves paginated activity material list page data
func (uc *GetActivityMaterialListPageDataUseCase) Execute(ctx context.Context, req *pb.GetActivityMaterialListPageDataRequest) (*pb.GetActivityMaterialListPageDataResponse, error) {
	return uc.Repo.GetActivityMaterialListPageData(ctx, req)
}

// =============================================================================
// GetActivityMaterialItemPageData
// =============================================================================

// GetActivityMaterialItemPageDataUseCase handles single item page data
type GetActivityMaterialItemPageDataUseCase struct {
	Repo    pb.ActivityMaterialDomainServiceServer
	AuthSvc ports.AuthorizationService
}

// Execute retrieves a single activity material with all related data
func (uc *GetActivityMaterialItemPageDataUseCase) Execute(ctx context.Context, req *pb.GetActivityMaterialItemPageDataRequest) (*pb.GetActivityMaterialItemPageDataResponse, error) {
	return uc.Repo.GetActivityMaterialItemPageData(ctx, req)
}
