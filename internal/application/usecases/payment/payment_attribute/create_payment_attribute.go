package payment_attribute

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"leapfor.xyz/espyna/internal/application/ports"
	contextutil "leapfor.xyz/espyna/internal/application/shared/context"
	attributepb "leapfor.xyz/esqyma/golang/v1/domain/common"
	paymentpb "leapfor.xyz/esqyma/golang/v1/domain/payment/payment"
	paymentattributepb "leapfor.xyz/esqyma/golang/v1/domain/payment/payment_attribute"
)

// CreatePaymentAttributeUseCase handles the business logic for creating payment attributes
// CreatePaymentAttributeRepositories groups all repository dependencies
type CreatePaymentAttributeRepositories struct {
	PaymentAttribute paymentattributepb.PaymentAttributeDomainServiceServer // Primary entity repository
	Payment          paymentpb.PaymentDomainServiceServer
	Attribute        attributepb.AttributeDomainServiceServer
}

// CreatePaymentAttributeServices groups all business service dependencies
type CreatePaymentAttributeServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// CreatePaymentAttributeUseCase handles the business logic for creating payment attributes
type CreatePaymentAttributeUseCase struct {
	repositories CreatePaymentAttributeRepositories
	services     CreatePaymentAttributeServices
}

// NewCreatePaymentAttributeUseCase creates a new CreatePaymentAttributeUseCase
func NewCreatePaymentAttributeUseCase(
	repositories CreatePaymentAttributeRepositories,
	services CreatePaymentAttributeServices,
) *CreatePaymentAttributeUseCase {
	return &CreatePaymentAttributeUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the create payment attribute operation
func (uc *CreatePaymentAttributeUseCase) Execute(ctx context.Context, req *paymentattributepb.CreatePaymentAttributeRequest) (*paymentattributepb.CreatePaymentAttributeResponse, error) {
	// Check if transaction service is available and supports transactions
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes payment attribute creation within a transaction
func (uc *CreatePaymentAttributeUseCase) executeWithTransaction(ctx context.Context, req *paymentattributepb.CreatePaymentAttributeRequest) (*paymentattributepb.CreatePaymentAttributeResponse, error) {
	var result *paymentattributepb.CreatePaymentAttributeResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "payment_attribute.errors.creation_failed", "Payment attribute creation failed [DEFAULT]"), err)
		}
		result = res
		return nil
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// executeCore contains the core business logic (moved from original Execute method)
func (uc *CreatePaymentAttributeUseCase) executeCore(ctx context.Context, req *paymentattributepb.CreatePaymentAttributeRequest) (*paymentattributepb.CreatePaymentAttributeResponse, error) {
	// TODO: Re-enable workspace-scoped authorization check once WorkspaceId is available
	// userID, err := contextutil.RequireUserIDFromContext(ctx)
	// if err != nil {
	// 	translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_attribute.errors.authorization_failed", "Authorization failed for payment attributes [DEFAULT]")
	// 	return nil, errors.New(translatedError)
	// }
	// permission := ports.EntityPermission(ports.EntityPaymentAttribute, ports.ActionCreate)
	// hasPerm, err := uc.services.AuthorizationService.HasPermission(ctx, userID, permission)
	// if err != nil {
	// 	translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_attribute.errors.authorization_failed", "Authorization failed for payment attributes [DEFAULT]")
	// 	return nil, errors.New(translatedError)
	// }
	// if !hasPerm {
	// 	translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_attribute.errors.authorization_failed", "Authorization failed for payment attributes [DEFAULT]")
	// 	return nil, errors.New(translatedError)
	// }

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_attribute.errors.validation_failed", "Input validation failed [DEFAULT]"), err)
	}

	// Business logic and enrichment
	if err := uc.enrichPaymentAttributeData(req.Data); err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_attribute.errors.enrichment_failed", "Business logic enrichment failed [DEFAULT]"), err)
	}

	// Entity reference validation
	if err := uc.validateEntityReferences(ctx, req.Data); err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_attribute.errors.entity_reference_validation_failed", "Entity reference validation failed [DEFAULT]"), err)
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_attribute.errors.business_rule_validation_failed", "Business rule validation failed [DEFAULT]"), err)
	}

	// Call repository
	resp, err := uc.repositories.PaymentAttribute.CreatePaymentAttribute(ctx, req)
	if err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_attribute.errors.creation_failed", "Payment attribute creation failed [DEFAULT]"), err)
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *CreatePaymentAttributeUseCase) validateInput(ctx context.Context, req *paymentattributepb.CreatePaymentAttributeRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_attribute.validation.request_required", "Request is required [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_attribute.validation.data_required", "Payment attribute data is required [DEFAULT]"))
	}
	if req.Data.PaymentId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_attribute.validation.payment_id_required", "Payment ID is required [DEFAULT]"))
	}
	if req.Data.AttributeId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_attribute.validation.attribute_id_required", "Attribute ID is required [DEFAULT]"))
	}
	if req.Data.Value == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_attribute.validation.value_required", "Attribute value is required [DEFAULT]"))
	}
	return nil
}

