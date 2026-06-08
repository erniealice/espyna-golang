package plan

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/registry/entityid"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"

	clientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client"
	productplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_plan"
	planpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/plan"
	priceplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_plan"
	priceschedulepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_schedule"
	productpriceplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/product_price_plan"
	subscriptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription"
)

// CustomizePlanForClientRepositories aggregates every repository the use case
// touches. The use case orchestrates a single transaction across the Plan,
// PricePlan, ProductPlan, ProductPricePlan, PriceSchedule, and (optionally)
// Subscription domains; see plan §4 (20260427-plan-client-scope) for the full
// algorithm.
type CustomizePlanForClientRepositories struct {
	Plan             planpb.PlanDomainServiceServer
	ProductPlan      productplanpb.ProductPlanDomainServiceServer
	PricePlan        priceplanpb.PricePlanDomainServiceServer
	ProductPricePlan productpriceplanpb.ProductPricePlanDomainServiceServer
	PriceSchedule    priceschedulepb.PriceScheduleDomainServiceServer
	Subscription     subscriptionpb.SubscriptionDomainServiceServer
	Client           clientpb.ClientDomainServiceServer
}

// CustomizePlanForClientServices groups all business service dependencies.
type CustomizePlanForClientServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	IDGenerator ports.IDGenerator
}

// CustomizePlanForClientRequest carries the inputs to the customize flow.
//
// NewScheduleName is the pre-built display name used when a brand-new client
// PriceSchedule has to be created. The handler resolves the lyngua-driven
// suffix ("Rate Cards" vs "Price Schedule") and concatenates it with the
// client's display name; the use case stays free of localization concerns.
type CustomizePlanForClientRequest struct {
	SourcePlanID      string // Plan to clone from (master or another client's)
	SourcePricePlanID string // PricePlan to clone from. Determines the source PriceSchedule.
	ClientID          string // Target client_id stamped on every cloned row
	SubscriptionID    string // Optional: when set, repoint subscription.price_plan_id atomically
	NewScheduleName   string // Pre-built name used when a new PriceSchedule has to be created
}

// CustomizePlanForClientResponse carries the resolved entities produced by the
// flow. Reused = true iff a matching client PriceSchedule was found and reused
// (no new schedule row was inserted).
type CustomizePlanForClientResponse struct {
	Plan          *planpb.Plan
	PricePlan     *priceplanpb.PricePlan
	PriceSchedule *priceschedulepb.PriceSchedule
	Reused        bool
}

// CustomizePlanForClientUseCase clones a Plan tree under a client's namespace.
// The entire 8-step flow (read source → resolve-or-create schedule → insert
// Plan → ProductPlan rows + remap → PricePlan → ProductPricePlan rows with
// remap → optional subscription repoint) runs in a single
// Transactor.ExecuteInTransaction.
type CustomizePlanForClientUseCase struct {
	repositories CustomizePlanForClientRepositories
	services     CustomizePlanForClientServices
}

// NewCustomizePlanForClientUseCase wires the use case.
func NewCustomizePlanForClientUseCase(
	repositories CustomizePlanForClientRepositories,
	services CustomizePlanForClientServices,
) *CustomizePlanForClientUseCase {
	return &CustomizePlanForClientUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute orchestrates the customize-for-client flow.
//
// Takes/returns proto types per Phase 0 of the block-decouple plan; internal
// helpers (validateInput, executeCore) keep their Go-struct signatures and
// the proto request is translated at the boundary.
func (uc *CustomizePlanForClientUseCase) Execute(
	ctx context.Context, req *planpb.CustomizePlanForClientRequest,
) (*planpb.CustomizePlanForClientResponse, error) {
	// Authorization — revenue:create OR (plan:create + price_plan:create).
	// We require both plan:create AND price_plan:create; revenue:create is
	// not consulted here because the use case never writes Revenue rows.
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityid.Plan, entityid.ActionCreate); err != nil {
		return nil, err
	}
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityid.PricePlan, entityid.ActionCreate); err != nil {
		return nil, err
	}

	// Translate proto request to the internal Go-struct used by executeCore.
	internalReq := &CustomizePlanForClientRequest{
		SourcePlanID:      req.GetSourcePlanId(),
		SourcePricePlanID: req.GetSourcePricePlanId(),
		ClientID:          req.GetClientId(),
		SubscriptionID:    req.GetSubscriptionId(),
		NewScheduleName:   req.GetNewScheduleName(),
	}

	if err := uc.validateInput(ctx, internalReq); err != nil {
		return nil, err
	}

	// Run the entire clone in a single transaction. Any failure rolls back
	// every insert (including a freshly-created PriceSchedule), preventing
	// orphan rows.
	var coreResult *CustomizePlanForClientResponse
	if uc.services.Transactor != nil &&
		uc.services.Transactor.SupportsTransactions() {
		if err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
			res, execErr := uc.executeCore(txCtx, internalReq)
			if execErr != nil {
				return execErr
			}
			coreResult = res
			return nil
		}); err != nil {
			return nil, err
		}
	} else {
		var execErr error
		coreResult, execErr = uc.executeCore(ctx, internalReq)
		if execErr != nil {
			return nil, execErr
		}
	}

	return wrapCustomizePlanResponse(coreResult), nil
}

