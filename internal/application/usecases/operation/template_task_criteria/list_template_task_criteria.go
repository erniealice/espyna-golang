package template_task_criteria

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/template_task_criteria"
)

type ListTemplateTaskCriteriaRepositories struct {
	TemplateTaskCriteria pb.TemplateTaskCriteriaDomainServiceServer
}

type ListTemplateTaskCriteriaServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ListTemplateTaskCriteriaUseCase handles the business logic for listing template task criteria
type ListTemplateTaskCriteriaUseCase struct {
	repositories ListTemplateTaskCriteriaRepositories
	services     ListTemplateTaskCriteriaServices
}

// NewListTemplateTaskCriteriaUseCase creates a new ListTemplateTaskCriteriaUseCase
func NewListTemplateTaskCriteriaUseCase(
	repositories ListTemplateTaskCriteriaRepositories,
	services ListTemplateTaskCriteriaServices,
) *ListTemplateTaskCriteriaUseCase {
	return &ListTemplateTaskCriteriaUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list template task criteria operation
func (uc *ListTemplateTaskCriteriaUseCase) Execute(ctx context.Context, req *pb.ListTemplateTaskCriteriasRequest) (*pb.ListTemplateTaskCriteriasResponse, error) {
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

// executeWithTransaction executes listing within a transaction
func (uc *ListTemplateTaskCriteriaUseCase) executeWithTransaction(ctx context.Context, req *pb.ListTemplateTaskCriteriasRequest) (*pb.ListTemplateTaskCriteriasResponse, error) {
	var result *pb.ListTemplateTaskCriteriasResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "template_task_criteria.errors.list_failed", "template task criteria listing failed: %w"), err)
		}
		result = res
		return nil
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// executeCore contains the core business logic for listing template task criteria
func (uc *ListTemplateTaskCriteriaUseCase) executeCore(ctx context.Context, req *pb.ListTemplateTaskCriteriasRequest) (*pb.ListTemplateTaskCriteriasResponse, error) {
	resp, err := uc.repositories.TemplateTaskCriteria.ListTemplateTaskCriterias(ctx, req)
	if err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "template_task_criteria.errors.list_failed", "template task criteria listing failed: %w"), err)
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *ListTemplateTaskCriteriaUseCase) validateInput(ctx context.Context, req *pb.ListTemplateTaskCriteriasRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "template_task_criteria.validation.request_required", "request is required"))
	}

	return nil
}