// enrichPaymentAttributeData adds generated fields and audit information
func (uc *CreatePaymentAttributeUseCase) enrichPaymentAttributeData(paymentAttribute *paymentattributepb.PaymentAttribute) error {
	now := time.Now()

	// Generate PaymentAttribute ID if not provided
	if paymentAttribute.Id == "" {
		if uc.services.IDService != nil {
			paymentAttribute.Id = uc.services.IDService.GenerateID()
		} else {
			// Fallback ID generation when service is not available
			paymentAttribute.Id = fmt.Sprintf("payment-attr-%d", now.UnixNano())
		}
	}

	// Set audit fields
	paymentAttribute.DateCreated = &[]int64{now.UnixMilli()}[0]
	paymentAttribute.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	paymentAttribute.DateModified = &[]int64{now.UnixMilli()}[0]
	paymentAttribute.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return nil
}

// validateBusinessRules enforces business constraints for payment attributes
func (uc *CreatePaymentAttributeUseCase) validateBusinessRules(ctx context.Context, paymentAttribute *paymentattributepb.PaymentAttribute) error {
	// Validate payment ID format
	if len(paymentAttribute.PaymentId) < 5 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_attribute.validation.payment_id_min_length", "Payment ID must be at least 5 characters long [DEFAULT]"))
	}

	// Validate attribute ID format
	if len(paymentAttribute.AttributeId) < 2 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_attribute.validation.attribute_id_min_length", "Attribute ID must be at least 2 characters long [DEFAULT]"))
	}

	// Validate attribute value length
	value := strings.TrimSpace(paymentAttribute.Value)
	if len(value) < 1 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_attribute.validation.value_not_empty", "Attribute value must not be empty [DEFAULT]"))
	}

	if len(value) > 500 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_attribute.validation.value_max_length", "Attribute value cannot exceed 500 characters [DEFAULT]"))
	}

	// Normalize value (trim spaces)
	paymentAttribute.Value = strings.TrimSpace(paymentAttribute.Value)

	// Business constraint: Payment attribute must be associated with a valid payment
	if paymentAttribute.PaymentId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_attribute.validation.payment_association_required", "Payment attribute must be associated with a payment [DEFAULT]"))
	}

	return nil
}

// validateEntityReferences validates that all referenced entities exist
func (uc *CreatePaymentAttributeUseCase) validateEntityReferences(ctx context.Context, paymentAttribute *paymentattributepb.PaymentAttribute) error {
	// Validate Payment entity reference
	if paymentAttribute.PaymentId != "" {
		payment, err := uc.repositories.Payment.ReadPayment(ctx, &paymentpb.ReadPaymentRequest{
			Data: &paymentpb.Payment{Id: paymentAttribute.PaymentId},
		})
		if err != nil {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_attribute.errors.payment_reference_validation_failed", "Failed to validate payment entity reference [DEFAULT]"), err)
		}
		if payment == nil || payment.Data == nil || len(payment.Data) == 0 {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_attribute.errors.payment_not_found", "Referenced payment with ID '%s' does not exist [DEFAULT]"), paymentAttribute.PaymentId)
		}
		if !payment.Data[0].Active {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_attribute.errors.payment_not_active", "Referenced payment with ID '%s' is not active [DEFAULT]"), paymentAttribute.PaymentId)
		}
	}

	// Validate Attribute entity reference
	if paymentAttribute.AttributeId != "" {
		attribute, err := uc.repositories.Attribute.ReadAttribute(ctx, &attributepb.ReadAttributeRequest{
			Data: &attributepb.Attribute{Id: paymentAttribute.AttributeId},
		})
		if err != nil {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_attribute.errors.attribute_reference_validation_failed", "Failed to validate attribute entity reference [DEFAULT]"), err)
		}
		if attribute == nil || attribute.Data == nil || len(attribute.Data) == 0 {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_attribute.errors.attribute_not_found", "Referenced attribute with ID '%s' does not exist [DEFAULT]"), paymentAttribute.AttributeId)
		}
		if !attribute.Data[0].Active {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_attribute.errors.attribute_not_active", "Referenced attribute with ID '%s' is not active [DEFAULT]"), paymentAttribute.AttributeId)
		}
	}

	return nil
}
