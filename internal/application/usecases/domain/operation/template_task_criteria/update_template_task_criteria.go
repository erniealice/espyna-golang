package template_task_criteria

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/template_task_criteria"
)

type UpdateTemplateTaskCriteriaRepositories struct {
	TemplateTaskCriteria pb.TemplateTaskCriteriaDomainServiceServer
}

type UpdateTemplateTaskCriteriaServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// UpdateTemplateTaskCriteriaUseCase handles the business logic for updating template task criteria
type UpdateTemplateTaskCriteriaUseCase struct {
	repositories UpdateTemplateTaskCriteriaRepositories
	services     UpdateTemplateTaskCriteriaServices
}

// NewUpdateTemplateTaskCriteriaUseCase creates a new UpdateTemplateTaskCriteriaUseCase
func NewUpdateTemplateTaskCriteriaUseCase(
	repositories UpdateTemplateTaskCriteriaRepositories,
	services UpdateTemplateTaskCriteriaServices,
) *UpdateTemplateTaskCriteriaUseCase {
	return &UpdateTemplateTaskCriteriaUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the update template task criteria operation
func (uc *UpdateTemplateTaskCriteriaUseCase) Execute(ctx context.Context, req *pb.UpdateTemplateTaskCriteriaRequest) (*pb.UpdateTemplateTaskCriteriaResponse, error) {
	// Authorization check
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.TemplateTaskCriteria,
		Action: entityid.ActionUpdate,
	}); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Business validation
	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		return nil, err
	}

	// Business enrichment
	enrichedData := uc.applyBusinessLogic(req.Data)

	// Use transaction service if available
	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req, enrichedData)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req, enrichedData)
}

// executeWithTransaction executes update within a transaction
func (uc *UpdateTemplateTaskCriteriaUseCase) executeWithTransaction(ctx context.Context, req *pb.UpdateTemplateTaskCriteriaRequest, enrichedData *pb.TemplateTaskCriteria) (*pb.UpdateTemplateTaskCriteriaResponse, error) {
	var result *pb.UpdateTemplateTaskCriteriaResponse

	err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
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

// executeCore contains the core business logic for updating a template task criteria
func (uc *UpdateTemplateTaskCriteriaUseCase) executeCore(ctx context.Context, req *pb.UpdateTemplateTaskCriteriaRequest, enrichedData *pb.TemplateTaskCriteria) (*pb.UpdateTemplateTaskCriteriaResponse, error) {
	_, err := uc.repositories.TemplateTaskCriteria.ReadTemplateTaskCriteria(ctx, &pb.ReadTemplateTaskCriteriaRequest{
		Data: &pb.TemplateTaskCriteria{Id: req.Data.Id},
	})
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "template_task_criteria.errors.not_found", "[ERR-DEFAULT] Template task criteria not found"))
	}

	resp, err := uc.repositories.TemplateTaskCriteria.UpdateTemplateTaskCriteria(ctx, &pb.UpdateTemplateTaskCriteriaRequest{
		Data: enrichedData,
	})
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "template_task_criteria.errors.update_failed", "[ERR-DEFAULT] Template task criteria update failed"))
	}
	return resp, nil
}

// applyBusinessLogic applies business rules and returns enriched data
func (uc *UpdateTemplateTaskCriteriaUseCase) applyBusinessLogic(data *pb.TemplateTaskCriteria) *pb.TemplateTaskCriteria {
	return data
}

// validateInput validates the input request
func (uc *UpdateTemplateTaskCriteriaUseCase) validateInput(ctx context.Context, req *pb.UpdateTemplateTaskCriteriaRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "template_task_criteria.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "template_task_criteria.validation.data_required", "[ERR-DEFAULT] Template task criteria data is required"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "template_task_criteria.validation.id_required", "[ERR-DEFAULT] Template task criteria ID is required"))
	}
	return nil
}

// validateBusinessRules enforces business constraints
func (uc *UpdateTemplateTaskCriteriaUseCase) validateBusinessRules(ctx context.Context, data *pb.TemplateTaskCriteria) error {
	if data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "template_task_criteria.validation.data_required", "[ERR-DEFAULT] Template task criteria data is required"))
	}
	if data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "template_task_criteria.validation.id_required", "[ERR-DEFAULT] Template task criteria ID is required"))
	}
	return nil
}
