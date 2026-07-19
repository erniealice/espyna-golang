package outcome_criteria

import (
	"context"
	"errors"
	"strings"
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

// checkCodeInvariants enforces the code contract against the MERGED post-update
// row: a field omitted from a partial update keeps its stored value, so the
// invariants must consider the row the UPDATE will actually produce, not just
// the supplied fields. This closes the re-home route (codex wave1-q1 finding
// 2B's non-concurrent leg) where moving a coded row to a fresh group WITHOUT
// resupplying its code escaped checking entirely.
//
// With effectiveCode = supplied code (normalized) else the stored code, and
// effectiveGroup = supplied criteria_group_id else the stored one:
//   - lineage stability: effectiveGroup's established code (if any) must equal
//     effectiveCode — mutating an established code, or re-homing a coded row
//     under a group anchored to a different code, is rejected. Filling a NULL
//     code with the established (or a fresh) code is allowed. Only CODED
//     versions anchor a lineage; NULL-code siblings are exempt by design.
//   - domain uniqueness: no OTHER criteria_group in the EXISTING row's (scope,
//     workspace_id, industry_code) domain may hold effectiveCode — across ALL
//     version statuses and both active states.
//
// The domain comes from the server-read existing row (trusted — never the
// client payload). Reads are single-statement anchor point lookups (bounded
// paginated fallback) and fail closed; the DB criteria_group anchor (composite
// FK + domain unique index) remains the authoritative race backstop. No-op for
// uncoded rows that stay uncoded.
func (uc *UpdateOutcomeCriteriaUseCase) checkCodeInvariants(ctx context.Context, data, existing *pb.OutcomeCriteria) error {
	if existing == nil {
		return nil
	}
	effectiveCode := existing.GetCode()
	if data.Code != nil && *data.Code != "" {
		effectiveCode = *data.Code
	}
	if effectiveCode == "" {
		return nil
	}
	effectiveGroup := existing.GetCriteriaGroupId()
	if data.GetCriteriaGroupId() != "" {
		effectiveGroup = data.GetCriteriaGroupId()
	}
	if strings.TrimSpace(effectiveGroup) == "" {
		// A coded row must belong to a real lineage (mirror of the CREATE-side
		// finding-11 guard): nothing to anchor the code to.
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "outcome_criteria.validation.code_requires_group", "[ERR-DEFAULT] Outcome criteria with a code requires a criteria group (criteria_group_id)"))
	}

	established, err := lineageEstablishedCode(ctx, uc.repositories.OutcomeCriteria, effectiveGroup)
	if err != nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "outcome_criteria.errors.code_check_failed", "[ERR-DEFAULT] Failed to validate outcome criteria code uniqueness"))
	}
	if established != "" && established != effectiveCode {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "outcome_criteria.validation.code_immutable", "[ERR-DEFAULT] Outcome criteria code cannot be changed once established for its lineage"))
	}

	// NEW-1 (Q1 re-review): the MERGED post-update row's (scope, workspace,
	// industry) domain must match the effective group's claimed domain. A
	// partial update keeps the stored value for any omitted field, so the
	// merged domain is: supplied scope (non-zero) else stored; supplied
	// industry (non-nil) else stored; the workspace is ALWAYS the server-read
	// existing row's (tenant immutable on update — the persistence decorator
	// strips any client-supplied workspace). Without this, an update supplying
	// a divergent scope/industry onto a coded row escapes both app checks and
	// the DB (the trigger never re-stamps an existing anchor; the domain unique
	// only fires on new anchors). NULL-code rows that stay uncoded are exempt
	// (early return above); fail closed on read errors.
	mergedDomain := &pb.OutcomeCriteria{
		Scope:        existing.GetScope(),
		WorkspaceId:  existing.WorkspaceId,
		IndustryCode: existing.IndustryCode,
	}
	if data.GetScope() != 0 {
		mergedDomain.Scope = data.GetScope()
	}
	if data.IndustryCode != nil {
		mergedDomain.IndustryCode = data.IndustryCode
	}
	scopeKey, wsKey, indKey, claimed, err := lineageClaimedDomain(ctx, uc.repositories.OutcomeCriteria, effectiveGroup)
	if err != nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "outcome_criteria.errors.code_check_failed", "[ERR-DEFAULT] Failed to validate outcome criteria code uniqueness"))
	}
	if claimed && criteriaDomainDiverges(mergedDomain, scopeKey, wsKey, indKey) {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "outcome_criteria.validation.code_domain_mismatch", "[ERR-DEFAULT] Outcome criteria scope/workspace/industry does not match the domain established for its criteria group"))
	}

	// Collision self-exclusion must use the EFFECTIVE lineage: on a re-home the
	// OLD group's standing domain claim is a real collision, not "self".
	merged := &pb.OutcomeCriteria{
		CriteriaGroupId: effectiveGroup,
		Scope:           existing.GetScope(),
		WorkspaceId:     existing.WorkspaceId,
		IndustryCode:    existing.IndustryCode,
	}
	taken, err := codeTakenByAnotherGroup(ctx, uc.repositories.OutcomeCriteria, effectiveCode, merged)
	if err != nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "outcome_criteria.errors.code_check_failed", "[ERR-DEFAULT] Failed to validate outcome criteria code uniqueness"))
	}
	if taken {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "outcome_criteria.errors.code_collision", "[ERR-DEFAULT] Outcome criteria code is already in use by another criterion in this domain"))
	}
	return nil
}
