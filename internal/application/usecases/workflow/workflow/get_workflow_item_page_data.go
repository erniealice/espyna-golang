package workflow

import (
	"context"
	"errors"
	"fmt"

	"leapfor.xyz/espyna/internal/application/ports"
	contextutil "leapfor.xyz/espyna/internal/application/shared/context"
	workflowpb "leapfor.xyz/esqyma/golang/v1/domain/workflow/workflow"
)

type GetWorkflowItemPageDataRepositories struct {
	Workflow workflowpb.WorkflowDomainServiceServer
}

type GetWorkflowItemPageDataServices struct {
	TransactionService ports.TransactionService
	TranslationService ports.TranslationService
}

// GetWorkflowItemPageDataUseCase handles the business logic for getting workflow item page data
type GetWorkflowItemPageDataUseCase struct {
	repositories GetWorkflowItemPageDataRepositories
	services     GetWorkflowItemPageDataServices
}

// NewGetWorkflowItemPageDataUseCase creates a new GetWorkflowItemPageDataUseCase
func NewGetWorkflowItemPageDataUseCase(
	repositories GetWorkflowItemPageDataRepositories,
	services GetWorkflowItemPageDataServices,
) *GetWorkflowItemPageDataUseCase {
	return &GetWorkflowItemPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get workflow item page data operation
func (uc *GetWorkflowItemPageDataUseCase) Execute(
	ctx context.Context,
	req *workflowpb.GetWorkflowItemPageDataRequest,
) (*workflowpb.GetWorkflowItemPageDataResponse, error) {
	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req.WorkflowId); err != nil {
		return nil, err
	}

	// Use transaction service if available
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes workflow item page data retrieval within a transaction
func (uc *GetWorkflowItemPageDataUseCase) executeWithTransaction(
	ctx context.Context,
	req *workflowpb.GetWorkflowItemPageDataRequest,
) (*workflowpb.GetWorkflowItemPageDataResponse, error) {
	var result *workflowpb.GetWorkflowItemPageDataResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
				txCtx,
				uc.services.TranslationService,
				"workflow.errors.item_page_data_failed",
				"workflow item page data retrieval failed: %w",
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

// executeCore contains the core business logic for getting workflow item page data
func (uc *GetWorkflowItemPageDataUseCase) executeCore(
	ctx context.Context,
	req *workflowpb.GetWorkflowItemPageDataRequest,
) (*workflowpb.GetWorkflowItemPageDataResponse, error) {
	// Create read request for the workflow
	readReq := &workflowpb.ReadWorkflowRequest{
		Data: &workflowpb.Workflow{
			Id: req.WorkflowId,
		},
	}

	// Retrieve the workflow
	readResp, err := uc.repositories.Workflow.ReadWorkflow(ctx, readReq)
	if err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"workflow.errors.read_failed",
			"failed to retrieve workflow: %w",
		), err)
	}

	if readResp == nil || len(readResp.Data) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"workflow.errors.not_found",
			"workflow not found",
		))
	}

	// Get the workflow (should be only one)
	workflow := readResp.Data[0]

	// Validate that we got the expected workflow
	if workflow.Id != req.WorkflowId {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"workflow.errors.id_mismatch",
			"retrieved workflow ID does not match requested ID",
		))
	}

	// TODO: In a real implementation, you might want to:
	// 1. Load related data (stage_templates, activity_templates) if not already populated
	// 2. Apply business rules for data visibility/access control
	// 3. Format data for optimal frontend consumption
	// 4. Add audit logging for item access
	// 5. Load hierarchical workflow structures and stage template relationships

	// For now, return the workflow as-is
	return &workflowpb.GetWorkflowItemPageDataResponse{
		Workflow: workflow,
		Success:  true,
	}, nil
}

// validateInput validates the input request
func (uc *GetWorkflowItemPageDataUseCase) validateInput(
	ctx context.Context,
	req *workflowpb.GetWorkflowItemPageDataRequest,
) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"workflow.validation.request_required",
			"request is required",
		))
	}

	if req.WorkflowId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"workflow.validation.id_required",
			"workflow ID is required",
		))
	}

	return nil
}

// validateBusinessRules enforces business constraints for reading workflow item page data
func (uc *GetWorkflowItemPageDataUseCase) validateBusinessRules(
	ctx context.Context,
	workflowId string,
) error {
	// Validate workflow ID format
	if len(workflowId) < 3 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"workflow.validation.id_too_short",
			"workflow ID is too short",
		))
	}

	// Additional business rules could be added here:
	// - Check user permissions to access this workflow
	// - Validate workflow belongs to the current user's organization/institution
	// - Check if workflow is in a state that allows viewing
	// - Rate limiting for workflow access
	// - Audit logging requirements
	// - Workflow domain specific validations (template access, etc.)

	return nil
}

// Optional: Helper methods for future enhancements

// loadRelatedData loads related entities like stage_templates and activity_template details
// This would be called from executeCore if needed
func (uc *GetWorkflowItemPageDataUseCase) loadRelatedData(
	ctx context.Context,
	workflow *workflowpb.Workflow,
) error {
	// TODO: Implement loading of related data
	// This could involve calls to stage_template and activity_template repositories
	// to populate the nested objects if they're not already loaded

	// Example implementation would be:
	// if workflow.StageTemplates == nil && workflow.Id != "" {
	//     // Load stage_template data
	// }
	// if workflow.ActivityTemplates == nil && workflow.Id != "" {
	//     // Load related activity_template data
	// }

	return nil
}

// applyDataTransformation applies any necessary data transformations for the frontend
func (uc *GetWorkflowItemPageDataUseCase) applyDataTransformation(
	ctx context.Context,
	workflow *workflowpb.Workflow,
) *workflowpb.Workflow {
	// TODO: Apply any transformations needed for optimal frontend consumption
	// This could include:
	// - Formatting dates
	// - Computing derived fields (e.g., stage template counts)
	// - Applying localization
	// - Sanitizing sensitive data
	// - Organizing hierarchical workflow structures

	return workflow
}

// checkAccessPermissions validates user has permission to access this workflow
func (uc *GetWorkflowItemPageDataUseCase) checkAccessPermissions(
	ctx context.Context,
	workflowId string,
) error {
	// TODO: Implement proper access control
	// This could involve:
	// - Checking user role/permissions
	// - Validating workflow belongs to user's institution
	// - Applying multi-tenant access controls
	// - Workflow domain specific permission checks

	return nil
}

// loadHierarchicalData loads workflow hierarchy (workflow -> stage_templates -> activity_templates)
func (uc *GetWorkflowItemPageDataUseCase) loadHierarchicalData(
	ctx context.Context,
	workflow *workflowpb.Workflow,
) error {
	// TODO: Implement hierarchical data loading for workflows
	// This would involve:
	// - Loading stage_templates that belong to this workflow
	// - Loading activity_templates associated with each stage_template
	// - Organizing the data in a hierarchical structure
	// - Applying proper template filtering

	return nil
}
