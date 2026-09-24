package subscription_group_outcome_export

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/shared/identity"
	exportpb "github.com/erniealice/esqyma/pkg/schema/v1/service/operation/subscription_group_outcome_export"
)

var canonicalReportCardAttributeCode = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

// GetClientReportCardUseCase authorizes and validates the single-member report
// projection used by both the client view and individual document download.
// The projection port derives workspace and principal scope from request context.
type GetClientReportCardUseCase struct {
	repositories Repositories
	services     Services
}

func (uc *GetClientReportCardUseCase) Execute(ctx context.Context, req *exportpb.GetSubscriptionGroupClientReportCardRequest) (*exportpb.GetSubscriptionGroupClientReportCardResponse, error) {
	scope, err := uc.services.reportScope(ctx)
	if err != nil {
		return nil, err
	}
	if err := validateClientReportCardRequest(req); err != nil {
		return nil, err
	}
	if uc.repositories.ClientReportCardQuery == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"subscription_group_outcome_export.errors.unavailable", "client report card projection query is not configured"))
	}
	requestIdentity, ok := identity.FromContext(ctx)
	if !ok || requestIdentity == nil || strings.TrimSpace(requestIdentity.WorkspaceID) == "" {
		return nil, fmt.Errorf("client report card requires an active workspace")
	}
	response, err := uc.repositories.ClientReportCardQuery.GetSubscriptionGroupClientReportCardScoped(ctx, req, scope)
	if err != nil {
		return nil, err
	}
	if err := validateClientReportCardResponse(req, response, requestIdentity.WorkspaceID); err != nil {
		return nil, err
	}
	return response, nil
}

func validateClientReportCardRequest(req *exportpb.GetSubscriptionGroupClientReportCardRequest) error {
	if req == nil {
		return fmt.Errorf("client report card request is required")
	}
	if strings.TrimSpace(req.GetSubscriptionGroupId()) == "" || req.GetSubscriptionGroupId() != strings.TrimSpace(req.GetSubscriptionGroupId()) {
		return fmt.Errorf("subscription_group_id must be nonempty and canonical")
	}
	if strings.TrimSpace(req.GetClientId()) == "" || req.GetClientId() != strings.TrimSpace(req.GetClientId()) {
		return fmt.Errorf("client_id must be nonempty and canonical")
	}
	seen := make(map[string]struct{}, len(req.GetClientAttributeCodes()))
	for _, code := range req.GetClientAttributeCodes() {
		if !canonicalReportCardAttributeCode.MatchString(code) {
			return fmt.Errorf("client attribute code must be canonical")
		}
		if _, duplicate := seen[code]; duplicate {
			return fmt.Errorf("client report card request contains duplicate attribute code %q", code)
		}
		seen[code] = struct{}{}
	}
	seenPlan := make(map[string]struct{}, len(req.GetPlanAttributeCodes()))
	for _, code := range req.GetPlanAttributeCodes() {
		if !canonicalReportCardAttributeCode.MatchString(code) {
			return fmt.Errorf("plan attribute code must be canonical")
		}
		if _, duplicate := seenPlan[code]; duplicate {
			return fmt.Errorf("client report card request contains duplicate plan attribute code %q", code)
		}
		seenPlan[code] = struct{}{}
	}
	return nil
}

