package supplier

import (
	"context"
	"errors"
	"strings"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	supplierpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/supplier"
)

// ReadSupplierRepositories groups all repository dependencies
type ReadSupplierRepositories struct {
	Supplier supplierpb.SupplierDomainServiceServer // Primary entity repository
}

// ReadSupplierServices groups all business service dependencies
type ReadSupplierServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ReadSupplierUseCase handles the business logic for reading a supplier
type ReadSupplierUseCase struct {
	repositories ReadSupplierRepositories
	services     ReadSupplierServices
}

// NewReadSupplierUseCase creates use case with grouped dependencies
func NewReadSupplierUseCase(
	repositories ReadSupplierRepositories,
	services ReadSupplierServices,
) *ReadSupplierUseCase {
	return &ReadSupplierUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewReadSupplierUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewReadSupplierUseCase with grouped parameters instead
func NewReadSupplierUseCaseUngrouped(supplierRepo supplierpb.SupplierDomainServiceServer) *ReadSupplierUseCase {
	repositories := ReadSupplierRepositories{
		Supplier: supplierRepo,
	}

	services := ReadSupplierServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewReadSupplierUseCase(repositories, services)
}

// Execute performs the read supplier operation
func (uc *ReadSupplierUseCase) Execute(ctx context.Context, req *supplierpb.ReadSupplierRequest) (*supplierpb.ReadSupplierResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		"supplier", ports.ActionRead); err != nil {
		return nil, err
	}

	// Input validation
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "supplier.validation.request_required", "Request is required for suppliers [DEFAULT]"))
	}

	if req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "supplier.validation.id_required", "Supplier ID is required [DEFAULT]"))
	}

	// Call repository
	resp, err := uc.repositories.Supplier.ReadSupplier(ctx, req)
	if err != nil {
		return nil, err
	}

	// Not found error
	if len(resp.Data) == 0 || resp.Data[0].Id == "" {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "supplier.errors.not_found", "Supplier with ID \"{supplierId}\" not found [DEFAULT]")
		translatedError = strings.ReplaceAll(translatedError, "{supplierId}", req.Data.Id)
		return nil, errors.New(translatedError)
	}

	return resp, nil
}
