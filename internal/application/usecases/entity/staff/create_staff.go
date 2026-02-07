package staff

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"

	"leapfor.xyz/espyna/internal/application/ports"
	contextutil "leapfor.xyz/espyna/internal/application/shared/context"
	staffpb "leapfor.xyz/esqyma/golang/v1/domain/entity/staff"
)

// CreateStaffRepositories groups all repository dependencies
type CreateStaffRepositories struct {
	Staff staffpb.StaffDomainServiceServer // Primary entity repository
}

// CreateStaffServices groups all business service dependencies
type CreateStaffServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// CreateStaffUseCase handles the business logic for creating staff
type CreateStaffUseCase struct {
	repositories CreateStaffRepositories
	services     CreateStaffServices
}

// NewCreateStaffUseCase creates use case with grouped dependencies
func NewCreateStaffUseCase(
	repositories CreateStaffRepositories,
	services CreateStaffServices,
) *CreateStaffUseCase {
	return &CreateStaffUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewCreateStaffUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewCreateStaffUseCase with grouped parameters instead
func NewCreateStaffUseCaseUngrouped(staffRepo staffpb.StaffDomainServiceServer) *CreateStaffUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := CreateStaffRepositories{
		Staff: staffRepo,
	}

	services := CreateStaffServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
		IDService:            ports.NewNoOpIDService(),
	}

	return NewCreateStaffUseCase(repositories, services)
}

// Execute performs the create staff operation
func (uc *CreateStaffUseCase) Execute(ctx context.Context, req *staffpb.CreateStaffRequest) (*staffpb.CreateStaffResponse, error) {
	// Check if transaction service is available and supports transactions
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes staff creation within a transaction
func (uc *CreateStaffUseCase) executeWithTransaction(ctx context.Context, req *staffpb.CreateStaffRequest) (*staffpb.CreateStaffResponse, error) {
	var result *staffpb.CreateStaffResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "staff.errors.creation_failed", "Staff creation failed [DEFAULT]")
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
func (uc *CreateStaffUseCase) executeCore(ctx context.Context, req *staffpb.CreateStaffRequest) (*staffpb.CreateStaffResponse, error) {
	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "staff.errors.authorization_failed", "")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityStaff, ports.ActionCreate)
	hasPerm, err := uc.services.AuthorizationService.HasPermission(ctx, userID, permission)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "staff.errors.authorization_failed", "")
		return nil, errors.New(translatedError)
	}
	if !hasPerm {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "staff.errors.authorization_failed", "")
		return nil, errors.New(translatedError)
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Business logic and enrichment
	if err := uc.enrichStaffData(req.Data); err != nil {
		return nil, err
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		return nil, err
	}

	// Call repository
	return uc.repositories.Staff.CreateStaff(ctx, req)
}

// validateInput validates the input request
func (uc *CreateStaffUseCase) validateInput(ctx context.Context, req *staffpb.CreateStaffRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "staff.validation.request_required", "Request is required for staff [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "staff.validation.data_required", "Staff data is required [DEFAULT]"))
	}
	if req.Data.User == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "staff.validation.user_data_required", "Staff user data is required [DEFAULT]"))
	}
	if req.Data.User.FirstName == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "staff.validation.first_name_required", "Staff first name is required [DEFAULT]"))
	}
	if req.Data.User.LastName == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "staff.validation.last_name_required", "Staff last name is required [DEFAULT]"))
	}
	if req.Data.User.EmailAddress == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "staff.validation.email_required", "Staff email address is required [DEFAULT]"))
	}
	return nil
}

// enrichStaffData adds generated fields and audit information
func (uc *CreateStaffUseCase) enrichStaffData(staff *staffpb.Staff) error {
	now := time.Now()

	// Generate Staff ID if not provided
	if staff.Id == "" {
		staff.Id = uc.services.IDService.GenerateID()
	}

	// Generate User ID if not provided
	if staff.User != nil && staff.User.Id == "" {
		staff.User.Id = uc.services.IDService.GenerateID()
	}

	// Set staff audit fields
	staff.DateCreated = &[]int64{now.Unix()}[0]
	staff.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	staff.DateModified = &[]int64{now.Unix()}[0]
	staff.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	staff.Active = true

	// Set user audit fields
	if staff.User != nil {
		staff.User.DateCreated = &[]int64{now.Unix()}[0]
		staff.User.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
		staff.User.DateModified = &[]int64{now.Unix()}[0]
		staff.User.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
		staff.User.Active = true

		// Set the UserId reference
		staff.UserId = staff.User.Id
	}

	return nil
}

// validateBusinessRules enforces business constraints
func (uc *CreateStaffUseCase) validateBusinessRules(ctx context.Context, staff *staffpb.Staff) error {
	if staff.User == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "staff.validation.user_data_required", "User data is required [DEFAULT]"))
	}

	// Validate email format
	if err := uc.validateEmail(staff.User.EmailAddress); err != nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "staff.validation.email_invalid", "Invalid email format [DEFAULT]"))
	}

	// Validate name length
	fullName := staff.User.FirstName + " " + staff.User.LastName
	if len(fullName) < 3 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "staff.validation.full_name_too_short", "Staff full name must be at least 3 characters long [DEFAULT]"))
	}

	if len(fullName) > 100 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "staff.validation.full_name_too_long", "Staff full name cannot exceed 100 characters [DEFAULT]"))
	}

	// Validate individual name parts
	if len(staff.User.FirstName) < 1 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "staff.validation.first_name_too_short", "First name must be at least 1 character long [DEFAULT]"))
	}

	if len(staff.User.LastName) < 1 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "staff.validation.last_name_too_short", "Last name must be at least 1 character long [DEFAULT]"))
	}

	return nil
}

// validateEmail validates email format
func (uc *CreateStaffUseCase) validateEmail(email string) error {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(email) {
		return errors.New("invalid email format")
	}
	return nil
}

// Additional validation methods can be added here as needed
