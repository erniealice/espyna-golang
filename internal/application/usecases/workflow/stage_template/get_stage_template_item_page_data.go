package stage_template

import (
	"context"
	"errors"
	"fmt"

	"leapfor.xyz/espyna/internal/application/ports"
	contextutil "leapfor.xyz/espyna/internal/application/shared/context"
	stageTemplatepb "leapfor.xyz/esqyma/golang/v1/domain/workflow/stage_template"
	workflowTemplatepb "leapfor.xyz/esqyma/golang/v1/domain/workflow/workflow_template"
)

type GetStageTemplateItemPageDataRepositories struct {
	StageTemplate    stageTemplatepb.StageTemplateDomainServiceServer       // Primary entity repository
	WorkflowTemplate workflowTemplatepb.WorkflowTemplateDomainServiceServer // Foreign key reference
}

type GetStageTemplateItemPageDataServices struct {
	TransactionService ports.TransactionService
	TranslationService ports.TranslationService
}

// GetStageTemplateItemPageDataUseCase handles the business logic for getting stage template item page data
type GetStageTemplateItemPageDataUseCase struct {
	repositories GetStageTemplateItemPageDataRepositories
	services     GetStageTemplateItemPageDataServices
}

// NewGetStageTemplateItemPageDataUseCase creates a new GetStageTemplateItemPageDataUseCase
func NewGetStageTemplateItemPageDataUseCase(
	repositories GetStageTemplateItemPageDataRepositories,
	services GetStageTemplateItemPageDataServices,
) *GetStageTemplateItemPageDataUseCase {
	return &GetStageTemplateItemPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get stage template item page data operation
func (uc *GetStageTemplateItemPageDataUseCase) Execute(
	ctx context.Context,
	req *stageTemplatepb.GetStageTemplateItemPageDataRequest,
) (*stageTemplatepb.GetStageTemplateItemPageDataResponse, error) {
	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req.StageTemplateId); err != nil {
		return nil, err
	}

	// Use transaction service if available
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes stage template item page data retrieval within a transaction
func (uc *GetStageTemplateItemPageDataUseCase) executeWithTransaction(
	ctx context.Context,
	req *stageTemplatepb.GetStageTemplateItemPageDataRequest,
) (*stageTemplatepb.GetStageTemplateItemPageDataResponse, error) {
	var result *stageTemplatepb.GetStageTemplateItemPageDataResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
				txCtx,
				uc.services.TranslationService,
				"stage_template.errors.item_page_data_failed",
				"stage template item page data retrieval failed: %w",
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

// executeCore contains the core business logic for getting stage template item page data
func (uc *GetStageTemplateItemPageDataUseCase) executeCore(
	ctx context.Context,
	req *stageTemplatepb.GetStageTemplateItemPageDataRequest,
) (*stageTemplatepb.GetStageTemplateItemPageDataResponse, error) {
	// Create read request for the stage template
	readReq := &stageTemplatepb.ReadStageTemplateRequest{
		Data: &stageTemplatepb.StageTemplate{
			Id: req.StageTemplateId,
		},
	}

	// Retrieve the stage template
	readResp, err := uc.repositories.StageTemplate.ReadStageTemplate(ctx, readReq)
	if err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"stage_template.errors.read_failed",
			"failed to retrieve stage template: %w",
		), err)
	}

	if readResp == nil || len(readResp.Data) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"stage_template.errors.not_found",
			"stage template not found",
		))
	}

	// Get the stage template (should be only one)
	stageTemplate := readResp.Data[0]

	// Validate that we got the expected stage template
	if stageTemplate.Id != req.StageTemplateId {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"stage_template.errors.id_mismatch",
			"retrieved stage template ID does not match requested ID",
		))
	}

	// TODO: In a real implementation, you might want to:
	// 1. Load related data (workflow, activity_templates) if not already populated
	// 2. Apply business rules for data visibility/access control
	// 3. Format data for optimal frontend consumption
	// 4. Add audit logging for item access
	// 5. Load hierarchical workflow structures and activity template relationships

	// For now, return the stage template as-is
	return &stageTemplatepb.GetStageTemplateItemPageDataResponse{
		StageTemplate: stageTemplate,
		Success:       true,
	}, nil
}

// validateInput validates the input request
func (uc *GetStageTemplateItemPageDataUseCase) validateInput(
	ctx context.Context,
	req *stageTemplatepb.GetStageTemplateItemPageDataRequest,
) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"stage_template.validation.request_required",
			"request is required",
		))
	}

	if req.StageTemplateId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"stage_template.validation.id_required",
			"stage template ID is required",
		))
	}

	return nil
}

// validateBusinessRules enforces business constraints for reading stage template item page data
func (uc *GetStageTemplateItemPageDataUseCase) validateBusinessRules(
	ctx context.Context,
	stageTemplateId string,
) error {
	// Validate stage template ID format
	if len(stageTemplateId) < 3 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"stage_template.validation.id_too_short",
			"stage template ID is too short",
		))
	}

	// Additional business rules could be added here:
	// - Check user permissions to access this stage template
	// - Validate stage template belongs to the current user's organization/institution
	// - Check if stage template is in a state that allows viewing
	// - Rate limiting for stage template access
	// - Audit logging requirements
	// - Workflow domain specific validations (template access, etc.)

	return nil
}

// Optional: Helper methods for future enhancements

// loadRelatedData loads related entities like workflow and activity_template details
// This would be called from executeCore if needed
func (uc *GetStageTemplateItemPageDataUseCase) loadRelatedData(
	ctx context.Context,
	stageTemplate *stageTemplatepb.StageTemplate,
) error {
	// TODO: Implement loading of related data
	// This could involve calls to workflow and activity_template repositories
	// to populate the nested objects if they're not already loaded

	// Example implementation would be:
	// if stageTemplate.Workflow == nil && stageTemplate.WorkflowId != "" {
	//     // Load workflow data
	// }
	// if stageTemplate.ActivityTemplates == nil && stageTemplate.Id != "" {
	//     // Load related activity_template data
	// }

	return nil
}

// applyDataTransformation applies any necessary data transformations for the frontend
func (uc *GetStageTemplateItemPageDataUseCase) applyDataTransformation(
	ctx context.Context,
	stageTemplate *stageTemplatepb.StageTemplate,
) *stageTemplatepb.StageTemplate {
	// TODO: Apply any transformations needed for optimal frontend consumption
	// This could include:
	// - Formatting dates
	// - Computing derived fields (e.g., activity template counts)
	// - Applying localization
	// - Sanitizing sensitive data
	// - Organizing hierarchical workflow structures

	return stageTemplate
}

// checkAccessPermissions validates user has permission to access this stage template
func (uc *GetStageTemplateItemPageDataUseCase) checkAccessPermissions(
	ctx context.Context,
	stageTemplateId string,
) error {
	// TODO: Implement proper access control
	// This could involve:
	// - Checking user role/permissions
	// - Validating stage template belongs to user's institution
	// - Applying multi-tenant access controls
	// - Workflow domain specific permission checks

	return nil
}

// loadHierarchicalData loads stage template hierarchy (stage_template -> activity_templates)
func (uc *GetStageTemplateItemPageDataUseCase) loadHierarchicalData(
	ctx context.Context,
	stageTemplate *stageTemplatepb.StageTemplate,
) error {
	// TODO: Implement hierarchical data loading for stage templates
	// This would involve:
	// - Loading activity_templates that belong to this stage template
	// - Organizing the data in a hierarchical structure
	// - Applying proper template filtering

	return nil
}
