package balance_attribute

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	attributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	balancepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/balance"
	balanceattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/balance_attribute"
)

// UpdateBalanceAttributeRepositories groups all repository dependencies
type UpdateBalanceAttributeRepositories struct {
	BalanceAttribute balanceattributepb.BalanceAttributeDomainServiceServer // Primary entity repository
	Balance          balancepb.BalanceDomainServiceServer                   // Entity reference validation
	Attribute        attributepb.AttributeDomainServiceServer               // Entity reference validation
}

// UpdateBalanceAttributeServices groups all business service dependencies
type UpdateBalanceAttributeServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService ports.TransactionService
	TranslationService ports.TranslationService
}

// UpdateBalanceAttributeUseCase handles the business logic for updating balance attributes
type UpdateBalanceAttributeUseCase struct {
	repositories UpdateBalanceAttributeRepositories
	services     UpdateBalanceAttributeServices
}

// NewUpdateBalanceAttributeUseCase creates a new UpdateBalanceAttributeUseCase
func NewUpdateBalanceAttributeUseCase(
	repositories UpdateBalanceAttributeRepositories,
	services UpdateBalanceAttributeServices,
) *UpdateBalanceAttributeUseCase {
	return &UpdateBalanceAttributeUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the update balance attribute operation
func (uc *UpdateBalanceAttributeUseCase) Execute(ctx context.Context, req *balanceattributepb.UpdateBalanceAttributeRequest) (*balanceattributepb.UpdateBalanceAttributeResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityBalanceAttribute, ports.ActionUpdate); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Business logic and enrichment
	if err := uc.enrichBalanceAttributeData(req.Data); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "balance_attribute.errors.enrichment_failed", "Business logic enrichment failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Entity reference validation
	if err := uc.validateEntityReferences(ctx, req.Data); err != nil {
		return nil, err
	}

	// Call repository
	resp, err := uc.repositories.BalanceAttribute.UpdateBalanceAttribute(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "balance_attribute.errors.update_failed", "Balance attribute update failed [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *UpdateBalanceAttributeUseCase) validateInput(ctx context.Context, req *balanceattributepb.UpdateBalanceAttributeRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "balance_attribute.validation.request_required", "Request is required [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "balance_attribute.validation.data_required", "Data is required [DEFAULT]"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "balance_attribute.validation.id_required", "Balance attribute ID is required [DEFAULT]"))
	}
	return nil
}

// enrichBalanceAttributeData updates audit information
func (uc *UpdateBalanceAttributeUseCase) enrichBalanceAttributeData(balanceAttribute *balanceattributepb.BalanceAttribute) error {
	now := time.Now()

	// Update audit fields
	balanceAttribute.DateModified = &[]int64{now.UnixMilli()}[0]
	balanceAttribute.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return nil
}

// validateEntityReferences validates that all referenced entities exist
func (uc *UpdateBalanceAttributeUseCase) validateEntityReferences(ctx context.Context, balanceAttribute *balanceattributepb.BalanceAttribute) error {
	// Validate Balance entity reference (if being updated)
	if balanceAttribute.BalanceId != "" {
		balance, err := uc.repositories.Balance.ReadBalance(ctx, &balancepb.ReadBalanceRequest{
			Data: &balancepb.Balance{Id: balanceAttribute.BalanceId},
		})
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "balance_attribute.errors.balance_reference_validation_failed", "Failed to validate balance entity reference [DEFAULT]")
			return fmt.Errorf("%s: %w", translatedError, err)
		}
		if balance == nil || balance.Data == nil || len(balance.Data) == 0 {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "balance_attribute.errors.balance_not_found", "Balance not found [DEFAULT]")
			translatedError = strings.ReplaceAll(translatedError, "{balanceId}", balanceAttribute.BalanceId)
			return errors.New(translatedError)
		}
		if !balance.Data[0].Active {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "balance_attribute.errors.balance_not_active", "Referenced balance with ID '{balanceId}' is not active [DEFAULT]")
			translatedError = strings.ReplaceAll(translatedError, "{balanceId}", balanceAttribute.BalanceId)
			return errors.New(translatedError)
		}
	}

	// Validate Attribute entity reference (if being updated)
	if balanceAttribute.AttributeId != "" {
		attribute, err := uc.repositories.Attribute.ReadAttribute(ctx, &attributepb.ReadAttributeRequest{
			Data: &attributepb.Attribute{Id: balanceAttribute.AttributeId},
		})
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "balance_attribute.errors.attribute_reference_validation_failed", "Failed to validate attribute entity reference [DEFAULT]")
			return fmt.Errorf("%s: %w", translatedError, err)
		}
		if attribute == nil || attribute.Data == nil || len(attribute.Data) == 0 {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "balance_attribute.errors.attribute_not_found", "Attribute not found [DEFAULT]")
			translatedError = strings.ReplaceAll(translatedError, "{attributeId}", balanceAttribute.AttributeId)
			return errors.New(translatedError)
		}
		if !attribute.Data[0].Active {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "balance_attribute.errors.attribute_not_active", "Referenced attribute with ID '{attributeId}' is not active [DEFAULT]")
			translatedError = strings.ReplaceAll(translatedError, "{attributeId}", balanceAttribute.AttributeId)
			return errors.New(translatedError)
		}
	}

	return nil
}
