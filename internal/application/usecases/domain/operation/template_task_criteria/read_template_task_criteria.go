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

type ReadTemplateTaskCriteriaRepositories struct {
	TemplateTaskCriteria pb.TemplateTaskCriteriaDomainServiceServer
}

type ReadTemplateTaskCriteriaServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// ReadTemplateTaskCriteriaUseCase handles the business logic for reading template task criteria
type ReadTemplateTaskCriteriaUseCase struct {
	repositories ReadTemplateTaskCriteriaRepositories
	services     ReadTemplateTaskCriteriaServices
}

// NewReadTemplateTaskCriteriaUseCase creates a new ReadTemplateTaskCriteriaUseCase
func NewReadTemplateTaskCriteriaUseCase(
	repositories ReadTemplateTaskCriteriaRepositories,
	services ReadTemplateTaskCriteriaServices,
) *ReadTemplateTaskCriteriaUseCase {
	return &ReadTemplateTaskCriteriaUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the read template task criteria operation
func (uc *ReadTemplateTaskCriteriaUseCase) Execute(ctx context.Context, req *pb.ReadTemplateTaskCriteriaRequest) (*pb.ReadTemplateTaskCriteriaResponse, error) {
	// Authorization check
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.TemplateTaskCriteria,
		Action: entityid.ActionRead,
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

// executeWithTransaction executes reading within a transaction
func (uc *ReadTemplateTaskCriteriaUseCase) executeWithTransaction(ctx context.Context, req *pb.ReadTemplateTaskCriteriaRequest) (*pb.ReadTemplateTaskCriteriaResponse, error) {
	var result *pb.ReadTemplateTaskCriteriaResponse

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

// executeCore contains the core business logic for reading a template task criteria
func (uc *ReadTemplateTaskCriteriaUseCase) executeCore(ctx context.Context, req *pb.ReadTemplateTaskCriteriaRequest) (*pb.ReadTemplateTaskCriteriaResponse, error) {
	resp, err := uc.repositories.TemplateTaskCriteria.ReadTemplateTaskCriteria(ctx, req)
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "template_task_criteria.errors.not_found", "[ERR-DEFAULT] Template task criteria not found"))
	}
	if resp == nil || len(resp.Data) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "template_task_criteria.errors.not_found", "[ERR-DEFAULT] Template task criteria not found"))
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *ReadTemplateTaskCriteriaUseCase) validateInput(ctx context.Context, req *pb.ReadTemplateTaskCriteriaRequest) error {
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
