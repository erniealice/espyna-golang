package user

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	userpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/user"
)

// CreateUserRepositories groups all repository dependencies
type CreateUserRepositories struct {
	User userpb.UserDomainServiceServer // Primary entity repository
}

// CreateUserServices groups all business service dependencies
type CreateUserServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// CreateUserUseCase handles the business logic for creating users
type CreateUserUseCase struct {
	repositories CreateUserRepositories
	services     CreateUserServices
}

// NewCreateUserUseCase creates use case with grouped dependencies
func NewCreateUserUseCase(
	repositories CreateUserRepositories,
	services CreateUserServices,
) *CreateUserUseCase {
	return &CreateUserUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewCreateUserUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewCreateUserUseCase with grouped parameters instead
func NewCreateUserUseCaseUngrouped(userRepo userpb.UserDomainServiceServer, authorizationService ports.AuthorizationService) *CreateUserUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := CreateUserRepositories{
		User: userRepo,
	}

	services := CreateUserServices{
		AuthorizationService: authorizationService,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
		IDService:            ports.NewNoOpIDService(),
	}

	return NewCreateUserUseCase(repositories, services)
}

// Execute performs the create user operation
func (uc *CreateUserUseCase) Execute(ctx context.Context, req *userpb.CreateUserRequest) (*userpb.CreateUserResponse, error) {
	// Check if transaction service is available and supports transactions
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes user creation within a transaction
func (uc *CreateUserUseCase) executeWithTransaction(ctx context.Context, req *userpb.CreateUserRequest) (*userpb.CreateUserResponse, error) {
	var result *userpb.CreateUserResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "user.errors.creation_failed", "User creation failed [DEFAULT]")
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
func (uc *CreateUserUseCase) executeCore(ctx context.Context, req *userpb.CreateUserRequest) (*userpb.CreateUserResponse, error) {

	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityUser, ports.ActionCreate); err != nil {
		return nil, err
	}

	// Business validation
	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		return nil, err
	}

	// Business enrichment
	enrichedUser := uc.applyBusinessLogic(req.Data)

	// Delegate to repository
	resp, err := uc.repositories.User.CreateUser(ctx, &userpb.CreateUserRequest{
		Data: enrichedUser,
	})
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "user.errors.creation_failed", "User creation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// applyBusinessLogic applies business rules and returns enriched user
func (uc *CreateUserUseCase) applyBusinessLogic(user *userpb.User) *userpb.User {
	now := time.Now()

	// Business logic: Generate User ID if not provided
	if user.Id == "" {
		user.Id = uc.services.IDService.GenerateID()
	}

	// Business logic: Set active status for new users
	user.Active = true

	// Business logic: Set creation audit fields
	user.DateCreated = &[]int64{now.Unix()}[0]
	user.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	user.DateModified = &[]int64{now.Unix()}[0]
	user.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return user
}

// validateBusinessRules enforces business constraints
func (uc *CreateUserUseCase) validateBusinessRules(ctx context.Context, user *userpb.User) error {
	// Business rule: Required data validation
	if user == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "user.validation.data_required", ""))
	}
	if user.FirstName == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "user.validation.first_name_required", ""))
	}
	if user.LastName == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "user.validation.last_name_required", "User last name is required [DEFAULT]"))
	}
	if user.EmailAddress == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "user.validation.email_required", "User email address is required [DEFAULT]"))
	}

	// Business rule: Email format validation
	if err := uc.validateEmail(user.EmailAddress); err != nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "user.validation.email_invalid", ""))
	}

	// Business rule: Name length constraints
	fullName := user.FirstName + " " + user.LastName
	if len(fullName) < 3 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "user.validation.full_name_too_short", "User full name must be at least 3 characters long [DEFAULT]"))
	}

	if len(fullName) > 100 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "user.validation.full_name_too_long", "User full name cannot exceed 100 characters [DEFAULT]"))
	}

	// Business rule: Individual name part validation
	if len(user.FirstName) < 1 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "user.validation.first_name_too_short", "First name must be at least 1 character long [DEFAULT]"))
	}

	if len(user.LastName) < 1 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "user.validation.last_name_too_short", "Last name must be at least 1 character long [DEFAULT]"))
	}

	return nil
}

// validateEmail validates email format
func (uc *CreateUserUseCase) validateEmail(email string) error {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(email) {
		return errors.New("invalid email format")
	}
	return nil
}

// Additional validation methods can be added here as needed
