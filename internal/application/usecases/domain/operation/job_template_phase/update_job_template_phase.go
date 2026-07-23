package job_template_phase

import (
	"context"
	"errors"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	jobtemplatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template_phase"
	scoringschemepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/scoring_scheme"
)

// UpdateJobTemplatePhaseRepositories groups repository dependencies. JobTemplate
// (owning template) and ScoringScheme anchor the fail-closed cross-workspace FK
// guards (red-team HIGH #2).
type UpdateJobTemplatePhaseRepositories struct {
	JobTemplatePhase pb.JobTemplatePhaseDomainServiceServer
	JobTemplate      jobtemplatepb.JobTemplateDomainServiceServer
	ScoringScheme    scoringschemepb.ScoringSchemeDomainServiceServer
}

type UpdateJobTemplatePhaseServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// UpdateJobTemplatePhaseUseCase handles the business logic for updating job template phases
type UpdateJobTemplatePhaseUseCase struct {
	repositories UpdateJobTemplatePhaseRepositories
	services     UpdateJobTemplatePhaseServices
}

// NewUpdateJobTemplatePhaseUseCase creates a new UpdateJobTemplatePhaseUseCase
func NewUpdateJobTemplatePhaseUseCase(
	repositories UpdateJobTemplatePhaseRepositories,
	services UpdateJobTemplatePhaseServices,
) *UpdateJobTemplatePhaseUseCase {
	return &UpdateJobTemplatePhaseUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the update job template phase operation
func (uc *UpdateJobTemplatePhaseUseCase) Execute(ctx context.Context, req *pb.UpdateJobTemplatePhaseRequest) (*pb.UpdateJobTemplatePhaseResponse, error) {
	// Authorization check
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.JobTemplatePhase,
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
func (uc *UpdateJobTemplatePhaseUseCase) executeWithTransaction(ctx context.Context, req *pb.UpdateJobTemplatePhaseRequest, enrichedData *pb.JobTemplatePhase) (*pb.UpdateJobTemplatePhaseResponse, error) {
	var result *pb.UpdateJobTemplatePhaseResponse

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

// executeCore contains the core business logic for updating a job template phase
func (uc *UpdateJobTemplatePhaseUseCase) executeCore(ctx context.Context, req *pb.UpdateJobTemplatePhaseRequest, enrichedData *pb.JobTemplatePhase) (*pb.UpdateJobTemplatePhaseResponse, error) {
	// First, check if the entity exists
	existingResp, err := uc.repositories.JobTemplatePhase.ReadJobTemplatePhase(ctx, &pb.ReadJobTemplatePhaseRequest{
		Data: &pb.JobTemplatePhase{Id: req.Data.Id},
	})
	if err != nil || existingResp == nil || len(existingResp.GetData()) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "job_template_phase.errors.not_found", "[ERR-DEFAULT] Job template phase not found"))
	}
	existing := existingResp.GetData()[0]

	// Fail-closed cross-workspace FK guard on the effective (post-update) FKs.
	// The caller must own the existing phase (its owning template in-workspace),
	// and any repointed job_template_id / scoring_scheme_id must be too. FKs are
	// mutable — an omitted field keeps the persisted value.
	wsID := contextutil.ExtractWorkspaceIDFromContext(ctx)
	if err := requireOwningTemplateInWorkspace(ctx, uc.repositories.JobTemplate, uc.services.Translator, wsID, existing.GetJobTemplateId()); err != nil {
		return nil, err
	}
	effectiveTemplateID := enrichedData.GetJobTemplateId()
	if effectiveTemplateID == "" {
		effectiveTemplateID = existing.GetJobTemplateId()
	}
	if effectiveTemplateID != existing.GetJobTemplateId() {
		if err := requireOwningTemplateInWorkspace(ctx, uc.repositories.JobTemplate, uc.services.Translator, wsID, effectiveTemplateID); err != nil {
			return nil, err
		}
	}
	effectiveScheme := enrichedData.GetScoringSchemeId()
	if effectiveScheme == "" {
		effectiveScheme = existing.GetScoringSchemeId()
	}
	if err := requireScoringSchemeInWorkspace(ctx, uc.repositories.ScoringScheme, uc.services.Translator, wsID, effectiveScheme); err != nil {
		return nil, err
	}

	resp, err := uc.repositories.JobTemplatePhase.UpdateJobTemplatePhase(ctx, &pb.UpdateJobTemplatePhaseRequest{
		Data: enrichedData,
	})
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "job_template_phase.errors.update_failed", "[ERR-DEFAULT] Job template phase update failed"))
	}
	return resp, nil
}

// applyBusinessLogic applies business rules and returns enriched data
func (uc *UpdateJobTemplatePhaseUseCase) applyBusinessLogic(data *pb.JobTemplatePhase) *pb.JobTemplatePhase {
	now := time.Now()

	// Business logic: Update modification audit fields
	data.DateModified = &[]int64{now.UnixMilli()}[0]
	data.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return data
}

// validateInput validates the input request
func (uc *UpdateJobTemplatePhaseUseCase) validateInput(ctx context.Context, req *pb.UpdateJobTemplatePhaseRequest) error {
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

// validateBusinessRules enforces business constraints
func (uc *UpdateJobTemplatePhaseUseCase) validateBusinessRules(ctx context.Context, data *pb.JobTemplatePhase) error {
	if data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "job_template_phase.validation.data_required", "[ERR-DEFAULT] Job template phase data is required"))
	}
	if data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "job_template_phase.validation.id_required", "[ERR-DEFAULT] Job template phase ID is required"))
	}
	// Validate Name only if provided (partial update support)
	if data.Name != "" && len(data.Name) > 100 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "job_template_phase.validation.name_too_long", "[ERR-DEFAULT] Job template phase name is too long"))
	}

	return nil
}
