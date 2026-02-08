package workflow_template

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	workflow_templatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/workflow/workflow_template"
)

type GetWorkflowTemplateItemPageDataRepositories struct {
	WorkflowTemplate workflow_templatepb.WorkflowTemplateDomainServiceServer
}

type GetWorkflowTemplateItemPageDataServices struct {
	TransactionService ports.TransactionService
	TranslationService ports.TranslationService
}

// GetWorkflowTemplateItemPageDataUseCase handles the business logic for getting workflow template item page data
type GetWorkflowTemplateItemPageDataUseCase struct {
	repositories GetWorkflowTemplateItemPageDataRepositories
	services     GetWorkflowTemplateItemPageDataServices
}

// NewGetWorkflowTemplateItemPageDataUseCase creates a new GetWorkflowTemplateItemPageDataUseCase
func NewGetWorkflowTemplateItemPageDataUseCase(
	repositories GetWorkflowTemplateItemPageDataRepositories,
	services GetWorkflowTemplateItemPageDataServices,
) *GetWorkflowTemplateItemPageDataUseCase {
	return &GetWorkflowTemplateItemPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get workflow template item page data operation
func (uc *GetWorkflowTemplateItemPageDataUseCase) Execute(
	ctx context.Context,
	req *workflow_templatepb.GetWorkflowTemplateItemPageDataRequest,
) (*workflow_templatepb.GetWorkflowTemplateItemPageDataResponse, error) {
	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req.WorkflowTemplateId); err != nil {
		return nil, err
	}

	// Use transaction service if available
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes workflow template item page data retrieval within a transaction
func (uc *GetWorkflowTemplateItemPageDataUseCase) executeWithTransaction(
	ctx context.Context,
	req *workflow_templatepb.GetWorkflowTemplateItemPageDataRequest,
) (*workflow_templatepb.GetWorkflowTemplateItemPageDataResponse, error) {
	var result *workflow_templatepb.GetWorkflowTemplateItemPageDataResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
				txCtx,
				uc.services.TranslationService,
				"workflow_template.errors.item_page_data_failed",
				"workflow template item page data retrieval failed: %w",
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

// executeCore contains the core business logic for getting workflow template item page data
func (uc *GetWorkflowTemplateItemPageDataUseCase) executeCore(
	ctx context.Context,
	req *workflow_templatepb.GetWorkflowTemplateItemPageDataRequest,
) (*workflow_templatepb.GetWorkflowTemplateItemPageDataResponse, error) {
	// Create read request for the workflow template
	readReq := &workflow_templatepb.ReadWorkflowTemplateRequest{
		Data: &workflow_templatepb.WorkflowTemplate{
			Id: req.WorkflowTemplateId,
		},
	}

	// Retrieve the workflow template
	readResp, err := uc.repositories.WorkflowTemplate.ReadWorkflowTemplate(ctx, readReq)
	if err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"workflow_template.errors.read_failed",
			"failed to retrieve workflow template: %w",
		), err)
	}

	if readResp == nil || len(readResp.Data) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"workflow_template.errors.not_found",
			"workflow template not found",
		))
	}

	// Get the workflow template (should be only one)
	workflowTemplate := readResp.Data[0]

	// Validate that we got the expected workflow template
	if workflowTemplate.Id != req.WorkflowTemplateId {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"workflow_template.errors.id_mismatch",
			"retrieved workflow template ID does not match requested ID",
		))
	}

	// Apply any necessary business logic or data transformations
	workflowTemplate = uc.applyDataTransformation(ctx, workflowTemplate)

	// TODO: In a real implementation, you might want to:
	// 1. Load related data (stage_templates, activity_templates) if not already populated
	// 2. Apply business rules for data visibility/access control
	// 3. Format data for optimal frontend consumption
	// 4. Add audit logging for item access
	// 5. Load hierarchical workflow template structures and stage template relationships

	// For now, return the workflow template as-is
	return &workflow_templatepb.GetWorkflowTemplateItemPageDataResponse{
		WorkflowTemplate: workflowTemplate,
		Success:          true,
	}, nil
}

// validateInput validates the input request
func (uc *GetWorkflowTemplateItemPageDataUseCase) validateInput(
	ctx context.Context,
	req *workflow_templatepb.GetWorkflowTemplateItemPageDataRequest,
) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"workflow_template.validation.request_required",
			"request is required",
		))
	}

	if req.WorkflowTemplateId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"workflow_template.validation.id_required",
			"workflow template ID is required",
		))
	}

	return nil
}

// validateBusinessRules enforces business constraints for reading workflow template item page data
func (uc *GetWorkflowTemplateItemPageDataUseCase) validateBusinessRules(
	ctx context.Context,
	workflowTemplateId string,
) error {
	// Validate workflow template ID format
	if len(workflowTemplateId) < 3 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"workflow_template.validation.id_too_short",
			"workflow template ID is too short",
		))
	}

	// Additional business rules could be added here:
	// - Check user permissions to access this workflow template
	// - Validate workflow template belongs to the current user's organization/institution
	// - Check if workflow template is in a state that allows viewing
	// - Rate limiting for workflow template access
	// - Audit logging requirements
	// - Workflow template domain specific validations (template access, etc.)

	return nil
}

// applyDataTransformation applies any necessary data transformations for the frontend
func (uc *GetWorkflowTemplateItemPageDataUseCase) applyDataTransformation(
	ctx context.Context,
	workflowTemplate *workflow_templatepb.WorkflowTemplate,
) *workflow_templatepb.WorkflowTemplate {
	// TODO: Apply any transformations needed for optimal frontend consumption
	// This could include:
	// - Formatting dates
	// - Computing derived fields (e.g., stage template counts)
	// - Applying localization
	// - Sanitizing sensitive data
	// - Organizing hierarchical workflow template structures

	return workflowTemplate
}

// Optional: Helper methods for future enhancements

// loadRelatedData loads related entities like stage_templates and activity_template details
// This would be called from executeCore if needed
func (uc *GetWorkflowTemplateItemPageDataUseCase) loadRelatedData(
	ctx context.Context,
	workflowTemplate *workflow_templatepb.WorkflowTemplate,
) error {
	// TODO: Implement loading of related data
	// This could involve calls to stage_template and activity_template repositories
	// to populate the nested objects if they're not already loaded

	// Example implementation would be:
	// if workflowTemplate.StageTemplates == nil && workflowTemplate.Id != "" {
	//     // Load stage_template data
	// }
	// if workflowTemplate.ActivityTemplates == nil && workflowTemplate.Id != "" {
	//     // Load related activity_template data
	// }

	return nil
}

// checkAccessPermissions validates user has permission to access this workflow template
func (uc *GetWorkflowTemplateItemPageDataUseCase) checkAccessPermissions(
	ctx context.Context,
	workflowTemplateId string,
) error {
	// TODO: Implement proper access control
	// This could involve:
	// - Checking user role/permissions
	// - Validating workflow template belongs to user's institution
	// - Applying multi-tenant access controls
	// - Workflow template domain specific permission checks

	return nil
}

// loadHierarchicalData loads workflow template hierarchy (workflow_template -> stage_templates -> activity_templates)
func (uc *GetWorkflowTemplateItemPageDataUseCase) loadHierarchicalData(
	ctx context.Context,
	workflowTemplate *workflow_templatepb.WorkflowTemplate,
) error {
	// TODO: Implement hierarchical data loading for workflow templates
	// This would involve:
	// - Loading stage_templates that belong to this workflow template
	// - Loading activity_templates associated with each stage_template
	// - Organizing the data in a hierarchical structure
	// - Applying proper template filtering

	return nil
}
