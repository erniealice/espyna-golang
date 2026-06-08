package plan

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/registry/entityid"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	planpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/plan"
	priceplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_plan"
)

// UpdatePlanRepositories groups all repository dependencies
type UpdatePlanRepositories struct {
	Plan      planpb.PlanDomainServiceServer           // Primary entity repository
	PricePlan priceplanpb.PricePlanDomainServiceServer // Cascade target for client_id sync (plan §3.2)
}

// UpdatePlanServices groups all business service dependencies
type UpdatePlanServices struct {
	Authorizer       ports.Authorizer // Current: RBAC and permissions
	Transactor       ports.Transactor // Current: Database transactions
	Translator       ports.Translator
	ReferenceChecker ports.ReferenceChecker // Plan §3.1 — client_id reassignment guard
}

// UpdatePlanUseCase handles the business logic for updating plans
type UpdatePlanUseCase struct {
	repositories UpdatePlanRepositories
	services     UpdatePlanServices
}

// NewUpdatePlanUseCase creates a new UpdatePlanUseCase
func NewUpdatePlanUseCase(
	repositories UpdatePlanRepositories,
	services UpdatePlanServices,
) *UpdatePlanUseCase {
	return &UpdatePlanUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the update plan operation
func (uc *UpdatePlanUseCase) Execute(ctx context.Context, req *planpb.UpdatePlanRequest) (*planpb.UpdatePlanResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityid.Plan, entityid.ActionUpdate); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Business logic and enrichment
	if err := uc.enrichPlanData(req.Data); err != nil {
		return nil, err
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		return nil, err
	}

	// Transaction wrap when supported — needed for the client_id cascade to
	// child PricePlans (plan §3.2).
	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		var result *planpb.UpdatePlanResponse
		txErr := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
			res, err := uc.executeCore(txCtx, req)
			if err != nil {
				return err
			}
			result = res
			return nil
		})
		if txErr != nil {
			return nil, txErr
		}
		return result, nil
	}
	return uc.executeCore(ctx, req)
}

// executeCore enforces the client_id reassignment guard (§3.1), the Plan →
// PricePlan client_id cascade (§3.2), and finally writes the Plan row.
func (uc *UpdatePlanUseCase) executeCore(ctx context.Context, req *planpb.UpdatePlanRequest) (*planpb.UpdatePlanResponse, error) {
	planID := ""
	if req.Data.Id != nil {
		planID = *req.Data.Id
	}

	// Read the existing plan — needed for both the client_id-change guard and
	// the cascade decision.
	existing, err := uc.readExistingPlan(ctx, planID)
	if err != nil {
		return nil, err
	}

	oldClientID := existing.GetClientId()
	newClientID := req.Data.GetClientId()

	clientIDChanging := oldClientID != newClientID

	// §3.1 — block client_id reassignment when any child PricePlan is attached
	// to an active subscription.
	if clientIDChanging && uc.services.ReferenceChecker != nil {
		locked, refErr := uc.services.ReferenceChecker.GetPlanClientScopeLockedIDs(ctx, []string{planID})
		if refErr != nil {
			return nil, refErr
		}
		if locked[planID] {
			msg := contextutil.GetTranslatedMessageWithContext(
				ctx, uc.services.Translator,
				"plan.errors.clientScopeLocked",
				"Cannot change this plan's client scope while one or more of its price plans is attached to an active subscription. Detach the subscriptions first or create a new plan. [DEFAULT]",
			)
			return nil, errors.New(msg)
		}
	}

	// parent_id is immutable after insert. Always overwrite req.Data with the
	// stored value regardless of body input — the only path that ever sets
	// parent_id is CustomizePlanForClient. This makes the field invisible to
	// the update flow and defends against API misuse.
	req.Data.ParentId = existing.ParentId

	// Persist the Plan update first.
	resp, err := uc.repositories.Plan.UpdatePlan(ctx, req)
	if err != nil {
		log.Printf("UpdatePlan repo error: planID=%s clientIDChanging=%v old=%q new=%q parentID=%v err=%v",
			planID, clientIDChanging, oldClientID, newClientID, req.Data.ParentId, err)
		return nil, fmt.Errorf("%s: %w",
			contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "plan.errors.update_failed", "plan update failed"),
			err)
	}

	// §3.2 cascade — propagate the new client_id to every child PricePlan.
	// Skipped when client_id is unchanged or PricePlan repo is unwired (some
	// callers don't need cascade in their composition).
	if clientIDChanging && uc.repositories.PricePlan != nil {
		if cascadeErr := uc.cascadeClientIDToPricePlans(ctx, planID, newClientID); cascadeErr != nil {
			return nil, cascadeErr
		}
	}

	return resp, nil
}

