package subscription

import (
	"context"
	"errors"
	"fmt"

	planjobtemplatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/plan_job_template"
)

type compositionSpawnEntry struct {
	templateID          string
	parentTemplateID    string
	useSubscriptionName bool
}

func listActivePlanComposition(ctx context.Context, repo planjobtemplatepb.PlanJobTemplateDomainServiceServer, planID string) ([]*planjobtemplatepb.PlanJobTemplate, error) {
	if repo == nil || planID == "" {
		return nil, nil
	}
	resp, err := repo.ListPlanJobTemplatesByPlan(ctx, &planjobtemplatepb.ListPlanJobTemplatesByPlanRequest{PlanId: planID})
	if err != nil {
		return nil, fmt.Errorf("list_plan_job_templates_by_plan: %w", err)
	}
	if resp == nil {
		return nil, nil
	}
	rows := make([]*planjobtemplatepb.PlanJobTemplate, 0, len(resp.GetPlanJobTemplates()))
	for _, row := range resp.GetPlanJobTemplates() {
		if row != nil && row.GetActive() {
			rows = append(rows, row)
		}
	}
	return rows, nil
}

func standaloneCompositionTemplateID(rows []*planjobtemplatepb.PlanJobTemplate) (string, error) {
	if len(rows) != 1 {
		return "", errors.New("cyclic or usage-based plan composition requires exactly one standalone entry")
	}
	row := rows[0]
	if row.GetCompositionEntryPattern() != planjobtemplatepb.PlanJobTemplateCompositionEntryPattern_PLAN_JOB_TEMPLATE_COMPOSITION_ENTRY_PATTERN_STANDALONE_ENTRY || row.GetJobTemplateId() == "" {
		return "", errors.New("cyclic or usage-based plan composition cannot use bundle_entry")
	}
	return row.GetJobTemplateId(), nil
}

func (uc *MaterializeJobsForSubscriptionUseCase) buildCompositionSpawnEntries(ctx context.Context, rows []*planjobtemplatepb.PlanJobTemplate) ([]compositionSpawnEntry, error) {
	entries := make([]compositionSpawnEntry, 0, len(rows))
	for _, row := range rows {
		switch row.GetCompositionEntryPattern() {
		case planjobtemplatepb.PlanJobTemplateCompositionEntryPattern_PLAN_JOB_TEMPLATE_COMPOSITION_ENTRY_PATTERN_BUNDLE_ENTRY:
			if row.GetJobTemplateId() == "" {
				return nil, errors.New("bundle composition entry is missing job_template_id")
			}
			entries = append(entries, compositionSpawnEntry{templateID: row.GetJobTemplateId()})
		case planjobtemplatepb.PlanJobTemplateCompositionEntryPattern_PLAN_JOB_TEMPLATE_COMPOSITION_ENTRY_PATTERN_STANDALONE_ENTRY:
			rootID := row.GetJobTemplateId()
			if rootID == "" {
				return nil, errors.New("standalone composition entry is missing job_template_id")
			}
			entries = append(entries, compositionSpawnEntry{templateID: rootID, useSubscriptionName: true})
			relations, err := uc.listChildRelations(ctx, rootID)
			if err != nil {
				return nil, err
			}
			for _, rel := range relations {
				if rel.GetActive() && rel.GetChildTemplateId() != "" && rel.GetChildTemplateId() != rootID {
					entries = append(entries, compositionSpawnEntry{templateID: rel.GetChildTemplateId(), parentTemplateID: rootID})
				}
			}
		default:
			return nil, fmt.Errorf("unsupported composition entry pattern %s", row.GetCompositionEntryPattern())
		}
	}
	return entries, nil
}

func (uc *MaterializeJobsForSubscriptionUseCase) buildLegacySpawnEntries(ctx context.Context, rootID string) ([]compositionSpawnEntry, error) {
	relations, err := uc.listChildRelations(ctx, rootID)
	if err != nil {
		return nil, err
	}
	entries := []compositionSpawnEntry{{templateID: rootID, useSubscriptionName: true}}
	for _, rel := range relations {
		if rel.GetActive() && rel.GetChildTemplateId() != "" && rel.GetChildTemplateId() != rootID {
			entries = append(entries, compositionSpawnEntry{templateID: rel.GetChildTemplateId(), parentTemplateID: rootID})
		}
	}
	return entries, nil
}

// PlanDeclaresComposition lets the strict Subscription.Create preflight prove
// that a rootless Plan still has an explicit materialization contract.
func (a *MaterializeJobsForSubscriptionInstantiator) PlanDeclaresComposition(ctx context.Context, planID string) (bool, error) {
	if a == nil || a.UseCase == nil {
		return false, errors.New("materializer unavailable")
	}
	rows, err := listActivePlanComposition(ctx, a.UseCase.repositories.PlanJobTemplate, planID)
	return len(rows) > 0, err
}