// wrapCustomizePlanResponse converts the internal Go-struct response to its
// proto representation. The Plan is exposed as a proto pointer; PricePlan
// and PriceSchedule surface via their IDs only (cross-package proto imports
// would circle back through plan.proto).
func wrapCustomizePlanResponse(r *CustomizePlanForClientResponse) *planpb.CustomizePlanForClientResponse {
	if r == nil {
		return &planpb.CustomizePlanForClientResponse{Success: true}
	}
	return &planpb.CustomizePlanForClientResponse{
		Success:            true,
		NewPlanId:          r.Plan.GetId(),
		NewPricePlanId:     r.PricePlan.GetId(),
		NewPriceScheduleId: r.PriceSchedule.GetId(),
		Reused:             r.Reused,
		Plan:               r.Plan,
	}
}

// validateInput checks for required IDs in the request. Localized
// translations are deferred to the same lyngua keys used elsewhere in the
// package.
func (uc *CustomizePlanForClientUseCase) validateInput(ctx context.Context, req *CustomizePlanForClientRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"plan.validation.request_required",
			"request is required [DEFAULT]",
		))
	}
	if req.SourcePlanID == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"plan.validation.source_plan_id_required",
			"source plan ID is required [DEFAULT]",
		))
	}
	if req.SourcePricePlanID == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"plan.validation.source_price_plan_id_required",
			"source price plan ID is required [DEFAULT]",
		))
	}
	if req.ClientID == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"plan.validation.client_id_required",
			"client ID is required [DEFAULT]",
		))
	}
	return nil
}

