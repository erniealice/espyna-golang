package subscription_group_outcome_export

import (
	"context"
	"fmt"
	"strings"

	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/subscription_group_document_template"
	exportpb "github.com/erniealice/esqyma/pkg/schema/v1/service/operation/subscription_group_outcome_export"
)

type ResolveDocumentUseCase struct {
	repositories Repositories
	services     Services
}

func (uc *ResolveDocumentUseCase) Execute(ctx context.Context, req *exportpb.ResolveSubscriptionGroupOutcomeDocumentForRenderRequest) (*exportpb.ResolveSubscriptionGroupOutcomeDocumentForRenderResponse, error) {
	scope, err := uc.services.reportScope(ctx)
	if err != nil {
		return nil, err
	}
	if err := validateResolveRequest(req); err != nil {
		return nil, err
	}
	if uc.repositories.Query == nil {
		return nil, fmt.Errorf("subscription group outcome document resolver is unavailable")
	}
	response, err := uc.repositories.Query.ResolveSubscriptionGroupOutcomeDocumentForRenderScoped(ctx, req, scope)
	if err != nil {
		return nil, err
	}
	if response == nil || !response.GetSuccess() {
		return nil, fmt.Errorf("subscription group outcome document resolver returned an incomplete response")
	}
	if !response.GetFound() {
		if response.GetDocument() != nil {
			return nil, fmt.Errorf("document resolver miss contains locator metadata")
		}
		return response, nil
	}
	document := response.GetDocument()
	if document == nil || strings.TrimSpace(document.GetStorageContainer()) == "" || strings.TrimSpace(document.GetStorageKey()) == "" {
		return nil, fmt.Errorf("document resolver hit contains an incomplete locator")
	}
	if document.GetRenderProfile() != req.GetRenderProfile() || document.GetJobCategoryId() != req.GetJobCategoryId() {
		return nil, fmt.Errorf("document resolver hit does not match the trusted profile/category")
	}
	return response, nil
}

func validateResolveRequest(req *exportpb.ResolveSubscriptionGroupOutcomeDocumentForRenderRequest) error {
	if req == nil {
		return fmt.Errorf("document render request is required")
	}
	if strings.TrimSpace(req.GetSubscriptionGroupId()) == "" || req.GetSubscriptionGroupId() != strings.TrimSpace(req.GetSubscriptionGroupId()) {
		return fmt.Errorf("subscription_group_id must be nonempty and canonical")
	}
	if strings.TrimSpace(req.GetJobCategoryId()) == "" || req.GetJobCategoryId() != strings.TrimSpace(req.GetJobCategoryId()) {
		return fmt.Errorf("job_category_id must be nonempty and canonical")
	}
	if req.GetRenderProfile() != pb.RenderProfile_RENDER_PROFILE_SUBSCRIPTION_GROUP_OUTCOME_MATRIX_SINGLE_PERIOD_11_V1 {
		return fmt.Errorf("unsupported render profile")
	}
	if req.ExpectedPlanId != nil && (strings.TrimSpace(req.GetExpectedPlanId()) == "" || req.GetExpectedPlanId() != strings.TrimSpace(req.GetExpectedPlanId())) {
		return fmt.Errorf("expected_plan_id must be nonempty and canonical when supplied")
	}
	if req.ExpectedPriceScheduleId != nil && (strings.TrimSpace(req.GetExpectedPriceScheduleId()) == "" || req.GetExpectedPriceScheduleId() != strings.TrimSpace(req.GetExpectedPriceScheduleId())) {
		return fmt.Errorf("expected_price_schedule_id must be nonempty and canonical when supplied")
	}
	return nil
}
