package template_task_criteria

import (
	"context"
	"errors"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/template_task_criteria"
)

type CreateTemplateTaskCriteriaRepositories struct {
	TemplateTaskCriteria pb.TemplateTaskCriteriaDomainServiceServer
}

type CreateTemplateTaskCriteriaServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// CreateTemplateTaskCriteriaUseCase handles the business logic for creating template task criteria
type CreateTemplateTaskCriteriaUseCase struct {
	repositories CreateTemplateTaskCriteriaRepositories
	services     CreateTemplateTaskCriteriaServices
}

// NewCreateTemplateTaskCriteriaUseCase creates a new CreateTemplateTaskCriteriaUseCase
func NewCreateTemplateTaskCriteriaUseCase(
	repositories CreateTemplateTaskCriteriaRepositories,
	services CreateTemplateTaskCriteriaServices,
) *CreateTemplateTaskCriteriaUseCase {
	return &CreateTemplateTaskCriteriaUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the create template task criteria operation
func (uc *CreateTemplateTaskCriteriaUseCase) Execute(ctx context.Context, req *pb.CreateTemplateTaskCriteriaRequest) (*pb.CreateTemplateTaskCriteriaResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityTemplateTaskCriteria, ports.ActionCreate); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "template_task_criteria.validation.data_required", "[ERR-DEFAULT] Template task criteria data is required"))
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
func (uc *CreateTemplateTaskCriteriaUseCase) executeWithTransaction(ctx context.Context, req *pb.CreateTemplateTaskCriteriaRequest, enrichedData *pb.TemplateTaskCriteria) (*pb.CreateTemplateTaskCriteriaResponse, error) {
	var result *pb.CreateTemplateTaskCriteriaResponse
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

// executeCore contains the core business logic for creating a template task criteria
func (uc *CreateTemplateTaskCriteriaUseCase) executeCore(ctx context.Context, req *pb.CreateTemplateTaskCriteriaRequest, enrichedData *pb.TemplateTaskCriteria) (*pb.CreateTemplateTaskCriteriaResponse, error) {
	resp, err := uc.repositories.TemplateTaskCriteria.CreateTemplateTaskCriteria(ctx, &pb.CreateTemplateTaskCriteriaRequest{
		Data: enrichedData,
	})
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "template_task_criteria.errors.creation_failed", "[ERR-DEFAULT] Template task criteria creation failed"))
	}
	return resp, nil
}

// applyBusinessLogic applies business rules and returns enriched data
func (uc *CreateTemplateTaskCriteriaUseCase) applyBusinessLogic(data *pb.TemplateTaskCriteria) *pb.TemplateTaskCriteria {
	now := time.Now()

	if data.Id == "" {
		data.Id = uc.services.IDService.GenerateID()
	}

	data.DateCreated = &[]int64{now.UnixMilli()}[0]
	data.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]

	return data
}

// validateBusinessRules enforces business constraints
func (uc *CreateTemplateTaskCriteriaUseCase) validateBusinessRules(ctx context.Context, data *pb.TemplateTaskCriteria) error {
	if data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "template_task_criteria.validation.data_required", "[ERR-DEFAULT] Template task criteria data is required"))
	}
	if data.JobTemplateTaskId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "template_task_criteria.validation.task_id_required", "[ERR-DEFAULT] Job template task ID is required"))
	}
	if data.OutcomeCriteriaId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "template_task_criteria.validation.criteria_id_required", "[ERR-DEFAULT] Outcome criteria ID is required"))
	}

	return nil
}