func validateClientReportCardResponse(req *exportpb.GetSubscriptionGroupClientReportCardRequest, response *exportpb.GetSubscriptionGroupClientReportCardResponse, workspaceID string) error {
	if response == nil {
		return ports.ErrClientReportNotFound
	}
	if !response.GetSuccess() {
		return fmt.Errorf("client report card query returned an incomplete response")
	}
	if response.GetReportCard() == nil {
		return ports.ErrClientReportNotFound
	}
	card := response.GetReportCard()
	if card.GetContext() == nil || card.GetContext().GetSubscriptionGroupId() != req.GetSubscriptionGroupId() {
		return fmt.Errorf("client report card context does not match the requested subscription group")
	}
	if len(card.GetClientSubscriptionIds()) == 0 || card.GetClient() == nil || card.GetClient().GetClientId() != req.GetClientId() {
		return fmt.Errorf("client report card does not prove exact group membership")
	}
	subscriptionIDs := make(map[string]struct{}, len(card.GetClientSubscriptionIds()))
	for _, subscriptionID := range card.GetClientSubscriptionIds() {
		if strings.TrimSpace(subscriptionID) == "" {
			return fmt.Errorf("client report card contains an invalid subscription membership")
		}
		if _, duplicate := subscriptionIDs[subscriptionID]; duplicate {
			return fmt.Errorf("client report card contains a duplicate subscription membership")
		}
		subscriptionIDs[subscriptionID] = struct{}{}
	}

	requestedAttributes := make(map[string]struct{}, len(req.GetClientAttributeCodes()))
	for _, code := range req.GetClientAttributeCodes() {
		requestedAttributes[code] = struct{}{}
	}
	attributeCodes := make(map[string]struct{}, len(card.GetAttributes()))
	for _, attribute := range card.GetAttributes() {
		if attribute == nil || strings.TrimSpace(attribute.GetCode()) == "" {
			return fmt.Errorf("client report card contains an invalid attribute")
		}
		if _, requested := requestedAttributes[attribute.GetCode()]; !requested {
			return fmt.Errorf("client report card contains an unrequested attribute")
		}
		if _, duplicate := attributeCodes[attribute.GetCode()]; duplicate {
			return fmt.Errorf("client report card contains a duplicate attribute")
		}
		attributeCodes[attribute.GetCode()] = struct{}{}
	}
	requestedPlanAttributes := make(map[string]struct{}, len(req.GetPlanAttributeCodes()))
	for _, code := range req.GetPlanAttributeCodes() {
		requestedPlanAttributes[code] = struct{}{}
	}
	planAttributeCodes := make(map[string]struct{}, len(card.GetPlanAttributes()))
	for _, attribute := range card.GetPlanAttributes() {
		if attribute == nil || strings.TrimSpace(attribute.GetCode()) == "" {
			return fmt.Errorf("client report card contains an invalid plan attribute")
		}
		if _, requested := requestedPlanAttributes[attribute.GetCode()]; !requested {
			return fmt.Errorf("client report card contains an unrequested plan attribute")
		}
		if _, duplicate := planAttributeCodes[attribute.GetCode()]; duplicate {
			return fmt.Errorf("client report card contains a duplicate plan attribute")
		}
		planAttributeCodes[attribute.GetCode()] = struct{}{}
	}

	jobIDs := make(map[string]struct{}, len(card.GetJobs()))
	for _, job := range card.GetJobs() {
		if job == nil || strings.TrimSpace(job.GetId()) == "" {
			return fmt.Errorf("client report card contains a job outside the exact subscription")
		}
		if _, memberJob := subscriptionIDs[strings.TrimSpace(job.GetOriginId())]; !memberJob {
			return fmt.Errorf("client report card contains a job outside the exact subscriptions")
		}
		if job.GetWorkspaceId() != workspaceID {
			return fmt.Errorf("client report card contains a job outside the active workspace")
		}
		if job.ClientId != nil && job.GetClientId() != req.GetClientId() {
			return fmt.Errorf("client report card contains a job for another client")
		}
		if _, duplicate := jobIDs[job.GetId()]; duplicate {
			return fmt.Errorf("client report card contains a duplicate job")
		}
		jobIDs[job.GetId()] = struct{}{}
	}

	gateIDs := make(map[string]struct{}, len(card.GetRenderGateJobIds()))
	for _, jobID := range card.GetRenderGateJobIds() {
		if strings.TrimSpace(jobID) == "" {
			return fmt.Errorf("client report card contains an invalid render-gate job id")
		}
		if _, duplicate := gateIDs[jobID]; duplicate {
			return fmt.Errorf("client report card contains a duplicate render-gate job id")
		}
		gateIDs[jobID] = struct{}{}
	}
	for jobID := range jobIDs {
		if _, covered := gateIDs[jobID]; !covered {
			return fmt.Errorf("client report card render gate omits job %q", jobID)
		}
	}
	for jobID := range gateIDs {
		if _, projected := jobIDs[jobID]; !projected {
			return fmt.Errorf("client report card render gate contains an unprojected job")
		}
	}
	if err := validateRenderGateSheets(req.GetSubscriptionGroupId(), workspaceID, card); err != nil {
		return err
	}

	if err := validateProjectedClientIDs(req.GetClientId(), workspaceID, card); err != nil {
		return err
	}
	return nil
}

