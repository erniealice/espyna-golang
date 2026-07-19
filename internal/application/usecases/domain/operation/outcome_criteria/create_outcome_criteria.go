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

type CreateOutcomeCriteriaRepositories struct {
	OutcomeCriteria pb.OutcomeCriteriaDomainServiceServer
}

type CreateOutcomeCriteriaServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
}

// CreateOutcomeCriteriaUseCase handles the business logic for creating outcome criteria
type CreateOutcomeCriteriaUseCase struct {
	repositories CreateOutcomeCriteriaRepositories
	services     CreateOutcomeCriteriaServices
}

// NewCreateOutcomeCriteriaUseCase creates a new CreateOutcomeCriteriaUseCase
func NewCreateOutcomeCriteriaUseCase(
	repositories CreateOutcomeCriteriaRepositories,
	services CreateOutcomeCriteriaServices,
) *CreateOutcomeCriteriaUseCase {
	return &CreateOutcomeCriteriaUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the create outcome criteria operation
func (uc *CreateOutcomeCriteriaUseCase) Execute(ctx context.Context, req *pb.CreateOutcomeCriteriaRequest) (*pb.CreateOutcomeCriteriaResponse, error) {
	// Authorization check
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.OutcomeCriteria,
		Action: entityid.ActionCreate,
	}); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "outcome_criteria.validation.data_required", "[ERR-DEFAULT] Outcome criteria data is required"))
	}

	// Stamp the authoritative tenant BEFORE any validation or invariant read
	// (codex wave1-q1 finding 10): the persistence decorator strips any
	// client-supplied workspace and injects the trusted context value at write
	// time, so validating against the request's own workspace let an omitted or
	// forged value false-allow a same-domain collision. Stamping first makes the
	// checked object and the persisted object carry the SAME tenant key.
	stampTrustedWorkspace(ctx, req.Data)

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

// executeWithTransaction executes creation within a transaction
func (uc *CreateOutcomeCriteriaUseCase) executeWithTransaction(ctx context.Context, req *pb.CreateOutcomeCriteriaRequest, enrichedData *pb.OutcomeCriteria) (*pb.CreateOutcomeCriteriaResponse, error) {
	var result *pb.CreateOutcomeCriteriaResponse
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

// executeCore contains the core business logic for creating an outcome criteria
func (uc *CreateOutcomeCriteriaUseCase) executeCore(ctx context.Context, req *pb.CreateOutcomeCriteriaRequest, enrichedData *pb.OutcomeCriteria) (*pb.CreateOutcomeCriteriaResponse, error) {
	// Enforce code-lineage/collision invariants inside the (transactional) core.
	// The reads are single-statement point lookups against the criteria_group
	// anchor (each internally consistent); cross-statement serialization is NOT
	// claimed here — the anchor's composite FK + domain unique index are the
	// authoritative race backstops, and this is the friendlier early rejection.
	if err := uc.checkCodeInvariants(ctx, enrichedData); err != nil {
		return nil, err
	}

	resp, err := uc.repositories.OutcomeCriteria.CreateOutcomeCriteria(ctx, &pb.CreateOutcomeCriteriaRequest{
		Data: enrichedData,
	})
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "outcome_criteria.errors.creation_failed", "[ERR-DEFAULT] Outcome criteria creation failed"))
	}
	return resp, nil
}

// applyBusinessLogic applies business rules and returns enriched data
func (uc *CreateOutcomeCriteriaUseCase) applyBusinessLogic(data *pb.OutcomeCriteria) *pb.OutcomeCriteria {
	now := time.Now()

	// Business logic: Generate ID if not provided
	if data.Id == "" {
		data.Id = uc.services.IDGenerator.GenerateID()
	}

	// Business logic: Set active status for new criteria
	data.Active = true

	// Business logic: Set creation audit fields
	data.DateCreated = &[]int64{now.UnixMilli()}[0]
	data.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	data.DateModified = &[]int64{now.UnixMilli()}[0]
	data.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return data
}

// validateBusinessRules enforces business constraints
func (uc *CreateOutcomeCriteriaUseCase) validateBusinessRules(ctx context.Context, data *pb.OutcomeCriteria) error {
	if data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "outcome_criteria.validation.data_required", "[ERR-DEFAULT] Outcome criteria data is required"))
	}
	if data.Name == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "outcome_criteria.validation.name_required", "[ERR-DEFAULT] Outcome criteria name is required"))
	}
	if len(data.Name) > 100 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "outcome_criteria.validation.name_too_long", "[ERR-DEFAULT] Outcome criteria name is too long"))
	}

	// Normalize + shape-validate the code when supplied (mirror of the DB CHECK).
	if data.Code != nil && *data.Code != "" {
		norm, ok := normalizeCriteriaCode(*data.Code)
		if !ok {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "outcome_criteria.validation.code_invalid", "[ERR-DEFAULT] Outcome criteria code must be a normalized path segment (lowercase letter, then letters/digits/underscores)"))
		}
		data.Code = &norm

		// A coded criterion MUST belong to a real lineage (codex wave1-q1
		// finding 11): a code is anchored per criteria_group, so a coded create
		// with no group id has nothing to anchor to — the DB trigger would
		// reject it with an opaque constraint error; reject it here with a
		// useful validation error instead.
		if strings.TrimSpace(data.CriteriaGroupId) == "" {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "outcome_criteria.validation.code_requires_group", "[ERR-DEFAULT] Outcome criteria with a code requires a criteria group (criteria_group_id)"))
		}
	}

	return nil
}

