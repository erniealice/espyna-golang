package supplier

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	supplierpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/supplier"
	userpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/user"
)

// CreateSupplierRepositories groups all repository dependencies
type CreateSupplierRepositories struct {
	Supplier supplierpb.SupplierDomainServiceServer // Primary entity repository
	User     userpb.UserDomainServiceServer         // User repository for embedded user data
}

// CreateSupplierServices groups all business service dependencies
type CreateSupplierServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// CreateSupplierUseCase handles the business logic for creating suppliers
type CreateSupplierUseCase struct {
	repositories CreateSupplierRepositories
	services     CreateSupplierServices
}

// NewCreateSupplierUseCase creates use case with grouped dependencies
func NewCreateSupplierUseCase(
	repositories CreateSupplierRepositories,
	services CreateSupplierServices,
) *CreateSupplierUseCase {
	return &CreateSupplierUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewCreateSupplierUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewCreateSupplierUseCase with grouped parameters instead
func NewCreateSupplierUseCaseUngrouped(supplierRepo supplierpb.SupplierDomainServiceServer) *CreateSupplierUseCase {
	repositories := CreateSupplierRepositories{
		Supplier: supplierRepo,
	}

	services := CreateSupplierServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
		IDService:            ports.NewNoOpIDService(),
	}

	return NewCreateSupplierUseCase(repositories, services)
}

// Execute performs the create supplier operation
func (uc *CreateSupplierUseCase) Execute(ctx context.Context, req *supplierpb.CreateSupplierRequest) (*supplierpb.CreateSupplierResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		"supplier", ports.ActionCreate); err != nil {
		return nil, err
	}

	// Input validation
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "supplier.validation.request_required", "Request is required for suppliers [DEFAULT]"))
	}

	// Business validation
	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		return nil, err
	}

	// Business enrichment
	enrichedSupplier := uc.applyBusinessLogic(req.Data)

	// Use transaction service if available
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, enrichedSupplier)
	}

	// Fallback to direct repository call
	return uc.executeCore(ctx, enrichedSupplier)
}

// executeWithTransaction executes supplier creation within a transaction
func (uc *CreateSupplierUseCase) executeWithTransaction(ctx context.Context, enrichedSupplier *supplierpb.Supplier) (*supplierpb.CreateSupplierResponse, error) {
	var result *supplierpb.CreateSupplierResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, enrichedSupplier)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "supplier.errors.creation_failed", "Supplier creation failed [DEFAULT]")
			return fmt.Errorf("%s: %w", translatedError, err)
		}
		result = res
		return nil
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// executeCore contains the core business logic for creating a supplier
func (uc *CreateSupplierUseCase) executeCore(ctx context.Context, enrichedSupplier *supplierpb.Supplier) (*supplierpb.CreateSupplierResponse, error) {
	// Step 1: Find or create User record (if User repository is available and user data exists)
	if uc.repositories.User != nil && enrichedSupplier.User != nil {
		user, err := uc.findOrCreateUser(ctx, enrichedSupplier.User)
		if err != nil {
			return nil, fmt.Errorf("failed to find or create user record: %w", err)
		}

		// Update the supplier's UserId reference with the user's ID
		enrichedSupplier.UserId = user.Id
		enrichedSupplier.User = user
	}

	// Step 2: Create Supplier record (with reference to the User)
	return uc.repositories.Supplier.CreateSupplier(ctx, &supplierpb.CreateSupplierRequest{
		Data: enrichedSupplier,
	})
}