// executeCore runs steps 1–8 of plan §4.2 against the given (possibly
// transactional) context.
func (uc *CustomizePlanForClientUseCase) executeCore(
	ctx context.Context, req *CustomizePlanForClientRequest,
) (*CustomizePlanForClientResponse, error) {
	// 1. Read source Plan + PricePlan + ProductPlan rows + ProductPricePlan
	//    rows + parent PriceSchedule.
	sourcePlan, err := uc.readPlan(ctx, req.SourcePlanID)
	if err != nil {
		return nil, err
	}
	sourcePricePlan, err := uc.readPricePlan(ctx, req.SourcePricePlanID)
	if err != nil {
		return nil, err
	}

	// 2. Defense — sourcePricePlan.plan_id MUST match sourcePlan.id.
	if sourcePricePlan.GetPlanId() != sourcePlan.GetId() {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"plan.errors.source_mismatch",
			"source price plan does not belong to source plan [DEFAULT]",
		))
	}

	// Source PricePlan already customized for this client — surface a no-op
	// rather than producing a duplicate clone (per plan §11 risk row).
	if sourcePricePlan.GetClientId() != "" && sourcePricePlan.GetClientId() == req.ClientID {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"plan.errors.already_customized",
			"this package is already customized for the target client [DEFAULT]",
		))
	}

	// 3. Validate target client exists + is active.
	client, err := uc.readClient(ctx, req.ClientID)
	if err != nil {
		return nil, err
	}

	// SubscriptionID set + client mismatch → reject before any write.
	var existingSub *subscriptionpb.Subscription
	if req.SubscriptionID != "" {
		existingSub, err = uc.readSubscription(ctx, req.SubscriptionID)
		if err != nil {
			return nil, err
		}
		if existingSub.GetClientId() != req.ClientID {
			return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
				ctx, uc.services.Translator,
				"plan.errors.subscription_client_mismatch",
				"subscription belongs to a different client [DEFAULT]",
			))
		}
	}

	sourceSchedule := uc.readPriceScheduleOptional(ctx, sourcePricePlan.GetPriceScheduleId())

	// Workspace identity used by the schedule reuse lookup.
	workspaceID := contextutil.ExtractWorkspaceIDFromContext(ctx)

	// 4. Resolve-or-create PriceSchedule for the target client.
	derivedName := req.NewScheduleName
	if derivedName == "" {
		derivedName = uc.fallbackScheduleName(client)
	}
	resolvedSchedule, reused, err := ResolveOrCreateClientPriceSchedule(
		ctx,
		&ResolveOrCreateClientScheduleRepos{PriceSchedule: uc.repositories.PriceSchedule},
		uc.services.IDGenerator,
		workspaceID,
		scheduleLocationID(sourceSchedule),
		req.ClientID,
		derivedName,
		sourceSchedule,
	)
	if err != nil {
		return nil, fmt.Errorf("resolve-or-create client schedule: %w", err)
	}

	// 5. Insert Plan (clone).
	clonedPlan, err := uc.clonePlan(ctx, sourcePlan, req.ClientID, clientDisplayName(client))
	if err != nil {
		return nil, fmt.Errorf("clone plan: %w", err)
	}

	// 6. Insert ProductPlans (clone all from source) — build the remap.
	productPlanRemap, err := uc.cloneProductPlans(ctx, sourcePlan.GetId(), clonedPlan.GetId())
	if err != nil {
		return nil, fmt.Errorf("clone product plans: %w", err)
	}

	// 7. Insert PricePlan (clone).
	clonedPricePlan, err := uc.clonePricePlan(ctx, sourcePricePlan, clonedPlan.GetId(), resolvedSchedule.GetId(), req.ClientID, clientDisplayName(client))
	if err != nil {
		return nil, fmt.Errorf("clone price plan: %w", err)
	}

	// 8. Insert ProductPricePlans (clone all from source, remapping
	//    product_plan_id).
	if err := uc.cloneProductPricePlans(ctx, sourcePricePlan.GetId(), clonedPricePlan.GetId(), productPlanRemap); err != nil {
		return nil, fmt.Errorf("clone product price plans: %w", err)
	}

	// 9. Optional subscription repoint.
	if existingSub != nil {
		existingSub.PricePlanId = clonedPricePlan.GetId()
		now := time.Now()
		existingSub.DateModified = &[]int64{now.UnixMilli()}[0]
		existingSub.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
		if _, err := uc.repositories.Subscription.UpdateSubscription(ctx, &subscriptionpb.UpdateSubscriptionRequest{
			Data: existingSub,
		}); err != nil {
			return nil, fmt.Errorf("repoint subscription: %w", err)
		}
	}

	return &CustomizePlanForClientResponse{
		Plan:          clonedPlan,
		PricePlan:     clonedPricePlan,
		PriceSchedule: resolvedSchedule,
		Reused:        reused,
	}, nil
}

// ----- step helpers --------------------------------------------------------

func (uc *CustomizePlanForClientUseCase) readPlan(ctx context.Context, id string) (*planpb.Plan, error) {
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

func (uc *CustomizePlanForClientUseCase) readPricePlan(ctx context.Context, id string) (*priceplanpb.PricePlan, error) {
	resp, err := uc.repositories.PricePlan.ReadPricePlan(ctx, &priceplanpb.ReadPricePlanRequest{
		Data: &priceplanpb.PricePlan{Id: id},
	})
	if err != nil || resp == nil || len(resp.GetData()) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"price_plan.errors.not_found",
			"price plan not found [DEFAULT]",
		))
	}
	return resp.GetData()[0], nil
}

