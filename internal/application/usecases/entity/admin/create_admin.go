package admin

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	adminpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/admin"
)

// CreateAdminRepositories groups all repository dependencies
type CreateAdminRepositories struct {
	Admin adminpb.AdminDomainServiceServer // Primary entity repository
}

// CreateAdminServices groups all business service dependencies
type CreateAdminServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// CreateAdminUseCase handles the business logic for creating admins
type CreateAdminUseCase struct {
	repositories CreateAdminRepositories
	services     CreateAdminServices
}

// NewCreateAdminUseCase creates use case with grouped dependencies
func NewCreateAdminUseCase(
	repositories CreateAdminRepositories,
	services CreateAdminServices,
) *CreateAdminUseCase {
	return &CreateAdminUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the create admin operation
func (uc *CreateAdminUseCase) Execute(ctx context.Context, req *adminpb.CreateAdminRequest) (*adminpb.CreateAdminResponse, error) {
	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "admin.errors.authorization_failed", "")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityAdmin, ports.ActionCreate)
	hasPerm, err := uc.services.AuthorizationService.HasPermission(ctx, userID, permission)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "admin.errors.authorization_failed", "")
		return nil, errors.New(translatedError)
	}
	if !hasPerm {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "admin.errors.authorization_failed", "")
		return nil, errors.New(translatedError)
	}

	// Input validation
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "admin.validation.request_required", ""))
	}

	// Check if transaction service is available and supports transactions
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes admin creation within a transaction
func (uc *CreateAdminUseCase) executeWithTransaction(ctx context.Context, req *adminpb.CreateAdminRequest) (*adminpb.CreateAdminResponse, error) {
	var result *adminpb.CreateAdminResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "admin.errors.creation_failed", "")
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

// executeCore contains the core business logic (moved from original Execute method)
func (uc *CreateAdminUseCase) executeCore(ctx context.Context, req *adminpb.CreateAdminRequest) (*adminpb.CreateAdminResponse, error) {
	// Business validation
	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		return nil, err
	}

	// Business enrichment
	enrichedAdmin := uc.applyBusinessLogic(req.Data)

	// Delegate to repository
	return uc.repositories.Admin.CreateAdmin(ctx, &adminpb.CreateAdminRequest{
		Data: enrichedAdmin,
	})
}

// applyBusinessLogic applies business rules and returns enriched admin
func (uc *CreateAdminUseCase) applyBusinessLogic(admin *adminpb.Admin) *adminpb.Admin {
	now := time.Now()

	// Business logic: Generate Admin ID if not provided
	if admin.Id == "" {
		admin.Id = uc.services.IDService.GenerateID()
	}

	// Business logic: Generate User ID if not provided
	if admin.User != nil && admin.User.Id == "" {
		admin.User.Id = uc.services.IDService.GenerateID()
	}

	// Business logic: Set active status for new admins
	admin.Active = true

	// Business logic: Set creation audit fields
	admin.DateCreated = &[]int64{now.UnixMilli()}[0]
	admin.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	admin.DateModified = &[]int64{now.UnixMilli()}[0]
	admin.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	// Business logic: Set user audit fields
	if admin.User != nil {
		admin.User.DateCreated = &[]int64{now.UnixMilli()}[0]
		admin.User.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
		admin.User.DateModified = &[]int64{now.UnixMilli()}[0]
		admin.User.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
		admin.User.Active = true

		// Business logic: Set the UserId reference
		admin.UserId = admin.User.Id
	}

	return admin
}

// validateBusinessRules enforces business constraints
func (uc *CreateAdminUseCase) validateBusinessRules(ctx context.Context, admin *adminpb.Admin) error {
	// Business rule: Required data validation
	if admin == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "admin.validation.data_required", ""))
	}
	if admin.User == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "admin.validation.user_data_required", ""))
	}
	if admin.User.FirstName == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "admin.validation.first_name_required", ""))
	}
	if admin.User.LastName == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "admin.validation.last_name_required", ""))
	}
	if admin.User.EmailAddress == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "admin.validation.email_required", ""))
	}

	// Business rule: Email format validation
	if err := uc.validateEmail(admin.User.EmailAddress); err != nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "admin.validation.email_invalid", ""))
	}

	// Business rule: Name length constraints
	fullName := admin.User.FirstName + " " + admin.User.LastName
	if len(fullName) < 3 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "admin.validation.full_name_too_short", ""))
	}

	if len(fullName) > 100 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "admin.validation.full_name_too_long", ""))
	}

	// Business rule: Individual name part validation
	if len(admin.User.FirstName) < 1 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "admin.validation.first_name_too_short", ""))
	}

	if len(admin.User.LastName) < 1 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "admin.validation.last_name_too_short", ""))
	}

	return nil
}

// validateEmail validates email format
func (uc *CreateAdminUseCase) validateEmail(email string) error {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(email) {
		return errors.New("invalid email format")
	}
	return nil
}

// Additional validation methods can be added here as needed