// readExistingPlan loads the plan currently in the store. Returns a "plan not
// found" translated error when the row does not exist.
func (uc *UpdatePlanUseCase) readExistingPlan(ctx context.Context, planID string) (*planpb.Plan, error) {
	id := planID
	resp, err := uc.repositories.Plan.ReadPlan(ctx, &planpb.ReadPlanRequest{
		Data: &planpb.Plan{Id: &id},
	})
	if err != nil || resp == nil || len(resp.GetData()) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"plan.errors.not_found",
			"plan not found [DEFAULT]",
		))
	}
	return resp.GetData()[0], nil
}

// cascadeClientIDToPricePlans sets every child PricePlan's client_id to match
// the parent Plan. Empty newClientID = revert to master (NULL on the column —
// represented by an empty string on the proto getter). Best-effort per row;
// the first failure aborts the cascade and bubbles up so the surrounding
// transaction can roll back.
func (uc *UpdatePlanUseCase) cascadeClientIDToPricePlans(ctx context.Context, planID, newClientID string) error {
	listResp, err := uc.repositories.PricePlan.ListPricePlans(ctx, &priceplanpb.ListPricePlansRequest{})
	if err != nil || listResp == nil {
		return nil
	}
	for _, pp := range listResp.GetData() {
		if pp.GetPlanId() != planID {
			continue
		}
		if pp.GetClientId() == newClientID {
			continue
		}
		pp.ClientId = stringPtrOrNil(newClientID)
		if _, updErr := uc.repositories.PricePlan.UpdatePricePlan(ctx, &priceplanpb.UpdatePricePlanRequest{
			Data: pp,
		}); updErr != nil {
			return updErr
		}
	}
	return nil
}

// stringPtrOrNil maps "" → nil (representing NULL in the underlying column),
// any non-empty value → its pointer. Used only for the optional client_id
// proto field.
func stringPtrOrNil(s string) *string {
	if s == "" {
		return nil
	}
	v := s
	return &v
}

// validateInput validates the input request
func (uc *UpdatePlanUseCase) validateInput(ctx context.Context, req *planpb.UpdatePlanRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "plan.validation.request_required", "request is required"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "plan.validation.data_required", "plan data is required"))
	}
	if req.Data.Id == nil || *req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "plan.validation.id_required", "plan ID is required"))
	}
	return nil
}

// enrichPlanData adds audit information for updates
func (uc *UpdatePlanUseCase) enrichPlanData(plan *planpb.Plan) error {
	now := time.Now()

	// Update modification timestamp
	plan.DateModified = &[]int64{now.UnixMilli()}[0]
	plan.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return nil
}

// validateBusinessRules enforces business constraints
func (uc *UpdatePlanUseCase) validateBusinessRules(ctx context.Context, plan *planpb.Plan) error {
	// Validate plan ID format
	if plan.Id == nil || len(*plan.Id) < 3 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "plan.validation.id_too_short", "plan ID must be at least 3 characters long"))
	}

	// Validate name is required
	if strings.TrimSpace(plan.Name) == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "plan.validation.name_required", "plan name is required"))
	}

	if len(plan.Name) > 100 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "plan.validation.name_too_long", "plan name cannot exceed 100 characters"))
	}

	// Validate description length (only if provided)
	if plan.Description != nil && len(*plan.Description) > 500 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "plan.validation.description_too_long", "plan description cannot exceed 500 characters"))
	}

	return nil
}
