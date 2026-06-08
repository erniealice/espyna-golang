package job_template_phase

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template_phase"
)

type ListByJobTemplateRepositories struct {
	JobTemplatePhase pb.JobTemplatePhaseDomainServiceServer
}

type ListByJobTemplateServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// ListByJobTemplateUseCase handles the business logic for listing phases by job template
type ListByJobTemplateUseCase struct {
	repositories ListByJobTemplateRepositories
	services     ListByJobTemplateServices
}

// NewListByJobTemplateUseCase creates a new ListByJobTemplateUseCase
func NewListByJobTemplateUseCase(
	repositories ListByJobTemplateRepositories,
	services ListByJobTemplateServices,
) *ListByJobTemplateUseCase {
	return &ListByJobTemplateUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list by job template operation
func (uc *ListByJobTemplateUseCase) Execute(ctx context.Context, req *pb.ListByJobTemplateRequest) (*pb.ListByJobTemplateResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityid.JobTemplatePhase, entityid.ActionList); err != nil {
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

// executeWithTransaction executes list by job template within a transaction
func (uc *ListByJobTemplateUseCase) executeWithTransaction(ctx context.Context, req *pb.ListByJobTemplateRequest) (*pb.ListByJobTemplateResponse, error) {
	var result *pb.ListByJobTemplateResponse

	err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.Translator, "job_template_phase.errors.list_by_job_template_failed", "job template phase listing by template failed: %w"), err)
		}
		result = res
		return nil
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// executeCore contains the core business logic for listing phases by job template
func (uc *ListByJobTemplateUseCase) executeCore(ctx context.Context, req *pb.ListByJobTemplateRequest) (*pb.ListByJobTemplateResponse, error) {
	resp, err := uc.repositories.JobTemplatePhase.ListByJobTemplate(ctx, req)
	if err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "job_template_phase.errors.list_by_job_template_failed", "failed to list job template phases by template: %w"), err)
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *ListByJobTemplateUseCase) validateInput(ctx context.Context, req *pb.ListByJobTemplateRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "job_template_phase.validation.request_required", "request is required"))
	}
	if req.JobTemplateId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "job_template_phase.validation.job_template_id_required", "job template ID is required"))
	}

	return nil
}
