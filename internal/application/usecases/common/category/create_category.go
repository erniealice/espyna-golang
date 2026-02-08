package category

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	categorypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
)

// CreateCategoryUseCase handles the business logic for creating categories
// CreateCategoryRepositories groups all repository dependencies
type CreateCategoryRepositories struct {
	Category categorypb.CategoryDomainServiceServer // Primary entity repository
}

// CreateCategoryServices groups all business service dependencies
type CreateCategoryServices struct {
	TransactionService ports.TransactionService
	TranslationService ports.TranslationService
	IDService          ports.IDService
}

// CreateCategoryUseCase handles the business logic for creating categories
type CreateCategoryUseCase struct {
	repositories CreateCategoryRepositories
	services     CreateCategoryServices
}

// NewCreateCategoryUseCase creates use case with grouped dependencies
func NewCreateCategoryUseCase(
	repositories CreateCategoryRepositories,
	services CreateCategoryServices,
) *CreateCategoryUseCase {
	return &CreateCategoryUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewCreateCategoryUseCaseUngrouped creates a new CreateCategoryUseCase
// Deprecated: Use NewCreateCategoryUseCase with grouped parameters instead
func NewCreateCategoryUseCaseUngrouped(categoryRepo categorypb.CategoryDomainServiceServer) *CreateCategoryUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := CreateCategoryRepositories{
		Category: categoryRepo,
	}

	services := CreateCategoryServices{
		TransactionService: ports.NewNoOpTransactionService(),
		TranslationService: ports.NewNoOpTranslationService(),
		IDService:          ports.NewNoOpIDService(),
	}

	return NewCreateCategoryUseCase(repositories, services)
}

// Execute performs the create category operation
func (uc *CreateCategoryUseCase) Execute(ctx context.Context, req *categorypb.CreateCategoryRequest) (*categorypb.CreateCategoryResponse, error) {
	// Check if transaction service is available and supports transactions
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes category creation within a transaction
func (uc *CreateCategoryUseCase) executeWithTransaction(ctx context.Context, req *categorypb.CreateCategoryRequest) (*categorypb.CreateCategoryResponse, error) {
	var result *categorypb.CreateCategoryResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return fmt.Errorf("category creation failed: %w", err)
		}
		result = res
		return nil
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// executeCore contains the core business logic (moved from original Execute method)
func (uc *CreateCategoryUseCase) executeCore(ctx context.Context, req *categorypb.CreateCategoryRequest) (*categorypb.CreateCategoryResponse, error) {
	// Input validation
	if err := uc.validateInput(req); err != nil {
		return nil, fmt.Errorf("input validation failed: %w", err)
	}

	// Business logic and enrichment
	if err := uc.enrichCategoryData(req.Data); err != nil {
		return nil, fmt.Errorf("business logic enrichment failed: %w", err)
	}

	// Business rule validation
	if err := uc.validateBusinessRules(req.Data); err != nil {
		return nil, fmt.Errorf("business rule validation failed: %w", err)
	}

	// Call repository
	return uc.repositories.Category.CreateCategory(ctx, req)
}

// validateInput validates the input request
func (uc *CreateCategoryUseCase) validateInput(req *categorypb.CreateCategoryRequest) error {
	if req == nil {
		return errors.New("request is required")
	}
	if req.Data == nil {
		return errors.New("category data is required")
	}
	if req.Data.Name == "" {
		return errors.New("category name is required")
	}
	if req.Data.Code == "" {
		return errors.New("category code is required")
	}
	if req.Data.Module == "" {
		return errors.New("category module is required")
	}
	return nil
}

// enrichCategoryData adds generated fields and audit information
func (uc *CreateCategoryUseCase) enrichCategoryData(category *categorypb.Category) error {
	now := time.Now()

	// Generate Category ID if not provided
	if category.Id == "" {
		category.Id = uc.services.IDService.GenerateID()
	}

	// Set category audit fields
	category.DateCreated = &[]int64{now.Unix()}[0]
	category.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	category.DateModified = &[]int64{now.Unix()}[0]
	category.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	category.Active = true

	return nil
}

// validateBusinessRules enforces business constraints
func (uc *CreateCategoryUseCase) validateBusinessRules(category *categorypb.Category) error {
	// Validate name length
	if len(strings.TrimSpace(category.Name)) == 0 {
		return errors.New("category name cannot be empty")
	}

	if len(category.Name) < 2 {
		return errors.New("category name must be at least 2 characters long")
	}

	if len(category.Name) > 100 {
		return errors.New("category name cannot exceed 100 characters")
	}

	// Validate code length and format
	if len(category.Code) < 2 {
		return errors.New("category code must be at least 2 characters long")
	}

	if len(category.Code) > 50 {
		return errors.New("category code cannot exceed 50 characters")
	}

	// Validate description length if provided
	if category.Description != "" && len(category.Description) > 500 {
		return errors.New("category description cannot exceed 500 characters")
	}

	// Validate module is one of the allowed values
	allowedModules := map[string]bool{
		"client":    true,
		"product":   true,
		"location":  true,
		"event":     true,
		"payment":   true,
		"subscription": true,
		"workflow":  true,
		"staff":     true,
		"delegate":  true,
	}
	if !allowedModules[category.Module] {
		return fmt.Errorf("category module must be one of: client, product, location, event, payment, subscription, workflow, staff, delegate")
	}

	// Validate parent_id if provided
	if category.ParentId != nil && *category.ParentId != "" {
		// In a real implementation, you would verify the parent category exists
		// This is a placeholder for that validation
	}

	return nil
}
