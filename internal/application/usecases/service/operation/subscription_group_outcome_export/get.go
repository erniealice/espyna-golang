package subscription_group_outcome_export

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	exportpb "github.com/erniealice/esqyma/pkg/schema/v1/service/operation/subscription_group_outcome_export"
)

var canonicalPhaseCode = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

type GetUseCase struct {
	repositories Repositories
	services     Services
}

func (uc *GetUseCase) Execute(ctx context.Context, req *exportpb.GetSubscriptionGroupOutcomeExportRequest) (*exportpb.GetSubscriptionGroupOutcomeExportResponse, error) {
	scope, err := uc.services.reportScope(ctx)
	if err != nil {
		return nil, err
	}
	if err := validateGetRequest(ctx, uc.services, req); err != nil {
		return nil, err
	}
	if uc.repositories.Query == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"subscription_group_outcome_export.errors.unavailable", "subscription group outcome export is unavailable"))
	}
	response, err := uc.repositories.Query.GetSubscriptionGroupOutcomeExportScoped(ctx, req, scope)
	if err != nil {
		return nil, err
	}
	if err := validateExportResponse(req, response); err != nil {
		// Preserve the scoped context/options alongside a post-query validation
		// error. In-process HTTP composition can distinguish an unavailable
		// category/period (400, proven from these options) from a corrupt matrix
		// or dependency failure (503) without issuing a second options statement.
		return response, err
	}
	return response, nil
}

func validateGetRequest(ctx context.Context, services Services, req *exportpb.GetSubscriptionGroupOutcomeExportRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, services.Translator,
			"subscription_group_outcome_export.validation.request_required", "subscription group outcome export request is required"))
	}
	groupID := strings.TrimSpace(req.GetSubscriptionGroupId())
	if groupID == "" || groupID != req.GetSubscriptionGroupId() {
		return fmt.Errorf("subscription_group_id must be nonempty and canonical")
	}
	categoryID := strings.TrimSpace(req.GetJobCategoryId())
	if req.JobCategoryId != nil && (categoryID == "" || categoryID != req.GetJobCategoryId()) {
		return fmt.Errorf("job_category_id must be nonempty and canonical when supplied")
	}
	switch selector := req.GetOutcomeSelector().(type) {
	case nil:
		return nil
	case *exportpb.GetSubscriptionGroupOutcomeExportRequest_JobTemplatePhaseCode:
		if categoryID == "" {
			return fmt.Errorf("job_category_id is required with an outcome selector")
		}
		if selector.JobTemplatePhaseCode != strings.TrimSpace(selector.JobTemplatePhaseCode) || !canonicalPhaseCode.MatchString(selector.JobTemplatePhaseCode) {
			return fmt.Errorf("job_template_phase_code must be a canonical path segment")
		}
	case *exportpb.GetSubscriptionGroupOutcomeExportRequest_FinalOutcome:
		if categoryID == "" {
			return fmt.Errorf("job_category_id is required with an outcome selector")
		}
		if !selector.FinalOutcome {
			return fmt.Errorf("final_outcome must be true when selected")
		}
	default:
		return fmt.Errorf("unsupported outcome selector %T", selector)
	}
	return nil
}

