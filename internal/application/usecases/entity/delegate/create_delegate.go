package delegate

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	delegatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/delegate"
)

// CreateDelegateRepositories groups all repository dependencies
type CreateDelegateRepositories struct {
	Delegate delegatepb.DelegateDomainServiceServer // Primary entity repository
}

// CreateDelegateServices groups all business service dependencies
type CreateDelegateServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// CreateDelegateUseCase handles the business logic for creating delegates
type CreateDelegateUseCase struct {
	repositories CreateDelegateRepositories
	services     CreateDelegateServices
}

// NewCreateDelegateUseCase creates use case with grouped dependencies
func NewCreateDelegateUseCase(
	repositories CreateDelegateRepositories,
	services CreateDelegateServices,
) *CreateDelegateUseCase {
	return &CreateDelegateUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewCreateDelegateUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewCreateDelegateUseCase with grouped parameters instead
func NewCreateDelegateUseCaseUngrouped(delegateRepo delegatepb.DelegateDomainServiceServer) *CreateDelegateUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := CreateDelegateRepositories{
		Delegate: delegateRepo,
	}

	services := CreateDelegateServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
		IDService:            ports.NewNoOpIDService(),
	}

	return NewCreateDelegateUseCase(repositories, services)
}

// Execute performs the create delegate operation
func (uc *CreateDelegateUseCase) Execute(ctx context.Context, req *delegatepb.CreateDelegateRequest) (*delegatepb.CreateDelegateResponse, error) {
	// Check if transaction service is available and supports transactions
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes delegate creation within a transaction
func (uc *CreateDelegateUseCase) executeWithTransaction(ctx context.Context, req *delegatepb.CreateDelegateRequest) (*delegatepb.CreateDelegateResponse, error) {
	var result *delegatepb.CreateDelegateResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "delegate.errors.creation_failed", "Delegate creation failed [DEFAULT]")
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
func (uc *CreateDelegateUseCase) executeCore(ctx context.Context, req *delegatepb.CreateDelegateRequest) (*delegatepb.CreateDelegateResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityDelegate, ports.ActionCreate); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Business logic and enrichment
	if err := uc.enrichDelegateData(req.Data); err != nil {
		return nil, err
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		return nil, err
	}

	// Call repository
	return uc.repositories.Delegate.CreateDelegate(ctx, req)
}

// validateInput validates the input request
func (uc *CreateDelegateUseCase) validateInput(ctx context.Context, req *delegatepb.CreateDelegateRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate.validation.request_required", "Request is required for delegates [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate.validation.data_required", "Delegate data is required [DEFAULT]"))
	}
	if req.Data.User == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate.validation.user_data_required", "Delegate user data is required [DEFAULT]"))
	}
	if req.Data.User.FirstName == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate.validation.first_name_required", "Delegate first name is required [DEFAULT]"))
	}
	if req.Data.User.LastName == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate.validation.last_name_required", "Delegate last name is required [DEFAULT]"))
	}
	if req.Data.User.EmailAddress == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate.validation.email_required", "Delegate email address is required [DEFAULT]"))
	}
	return nil
}

// enrichDelegateData adds generated fields and audit information
func (uc *CreateDelegateUseCase) enrichDelegateData(delegate *delegatepb.Delegate) error {
	now := time.Now()

	// Generate Delegate ID if not provided
	if delegate.Id == "" {
		delegate.Id = uc.services.IDService.GenerateID()
	}

	// Generate User ID if not provided
	if delegate.User != nil && delegate.User.Id == "" {
		delegate.User.Id = uc.services.IDService.GenerateID()
	}

	// Set delegate audit fields
	delegate.DateCreated = &[]int64{now.UnixMilli()}[0]
	delegate.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	delegate.DateModified = &[]int64{now.UnixMilli()}[0]
	delegate.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	delegate.Active = true

	// Set user audit fields
	if delegate.User != nil {
		delegate.User.DateCreated = &[]int64{now.UnixMilli()}[0]
		delegate.User.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
		delegate.User.DateModified = &[]int64{now.UnixMilli()}[0]
		delegate.User.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
		delegate.User.Active = true

		// Set the UserId reference
		delegate.UserId = delegate.User.Id
	}

	return nil
}

// validateBusinessRules enforces business constraints
func (uc *CreateDelegateUseCase) validateBusinessRules(ctx context.Context, delegate *delegatepb.Delegate) error {
	if delegate.User == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate.validation.user_data_required", "User data is required [DEFAULT]"))
	}

	// Validate email format
	if err := uc.validateEmail(delegate.User.EmailAddress); err != nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate.validation.email_invalid", "Invalid email format [DEFAULT]"))
	}

	// Validate name length
	fullName := delegate.User.FirstName + " " + delegate.User.LastName
	if len(fullName) < 3 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate.validation.full_name_too_short", "Delegate full name must be at least 3 characters long [DEFAULT]"))
	}

	if len(fullName) > 100 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate.validation.full_name_too_long", "Delegate full name cannot exceed 100 characters [DEFAULT]"))
	}

	// Validate individual name parts
	if len(delegate.User.FirstName) < 1 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate.validation.first_name_too_short", "First name must be at least 1 character long [DEFAULT]"))
	}

	if len(delegate.User.LastName) < 1 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate.validation.last_name_too_short", "Last name must be at least 1 character long [DEFAULT]"))
	}

	return nil
}

// validateEmail validates email format
func (uc *CreateDelegateUseCase) validateEmail(email string) error {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(email) {
		return errors.New("invalid email format")
	}
	return nil
}

// Additional validation methods can be added here as needed
