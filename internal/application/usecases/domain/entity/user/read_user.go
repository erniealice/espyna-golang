package user

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	userpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/user"
)

// ReadUserRepositories groups all repository dependencies
type ReadUserRepositories struct {
	User userpb.UserDomainServiceServer // Primary entity repository
}

// ReadUserServices groups all business service dependencies
type ReadUserServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
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
		Authorizer: nil,
		Transactor: ports.NewNoOpTransactor(),
		Translator:       ports.NewNoOpTranslator(),
		ActionGatekeeper: actiongate.NewActionGatekeeper(nil, ports.NewNoOpTranslator()),
	}

	return NewReadUserUseCase(repositories, services)
}

// Execute performs the read user operation
func (uc *ReadUserUseCase) Execute(ctx context.Context, req *userpb.ReadUserRequest) (*userpb.ReadUserResponse, error) {
	// Authorization check
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.User,
		Action: entityid.ActionRead,
	}); err != nil {
		return nil, err
	}

	// Check if transaction service is available and supports transactions
	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes user read within a transaction
func (uc *ReadUserUseCase) executeWithTransaction(ctx context.Context, req *userpb.ReadUserRequest) (*userpb.ReadUserResponse, error) {
	var result *userpb.ReadUserResponse

	err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.Translator, "user.errors.read_failed", "User read failed [DEFAULT]")
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
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "user.errors.not_found", "User with ID \"{userId}\" not found [DEFAULT]")
		translatedError = strings.ReplaceAll(translatedError, "{userId}", req.Data.Id)
		return nil, errors.New(translatedError)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *ReadUserUseCase) validateInput(ctx context.Context, req *userpb.ReadUserRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "user.validation.request_required", "Request is required for users [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "user.validation.data_required", "User data is required [DEFAULT]"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "user.validation.id_required", "User ID is required [DEFAULT]"))
	}
	return nil
}