func validateExportResponse(req *exportpb.GetSubscriptionGroupOutcomeExportRequest, response *exportpb.GetSubscriptionGroupOutcomeExportResponse) error {
	if response == nil || !response.GetSuccess() || response.GetContext() == nil {
		return fmt.Errorf("subscription group outcome export returned an incomplete response")
	}
	if response.GetContext().GetSubscriptionGroupId() != req.GetSubscriptionGroupId() {
		return fmt.Errorf("subscription group outcome export context does not match the request")
	}
	categories := make(map[string]*exportpb.JobCategoryOption, len(response.GetJobCategories()))
	for _, category := range response.GetJobCategories() {
		if category == nil || strings.TrimSpace(category.GetJobCategoryId()) == "" {
			return fmt.Errorf("subscription group outcome export contains an invalid category option")
		}
		if _, duplicate := categories[category.GetJobCategoryId()]; duplicate {
			return fmt.Errorf("subscription group outcome export contains duplicate category %q", category.GetJobCategoryId())
		}
		categories[category.GetJobCategoryId()] = category
		phaseCodes := make(map[string]struct{}, len(category.GetJobTemplatePhases()))
		for _, phase := range category.GetJobTemplatePhases() {
			if phase == nil || !canonicalPhaseCode.MatchString(phase.GetCode()) {
				return fmt.Errorf("subscription group outcome export contains an invalid phase option")
			}
			if _, duplicate := phaseCodes[phase.GetCode()]; duplicate {
				return fmt.Errorf("subscription group outcome export contains duplicate phase %q", phase.GetCode())
			}
			phaseCodes[phase.GetCode()] = struct{}{}
		}
	}

	selector := req.GetOutcomeSelector()
	if selector == nil {
		if len(response.GetJobTemplateColumns()) != 0 || len(response.GetClientRows()) != 0 {
			return fmt.Errorf("options response unexpectedly contains a matrix")
		}
		if req.JobCategoryId != nil {
			if _, offered := categories[req.GetJobCategoryId()]; !offered {
				return fmt.Errorf("requested job category is not present in the scoped options")
			}
		}
		return nil
	}
	selectedCategory, ok := categories[req.GetJobCategoryId()]
	if !ok {
		return fmt.Errorf("selected job category is not present in the scoped options")
	}
	switch selected := selector.(type) {
	case *exportpb.GetSubscriptionGroupOutcomeExportRequest_JobTemplatePhaseCode:
		matched := false
		for _, phase := range selectedCategory.GetJobTemplatePhases() {
			if phase.GetCode() == selected.JobTemplatePhaseCode {
				if phase.GetAmbiguous() {
					return fmt.Errorf("selected job template phase is ambiguous")
				}
				matched = true
				break
			}
		}
		if !matched {
			return fmt.Errorf("selected job template phase is not available for the category")
		}
	case *exportpb.GetSubscriptionGroupOutcomeExportRequest_FinalOutcome:
		if !selectedCategory.GetFinalOutcomeAvailable() {
			return fmt.Errorf("final outcome is not available for the category")
		}
	}

	columnIDs := make(map[string]struct{}, len(response.GetJobTemplateColumns()))
	for _, column := range response.GetJobTemplateColumns() {
		if column == nil || strings.TrimSpace(column.GetJobTemplateId()) == "" {
			return fmt.Errorf("subscription group outcome export contains an invalid job template column")
		}
		if _, duplicate := columnIDs[column.GetJobTemplateId()]; duplicate {
			return fmt.Errorf("subscription group outcome export contains duplicate job template column %q", column.GetJobTemplateId())
		}
		columnIDs[column.GetJobTemplateId()] = struct{}{}
	}
	if len(columnIDs) == 0 {
		return fmt.Errorf("selected matrix contains no job template columns")
	}
	clientIDs := make(map[string]struct{}, len(response.GetClientRows()))
	for _, row := range response.GetClientRows() {
		if row == nil || strings.TrimSpace(row.GetClientId()) == "" {
			return fmt.Errorf("subscription group outcome export contains an invalid client row")
		}
		if _, duplicate := clientIDs[row.GetClientId()]; duplicate {
			return fmt.Errorf("subscription group outcome export contains duplicate client %q", row.GetClientId())
		}
		clientIDs[row.GetClientId()] = struct{}{}
		if len(row.GetCells()) != len(columnIDs) {
			return fmt.Errorf("client %q has %d cells; want %d", row.GetClientId(), len(row.GetCells()), len(columnIDs))
		}
		cellIDs := make(map[string]struct{}, len(row.GetCells()))
		for _, cell := range row.GetCells() {
			if cell == nil || cell.GetEnrollmentEvidence() == nil {
				return fmt.Errorf("client %q contains a cell without enrollment evidence", row.GetClientId())
			}
			if _, known := columnIDs[cell.GetJobTemplateId()]; !known {
				return fmt.Errorf("client %q contains an unknown job template cell", row.GetClientId())
			}
			if _, duplicate := cellIDs[cell.GetJobTemplateId()]; duplicate {
				return fmt.Errorf("client %q contains a duplicate job template cell", row.GetClientId())
			}
			cellIDs[cell.GetJobTemplateId()] = struct{}{}
		}
	}
	return nil
}
