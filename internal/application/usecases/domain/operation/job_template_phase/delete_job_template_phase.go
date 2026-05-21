package job_template_phase

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template_phase"
)

type DeleteJobTemplatePhaseRepositories struct {
	JobTemplatePhase pb.JobTemplatePhaseDomainServiceServer
}

type DeleteJobTemplatePhaseServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// DeleteJobTemplatePhaseUseCase handles the business logic for deleting job template phases
type DeleteJobTemplatePhaseUseCase struct {
	repositories DeleteJobTemplatePhaseRepositories
	services     DeleteJobTemplatePhaseServices
}

// NewDeleteJobTemplatePhaseUseCase creates a new DeleteJobTemplatePhaseUseCase
func NewDeleteJobTemplatePhaseUseCase(
	repositories DeleteJobTemplatePhaseRepositories,
	services DeleteJobTemplatePhaseServices,
) *DeleteJobTemplatePhaseUseCase {
	return &DeleteJobTemplatePhaseUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the delete job template phase operation
func (uc *DeleteJobTemplatePhaseUseCase) Execute(ctx context.Context, req *pb.DeleteJobTemplatePhaseRequest) (*pb.DeleteJobTemplatePhaseResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		ports.EntityJobTemplatePhase, ports.ActionDelete); err != nil {
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

// executeWithTransaction executes deletion within a transaction
func (uc *DeleteJobTemplatePhaseUseCase) executeWithTransaction(ctx context.Context, req *pb.DeleteJobTemplatePhaseRequest) (*pb.DeleteJobTemplatePhaseResponse, error) {
	var result *pb.DeleteJobTemplatePhaseResponse

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

// executeCore contains the core business logic for deleting a job template phase
func (uc *DeleteJobTemplatePhaseUseCase) executeCore(ctx context.Context, req *pb.DeleteJobTemplatePhaseRequest) (*pb.DeleteJobTemplatePhaseResponse, error) {
	// First, check if the entity exists
	_, err := uc.repositories.JobTemplatePhase.ReadJobTemplatePhase(ctx, &pb.ReadJobTemplatePhaseRequest{
		Data: &pb.JobTemplatePhase{Id: req.Data.Id},
	})
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "job_template_phase.errors.not_found", "[ERR-DEFAULT] Job template phase not found"))
	}

	resp, err := uc.repositories.JobTemplatePhase.DeleteJobTemplatePhase(ctx, req)
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "job_template_phase.errors.deletion_failed", "[ERR-DEFAULT] Job template phase deletion failed"))
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *DeleteJobTemplatePhaseUseCase) validateInput(ctx context.Context, req *pb.DeleteJobTemplatePhaseRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "job_template_phase.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "job_template_phase.validation.data_required", "[ERR-DEFAULT] Job template phase data is required"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "job_template_phase.validation.id_required", "[ERR-DEFAULT] Job template phase ID is required"))
	}
	return nil
}