// findOrCreateUser finds an existing user by email or creates a new one
func (uc *CreateSupplierUseCase) findOrCreateUser(ctx context.Context, userData *userpb.User) (*userpb.User, error) {
	if userData.EmailAddress == "" {
		return nil, fmt.Errorf("email address is required to find or create user")
	}

	// Step 1: Try to find existing user by email
	listResp, err := uc.repositories.User.ListUsers(ctx, &userpb.ListUsersRequest{
		Filters: &commonpb.FilterRequest{
			Filters: []*commonpb.TypedFilter{
				{
					Field: "email_address",
					FilterType: &commonpb.TypedFilter_StringFilter{
						StringFilter: &commonpb.StringFilter{
							Value:    userData.EmailAddress,
							Operator: commonpb.StringOperator_STRING_EQUALS,
						},
					},
				},
			},
		},
	})

	// If we found an existing user, return it
	if err == nil && listResp != nil && len(listResp.Data) > 0 {
		return listResp.Data[0], nil
	}

	// Step 2: No existing user found - create a new one
	createResp, err := uc.repositories.User.CreateUser(ctx, &userpb.CreateUserRequest{
		Data: userData,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	if createResp == nil || len(createResp.Data) == 0 {
		return nil, fmt.Errorf("user creation returned no data")
	}

	return createResp.Data[0], nil
}

// applyBusinessLogic applies business rules and returns enriched supplier
func (uc *CreateSupplierUseCase) applyBusinessLogic(supplier *supplierpb.Supplier) *supplierpb.Supplier {
	now := time.Now()

	// Business logic: Generate Supplier ID if not provided
	if supplier.Id == "" {
		if uc.services.IDService != nil {
			supplier.Id = uc.services.IDService.GenerateID()
		} else {
			supplier.Id = fmt.Sprintf("supplier-%d", now.UnixNano())
		}
	}

	// Business logic: Generate User ID if not provided
	if supplier.User != nil {
		if supplier.User.Id == "" {
			if uc.services.IDService != nil {
				supplier.User.Id = uc.services.IDService.GenerateID()
			} else {
				supplier.User.Id = fmt.Sprintf("user-%d", now.UnixNano())
			}
		}
	}

	// Business logic: Generate internal_id if not provided
	if supplier.InternalId == "" {
		if uc.services.IDService != nil {
			supplier.InternalId = uc.services.IDService.GenerateID()
		} else {
			supplier.InternalId = fmt.Sprintf("internal-%d", now.UnixNano())
		}
	}

	// Business logic: Set active status for new suppliers
	supplier.Active = true

	// Business logic: Set creation audit fields
	supplier.DateCreated = &[]int64{now.UnixMilli()}[0]
	supplier.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	supplier.DateModified = &[]int64{now.UnixMilli()}[0]
	supplier.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	// Business logic: Set user audit fields
	if supplier.User != nil {
		supplier.User.DateCreated = &[]int64{now.UnixMilli()}[0]
		supplier.User.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
		supplier.User.DateModified = &[]int64{now.UnixMilli()}[0]
		supplier.User.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
		supplier.User.Active = true

		// Business logic: Set the UserId reference
		supplier.UserId = supplier.User.Id
	}

	return supplier
}

// validateBusinessRules enforces business constraints
func (uc *CreateSupplierUseCase) validateBusinessRules(ctx context.Context, supplier *supplierpb.Supplier) error {
	// Business rule: Required data validation
	if supplier == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "supplier.validation.data_required", "Supplier data is required [DEFAULT]"))
	}

	// Business rule: Company name is required
	if supplier.CompanyName == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "supplier.validation.company_name_required", "Supplier company name is required [DEFAULT]"))
	}

	// Business rule: Supplier type is required
	if supplier.SupplierType == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "supplier.validation.supplier_type_required", "Supplier type is required [DEFAULT]"))
	}

	// Business rule: Company name length constraints
	if len(supplier.CompanyName) < 2 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "supplier.validation.company_name_too_short", "Company name must be at least 2 characters long [DEFAULT]"))
	}

	if len(supplier.CompanyName) > 200 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "supplier.validation.company_name_too_long", "Company name cannot exceed 200 characters [DEFAULT]"))
	}

	// Business rule: Internal ID format validation
	if supplier.InternalId != "" {
		if len(supplier.InternalId) < 3 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "supplier.validation.internal_id_too_short", "Internal ID must be at least 3 characters long [DEFAULT]"))
		}
		if len(supplier.InternalId) > 50 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "supplier.validation.internal_id_too_long", "Internal ID cannot exceed 50 characters [DEFAULT]"))
		}
	}

	return nil
}
