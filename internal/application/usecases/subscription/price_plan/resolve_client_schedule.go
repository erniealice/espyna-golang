package price_plan

import (
	"context"
	"errors"
	"log"
	"strings"
	"time"

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
	submittedScheduleID := data.GetPriceScheduleId()

	if parentClientID == "" {
		// 2026-05-03 — Master parent: enforce the symmetric mutex. A master
		// plan cannot attach to a client-scoped schedule (the view layer
		// already filters this out, but a crafted POST or stale dropdown
		// can still submit it). Reject with the same lyngua error used for
		// the cross-client mismatch — operator sees one consistent message.
		if submittedScheduleID != "" && priceScheduleRepo != nil {
			readResp, err := priceScheduleRepo.ReadPriceSchedule(ctx, &priceschedulepb.ReadPriceScheduleRequest{
				Data: &priceschedulepb.PriceSchedule{Id: submittedScheduleID},
			})
			if err == nil && readResp != nil && len(readResp.GetData()) > 0 {
				if sched := readResp.GetData()[0]; sched.GetClientId() != "" {
					msg := contextutil.GetTranslatedMessageWithContext(
						ctx, translation,
						"price_plan.errors.scheduleClientMismatch",
						"Selected schedule belongs to a different client and cannot be attached to this price plan. [DEFAULT]",
					)
					return errors.New(msg)
				}
			}
		}
		// Master parent — no schedule auto-creation. Return without touching
		// the body's price_schedule_id.
		return nil
	}

	// Operator picked a schedule explicitly — verify scope match. The picker
	// must now resolve to the same client; master schedules are no longer an
	// allowed attachment target for client-scoped plans (mutually-exclusive
	// rule shipped 2026-05-03 alongside the view-layer filter).
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
		if schedClientID != parentClientID {
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

	// Auto-create schedule name: ALWAYS "{client.name} - {suffix}" so the
	// rate-card list scans by client. Suffix comes from the centymo handler
	// via context (lyngua-resolved, tier-correct); falls back to "Price
	// Schedule" when unset (English / general tier default).
	// 2026-05-03 — Append a wall-clock + IANA-tz suffix so multiple
	// rate cards minted over the course of a client relationship (e.g.
	// renewals on new effective dates) can be scanned by recency. The
	// shape mirrors pyeza-golang/types.AppendTimestamp; espyna does not
	// import pyeza so the format constants are inlined.
	suffix := contextutil.ExtractClientScheduleSuffixFromContext(ctx)
	base := buildClientScheduleBase(client, parentClientID, suffix)
	derivedName := appendScheduleNameTimestamp(base, time.Now(), tzFromClient(client))

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

// buildClientScheduleBase returns "{client.name} - {suffix}", with fallbacks
// down the client-name chain (entity name → User first+last → bare client
// id). Mirrors the centymo drawer's preview construction so render-time and
// save-time names share the same shape.
func buildClientScheduleBase(client *clientpb.Client, clientID, suffix string) string {
	name := ""
	if client != nil {
		name = strings.TrimSpace(client.GetName())
		if name == "" {
			if u := client.GetUser(); u != nil {
				full := strings.TrimSpace(u.GetFirstName() + " " + u.GetLastName())
				if full != "" {
					name = full
				}
			}
		}
	}
	if name == "" {
		name = clientID
	}
	if name == "" {
		return suffix
	}
	return name + " - " + suffix
}

// appendScheduleNameTimestamp produces "{base} - 2026-05-03 14:30:00 Asia/Manila".
// Mirrors pyeza-golang/types.AppendTimestamp; format constants inlined because
// espyna does not depend on pyeza. Nil tz falls back to UTC.
func appendScheduleNameTimestamp(base string, now time.Time, tz *time.Location) string {
	if tz == nil {
		tz = time.UTC
	}
	return base + " - " + now.In(tz).Format("2006-01-02 15:04:05") + " " + tz.String()
}

// tzFromClient pulls Client.User.Timezone and loads it as *time.Location.
// Falls back to "Asia/Manila" (the dev anchor used elsewhere in the monorepo),
// then UTC as a last resort. Bad IANA names log a warning and fall through.
func tzFromClient(client *clientpb.Client) *time.Location {
	if client != nil {
		if u := client.GetUser(); u != nil {
			if name := strings.TrimSpace(u.GetTimezone()); name != "" {
				if loc, err := time.LoadLocation(name); err == nil {
					return loc
				} else {
					log.Printf("invalid client timezone %q for client %s: %v; falling back to default", name, client.GetId(), err)
				}
			}
		}
	}
	if loc, err := time.LoadLocation("Asia/Manila"); err == nil {
		return loc
	}
	return time.UTC
}
