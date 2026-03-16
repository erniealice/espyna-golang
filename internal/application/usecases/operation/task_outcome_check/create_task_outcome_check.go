package task_outcome_check

import (
	"context"
	"errors"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/task_outcome_check"
)

type CreateTaskOutcomeCheckRepositories struct {
	TaskOutcomeCheck pb.TaskOutcomeCheckDomainServiceServer
}

type CreateTaskOutcomeCheckServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// CreateTaskOutcomeCheckUseCase handles the business logic for creating task outcome checks
type CreateTaskOutcomeCheckUseCase struct {
	repositories CreateTaskOutcomeCheckRepositories
	services     CreateTaskOutcomeCheckServices
}

// NewCreateTaskOutcomeCheckUseCase creates a new CreateTaskOutcomeCheckUseCase
func NewCreateTaskOutcomeCheckUseCase(
	repositories CreateTaskOutcomeCheckRepositories,
	services CreateTaskOutcomeCheckServices,
) *CreateTaskOutcomeCheckUseCase {
	return &CreateTaskOutcomeCheckUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the create task outcome check operation
func (uc *CreateTaskOutcomeCheckUseCase) Execute(ctx context.Context, req *pb.CreateTaskOutcomeCheckRequest) (*pb.CreateTaskOutcomeCheckResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityTaskOutcomeCheck, ports.ActionCreate); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "task_outcome_check.validation.data_required", "[ERR-DEFAULT] Task outcome check data is required"))
	}

	// Business validation
	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		return nil, err
	}

	// Business enrichment
	enrichedData := uc.applyBusinessLogic(req.Data)

	// Use transaction service if available
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req, enrichedData)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req, enrichedData)
}

// executeWithTransaction executes creation within a transaction
func (uc *CreateTaskOutcomeCheckUseCase) executeWithTransaction(ctx context.Context, req *pb.CreateTaskOutcomeCheckRequest, enrichedData *pb.TaskOutcomeCheck) (*pb.CreateTaskOutcomeCheckResponse, error) {
	var result *pb.CreateTaskOutcomeCheckResponse
	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req, enrichedData)
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

// executeCore contains the core business logic for creating a task outcome check
func (uc *CreateTaskOutcomeCheckUseCase) executeCore(ctx context.Context, req *pb.CreateTaskOutcomeCheckRequest, enrichedData *pb.TaskOutcomeCheck) (*pb.CreateTaskOutcomeCheckResponse, error) {
	resp, err := uc.repositories.TaskOutcomeCheck.CreateTaskOutcomeCheck(ctx, &pb.CreateTaskOutcomeCheckRequest{
		Data: enrichedData,
	})
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "task_outcome_check.errors.creation_failed", "[ERR-DEFAULT] Task outcome check creation failed"))
	}
	return resp, nil
}

// applyBusinessLogic applies business rules and returns enriched data
func (uc *CreateTaskOutcomeCheckUseCase) applyBusinessLogic(data *pb.TaskOutcomeCheck) *pb.TaskOutcomeCheck {
	now := time.Now()

	if data.Id == "" {
		data.Id = uc.services.IDService.GenerateID()
	}

	data.DateCreated = &[]int64{now.UnixMilli()}[0]
	data.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]

	return data
}

// validateBusinessRules enforces business constraints
func (uc *CreateTaskOutcomeCheckUseCase) validateBusinessRules(ctx context.Context, data *pb.TaskOutcomeCheck) error {
	if data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "task_outcome_check.validation.data_required", "[ERR-DEFAULT] Task outcome check data is required"))
	}
	if data.TaskOutcomeId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "task_outcome_check.validation.task_outcome_id_required", "[ERR-DEFAULT] Task outcome ID is required"))
	}

	return nil
}
