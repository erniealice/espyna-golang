package task_outcome_check

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/task_outcome_check"
)

type ListTaskOutcomeChecksRepositories struct {
	TaskOutcomeCheck pb.TaskOutcomeCheckDomainServiceServer
}

type ListTaskOutcomeChecksServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// ListTaskOutcomeChecksUseCase handles the business logic for listing task outcome checks
type ListTaskOutcomeChecksUseCase struct {
	repositories ListTaskOutcomeChecksRepositories
	services     ListTaskOutcomeChecksServices
}

// NewListTaskOutcomeChecksUseCase creates a new ListTaskOutcomeChecksUseCase
func NewListTaskOutcomeChecksUseCase(
	repositories ListTaskOutcomeChecksRepositories,
	services ListTaskOutcomeChecksServices,
) *ListTaskOutcomeChecksUseCase {
	return &ListTaskOutcomeChecksUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list task outcome checks operation
func (uc *ListTaskOutcomeChecksUseCase) Execute(ctx context.Context, req *pb.ListTaskOutcomeChecksRequest) (*pb.ListTaskOutcomeChecksResponse, error) {
	// Authorization check
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.TaskOutcomeCheck,
		Action: entityid.ActionList,
	}); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Use transaction service if available
	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes listing within a transaction
func (uc *ListTaskOutcomeChecksUseCase) executeWithTransaction(ctx context.Context, req *pb.ListTaskOutcomeChecksRequest) (*pb.ListTaskOutcomeChecksResponse, error) {
	var result *pb.ListTaskOutcomeChecksResponse

	err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.Translator, "task_outcome_check.errors.list_failed", "task outcome check listing failed: %w"), err)
		}
		result = res
		return nil
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// executeCore contains the core business logic for listing task outcome checks
func (uc *ListTaskOutcomeChecksUseCase) executeCore(ctx context.Context, req *pb.ListTaskOutcomeChecksRequest) (*pb.ListTaskOutcomeChecksResponse, error) {
	resp, err := uc.repositories.TaskOutcomeCheck.ListTaskOutcomeChecks(ctx, req)
	if err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "task_outcome_check.errors.list_failed", "task outcome check listing failed: %w"), err)
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *ListTaskOutcomeChecksUseCase) validateInput(ctx context.Context, req *pb.ListTaskOutcomeChecksRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "task_outcome_check.validation.request_required", "request is required"))
	}

	return nil
}
