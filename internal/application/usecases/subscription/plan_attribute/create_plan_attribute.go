package plan_attribute

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"leapfor.xyz/espyna/internal/application/ports"
	contextutil "leapfor.xyz/espyna/internal/application/shared/context"
	attributepb "leapfor.xyz/esqyma/golang/v1/domain/common"
	planpb "leapfor.xyz/esqyma/golang/v1/domain/subscription/plan"
	planattributepb "leapfor.xyz/esqyma/golang/v1/domain/subscription/plan_attribute"
)

// CreatePlanAttributeRepositories groups all repository dependencies
type CreatePlanAttributeRepositories struct {
	PlanAttribute planattributepb.PlanAttributeDomainServiceServer // Primary entity repository
	Plan          planpb.PlanDomainServiceServer                   // Entity reference validation
	Attribute     attributepb.AttributeDomainServiceServer         // Entity reference validation
}

// CreatePlanAttributeServices groups all business service dependencies
type CreatePlanAttributeServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// CreatePlanAttributeUseCase handles the business logic for creating plan attributes
type CreatePlanAttributeUseCase struct {
	repositories CreatePlanAttributeRepositories
	services     CreatePlanAttributeServices
}

// NewCreatePlanAttributeUseCase creates use case with grouped dependencies
func NewCreatePlanAttributeUseCase(
	repositories CreatePlanAttributeRepositories,
	services CreatePlanAttributeServices,
) *CreatePlanAttributeUseCase {
	return &CreatePlanAttributeUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the create plan attribute operation
func (uc *CreatePlanAttributeUseCase) Execute(ctx context.Context, req *planattributepb.CreatePlanAttributeRequest) (*planattributepb.CreatePlanAttributeResponse, error) {
	// Input validation (must be done first to avoid nil pointer access)
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// TODO: Re-enable workspace-scoped authorization check once Plan.WorkspaceId is available
	// if uc.services.AuthorizationService != nil && uc.services.AuthorizationService.IsEnabled() {
	// 	userID := contextutil.ExtractUserIDFromContext(ctx)
	// 	if userID == "" {
	// 		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "plan_attribute.errors.user_not_authenticated", "User not authenticated [DEFAULT]"))
	// 	}
	//
	// 	permission := ports.EntityPermission(ports.EntityPlanAttribute, ports.ActionCreate)
	// 	hasPerm, err := uc.services.AuthorizationService.HasPermission(ctx, userID, permission)
	// 	if err != nil {
	// 		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "plan_attribute.errors.authorization_failed", "Authorization failed [DEFAULT]")
	// 		return nil, fmt.Errorf("%s: %w", translatedError, err)
	// 	}
	// 	if !hasPerm {
	// 		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "plan_attribute.errors.authorization_failed", "Authorization failed [DEFAULT]")
	// 		return nil, errors.New(translatedError)
	// 	}
	// }

	// Business logic and enrichment
	if err := uc.enrichPlanAttributeData(req.Data); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "plan_attribute.errors.enrichment_failed", "Business logic enrichment failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Entity reference validation
	if err := uc.validateEntityReferences(ctx, req.Data); err != nil {
		return nil, err
	}

	// Call repository
	resp, err := uc.repositories.PlanAttribute.CreatePlanAttribute(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "plan_attribute.errors.creation_failed", "Plan attribute creation failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *CreatePlanAttributeUseCase) validateInput(ctx context.Context, req *planattributepb.CreatePlanAttributeRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "plan_attribute.validation.request_required", "Request is required [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "plan_attribute.validation.data_required", "Data is required [DEFAULT]"))
	}
	if req.Data.PlanId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "plan_attribute.validation.plan_id_required", "Plan ID is required [DEFAULT]"))
	}
	if req.Data.AttributeId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "plan_attribute.validation.attribute_id_required", "Attribute ID is required [DEFAULT]"))
	}
	if req.Data.Value == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "plan_attribute.validation.value_required", "Value is required [DEFAULT]"))
	}
	return nil
}

// enrichPlanAttributeData adds generated fields and audit information
func (uc *CreatePlanAttributeUseCase) enrichPlanAttributeData(planAttribute *planattributepb.PlanAttribute) error {
	now := time.Now()

	// Generate PlanAttribute ID
	if planAttribute.Id == "" {
		planAttribute.Id = uc.services.IDService.GenerateID()
	}

	// Set plan attribute audit fields
	planAttribute.DateCreated = &[]int64{now.UnixMilli()}[0]
	planAttribute.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	planAttribute.DateModified = &[]int64{now.UnixMilli()}[0]
	planAttribute.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	planAttribute.Active = true

	return nil
}

// validateEntityReferences validates that all referenced entities exist
func (uc *CreatePlanAttributeUseCase) validateEntityReferences(ctx context.Context, planAttribute *planattributepb.PlanAttribute) error {
	// Validate Plan entity reference
	if planAttribute.PlanId != "" {
		plan, err := uc.repositories.Plan.ReadPlan(ctx, &planpb.ReadPlanRequest{
			Data: &planpb.Plan{Id: &planAttribute.PlanId},
		})
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "plan_attribute.errors.plan_reference_validation_failed", "Failed to validate plan entity reference [DEFAULT]")
			return fmt.Errorf("%s: %w", translatedError, err)
		}
		if plan == nil || plan.Data == nil || len(plan.Data) == 0 {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "plan_attribute.errors.plan_not_found", "Plan not found [DEFAULT]")
			translatedError = strings.ReplaceAll(translatedError, "{planId}", planAttribute.PlanId)
			return errors.New(translatedError)
		}
		if !plan.Data[0].Active {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "plan_attribute.errors.plan_not_active", "Referenced plan with ID '{planId}' is not active [DEFAULT]")
			translatedError = strings.ReplaceAll(translatedError, "{planId}", planAttribute.PlanId)
			return errors.New(translatedError)
		}
	}

	// Validate Attribute entity reference
	if planAttribute.AttributeId != "" {
		attribute, err := uc.repositories.Attribute.ReadAttribute(ctx, &attributepb.ReadAttributeRequest{
			Data: &attributepb.Attribute{Id: planAttribute.AttributeId},
		})
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "plan_attribute.errors.attribute_reference_validation_failed", "Failed to validate attribute entity reference [DEFAULT]")
			return fmt.Errorf("%s: %w", translatedError, err)
		}
		if attribute == nil || attribute.Data == nil || len(attribute.Data) == 0 {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "plan_attribute.errors.attribute_not_found", "Attribute not found [DEFAULT]")
			translatedError = strings.ReplaceAll(translatedError, "{attributeId}", planAttribute.AttributeId)
			return errors.New(translatedError)
		}
		if !attribute.Data[0].Active {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "plan_attribute.errors.attribute_not_active", "Referenced attribute with ID '{attributeId}' is not active [DEFAULT]")
			translatedError = strings.ReplaceAll(translatedError, "{attributeId}", planAttribute.AttributeId)
			return errors.New(translatedError)
		}
	}

	return nil
}
