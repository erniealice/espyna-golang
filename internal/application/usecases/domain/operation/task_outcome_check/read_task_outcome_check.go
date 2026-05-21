package task_outcome_check

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/task_outcome_check"
)

type ReadTaskOutcomeCheckRepositories struct {
	TaskOutcomeCheck pb.TaskOutcomeCheckDomainServiceServer
}

type ReadTaskOutcomeCheckServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// ReadTaskOutcomeCheckUseCase handles the business logic for reading task outcome checks
type ReadTaskOutcomeCheckUseCase struct {
	repositories ReadTaskOutcomeCheckRepositories
	services     ReadTaskOutcomeCheckServices
}

// NewReadTaskOutcomeCheckUseCase creates a new ReadTaskOutcomeCheckUseCase
func NewReadTaskOutcomeCheckUseCase(
	repositories ReadTaskOutcomeCheckRepositories,
	services ReadTaskOutcomeCheckServices,
) *ReadTaskOutcomeCheckUseCase {
	return &ReadTaskOutcomeCheckUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the read task outcome check operation
func (uc *ReadTaskOutcomeCheckUseCase) Execute(ctx context.Context, req *pb.ReadTaskOutcomeCheckRequest) (*pb.ReadTaskOutcomeCheckResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		ports.EntityTaskOutcomeCheck, ports.ActionRead); err != nil {
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

// executeWithTransaction executes reading within a transaction
func (uc *ReadTaskOutcomeCheckUseCase) executeWithTransaction(ctx context.Context, req *pb.ReadTaskOutcomeCheckRequest) (*pb.ReadTaskOutcomeCheckResponse, error) {
	var result *pb.ReadTaskOutcomeCheckResponse

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

// executeCore contains the core business logic for reading a task outcome check
func (uc *ReadTaskOutcomeCheckUseCase) executeCore(ctx context.Context, req *pb.ReadTaskOutcomeCheckRequest) (*pb.ReadTaskOutcomeCheckResponse, error) {
	resp, err := uc.repositories.TaskOutcomeCheck.ReadTaskOutcomeCheck(ctx, req)
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "task_outcome_check.errors.not_found", "[ERR-DEFAULT] Task outcome check not found"))
	}
	if resp == nil || len(resp.Data) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "task_outcome_check.errors.not_found", "[ERR-DEFAULT] Task outcome check not found"))
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *ReadTaskOutcomeCheckUseCase) validateInput(ctx context.Context, req *pb.ReadTaskOutcomeCheckRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "task_outcome_check.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "task_outcome_check.validation.data_required", "[ERR-DEFAULT] Task outcome check data is required"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "task_outcome_check.validation.id_required", "[ERR-DEFAULT] Task outcome check ID is required"))
	}
	return nil
}
