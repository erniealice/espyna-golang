package activity_template

import (
	"context"
	"errors"
	"fmt"

	"leapfor.xyz/espyna/internal/application/ports"
	contextutil "leapfor.xyz/espyna/internal/application/shared/context"
	activityTemplatepb "leapfor.xyz/esqyma/golang/v1/domain/workflow/activity_template"
	stageTemplatepb "leapfor.xyz/esqyma/golang/v1/domain/workflow/stage_template"
)

type GetActivityTemplateItemPageDataRepositories struct {
	ActivityTemplate activityTemplatepb.ActivityTemplateDomainServiceServer
	StageTemplate    stageTemplatepb.StageTemplateDomainServiceServer // Foreign key reference
}

type GetActivityTemplateItemPageDataServices struct {
	TransactionService ports.TransactionService
	TranslationService ports.TranslationService
}

// GetActivityTemplateItemPageDataUseCase handles the business logic for getting activity template item page data
type GetActivityTemplateItemPageDataUseCase struct {
	repositories GetActivityTemplateItemPageDataRepositories
	services     GetActivityTemplateItemPageDataServices
}

// NewGetActivityTemplateItemPageDataUseCase creates a new GetActivityTemplateItemPageDataUseCase
func NewGetActivityTemplateItemPageDataUseCase(
	repositories GetActivityTemplateItemPageDataRepositories,
	services GetActivityTemplateItemPageDataServices,
) *GetActivityTemplateItemPageDataUseCase {
	return &GetActivityTemplateItemPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get activity template item page data operation
func (uc *GetActivityTemplateItemPageDataUseCase) Execute(
	ctx context.Context,
	req *activityTemplatepb.GetActivityTemplateItemPageDataRequest,
) (*activityTemplatepb.GetActivityTemplateItemPageDataResponse, error) {
	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req.ActivityTemplateId); err != nil {
		return nil, err
	}

	// Use transaction service if available
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes activity template item page data retrieval within a transaction
func (uc *GetActivityTemplateItemPageDataUseCase) executeWithTransaction(
	ctx context.Context,
	req *activityTemplatepb.GetActivityTemplateItemPageDataRequest,
) (*activityTemplatepb.GetActivityTemplateItemPageDataResponse, error) {
	var result *activityTemplatepb.GetActivityTemplateItemPageDataResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
				txCtx,
				uc.services.TranslationService,
				"activity_template.errors.item_page_data_failed",
				"activity template item page data retrieval failed: %w",
			), err)
		}
		result = res
		return nil
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// executeCore contains the core business logic for getting activity template item page data
func (uc *GetActivityTemplateItemPageDataUseCase) executeCore(
	ctx context.Context,
	req *activityTemplatepb.GetActivityTemplateItemPageDataRequest,
) (*activityTemplatepb.GetActivityTemplateItemPageDataResponse, error) {
	// Create read request for the activity template
	readReq := &activityTemplatepb.ReadActivityTemplateRequest{
		Data: &activityTemplatepb.ActivityTemplate{
			Id: req.ActivityTemplateId,
		},
	}

	// Retrieve the activity template
	readResp, err := uc.repositories.ActivityTemplate.ReadActivityTemplate(ctx, readReq)
	if err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"activity_template.errors.read_failed",
			"failed to retrieve activity template: %w",
		), err)
	}

	if readResp == nil || len(readResp.Data) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"activity_template.errors.not_found",
			"activity template not found",
		))
	}

	// Get the activity template (should be only one)
	activityTemplate := readResp.Data[0]

	// Validate that we got the expected activity template
	if activityTemplate.Id != req.ActivityTemplateId {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"activity_template.errors.id_mismatch",
			"retrieved activity template ID does not match requested ID",
		))
	}

	// TODO: In a real implementation, you might want to:
	// 1. Load related data (stage_template) if not already populated
	// 2. Apply business rules for data visibility/access control
	// 3. Format data for optimal frontend consumption
	// 4. Add audit logging for item access
	// 5. Load hierarchical workflow structures and stage template relationships

	// For now, return the activity template as-is
	return &activityTemplatepb.GetActivityTemplateItemPageDataResponse{
		ActivityTemplate: activityTemplate,
		Success:          true,
	}, nil
}

// validateInput validates the input request
func (uc *GetActivityTemplateItemPageDataUseCase) validateInput(
	ctx context.Context,
	req *activityTemplatepb.GetActivityTemplateItemPageDataRequest,
) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"activity_template.validation.request_required",
			"request is required",
		))
	}

	if req.ActivityTemplateId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"activity_template.validation.id_required",
			"activity template ID is required",
		))
	}

	return nil
}

// validateBusinessRules enforces business constraints for reading activity template item page data
func (uc *GetActivityTemplateItemPageDataUseCase) validateBusinessRules(
	ctx context.Context,
	activityTemplateId string,
) error {
	// Validate activity template ID format
	if len(activityTemplateId) < 3 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"activity_template.validation.id_too_short",
			"activity template ID is too short",
		))
	}

	// Additional business rules could be added here:
	// - Check user permissions to access this activity template
	// - Validate activity template belongs to the current user's organization/institution
	// - Check if activity template is in a state that allows viewing
	// - Rate limiting for activity template access
	// - Audit logging requirements
	// - Workflow domain specific validations (template access, etc.)

	return nil
}

// Optional: Helper methods for future enhancements

// loadRelatedData loads related entities like stage_template details
// This would be called from executeCore if needed
func (uc *GetActivityTemplateItemPageDataUseCase) loadRelatedData(
	ctx context.Context,
	activityTemplate *activityTemplatepb.ActivityTemplate,
) error {
	// TODO: Implement loading of related data
	// This could involve calls to stage_template repositories
	// to populate the nested objects if they're not already loaded

	// Example implementation would be:
	// if activityTemplate.StageTemplate == nil && activityTemplate.StageTemplateId != "" {
	//     // Load stage_template data
	// }

	return nil
}

// applyDataTransformation applies any necessary data transformations for the frontend
func (uc *GetActivityTemplateItemPageDataUseCase) applyDataTransformation(
	ctx context.Context,
	activityTemplate *activityTemplatepb.ActivityTemplate,
) *activityTemplatepb.ActivityTemplate {
	// TODO: Apply any transformations needed for optimal frontend consumption
	// This could include:
	// - Formatting dates
	// - Computing derived fields (e.g., duration calculations)
	// - Applying localization
	// - Sanitizing sensitive data
	// - Organizing hierarchical workflow structures

	return activityTemplate
}

// checkAccessPermissions validates user has permission to access this activity template
func (uc *GetActivityTemplateItemPageDataUseCase) checkAccessPermissions(
	ctx context.Context,
	activityTemplateId string,
) error {
	// TODO: Implement proper access control
	// This could involve:
	// - Checking user role/permissions
	// - Validating activity template belongs to user's institution
	// - Applying multi-tenant access controls
	// - Workflow domain specific permission checks

	return nil
}

// loadHierarchicalData loads activity template hierarchical relationships
func (uc *GetActivityTemplateItemPageDataUseCase) loadHierarchicalData(
	ctx context.Context,
	activityTemplate *activityTemplatepb.ActivityTemplate,
) error {
	// TODO: Implement hierarchical data loading for activity templates
	// This would involve:
	// - Loading stage_template that this activity template belongs to
	// - Organizing the data in a hierarchical structure
	// - Applying proper template filtering

	return nil
}