func (uc *CustomizePlanForClientUseCase) readClient(ctx context.Context, id string) (*clientpb.Client, error) {
	resp, err := uc.repositories.Client.ReadClient(ctx, &clientpb.ReadClientRequest{
		Data: &clientpb.Client{Id: id},
	})
	if err != nil || resp == nil || len(resp.GetData()) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"client.errors.not_found",
			"client not found [DEFAULT]",
		))
	}
	if !resp.GetData()[0].GetActive() {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"client.errors.not_active",
			"client is not active [DEFAULT]",
		))
	}
	return resp.GetData()[0], nil
}

func (uc *CustomizePlanForClientUseCase) readSubscription(ctx context.Context, id string) (*subscriptionpb.Subscription, error) {
	resp, err := uc.repositories.Subscription.ReadSubscription(ctx, &subscriptionpb.ReadSubscriptionRequest{
		Data: &subscriptionpb.Subscription{Id: id},
	})
	if err != nil || resp == nil || len(resp.GetData()) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"subscription.errors.not_found",
			"subscription not found [DEFAULT]",
		))
	}
	return resp.GetData()[0], nil
}

// readPriceScheduleOptional returns nil when the schedule cannot be resolved
// — the source PricePlan may not have one, in which case the resolve helper
// has to fall back to defaults for date/location bookkeeping.
func (uc *CustomizePlanForClientUseCase) readPriceScheduleOptional(ctx context.Context, id string) *priceschedulepb.PriceSchedule {
	if id == "" || uc.repositories.PriceSchedule == nil {
		return nil
	}
	resp, err := uc.repositories.PriceSchedule.ReadPriceSchedule(ctx, &priceschedulepb.ReadPriceScheduleRequest{
		Data: &priceschedulepb.PriceSchedule{Id: id},
	})
	if err != nil || resp == nil || len(resp.GetData()) == 0 {
		return nil
	}
	return resp.GetData()[0]
}

// clonePlan inserts the new Plan row stamped with the target client_id and
// a parent_id pointing at the master plan.
func (uc *CustomizePlanForClientUseCase) clonePlan(
	ctx context.Context, source *planpb.Plan, clientID, clientName string,
) (*planpb.Plan, error) {
	now := time.Now()
	newID := uc.generateID()
	clientCopy := clientID

	// Always flatten to the master — no grandchildren. If source is itself
	// a clone (source.GetParentId() != ""), we re-parent the new clone
	// to the SAME master. Two-level invariant enforced here, in code, since
	// the DB FK alone can't express it.
	parentID := source.GetParentId()
	if parentID == "" {
		parentID = source.GetId() // source is the master itself
	}

	clone := &planpb.Plan{
		Id:                 &newID,
		Name:               appendClientSuffix(source.GetName(), clientName),
		Description:        copyStringPtr(source.Description),
		Active:             true,
		ThumbnailUrl:       copyStringPtr(source.ThumbnailUrl),
		PlanLocations:      source.GetPlanLocations(),
		ClientId:           &clientCopy,
		ParentId:           &parentID,
		DateCreated:        ptrInt64(now.UnixMilli()),
		DateCreatedString:  ptrString(now.Format(time.RFC3339)),
		DateModified:       ptrInt64(now.UnixMilli()),
		DateModifiedString: ptrString(now.Format(time.RFC3339)),
	}
	resp, err := uc.repositories.Plan.CreatePlan(ctx, &planpb.CreatePlanRequest{Data: clone})
	if err != nil {
		return nil, err
	}
	if resp == nil || len(resp.GetData()) == 0 {
		return nil, errors.New("create plan returned no data")
	}
	return resp.GetData()[0], nil
}

