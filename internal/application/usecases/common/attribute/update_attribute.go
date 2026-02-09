package attribute

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	attributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
)

// UpdateAttributeUseCase handles the business logic for updating attributes
// UpdateAttributeRepositories groups all repository dependencies
type UpdateAttributeRepositories struct {
	Attribute attributepb.AttributeDomainServiceServer // Primary entity repository
}

// UpdateAttributeServices groups all business service dependencies
type UpdateAttributeServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// UpdateAttributeUseCase handles the business logic for updating attributes
type UpdateAttributeUseCase struct {
	repositories UpdateAttributeRepositories
	services     UpdateAttributeServices
}

// NewUpdateAttributeUseCase creates use case with grouped dependencies
func NewUpdateAttributeUseCase(
	repositories UpdateAttributeRepositories,
	services UpdateAttributeServices,
) *UpdateAttributeUseCase {
	return &UpdateAttributeUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewUpdateAttributeUseCaseUngrouped creates a new UpdateAttributeUseCase
// Deprecated: Use NewUpdateAttributeUseCase with grouped parameters instead
func NewUpdateAttributeUseCaseUngrouped(attributeRepo attributepb.AttributeDomainServiceServer) *UpdateAttributeUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := UpdateAttributeRepositories{
		Attribute: attributeRepo,
	}

	services := UpdateAttributeServices{
		TransactionService: ports.NewNoOpTransactionService(),
		TranslationService: ports.NewNoOpTranslationService(),
	}

	return NewUpdateAttributeUseCase(repositories, services)
}

// Execute performs the update attribute operation
func (uc *UpdateAttributeUseCase) Execute(ctx context.Context, req *attributepb.UpdateAttributeRequest) (*attributepb.UpdateAttributeResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		"attribute", ports.ActionUpdate); err != nil {
		return nil, err
	}

		// Check if transaction service is available and supports transactions
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes attribute update within a transaction
func (uc *UpdateAttributeUseCase) executeWithTransaction(ctx context.Context, req *attributepb.UpdateAttributeRequest) (*attributepb.UpdateAttributeResponse, error) {
	var result *attributepb.UpdateAttributeResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return fmt.Errorf("attribute update failed: %w", err)
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
func (uc *UpdateAttributeUseCase) executeCore(ctx context.Context, req *attributepb.UpdateAttributeRequest) (*attributepb.UpdateAttributeResponse, error) {
	// Input validation
	if err := uc.validateInput(req); err != nil {
		return nil, fmt.Errorf("input validation failed: %w", err)
	}

	// Business logic and enrichment
	if err := uc.enrichAttributeData(req.Data); err != nil {
		return nil, fmt.Errorf("business logic enrichment failed: %w", err)
	}

	// Business rule validation
	if err := uc.validateBusinessRules(req.Data); err != nil {
		return nil, fmt.Errorf("business rule validation failed: %w", err)
	}

	// Call repository
	return uc.repositories.Attribute.UpdateAttribute(ctx, req)
}

// validateInput validates the input request
func (uc *UpdateAttributeUseCase) validateInput(req *attributepb.UpdateAttributeRequest) error {
	if req == nil {
		return errors.New("request is required")
	}
	if req.Data == nil {
		return errors.New("attribute data is required")
	}
	if req.Data.Id == "" {
		return errors.New("attribute ID is required")
	}
	if req.Data.Name == "" {
		return errors.New("attribute name is required")
	}
	return nil
}

// enrichAttributeData adds generated fields and audit information
func (uc *UpdateAttributeUseCase) enrichAttributeData(attribute *attributepb.Attribute) error {
	now := time.Now()

	// Set attribute audit fields (preserve creation date)
	attribute.DateModified = &[]int64{now.Unix()}[0]
	attribute.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return nil
}

// validateBusinessRules enforces business constraints
func (uc *UpdateAttributeUseCase) validateBusinessRules(attribute *attributepb.Attribute) error {
	// Validate name length
	if len(strings.TrimSpace(attribute.Name)) == 0 {
		return errors.New("attribute name cannot be empty")
	}

	if len(attribute.Name) < 2 {
		return errors.New("attribute name must be at least 2 characters long")
	}

	if len(attribute.Name) > 100 {
		return errors.New("attribute name cannot exceed 100 characters")
	}

	// Validate description length if provided
	if attribute.Description != "" && len(attribute.Description) > 500 {
		return errors.New("attribute description cannot exceed 500 characters")
	}

	return nil
}
