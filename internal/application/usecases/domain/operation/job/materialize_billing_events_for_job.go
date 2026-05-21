package job

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"

	enumspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/enums"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job"
	jobphasepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_phase"
	jobtemplatephasepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template_phase"
	billingeventpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/billing_event"
	priceplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_plan"
	productpriceplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/product_price_plan"
	subscriptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription"
)

// MaterializeBillingEventsForJobRepositories groups every cross-domain repo
// the use case touches. Mirror of `JobRepositories` minus the unused outcome
// repos — kept narrow so it's clear at a glance which entities the
// materialization logic reads/writes.
type MaterializeBillingEventsForJobRepositories struct {
	Job              pb.JobDomainServiceServer
	JobTemplatePhase jobtemplatephasepb.JobTemplatePhaseDomainServiceServer
	JobPhase         jobphasepb.JobPhaseDomainServiceServer
	BillingEvent     billingeventpb.BillingEventDomainServiceServer
	// Subscription is required when the request omits SubscriptionID — used to
	// derive the price_plan_id for percent-precedence resolution.
	Subscription     subscriptionpb.SubscriptionDomainServiceServer
	PricePlan        priceplanpb.PricePlanDomainServiceServer
	ProductPricePlan productpriceplanpb.ProductPricePlanDomainServiceServer
}

// MaterializeBillingEventsForJobServices mirrors the standard service struct
// pattern used by every other use case in this package.
type MaterializeBillingEventsForJobServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	IDGenerator ports.IDGenerator
}

// MaterializeBillingEventsForJobUseCase generates BillingEvent rows for a Job
// whose billing_rule_type = MILESTONE. One event per JobTemplatePhase row
// where triggers_billing = true. Idempotent — never overwrites existing rows.
type MaterializeBillingEventsForJobUseCase struct {
	repositories MaterializeBillingEventsForJobRepositories
	services     MaterializeBillingEventsForJobServices
}

// NewMaterializeBillingEventsForJobUseCase wires the use case.
func NewMaterializeBillingEventsForJobUseCase(
	repositories MaterializeBillingEventsForJobRepositories,
	services MaterializeBillingEventsForJobServices,
) *MaterializeBillingEventsForJobUseCase {
	return &MaterializeBillingEventsForJobUseCase{
		repositories: repositories,
		services:     services,
	}
}

// MaterializeBillingEventsForJobRequest is the input contract.
type MaterializeBillingEventsForJobRequest struct {
	JobID          string
	SubscriptionID string // soft pointer — when empty, fall back to job.OriginId if origin_type=SUBSCRIPTION
}

// MaterializeBillingEventsForJobResponse echoes back the events created.
type MaterializeBillingEventsForJobResponse struct {
	Events []*billingeventpb.BillingEvent
}