// cloneProductPlans inserts a new ProductPlan row for every source row.
// Returns a map[oldProductPlanID]newProductPlanID used by the
// ProductPricePlan clone step to remap FK references.
func (uc *CustomizePlanForClientUseCase) cloneProductPlans(
	ctx context.Context, sourcePlanID, newPlanID string,
) (map[string]string, error) {
	remap := make(map[string]string)
	if uc.repositories.ProductPlan == nil {
		return remap, nil
	}
	listResp, err := uc.repositories.ProductPlan.ListByPlan(ctx, &productplanpb.ListProductPlansByPlanRequest{
		PlanId: sourcePlanID,
	})
	if err != nil || listResp == nil {
		return remap, nil
	}
	now := time.Now()
	for _, src := range listResp.GetProductPlans() {
		newID := uc.generateID()
		clone := &productplanpb.ProductPlan{
			Id:          newID,
			Name:        src.GetName(),
			Description: copyStringPtr(src.Description),
			Active:      true,
			ProductId:   src.GetProductId(),
			PlanId:      newPlanID,
			// ProductPlan.job_template_id (field 14) is reserved as of
			// 20260429 (auto-spawn-jobs-from-subscription) — the JobTemplate
			// anchor moved to Plan.job_template_id.
			ProductVariantId:   copyStringPtr(src.ProductVariantId),
			DateCreated:        ptrInt64(now.UnixMilli()),
			DateCreatedString:  ptrString(now.Format(time.RFC3339)),
			DateModified:       ptrInt64(now.UnixMilli()),
			DateModifiedString: ptrString(now.Format(time.RFC3339)),
		}
		if _, err := uc.repositories.ProductPlan.CreateProductPlan(ctx, &productplanpb.CreateProductPlanRequest{
			Data: clone,
		}); err != nil {
			return nil, err
		}
		remap[src.GetId()] = newID
	}
	return remap, nil
}

// clonePricePlan inserts the new PricePlan stamped with the cloned plan_id,
// the resolved price_schedule_id, and the target client_id.
func (uc *CustomizePlanForClientUseCase) clonePricePlan(
	ctx context.Context,
	source *priceplanpb.PricePlan,
	newPlanID, scheduleID, clientID, clientName string,
) (*priceplanpb.PricePlan, error) {
	now := time.Now()
	newID := uc.generateID()
	scheduleIDCopy := scheduleID
	clientCopy := clientID
	clone := &priceplanpb.PricePlan{
		Id:                   newID,
		PlanId:               newPlanID,
		Name:                 ptrString(appendClientSuffix(source.GetName(), clientName)),
		Description:          copyStringPtr(source.Description),
		Active:               true,
		BillingAmount:        source.GetBillingAmount(),
		BillingCurrency:      source.GetBillingCurrency(),
		BillingKind:          source.GetBillingKind(),
		AmountBasis:          source.GetAmountBasis(),
		BillingCycleValue:    copyInt32Ptr(source.BillingCycleValue),
		BillingCycleUnit:     copyStringPtr(source.BillingCycleUnit),
		DefaultTermValue:     copyInt32Ptr(source.DefaultTermValue),
		DefaultTermUnit:      copyStringPtr(source.DefaultTermUnit),
		DurationValue:        copyInt32Ptr(source.DurationValue),
		DurationUnit:         copyStringPtr(source.DurationUnit),
		ConfirmationTemplate: copyStringPtr(source.ConfirmationTemplate),
		ReceiptTemplate:      copyStringPtr(source.ReceiptTemplate),
		PriceScheduleId:      &scheduleIDCopy,
		ClientId:             &clientCopy,
		DateCreated:          ptrInt64(now.UnixMilli()),
		DateCreatedString:    ptrString(now.Format(time.RFC3339)),
		DateModified:         ptrInt64(now.UnixMilli()),
		DateModifiedString:   ptrString(now.Format(time.RFC3339)),
	}
	resp, err := uc.repositories.PricePlan.CreatePricePlan(ctx, &priceplanpb.CreatePricePlanRequest{Data: clone})
	if err != nil {
		return nil, err
	}
	if resp == nil || len(resp.GetData()) == 0 {
		return nil, errors.New("create price plan returned no data")
	}
	return resp.GetData()[0], nil
}

