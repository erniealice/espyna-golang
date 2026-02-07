package user

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"leapfor.xyz/espyna/internal/application/ports"
	contextutil "leapfor.xyz/espyna/internal/application/shared/context"
	userpb "leapfor.xyz/esqyma/golang/v1/domain/entity/user"
)

// ReadUserRepositories groups all repository dependencies
type ReadUserRepositories struct {
	User userpb.UserDomainServiceServer // Primary entity repository
}

// ReadUserServices groups all business service dependencies
type ReadUserServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ReadUserUseCase handles the business logic for reading a user
type ReadUserUseCase struct {
	repositories ReadUserRepositories
	services     ReadUserServices
}

// NewReadUserUseCase creates use case with grouped dependencies
func NewReadUserUseCase(
	repositories ReadUserRepositories,
	services ReadUserServices,
) *ReadUserUseCase {
	return &ReadUserUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewReadUserUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewReadUserUseCase with grouped parameters instead
func NewReadUserUseCaseUngrouped(userRepo userpb.UserDomainServiceServer) *ReadUserUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := ReadUserRepositories{
		User: userRepo,
	}

	services := ReadUserServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewReadUserUseCase(repositories, services)
}

// Execute performs the read user operation
func (uc *ReadUserUseCase) Execute(ctx context.Context, req *userpb.ReadUserRequest) (*userpb.ReadUserResponse, error) {
	// Check if transaction service is available and supports transactions
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes user read within a transaction
func (uc *ReadUserUseCase) executeWithTransaction(ctx context.Context, req *userpb.ReadUserRequest) (*userpb.ReadUserResponse, error) {
	var result *userpb.ReadUserResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "user.errors.read_failed", "User read failed [DEFAULT]")
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
func (uc *ReadUserUseCase) executeCore(ctx context.Context, req *userpb.ReadUserRequest) (*userpb.ReadUserResponse, error) {
	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Call repository
	resp, err := uc.repositories.User.ReadUser(ctx, req)
	if err != nil {
		return nil, err
	}

	// Not found error
	if len(resp.Data) == 0 || resp.Data[0].Id == "" { // Assuming resp.Data will be nil or have empty ID if not found
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "user.errors.not_found", "User with ID \"{userId}\" not found [DEFAULT]")
		translatedError = strings.ReplaceAll(translatedError, "{userId}", req.Data.Id)
		return nil, errors.New(translatedError)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *ReadUserUseCase) validateInput(ctx context.Context, req *userpb.ReadUserRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "user.validation.request_required", "Request is required for users [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "user.validation.data_required", "User data is required [DEFAULT]"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "user.validation.id_required", "User ID is required [DEFAULT]"))
	}
	return nil
}
