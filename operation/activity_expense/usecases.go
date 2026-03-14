package activity_expense

import (
	"context"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/activity_expense"
)

// ActivityExpenseRepositories groups all repository dependencies
type ActivityExpenseRepositories struct {
	ActivityExpense pb.ActivityExpenseDomainServiceServer
}

// ActivityExpenseServices groups all business service dependencies
type ActivityExpenseServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// UseCases contains all activity expense use cases
type UseCases struct {
	CreateActivityExpense          *CreateActivityExpenseUseCase
	ReadActivityExpense            *ReadActivityExpenseUseCase
	UpdateActivityExpense          *UpdateActivityExpenseUseCase
	DeleteActivityExpense          *DeleteActivityExpenseUseCase
	ListActivityExpenses           *ListActivityExpensesUseCase
	GetActivityExpenseListPageData *GetActivityExpenseListPageDataUseCase
	GetActivityExpenseItemPageData *GetActivityExpenseItemPageDataUseCase
}

// NewUseCases creates a new collection of activity expense use cases
func NewUseCases(
	repositories ActivityExpenseRepositories,
	services ActivityExpenseServices,
) *UseCases {
	return &UseCases{
		CreateActivityExpense: &CreateActivityExpenseUseCase{
			Repo:    repositories.ActivityExpense,
			AuthSvc: services.AuthorizationService,
			TxSvc:   services.TransactionService,
			I18nSvc: services.TranslationService,
			IDSvc:   services.IDService,
		},
		ReadActivityExpense: &ReadActivityExpenseUseCase{
			Repo:    repositories.ActivityExpense,
			AuthSvc: services.AuthorizationService,
		},
		UpdateActivityExpense: &UpdateActivityExpenseUseCase{
			Repo:    repositories.ActivityExpense,
			AuthSvc: services.AuthorizationService,
			TxSvc:   services.TransactionService,
			I18nSvc: services.TranslationService,
		},
		DeleteActivityExpense: &DeleteActivityExpenseUseCase{
			Repo:    repositories.ActivityExpense,
			AuthSvc: services.AuthorizationService,
			TxSvc:   services.TransactionService,
			I18nSvc: services.TranslationService,
		},
		ListActivityExpenses: &ListActivityExpensesUseCase{
			Repo:    repositories.ActivityExpense,
			AuthSvc: services.AuthorizationService,
		},
		GetActivityExpenseListPageData: &GetActivityExpenseListPageDataUseCase{
			Repo:    repositories.ActivityExpense,
			AuthSvc: services.AuthorizationService,
		},
		GetActivityExpenseItemPageData: &GetActivityExpenseItemPageDataUseCase{
			Repo:    repositories.ActivityExpense,
			AuthSvc: services.AuthorizationService,
		},
	}
}

// =============================================================================
// Create
// =============================================================================

// CreateActivityExpenseUseCase handles creating a new activity expense record
type CreateActivityExpenseUseCase struct {
	Repo    pb.ActivityExpenseDomainServiceServer
	AuthSvc ports.AuthorizationService
	TxSvc   ports.TransactionService
	I18nSvc ports.TranslationService
	IDSvc   ports.IDService
}

// Execute creates a new activity expense record
// Note: activity_id is the PK (1:1 with job_activity), not auto-generated
func (uc *CreateActivityExpenseUseCase) Execute(ctx context.Context, req *pb.CreateActivityExpenseRequest) (*pb.CreateActivityExpenseResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("activity expense data is required")
	}
	if req.Data.ActivityId == "" {
		return nil, fmt.Errorf("activity_id is required (must match parent job_activity)")
	}
	return uc.Repo.CreateActivityExpense(ctx, req)
}

// =============================================================================
// Read
// =============================================================================

// ReadActivityExpenseUseCase handles reading a single activity expense record
type ReadActivityExpenseUseCase struct {
	Repo    pb.ActivityExpenseDomainServiceServer
	AuthSvc ports.AuthorizationService
}

// Execute reads an activity expense by activity_id
func (uc *ReadActivityExpenseUseCase) Execute(ctx context.Context, req *pb.ReadActivityExpenseRequest) (*pb.ReadActivityExpenseResponse, error) {
	return uc.Repo.ReadActivityExpense(ctx, req)
}

// =============================================================================
// Update
// =============================================================================

// UpdateActivityExpenseUseCase handles updating an activity expense record
type UpdateActivityExpenseUseCase struct {
	Repo    pb.ActivityExpenseDomainServiceServer
	AuthSvc ports.AuthorizationService
	TxSvc   ports.TransactionService
	I18nSvc ports.TranslationService
}

// Execute updates an activity expense record
func (uc *UpdateActivityExpenseUseCase) Execute(ctx context.Context, req *pb.UpdateActivityExpenseRequest) (*pb.UpdateActivityExpenseResponse, error) {
	if req.Data == nil || req.Data.ActivityId == "" {
		return nil, fmt.Errorf("activity_id is required")
	}
	return uc.Repo.UpdateActivityExpense(ctx, req)
}

// =============================================================================
// Delete
// =============================================================================

// DeleteActivityExpenseUseCase handles deleting an activity expense record
type DeleteActivityExpenseUseCase struct {
	Repo    pb.ActivityExpenseDomainServiceServer
	AuthSvc ports.AuthorizationService
	TxSvc   ports.TransactionService
	I18nSvc ports.TranslationService
}

// Execute deletes an activity expense record
func (uc *DeleteActivityExpenseUseCase) Execute(ctx context.Context, req *pb.DeleteActivityExpenseRequest) (*pb.DeleteActivityExpenseResponse, error) {
	if req.Data == nil || req.Data.ActivityId == "" {
		return nil, fmt.Errorf("activity_id is required")
	}
	return uc.Repo.DeleteActivityExpense(ctx, req)
}

// =============================================================================
// List
// =============================================================================

// ListActivityExpensesUseCase handles listing activity expense records
type ListActivityExpensesUseCase struct {
	Repo    pb.ActivityExpenseDomainServiceServer
	AuthSvc ports.AuthorizationService
}

// Execute lists activity expense records with optional filters
func (uc *ListActivityExpensesUseCase) Execute(ctx context.Context, req *pb.ListActivityExpensesRequest) (*pb.ListActivityExpensesResponse, error) {
	return uc.Repo.ListActivityExpenses(ctx, req)
}

// =============================================================================
// GetActivityExpenseListPageData
// =============================================================================

// GetActivityExpenseListPageDataUseCase handles paginated list page data
type GetActivityExpenseListPageDataUseCase struct {
	Repo    pb.ActivityExpenseDomainServiceServer
	AuthSvc ports.AuthorizationService
}

// Execute retrieves paginated activity expense list page data
func (uc *GetActivityExpenseListPageDataUseCase) Execute(ctx context.Context, req *pb.GetActivityExpenseListPageDataRequest) (*pb.GetActivityExpenseListPageDataResponse, error) {
	return uc.Repo.GetActivityExpenseListPageData(ctx, req)
}

// =============================================================================
// GetActivityExpenseItemPageData
// =============================================================================

// GetActivityExpenseItemPageDataUseCase handles single item page data
type GetActivityExpenseItemPageDataUseCase struct {
	Repo    pb.ActivityExpenseDomainServiceServer
	AuthSvc ports.AuthorizationService
}

// Execute retrieves a single activity expense with all related data
func (uc *GetActivityExpenseItemPageDataUseCase) Execute(ctx context.Context, req *pb.GetActivityExpenseItemPageDataRequest) (*pb.GetActivityExpenseItemPageDataResponse, error) {
	return uc.Repo.GetActivityExpenseItemPageData(ctx, req)
}