// cloneProductPricePlans inserts a new ProductPricePlan for every row that
// referenced the source PricePlan, remapping product_plan_id through the
// caller-supplied map. Currency lock to the parent newPricePlan holds by
// construction (we copy source.billing_currency unchanged).
func (uc *CustomizePlanForClientUseCase) cloneProductPricePlans(
	ctx context.Context,
	sourcePricePlanID, newPricePlanID string,
	productPlanRemap map[string]string,
) error {
	if uc.repositories.ProductPricePlan == nil {
		return nil
	}
	listResp, err := uc.repositories.ProductPricePlan.ListProductPricePlans(ctx, &productpriceplanpb.ListProductPricePlansRequest{})
	if err != nil || listResp == nil {
		return nil
	}
	now := time.Now()
	for _, src := range listResp.GetData() {
		if src.GetPricePlanId() != sourcePricePlanID {
			continue
		}
		newProductPlanID, ok := productPlanRemap[src.GetProductPlanId()]
		if !ok {
			// No remap entry — skip rather than insert a dangling FK.
			// This can happen when the source has stale ProductPricePlan
			// rows pointing at deleted ProductPlan rows.
			continue
		}
		clone := &productpriceplanpb.ProductPricePlan{
			Id:                 uc.generateID(),
			PricePlanId:        newPricePlanID,
			ProductPlanId:      newProductPlanID,
			BillingAmount:      src.GetBillingAmount(),
			BillingCurrency:    src.GetBillingCurrency(),
			BillingTreatment:   src.GetBillingTreatment(),
			DateStart:          copyStringPtr(src.DateStart),
			DateEnd:            copyStringPtr(src.DateEnd),
			Active:             true,
			DateCreated:        ptrInt64(now.UnixMilli()),
			DateCreatedString:  ptrString(now.Format(time.RFC3339)),
			DateModified:       ptrInt64(now.UnixMilli()),
			DateModifiedString: ptrString(now.Format(time.RFC3339)),
		}
		if _, err := uc.repositories.ProductPricePlan.CreateProductPricePlan(ctx, &productpriceplanpb.CreateProductPricePlanRequest{
			Data: clone,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (uc *CustomizePlanForClientUseCase) generateID() string {
	if uc.services.IDGenerator != nil {
		return uc.services.IDGenerator.GenerateID()
	}
	return fmt.Sprintf("custom-%d", time.Now().UnixNano())
}

// fallbackScheduleName is used when the handler did not supply a derived
// name. Mirrors the lyngua-driven format ("{Client} - Price Schedule") with
// English defaults.
func (uc *CustomizePlanForClientUseCase) fallbackScheduleName(client *clientpb.Client) string {
	name := clientDisplayName(client)
	if name == "" {
		return "Price Schedule"
	}
	return name + " - Price Schedule"
}

// ----- helper utilities ----------------------------------------------------

// scheduleLocationID safely reads the location_id from a (possibly nil)
// schedule.
func scheduleLocationID(s *priceschedulepb.PriceSchedule) string {
	if s == nil {
		return ""
	}
	return s.GetLocationId()
}

// clientDisplayName picks the best human-readable label for the client, with
// the same fallback chain documented in the plan §4.4.1 edge case.
func clientDisplayName(c *clientpb.Client) string {
	if c == nil {
		return ""
	}
	if name := c.GetName(); name != "" {
		return name
	}
	// `Client.User` may not be exposed on the proto; fall back to the ID so
	// we always return a non-empty string when a row exists.
	if id := c.GetId(); id != "" {
		return id
	}
	return ""
}

// appendClientSuffix returns `"<name> (<clientName>)"` when both inputs are
// non-empty, else degrades gracefully.
func appendClientSuffix(name, clientName string) string {
	if name == "" {
		return clientName
	}
	if clientName == "" {
		return name
	}
	return name + " (" + clientName + ")"
}

// copyStringPtr returns a fresh pointer holding the same string. Returns nil
// when the input is nil so optional-field round-tripping stays clean.
func copyStringPtr(s *string) *string {
	if s == nil {
		return nil
	}
	v := *s
	return &v
}

// copyInt32Ptr mirrors copyStringPtr for *int32.
func copyInt32Ptr(v *int32) *int32 {
	if v == nil {
		return nil
	}
	c := *v
	return &c
}

func ptrString(s string) *string { v := s; return &v }
func ptrInt64(v int64) *int64    { c := v; return &c }
