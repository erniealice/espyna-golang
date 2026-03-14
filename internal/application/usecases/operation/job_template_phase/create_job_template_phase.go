package job_template_phase

import (
	"context"
	"errors"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template_phase"
)

type CreateJobTemplatePhaseRepositories struct {
	JobTemplatePhase pb.JobTemplatePhaseDomainServiceServer
}

type CreateJobTemplatePhaseServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// CreateJobTemplatePhaseUseCase handles the business logic for creating job template phases
type CreateJobTemplatePhaseUseCase struct {
	repositories CreateJobTemplatePhaseRepositories
	services     CreateJobTemplatePhaseServices
}

// NewCreateJobTemplatePhaseUseCase creates a new CreateJobTemplatePhaseUseCase
func NewCreateJobTemplatePhaseUseCase(
	repositories CreateJobTemplatePhaseRepositories,
	services CreateJobTemplatePhaseServices,
) *CreateJobTemplatePhaseUseCase {
	return &CreateJobTemplatePhaseUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the create job template phase operation
func (uc *CreateJobTemplatePhaseUseCase) Execute(ctx context.Context, req *pb.CreateJobTemplatePhaseRequest) (*pb.CreateJobTemplatePhaseResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityJobTemplatePhase, ports.ActionCreate); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_template_phase.validation.data_required", "[ERR-DEFAULT] Job template phase data is required"))
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
func (uc *CreateJobTemplatePhaseUseCase) executeWithTransaction(ctx context.Context, req *pb.CreateJobTemplatePhaseRequest, enrichedData *pb.JobTemplatePhase) (*pb.CreateJobTemplatePhaseResponse, error) {
	var result *pb.CreateJobTemplatePhaseResponse
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

// executeCore contains the core business logic for creating a job template phase
func (uc *CreateJobTemplatePhaseUseCase) executeCore(ctx context.Context, req *pb.CreateJobTemplatePhaseRequest, enrichedData *pb.JobTemplatePhase) (*pb.CreateJobTemplatePhaseResponse, error) {
	resp, err := uc.repositories.JobTemplatePhase.CreateJobTemplatePhase(ctx, &pb.CreateJobTemplatePhaseRequest{
		Data: enrichedData,
	})
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_template_phase.errors.creation_failed", "[ERR-DEFAULT] Job template phase creation failed"))
	}
	return resp, nil
}

// applyBusinessLogic applies business rules and returns enriched data
func (uc *CreateJobTemplatePhaseUseCase) applyBusinessLogic(data *pb.JobTemplatePhase) *pb.JobTemplatePhase {
	now := time.Now()

	// Business logic: Generate ID if not provided
	if data.Id == "" {
		data.Id = uc.services.IDService.GenerateID()
	}

	// Business logic: Set active status for new phases
	data.Active = true

	// Business logic: Set creation audit fields
	data.DateCreated = &[]int64{now.UnixMilli()}[0]
	data.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	data.DateModified = &[]int64{now.UnixMilli()}[0]
	data.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return data
}

// validateBusinessRules enforces business constraints
func (uc *CreateJobTemplatePhaseUseCase) validateBusinessRules(ctx context.Context, data *pb.JobTemplatePhase) error {
	if data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_template_phase.validation.data_required", "[ERR-DEFAULT] Job template phase data is required"))
	}
	if data.Name == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_template_phase.validation.name_required", "[ERR-DEFAULT] Job template phase name is required"))
	}
	if data.JobTemplateId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_template_phase.validation.job_template_id_required", "[ERR-DEFAULT] Job template ID is required"))
	}
	if len(data.Name) > 100 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_template_phase.validation.name_too_long", "[ERR-DEFAULT] Job template phase name is too long"))
	}

	return nil
}