// validateRenderGateSheets proves complete coverage by distinct expected
// approval sheets. Multiple selected-client phases can share one group sheet;
// NULL-template phases each retain their own singleton sheet identity.
func validateRenderGateSheets(groupID, workspaceID string, card *exportpb.ClientReportCardProjection) error {
	if card.GetRenderGateAppliedSubscriptionGroupId() != groupID {
		return fmt.Errorf("client report card render gate group echo does not match the requested group")
	}
	templatePhaseCounts := make(map[string]int32)
	singletonPhaseIDs := make(map[string]struct{})
	projectedJobs := make(map[string]struct{}, len(card.GetJobs()))
	for _, job := range card.GetJobs() {
		if job != nil {
			projectedJobs[job.GetId()] = struct{}{}
		}
	}
	projectedPhaseIDs := make(map[string]struct{}, len(card.GetJobPhases()))
	for _, phase := range card.GetJobPhases() {
		if phase == nil {
			return fmt.Errorf("client report card contains an invalid phase")
		}
		if _, projected := projectedJobs[phase.GetJobId()]; !projected || strings.TrimSpace(phase.GetId()) == "" {
			return fmt.Errorf("client report card contains a phase outside the projected jobs")
		}
		if _, duplicate := projectedPhaseIDs[phase.GetId()]; duplicate {
			return fmt.Errorf("client report card contains a duplicate phase")
		}
		projectedPhaseIDs[phase.GetId()] = struct{}{}
		if phase.GetWorkspaceId() != workspaceID {
			return fmt.Errorf("client report card contains a phase outside the active workspace")
		}
		if !phase.GetActive() {
			continue
		}
		if phase.GetTemplatePhaseId() == "" {
			if _, duplicate := singletonPhaseIDs[phase.GetId()]; duplicate {
				return fmt.Errorf("client report card contains a duplicate singleton phase")
			}
			singletonPhaseIDs[phase.GetId()] = struct{}{}
			continue
		}
		templatePhaseCounts[phase.GetTemplatePhaseId()]++
	}
	seenTemplateSheets := make(map[string]struct{}, len(templatePhaseCounts))
	seenSingletonSheets := make(map[string]struct{}, len(singletonPhaseIDs))
	for _, sheet := range card.GetRenderGateSheets() {
		if sheet == nil || sheet.GetAppliedSubscriptionGroupId() != groupID {
			return fmt.Errorf("client report card contains incomplete or foreign render-gate evidence")
		}
		templateID, singletonID := sheet.GetJobTemplatePhaseId(), sheet.GetJobPhaseId()
		if templateID != "" && singletonID != "" || templateID == "" && singletonID == "" {
			return fmt.Errorf("client report card contains mismatched render-gate sheet identity")
		}
		if templateID != "" {
			expectedCount, expected := templatePhaseCounts[templateID]
			if !expected || sheet.TargetCount < expectedCount || sheet.TargetCount <= 0 {
				return fmt.Errorf("client report card contains unrequested or incomplete template-phase gate evidence")
			}
			if _, duplicate := seenTemplateSheets[templateID]; duplicate {
				return fmt.Errorf("client report card contains duplicate template-phase gate evidence")
			}
			seenTemplateSheets[templateID] = struct{}{}
			continue
		}
		if _, expected := singletonPhaseIDs[singletonID]; !expected || sheet.TargetCount != 1 {
			return fmt.Errorf("client report card contains unrequested or incomplete singleton gate evidence")
		}
		if _, duplicate := seenSingletonSheets[singletonID]; duplicate {
			return fmt.Errorf("client report card contains duplicate singleton gate evidence")
		}
		seenSingletonSheets[singletonID] = struct{}{}
	}
	if len(seenTemplateSheets) != len(templatePhaseCounts) || len(seenSingletonSheets) != len(singletonPhaseIDs) {
		return fmt.Errorf("client report card render-gate evidence does not cover every active phase sheet")
	}
	return nil
}

func validateProjectedClientIDs(clientID, workspaceID string, card *exportpb.ClientReportCardProjection) error {
	projectedJobs := make(map[string]struct{}, len(card.GetJobs()))
	for _, job := range card.GetJobs() {
		if job != nil && strings.TrimSpace(job.GetId()) != "" {
			projectedJobs[job.GetId()] = struct{}{}
		}
	}
	projectedSummaries := make(map[string]struct{}, len(card.GetJobOutcomeSummaries()))
	for _, summary := range card.GetJobOutcomeSummaries() {
		if summary == nil || strings.TrimSpace(summary.GetId()) == "" || summary.GetWorkspaceId() != workspaceID || summary.GetClientId() != "" && summary.GetClientId() != clientID {
			return fmt.Errorf("client report card contains a job summary for another client")
		}
		if _, belongsToProjection := projectedJobs[summary.GetJobId()]; !belongsToProjection {
			return fmt.Errorf("client report card contains a job summary outside the projected jobs")
		}
		if _, duplicate := projectedSummaries[summary.GetId()]; duplicate {
			return fmt.Errorf("client report card contains a duplicate job summary")
		}
		projectedSummaries[summary.GetId()] = struct{}{}
	}
	for _, line := range card.GetJobOutcomeLines() {
		if line == nil || line.GetWorkspaceId() != workspaceID || line.GetClientId() != "" && line.GetClientId() != clientID {
			return fmt.Errorf("client report card contains an outcome line for another client")
		}
		if _, belongsToProjection := projectedSummaries[line.GetJobOutcomeSummaryId()]; !belongsToProjection {
			return fmt.Errorf("client report card contains an outcome line outside the projected summaries")
		}
	}
	return nil
}
