package template_task_criteria

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/template_task_criteria"
)

type ListByTemplateTaskRepositories struct {
	TemplateTaskCriteria pb.TemplateTaskCriteriaDomainServiceServer
}

type ListByTemplateTaskServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ListByTemplateTaskUseCase handles the business logic for listing criteria by template task
type ListByTemplateTaskUseCase struct {
	repositories ListByTemplateTaskRepositories
	services     ListByTemplateTaskServices
}

// NewListByTemplateTaskUseCase creates a new ListByTemplateTaskUseCase
func NewListByTemplateTaskUseCase(
	repositories ListByTemplateTaskRepositories,
	services ListByTemplateTaskServices,
) *ListByTemplateTaskUseCase {
	return &ListByTemplateTaskUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list by template task operation
func (uc *ListByTemplateTaskUseCase) Execute(ctx context.Context, req *pb.ListTemplateTaskCriteriasByTemplateTaskRequest) (*pb.ListTemplateTaskCriteriasByTemplateTaskResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityTemplateTaskCriteria, ports.ActionList); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Use transaction service if available
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes list by template task within a transaction
func (uc *ListByTemplateTaskUseCase) executeWithTransaction(ctx context.Context, req *pb.ListTemplateTaskCriteriasByTemplateTaskRequest) (*pb.ListTemplateTaskCriteriasByTemplateTaskResponse, error) {
	var result *pb.ListTemplateTaskCriteriasByTemplateTaskResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "template_task_criteria.errors.list_by_template_task_failed", "template task criteria listing by template task failed: %w"), err)
		}
		result = res
		return nil
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// executeCore contains the core business logic for listing criteria by template task
func (uc *ListByTemplateTaskUseCase) executeCore(ctx context.Context, req *pb.ListTemplateTaskCriteriasByTemplateTaskRequest) (*pb.ListTemplateTaskCriteriasByTemplateTaskResponse, error) {
	resp, err := uc.repositories.TemplateTaskCriteria.ListByTemplateTask(ctx, req)
	if err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "template_task_criteria.errors.list_by_template_task_failed", "failed to list template task criteria by template task: %w"), err)
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *ListByTemplateTaskUseCase) validateInput(ctx context.Context, req *pb.ListTemplateTaskCriteriasByTemplateTaskRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "template_task_criteria.validation.request_required", "request is required"))
	}
	if req.JobTemplateTaskId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "template_task_criteria.validation.task_id_required", "job template task ID is required"))
	}

	return nil
}
