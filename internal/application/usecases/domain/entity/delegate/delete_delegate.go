package delegate

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	delegatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/delegate"
)

// DeleteDelegateRepositories groups all repository dependencies
type DeleteDelegateRepositories struct {
	Delegate delegatepb.DelegateDomainServiceServer // Primary entity repository
}

// DeleteDelegateServices groups all business service dependencies
type DeleteDelegateServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// DeleteDelegateUseCase handles the business logic for deleting a delegate
type DeleteDelegateUseCase struct {
	repositories DeleteDelegateRepositories
	services     DeleteDelegateServices
}

// NewDeleteDelegateUseCase creates use case with grouped dependencies
func NewDeleteDelegateUseCase(
	repositories DeleteDelegateRepositories,
	services DeleteDelegateServices,
) *DeleteDelegateUseCase {
	return &DeleteDelegateUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewDeleteDelegateUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewDeleteDelegateUseCase with grouped parameters instead
func NewDeleteDelegateUseCaseUngrouped(delegateRepo delegatepb.DelegateDomainServiceServer) *DeleteDelegateUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := DeleteDelegateRepositories{
		Delegate: delegateRepo,
	}

	services := DeleteDelegateServices{
		Authorizer: nil,
		Transactor: ports.NewNoOpTransactor(),
		Translator: ports.NewNoOpTranslator(),
	}

	return NewDeleteDelegateUseCase(repositories, services)
}

// Execute performs the delete delegate operation
func (uc *DeleteDelegateUseCase) Execute(ctx context.Context, req *delegatepb.DeleteDelegateRequest) (*delegatepb.DeleteDelegateResponse, error) {
	// Authorization check
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.Delegate,
		Action: entityid.ActionDelete,
	}); err != nil {
		return nil, err
	}

	// Input validation
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "delegate.validation.request_required", "Request is required for delegates [DEFAULT]"))
	}
	if req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "delegate.validation.data_required", "Delegate data is required [DEFAULT]"))
	}

	if req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "delegate.validation.id_required", "Delegate ID is required [DEFAULT]"))
	}

	// Check if transaction service is available and supports transactions
	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes delegate deletion within a transaction
func (uc *DeleteDelegateUseCase) executeWithTransaction(ctx context.Context, req *delegatepb.DeleteDelegateRequest) (*delegatepb.DeleteDelegateResponse, error) {
	var result *delegatepb.DeleteDelegateResponse

	err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return err
		}
		result = res
		return nil
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// executeCore contains the core business logic for delegate deletion
func (uc *DeleteDelegateUseCase) executeCore(ctx context.Context, req *delegatepb.DeleteDelegateRequest) (*delegatepb.DeleteDelegateResponse, error) {
	// Call repository
	resp, err := uc.repositories.Delegate.DeleteDelegate(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}
