package supplier

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	supplierpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/supplier"
)

// UpdateSupplierRepositories groups all repository dependencies
type UpdateSupplierRepositories struct {
	Supplier supplierpb.SupplierDomainServiceServer // Primary entity repository
}

// UpdateSupplierServices groups all business service dependencies
type UpdateSupplierServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// UpdateSupplierUseCase handles the business logic for updating a supplier
type UpdateSupplierUseCase struct {
	repositories UpdateSupplierRepositories
	services     UpdateSupplierServices
}

// NewUpdateSupplierUseCase creates use case with grouped dependencies
func NewUpdateSupplierUseCase(
	repositories UpdateSupplierRepositories,
	services UpdateSupplierServices,
) *UpdateSupplierUseCase {
	return &UpdateSupplierUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewUpdateSupplierUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewUpdateSupplierUseCase with grouped parameters instead
func NewUpdateSupplierUseCaseUngrouped(supplierRepo supplierpb.SupplierDomainServiceServer) *UpdateSupplierUseCase {
	repositories := UpdateSupplierRepositories{
		Supplier: supplierRepo,
	}

	services := UpdateSupplierServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewUpdateSupplierUseCase(repositories, services)
}

// Execute performs the update supplier operation
func (uc *UpdateSupplierUseCase) Execute(ctx context.Context, req *supplierpb.UpdateSupplierRequest) (*supplierpb.UpdateSupplierResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		"supplier", ports.ActionUpdate); err != nil {
		return nil, err
	}

	// Input validation
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "supplier.validation.request_required", "Request is required for suppliers [DEFAULT]"))
	}

	if req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "supplier.validation.id_required", "Supplier ID is required [DEFAULT]"))
	}

	// Business logic validation
	if req.Data.CompanyName == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "supplier.validation.company_name_required", "Supplier company name is required [DEFAULT]"))
	}

	// Call repository
	resp, err := uc.repositories.Supplier.UpdateSupplier(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "supplier.errors.update_failed", "Supplier update failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}