// Execute drives the materialization flow:
//
//  1. Read Job + sanity checks (active, billing_rule_type = MILESTONE).
//  2. Read JobTemplate's phases via ListByJobTemplate (all rows, then filter
//     triggers_billing = true).
//  3. Read each JobPhase under this Job to map template_phase_id → instance id.
//  4. Resolve PricePlan for the subscription (used to compute percent-based
//     amounts and to gate ProductPricePlans).
//  5. For each triggers-billing template phase, resolve billable_amount:
//     fixed > percent > sum-of-gated-PPPs.
//  6. Insert one BillingEvent per phase (status=UNSPECIFIED, trigger=UNSPECIFIED).
func (uc *MaterializeBillingEventsForJobUseCase) Execute(
	ctx context.Context, req MaterializeBillingEventsForJobRequest,
) (*MaterializeBillingEventsForJobResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		"job", ports.ActionUpdate); err != nil {
		return nil, err
	}
	if req.JobID == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"job.validation.id_required",
			"job ID is required [DEFAULT]",
		))
	}
	if uc.repositories.Job == nil ||
		uc.repositories.JobTemplatePhase == nil ||
		uc.repositories.JobPhase == nil ||
		uc.repositories.BillingEvent == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"job.errors.materialize_repositories_unavailable",
			"materialize_billing_events_for_job is missing required repositories [DEFAULT]",
		))
	}

	// 1. Read Job
	jobResp, err := uc.repositories.Job.ReadJob(ctx, &pb.ReadJobRequest{Data: &pb.Job{Id: req.JobID}})
	if err != nil || jobResp == nil || len(jobResp.GetData()) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"job.errors.not_found",
			"job not found [DEFAULT]",
		))
	}
	job := jobResp.GetData()[0]

	if job.GetBillingRuleType() != enumspb.BillingRuleType_BILLING_RULE_TYPE_MILESTONE {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"job.errors.not_milestone_billing",
			"job is not milestone-billed [DEFAULT]",
		))
	}

	subscriptionID := req.SubscriptionID
	if subscriptionID == "" && job.GetOriginType() == enumspb.OriginType_ORIGIN_TYPE_SUBSCRIPTION {
		subscriptionID = job.GetOriginId()
	}
	if subscriptionID == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"job.errors.subscription_required",
			"subscription_id is required to materialize billing events [DEFAULT]",
		))
	}

	// 2. List template phases for the Job's template, filter triggers_billing.
	templateID := job.GetJobTemplateId()
	if templateID == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx, uc.services.Translator,
			"job.errors.template_required",
			"job_template_id is required to materialize billing events [DEFAULT]",
		))
	}
	tplResp, err := uc.repositories.JobTemplatePhase.ListByJobTemplate(
		ctx, &jobtemplatephasepb.ListByJobTemplateRequest{JobTemplateId: templateID},
	)
	if err != nil || tplResp == nil {
		return nil, fmt.Errorf("list_by_job_template: %w", err)
	}

	// 3. Map template_phase_id → JobPhase.id (and capture currency / amount inputs).
	jpResp, err := uc.repositories.JobPhase.ListByJob(
		ctx, &jobphasepb.ListJobPhasesByJobRequest{JobId: job.GetId()},
	)
	if err != nil || jpResp == nil {
		return nil, fmt.Errorf("list_job_phases_by_job: %w", err)
	}
	phaseByTemplate := make(map[string]*jobphasepb.JobPhase, len(jpResp.GetJobPhases()))
	for _, jp := range jpResp.GetJobPhases() {
		if id := jp.GetTemplatePhaseId(); id != "" {
			phaseByTemplate[id] = jp
		}
	}

	// 4. Resolve the parent PricePlan eagerly so percent-precedence
	// (`billing_percent_bps × pricePlan.billing_amount / 10000`) and
	// PPP-sum branches both have the data they need. The Job → Subscription
	// link is `job.origin_type=SUBSCRIPTION + job.origin_id=subscription_id`;
	// the Subscription carries the `price_plan_id`. Failure to resolve is
	// non-fatal — the use case falls through to the fixed/PPP-sum branches
	// for templates that don't need percent math.
	var (
		pricePlan *priceplanpb.PricePlan
		ppps      []*productpriceplanpb.ProductPricePlan
	)
	if uc.repositories.Subscription != nil && uc.repositories.PricePlan != nil {
		subResp, subErr := uc.repositories.Subscription.ReadSubscription(
			ctx, &subscriptionpb.ReadSubscriptionRequest{Data: &subscriptionpb.Subscription{Id: subscriptionID}},
		)
		if subErr == nil && subResp != nil && len(subResp.GetData()) > 0 {
			if planID := subResp.GetData()[0].GetPricePlanId(); planID != "" {
				ppResp, ppErr := uc.repositories.PricePlan.ReadPricePlan(
					ctx, &priceplanpb.ReadPricePlanRequest{Data: &priceplanpb.PricePlan{Id: planID}},
				)
				if ppErr == nil && ppResp != nil && len(ppResp.GetData()) > 0 {
					pricePlan = ppResp.GetData()[0]
				}
			}
		}
	}
	resolvePPPs := func() []*productpriceplanpb.ProductPricePlan {
		if ppps != nil {
			return ppps
		}
		if uc.repositories.ProductPricePlan == nil {
			return []*productpriceplanpb.ProductPricePlan{}
		}
		resp, err := uc.repositories.ProductPricePlan.ListProductPricePlans(
			ctx, &productpriceplanpb.ListProductPricePlansRequest{},
		)
		if err != nil || resp == nil {
			return []*productpriceplanpb.ProductPricePlan{}
		}
		// Filter to the resolved PricePlan's rows when known. When PricePlan
		// resolution fails, fall back to all rows — `resolveBillableAmount`
		// further filters by `job_template_phase_id` so the cross-plan
		// pollution risk is bounded to plans that share template-phase IDs
		// (rare but possible).
		planID := ""
		if pricePlan != nil {
			planID = pricePlan.GetId()
		}
		out := make([]*productpriceplanpb.ProductPricePlan, 0, len(resp.GetData()))
		for _, ppp := range resp.GetData() {
			if planID == "" || ppp.GetPricePlanId() == planID {
				out = append(out, ppp)
			}
		}
		ppps = out
		return out
	}

	// 5 + 6. Build + insert BillingEvent rows.
	now := time.Now()
	dc := now.UnixMilli()
	dcs := now.Format(time.RFC3339)
	var created []*billingeventpb.BillingEvent

	// Idempotency: skip phases already covered by an existing billing_event.
	existingByPhase := make(map[string]bool)
	if exResp, err := uc.repositories.BillingEvent.ListBySubscription(
		ctx, &billingeventpb.ListBillingEventsBySubscriptionRequest{SubscriptionId: subscriptionID},
	); err == nil && exResp != nil {
		for _, ev := range exResp.GetBillingEvents() {
			if ev.GetJobId() == job.GetId() {
				if v := ev.GetJobTemplatePhaseId(); v != "" {
					existingByPhase[v] = true
				}
			}
		}
	}

	for _, tpl := range tplResp.GetJobTemplatePhases() {
		if !tpl.GetTriggersBilling() {
			continue
		}
		if existingByPhase[tpl.GetId()] {
			continue
		}

		amount := resolveBillableAmount(tpl, pricePlan, resolvePPPs())
		// Resolve currency: a fixed billing_amount carries its own currency
		// (per plan §2.3); percent + PPP-sum paths inherit from the parent
		// PricePlan. Falls back to template-phase currency last.
		currency := ""
		if tpl.GetBillingAmount() > 0 {
			currency = tpl.GetBillingCurrency()
		}
		if currency == "" && pricePlan != nil {
			currency = pricePlan.GetBillingCurrency()
		}
		if currency == "" {
			currency = tpl.GetBillingCurrency()
		}

		ev := &billingeventpb.BillingEvent{
			Active:             true,
			SubscriptionId:     subscriptionID,
			BillableAmount:     amount,
			BillingCurrency:    currency,
			Status:             billingeventpb.BillingEventStatus_BILLING_EVENT_STATUS_UNSPECIFIED,
			Trigger:            billingeventpb.BillingEventTrigger_BILLING_EVENT_TRIGGER_UNSPECIFIED,
			DateCreated:        &dc,
			DateCreatedString:  &dcs,
			DateModified:       &dc,
			DateModifiedString: &dcs,
		}
		jobIDLocal := job.GetId()
		ev.JobId = &jobIDLocal
		tplID := tpl.GetId()
		ev.JobTemplatePhaseId = &tplID
		if jp := phaseByTemplate[tpl.GetId()]; jp != nil {
			jpID := jp.GetId()
			ev.JobPhaseId = &jpID
		}
		if uc.services.IDGenerator != nil {
			ev.Id = uc.services.IDGenerator.GenerateID()
		}
		resp, err := uc.repositories.BillingEvent.CreateBillingEvent(
			ctx, &billingeventpb.CreateBillingEventRequest{Data: ev},
		)
		if err != nil {
			return nil, fmt.Errorf("create billing_event: %w", err)
		}
		if resp != nil && len(resp.GetData()) > 0 {
			created = append(created, resp.GetData()[0])
		}
	}

	return &MaterializeBillingEventsForJobResponse{Events: created}, nil
}

// resolveBillableAmount applies the presence rules from milestone-billing
// plan §2.3:
//
//	fixed billing_amount > percent_bps × pricePlan.billing_amount > derived sum.
//
// Returns 0 when none of the inputs resolve. Caller decides whether 0 is a
// reject or "operator must edit".
func resolveBillableAmount(
	tpl *jobtemplatephasepb.JobTemplatePhase,
	pricePlan *priceplanpb.PricePlan,
	ppps []*productpriceplanpb.ProductPricePlan,
) int64 {
	if v := tpl.GetBillingAmount(); v > 0 {
		return v
	}
	if pct := tpl.GetBillingPercentBps(); pct > 0 && pricePlan != nil {
		return (pricePlan.GetBillingAmount() * int64(pct)) / 10000
	}
	var sum int64
	for _, ppp := range ppps {
		if ppp.GetJobTemplatePhaseId() == tpl.GetId() {
			sum += ppp.GetBillingAmount()
		}
	}
	return sum
}
