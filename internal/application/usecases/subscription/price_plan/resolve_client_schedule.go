package price_plan

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	planUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/subscription/plan"

	clientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client"
	planpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/plan"
	priceplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_plan"
	priceschedulepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_schedule"
)

// applyClientScopedScheduleRule enforces the schedule-scope invariants for
// PricePlan create/update under a client-scoped Plan (plan §3.2 / §4.4 of
// 20260427-plan-client-scope).
//
// When the parent Plan is master (`plan.client_id == ""`), this is a no-op.
//
// When the parent Plan is client-scoped:
//   - If schedule_id is set, verify it does not belong to a different client.
//     Mismatch → return scheduleClientMismatch. Master schedules
//     (sched.client_id == "") are accepted; the cascade leaves them as picked.
//   - If schedule_id is empty, look up an existing client-scoped schedule
//     (workspace + first PlanLocation + client) and reuse it. If none exists,
//     create a new one. The drawer surfaces a contextual hint ("will create"
//     vs "will be added to existing") so the operator knows which path runs
//     before they save.
//
// `data.PriceScheduleId` is mutated to the resolved value when this function
// resolved-or-created a schedule. The auto-created schedule defaults
// `date_time_start = now()`; operator can adjust dates from the schedule
// detail page after.
func applyClientScopedScheduleRule(
	ctx context.Context,
	data *priceplanpb.PricePlan,
	parentPlan *planpb.Plan,
	priceScheduleRepo priceschedulepb.PriceScheduleDomainServiceServer,
	clientRepo clientpb.ClientDomainServiceServer,
	idSvc ports.IDService,
	translation ports.TranslationService,
) error {
	if data == nil || parentPlan == nil {
		return nil
	}
	parentClientID := parentPlan.GetClientId()
	if parentClientID == "" {
		// Master parent — no schedule auto-creation. Return without touching
		// the body's price_schedule_id.
		return nil
	}

	submittedScheduleID := data.GetPriceScheduleId()

	// Operator picked a schedule explicitly — verify scope match. Master
	// schedules pass through unchanged; only cross-client picks are rejected.
	if submittedScheduleID != "" {
		if priceScheduleRepo == nil {
			return nil // No repo wired — defer to repository-level integrity.
		}
		readResp, err := priceScheduleRepo.ReadPriceSchedule(ctx, &priceschedulepb.ReadPriceScheduleRequest{
			Data: &priceschedulepb.PriceSchedule{Id: submittedScheduleID},
		})
		if err != nil || readResp == nil || len(readResp.GetData()) == 0 {
			// Defer to other validators — this checker only enforces the
			// client-scope match invariant.
			return nil
		}
		sched := readResp.GetData()[0]
		schedClientID := sched.GetClientId()
		if schedClientID != "" && schedClientID != parentClientID {
			msg := contextutil.GetTranslatedMessageWithContext(
				ctx, translation,
				"price_plan.errors.scheduleClientMismatch",
				"Selected schedule belongs to a different client and cannot be attached to this price plan. [DEFAULT]",
			)
			return errors.New(msg)
		}
		return nil
	}

	// Submitted price_schedule_id is empty — resolve-or-create a client-scoped
	// schedule. Drawer hint already told the operator which path will run.
	if priceScheduleRepo == nil {
		return nil
	}

	// Read the client for the auto-create name fallback. Failures are
	// non-fatal (the parent Plan's client_id was already validated upstream).
	var client *clientpb.Client
	if clientRepo != nil {
		clientResp, err := clientRepo.ReadClient(ctx, &clientpb.ReadClientRequest{
			Data: &clientpb.Client{Id: parentClientID},
		})
		if err == nil && clientResp != nil && len(clientResp.GetData()) > 0 {
			client = clientResp.GetData()[0]
		}
	}

	// Auto-create schedule name: prefer the parent Plan's own name (operator
	// already named the bundle) so the rate-card identity inherits intent.
	// Degrade to "<client.name> - Price Schedule" if the Plan was unnamed.
	derivedName := parentPlan.GetName()
	if derivedName == "" {
		derivedName = fallbackDerivedScheduleName(client)
	}

	workspaceID := contextutil.ExtractWorkspaceIDFromContext(ctx)

	// 1-to-1 invariant: at most one client-scoped PriceSchedule per client.
	// Pass locationID="" so the match key collapses to (workspace, client,
	// active) — across all locations. Customize-Plan-for-Client still passes
	// a real location because that flow clones from a specific source schedule
	// and the new client schedule inherits its location, but the new
	// PricePlan-side path doesn't have a "source location" to inherit and
	// shouldn't fragment a client into multiple per-location schedules.
	resolved, _, err := planUseCases.ResolveOrCreateClientPriceSchedule(
		ctx,
		&planUseCases.ResolveOrCreateClientScheduleRepos{PriceSchedule: priceScheduleRepo},
		idSvc,
		workspaceID,
		"", // see comment above — no location filter or stamp.
		parentClientID,
		derivedName,
		nil, // no template — helper defaults date_time_start = now().
	)
	if err != nil {
		return err
	}
	if resolved == nil {
		return errors.New("resolve-or-create client schedule returned no row")
	}
	resolvedID := resolved.GetId()
	data.PriceScheduleId = &resolvedID
	return nil
}

// fallbackDerivedScheduleName returns "<client.name> - Price Schedule" when
// the client has a name, else degrades to the literal "Price Schedule". Used
// only when the parent Plan also has no name to borrow from.
func fallbackDerivedScheduleName(client *clientpb.Client) string {
	if client == nil {
		return "Price Schedule"
	}
	name := client.GetName()
	if name == "" {
		return "Price Schedule"
	}
	return name + " - Price Schedule"
}
