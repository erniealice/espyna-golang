package template_task_criteria

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/template_task_criteria"
)

type DeleteTemplateTaskCriteriaRepositories struct {
	TemplateTaskCriteria pb.TemplateTaskCriteriaDomainServiceServer
}

type DeleteTemplateTaskCriteriaServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// DeleteTemplateTaskCriteriaUseCase handles the business logic for deleting template task criteria
type DeleteTemplateTaskCriteriaUseCase struct {
	repositories DeleteTemplateTaskCriteriaRepositories
	services     DeleteTemplateTaskCriteriaServices
}

// NewDeleteTemplateTaskCriteriaUseCase creates a new DeleteTemplateTaskCriteriaUseCase
func NewDeleteTemplateTaskCriteriaUseCase(
	repositories DeleteTemplateTaskCriteriaRepositories,
	services DeleteTemplateTaskCriteriaServices,
) *DeleteTemplateTaskCriteriaUseCase {
	return &DeleteTemplateTaskCriteriaUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the delete template task criteria operation
func (uc *DeleteTemplateTaskCriteriaUseCase) Execute(ctx context.Context, req *pb.DeleteTemplateTaskCriteriaRequest) (*pb.DeleteTemplateTaskCriteriaResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityTemplateTaskCriteria, ports.ActionDelete); err != nil {
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

// executeWithTransaction executes deletion within a transaction
func (uc *DeleteTemplateTaskCriteriaUseCase) executeWithTransaction(ctx context.Context, req *pb.DeleteTemplateTaskCriteriaRequest) (*pb.DeleteTemplateTaskCriteriaResponse, error) {
	var result *pb.DeleteTemplateTaskCriteriaResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
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

// executeCore contains the core business logic for deleting a template task criteria
func (uc *DeleteTemplateTaskCriteriaUseCase) executeCore(ctx context.Context, req *pb.DeleteTemplateTaskCriteriaRequest) (*pb.DeleteTemplateTaskCriteriaResponse, error) {
	_, err := uc.repositories.TemplateTaskCriteria.ReadTemplateTaskCriteria(ctx, &pb.ReadTemplateTaskCriteriaRequest{
		Data: &pb.TemplateTaskCriteria{Id: req.Data.Id},
	})
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "template_task_criteria.errors.not_found", "[ERR-DEFAULT] Template task criteria not found"))
	}

	resp, err := uc.repositories.TemplateTaskCriteria.DeleteTemplateTaskCriteria(ctx, req)
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "template_task_criteria.errors.deletion_failed", "[ERR-DEFAULT] Template task criteria deletion failed"))
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *DeleteTemplateTaskCriteriaUseCase) validateInput(ctx context.Context, req *pb.DeleteTemplateTaskCriteriaRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "template_task_criteria.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "template_task_criteria.validation.data_required", "[ERR-DEFAULT] Template task criteria data is required"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "template_task_criteria.validation.id_required", "[ERR-DEFAULT] Template task criteria ID is required"))
	}
	return nil
}