// stampTrustedWorkspace overwrites the proposal's workspace with the
// authoritative value from the request context — the SAME trusted source the
// persistence decorator injects at write time (WorkspaceAwareOperations reads
// identity from ctx; contextutil.ExtractWorkspaceIDFromContext reads identity
// first, then the server-stamped legacy keys). When the context carries no
// workspace (service-to-service call) the supplied value stands, matching the
// decorator's pass-through. MUST run before any validation or invariant read.
func stampTrustedWorkspace(ctx context.Context, data *pb.OutcomeCriteria) {
	if data == nil {
		return
	}
	if ws := contextutil.ExtractWorkspaceIDFromContext(ctx); ws != "" {
		data.WorkspaceId = &ws
	}
}

// checkCodeInvariants enforces, when the create supplies a (normalized) code:
//   - lineage stability: a create adding a NEW version under an EXISTING
//     criteria_group must carry that group's already-established code (a
//     brand-new group has none → any code is allowed). Only CODED versions
//     anchor a lineage; NULL-code siblings are exempt by design.
//   - domain uniqueness: a DIFFERENT criteria_group in the same (scope,
//     workspace_id, industry_code) domain must not already hold the code —
//     across ALL version statuses and both active states.
//
// Reads are single-statement anchor point lookups (bounded paginated fallback
// for providers without the seam) and fail closed; the DB criteria_group
// anchor (composite FK + domain unique index) remains the authoritative race
// backstop. No-op when no code is supplied. data.WorkspaceId MUST already be
// trusted-stamped (stampTrustedWorkspace).
func (uc *CreateOutcomeCriteriaUseCase) checkCodeInvariants(ctx context.Context, data *pb.OutcomeCriteria) error {
	if data.Code == nil || *data.Code == "" {
		return nil
	}
	newCode := *data.Code

	// Defense in depth for direct callers: validateBusinessRules already
	// rejects a coded create without a lineage.
	if strings.TrimSpace(data.CriteriaGroupId) == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "outcome_criteria.validation.code_requires_group", "[ERR-DEFAULT] Outcome criteria with a code requires a criteria group (criteria_group_id)"))
	}

	established, err := lineageEstablishedCode(ctx, uc.repositories.OutcomeCriteria, data.CriteriaGroupId)
	if err != nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "outcome_criteria.errors.code_check_failed", "[ERR-DEFAULT] Failed to validate outcome criteria code uniqueness"))
	}
	if established != "" && established != newCode {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "outcome_criteria.validation.code_immutable", "[ERR-DEFAULT] Outcome criteria code cannot be changed once established for its lineage"))
	}

	// NEW-1 (Q1 re-review): a coded write into an EXISTING group must carry the
	// group's claimed (scope, workspace, industry) domain. The DB cannot catch
	// divergence here — the populate trigger's ON CONFLICT DO NOTHING never
	// re-stamps an existing anchor and uq_criteria_group_domain_code only fires
	// on NEW anchors — so a divergent-domain coded version would silently ride
	// an anchor claiming a different domain. Mirror of the code-immutability
	// rejection; NULL-code writes are exempt (early return above); fail closed.
	scopeKey, wsKey, indKey, claimed, err := lineageClaimedDomain(ctx, uc.repositories.OutcomeCriteria, data.CriteriaGroupId)
	if err != nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "outcome_criteria.errors.code_check_failed", "[ERR-DEFAULT] Failed to validate outcome criteria code uniqueness"))
	}
	if claimed && criteriaDomainDiverges(data, scopeKey, wsKey, indKey) {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "outcome_criteria.validation.code_domain_mismatch", "[ERR-DEFAULT] Outcome criteria scope/workspace/industry does not match the domain established for its criteria group"))
	}

	taken, err := codeTakenByAnotherGroup(ctx, uc.repositories.OutcomeCriteria, newCode, data)
	if err != nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "outcome_criteria.errors.code_check_failed", "[ERR-DEFAULT] Failed to validate outcome criteria code uniqueness"))
	}
	if taken {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "outcome_criteria.errors.code_collision", "[ERR-DEFAULT] Outcome criteria code is already in use by another criterion in this domain"))
	}
	return nil
}
