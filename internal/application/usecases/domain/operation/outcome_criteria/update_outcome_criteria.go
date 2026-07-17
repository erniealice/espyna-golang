package outcome_criteria

import (
	"context"
	"errors"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/outcome_criteria"
)

type UpdateOutcomeCriteriaRepositories struct {
	OutcomeCriteria pb.OutcomeCriteriaDomainServiceServer
}

type UpdateOutcomeCriteriaServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// UpdateOutcomeCriteriaUseCase handles the business logic for updating outcome criteria
type UpdateOutcomeCriteriaUseCase struct {
	repositories UpdateOutcomeCriteriaRepositories
	services     UpdateOutcomeCriteriaServices
}

// NewUpdateOutcomeCriteriaUseCase creates a new UpdateOutcomeCriteriaUseCase
func NewUpdateOutcomeCriteriaUseCase(
	repositories UpdateOutcomeCriteriaRepositories,
	services UpdateOutcomeCriteriaServices,
) *UpdateOutcomeCriteriaUseCase {
	return &UpdateOutcomeCriteriaUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the update outcome criteria operation
func (uc *UpdateOutcomeCriteriaUseCase) Execute(ctx context.Context, req *pb.UpdateOutcomeCriteriaRequest) (*pb.UpdateOutcomeCriteriaResponse, error) {
	// Authorization check
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.OutcomeCriteria,
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
func (uc *UpdateOutcomeCriteriaUseCase) executeWithTransaction(ctx context.Context, req *pb.UpdateOutcomeCriteriaRequest, enrichedData *pb.OutcomeCriteria) (*pb.UpdateOutcomeCriteriaResponse, error) {
	var result *pb.UpdateOutcomeCriteriaResponse

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

// executeCore contains the core business logic for updating an outcome criteria
func (uc *UpdateOutcomeCriteriaUseCase) executeCore(ctx context.Context, req *pb.UpdateOutcomeCriteriaRequest, enrichedData *pb.OutcomeCriteria) (*pb.UpdateOutcomeCriteriaResponse, error) {
	// First, check if the entity exists (and capture the current row — it carries
	// the authoritative lineage + domain the code invariants are checked against).
	readResp, err := uc.repositories.OutcomeCriteria.ReadOutcomeCriteria(ctx, &pb.ReadOutcomeCriteriaRequest{
		Data: &pb.OutcomeCriteria{Id: req.Data.Id},
	})
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "outcome_criteria.errors.not_found", "[ERR-DEFAULT] Outcome criteria not found"))
	}
	var existing *pb.OutcomeCriteria
	if readResp != nil && len(readResp.Data) > 0 {
		existing = readResp.Data[0]
	}

	// Enforce code-lineage/collision invariants inside the (transactional) core.
	if err := uc.checkCodeInvariants(ctx, enrichedData, existing); err != nil {
		return nil, err
	}

	resp, err := uc.repositories.OutcomeCriteria.UpdateOutcomeCriteria(ctx, &pb.UpdateOutcomeCriteriaRequest{
		Data: enrichedData,
	})
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "outcome_criteria.errors.update_failed", "[ERR-DEFAULT] Outcome criteria update failed"))
	}
	return resp, nil
}

// applyBusinessLogic applies business rules and returns enriched data
func (uc *UpdateOutcomeCriteriaUseCase) applyBusinessLogic(data *pb.OutcomeCriteria) *pb.OutcomeCriteria {
	now := time.Now()

	// Business logic: Update modification audit fields
	data.DateModified = &[]int64{now.UnixMilli()}[0]
	data.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return data
}

// validateInput validates the input request
func (uc *UpdateOutcomeCriteriaUseCase) validateInput(ctx context.Context, req *pb.UpdateOutcomeCriteriaRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "outcome_criteria.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "outcome_criteria.validation.data_required", "[ERR-DEFAULT] Outcome criteria data is required"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "outcome_criteria.validation.id_required", "[ERR-DEFAULT] Outcome criteria ID is required"))
	}
	return nil
}

// validateBusinessRules enforces business constraints
func (uc *UpdateOutcomeCriteriaUseCase) validateBusinessRules(ctx context.Context, data *pb.OutcomeCriteria) error {
	if data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "outcome_criteria.validation.data_required", "[ERR-DEFAULT] Outcome criteria data is required"))
	}
	if data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "outcome_criteria.validation.id_required", "[ERR-DEFAULT] Outcome criteria ID is required"))
	}
	// Validate Name only if provided (partial update support)
	if data.Name != "" && len(data.Name) > 100 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "outcome_criteria.validation.name_too_long", "[ERR-DEFAULT] Outcome criteria name is too long"))
	}

	// Normalize + shape-validate the code when supplied (mirror of the DB CHECK).
	if data.Code != nil && *data.Code != "" {
		norm, ok := normalizeCriteriaCode(*data.Code)
		if !ok {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "outcome_criteria.validation.code_invalid", "[ERR-DEFAULT] Outcome criteria code must be a normalized path segment (lowercase letter, then letters/digits/underscores)"))
		}
		data.Code = &norm
	}

	return nil
}

// checkCodeInvariants enforces, when the update supplies a (normalized) code:
//   - lineage stability: once a criteria_group has an established non-NULL code,
//     the code cannot be mutated to a DIFFERENT one (filling NULL -> the
//     established code is allowed);
//   - domain uniqueness: a DIFFERENT criteria_group in the same (scope,
//     workspace_id, industry_code) domain must not already hold the code.
//
// Both checks read against the EXISTING row's authoritative lineage/domain. The
// DB partial unique index remains the ultimate race/backstop (and covers the rare
// re-home-plus-code edge this pre-check does not). No-op when no code is supplied.
func (uc *UpdateOutcomeCriteriaUseCase) checkCodeInvariants(ctx context.Context, data, existing *pb.OutcomeCriteria) error {
	if data.Code == nil || *data.Code == "" {
		return nil
	}
	if existing == nil {
		return nil
	}
	newCode := *data.Code

	established, err := lineageEstablishedCode(ctx, uc.repositories.OutcomeCriteria, existing.CriteriaGroupId)
	if err != nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "outcome_criteria.errors.code_check_failed", "[ERR-DEFAULT] Failed to validate outcome criteria code uniqueness"))
	}
	if established != "" && established != newCode {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "outcome_criteria.validation.code_immutable", "[ERR-DEFAULT] Outcome criteria code cannot be changed once established for its lineage"))
	}

	taken, err := codeTakenByAnotherGroup(ctx, uc.repositories.OutcomeCriteria, newCode, existing)
	if err != nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "outcome_criteria.errors.code_check_failed", "[ERR-DEFAULT] Failed to validate outcome criteria code uniqueness"))
	}
	if taken {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "outcome_criteria.errors.code_collision", "[ERR-DEFAULT] Outcome criteria code is already in use by another criterion in this domain"))
	}
	return nil
}
