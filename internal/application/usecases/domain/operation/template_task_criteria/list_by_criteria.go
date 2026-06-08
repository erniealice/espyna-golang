package template_task_criteria

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/template_task_criteria"
)

type ListByCriteriaRepositories struct {
	TemplateTaskCriteria pb.TemplateTaskCriteriaDomainServiceServer
}

type ListByCriteriaServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// ListByCriteriaUseCase handles the business logic for listing template task criteria by outcome criteria
type ListByCriteriaUseCase struct {
	repositories ListByCriteriaRepositories
	services     ListByCriteriaServices
}

// NewListByCriteriaUseCase creates a new ListByCriteriaUseCase
func NewListByCriteriaUseCase(
	repositories ListByCriteriaRepositories,
	services ListByCriteriaServices,
) *ListByCriteriaUseCase {
	return &ListByCriteriaUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list by criteria operation
func (uc *ListByCriteriaUseCase) Execute(ctx context.Context, req *pb.ListTemplateTaskCriteriasByCriteriaRequest) (*pb.ListTemplateTaskCriteriasByCriteriaResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityid.TemplateTaskCriteria, entityid.ActionList); err != nil {
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

// executeWithTransaction executes list by criteria within a transaction
func (uc *ListByCriteriaUseCase) executeWithTransaction(ctx context.Context, req *pb.ListTemplateTaskCriteriasByCriteriaRequest) (*pb.ListTemplateTaskCriteriasByCriteriaResponse, error) {
	var result *pb.ListTemplateTaskCriteriasByCriteriaResponse

	err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.Translator, "template_task_criteria.errors.list_by_criteria_failed", "template task criteria listing by criteria failed: %w"), err)
		}
		result = res
		return nil
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// executeCore contains the core business logic for listing template task criteria by outcome criteria
func (uc *ListByCriteriaUseCase) executeCore(ctx context.Context, req *pb.ListTemplateTaskCriteriasByCriteriaRequest) (*pb.ListTemplateTaskCriteriasByCriteriaResponse, error) {
	resp, err := uc.repositories.TemplateTaskCriteria.ListByCriteria(ctx, req)
	if err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "template_task_criteria.errors.list_by_criteria_failed", "failed to list template task criteria by criteria: %w"), err)
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *ListByCriteriaUseCase) validateInput(ctx context.Context, req *pb.ListTemplateTaskCriteriasByCriteriaRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "template_task_criteria.validation.request_required", "request is required"))
	}
	if req.OutcomeCriteriaId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "template_task_criteria.validation.criteria_id_required", "outcome criteria ID is required"))
	}

	return nil
}
