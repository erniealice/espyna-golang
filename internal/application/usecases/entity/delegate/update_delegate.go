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

// UpdateDelegateRepositories groups all repository dependencies
type UpdateDelegateRepositories struct {
	Delegate delegatepb.DelegateDomainServiceServer // Primary entity repository
}

// UpdateDelegateServices groups all business service dependencies
type UpdateDelegateServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// UpdateDelegateUseCase handles the business logic for updating a delegate
type UpdateDelegateUseCase struct {
	repositories UpdateDelegateRepositories
	services     UpdateDelegateServices
}

// NewUpdateDelegateUseCase creates use case with grouped dependencies
func NewUpdateDelegateUseCase(
	repositories UpdateDelegateRepositories,
	services UpdateDelegateServices,
) *UpdateDelegateUseCase {
	return &UpdateDelegateUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewUpdateDelegateUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewUpdateDelegateUseCase with grouped parameters instead
func NewUpdateDelegateUseCaseUngrouped(delegateRepo delegatepb.DelegateDomainServiceServer) *UpdateDelegateUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := UpdateDelegateRepositories{
		Delegate: delegateRepo,
	}

	services := UpdateDelegateServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewUpdateDelegateUseCase(repositories, services)
}

// Execute performs the update delegate operation
func (uc *UpdateDelegateUseCase) Execute(ctx context.Context, req *delegatepb.UpdateDelegateRequest) (*delegatepb.UpdateDelegateResponse, error) {
	// Check if transaction service is available and supports transactions
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes delegate update within a transaction
func (uc *UpdateDelegateUseCase) executeWithTransaction(ctx context.Context, req *delegatepb.UpdateDelegateRequest) (*delegatepb.UpdateDelegateResponse, error) {
	var result *delegatepb.UpdateDelegateResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "delegate.errors.update_failed", "Delegate update failed")
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
func (uc *UpdateDelegateUseCase) executeCore(ctx context.Context, req *delegatepb.UpdateDelegateRequest) (*delegatepb.UpdateDelegateResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityDelegate, ports.ActionUpdate); err != nil {
		return nil, err
	}

	// Input validation
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate.validation.request_required", "Request is required for delegates [DEFAULT]"))
	}
	if req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate.validation.data_required", "Delegate data is required [DEFAULT]"))
	}

	if req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate.validation.id_required", "Delegate ID is required [DEFAULT]"))
	}

	// Business logic validation
	if req.Data.User != nil && req.Data.User.EmailAddress == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate.validation.email_required", "Parent/Guardian email is required [DEFAULT]"))
	}

	// Email format validation
	if req.Data.User != nil && req.Data.User.EmailAddress != "" {
		emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
		if !emailRegex.MatchString(req.Data.User.EmailAddress) {
			return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate.validation.email_invalid", "Invalid email format [DEFAULT]"))
		}
	}

	// Enrich data with timestamps
	if err := uc.enrichDelegateData(req.Data); err != nil {
		return nil, fmt.Errorf("failed to enrich delegate data: %w", err)
	}

	// Call repository
	resp, err := uc.repositories.Delegate.UpdateDelegate(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "delegate.errors.update_failed", "Delegate update failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// enrichDelegateData adds timestamps and other enrichment to delegate data
func (uc *UpdateDelegateUseCase) enrichDelegateData(delegate *delegatepb.Delegate) error {
	now := time.Now()
	delegate.DateModified = &[]int64{now.UnixMilli()}[0] // Milliseconds for consistency
	delegate.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	// Also update User timestamps if User is provided
	if delegate.User != nil {
		delegate.User.DateModified = &[]int64{now.UnixMilli()}[0]
		delegate.User.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	}

	return nil
}
