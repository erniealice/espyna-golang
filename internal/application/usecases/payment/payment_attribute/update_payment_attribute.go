package payment_attribute

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	attributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	paymentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/payment/payment"
	paymentattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/payment/payment_attribute"
)

// UpdatePaymentAttributeUseCase handles the business logic for updating payment attributes
// UpdatePaymentAttributeRepositories groups all repository dependencies
type UpdatePaymentAttributeRepositories struct {
	PaymentAttribute paymentattributepb.PaymentAttributeDomainServiceServer // Primary entity repository
	Payment          paymentpb.PaymentDomainServiceServer
	Attribute        attributepb.AttributeDomainServiceServer
}

// UpdatePaymentAttributeServices groups all business service dependencies
type UpdatePaymentAttributeServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
}

// UpdatePaymentAttributeUseCase handles the business logic for updating payment attributes
type UpdatePaymentAttributeUseCase struct {
	repositories UpdatePaymentAttributeRepositories
	services     UpdatePaymentAttributeServices
}

// NewUpdatePaymentAttributeUseCase creates a new UpdatePaymentAttributeUseCase
func NewUpdatePaymentAttributeUseCase(
	repositories UpdatePaymentAttributeRepositories,
	services UpdatePaymentAttributeServices,
) *UpdatePaymentAttributeUseCase {
	return &UpdatePaymentAttributeUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the update payment attribute operation
func (uc *UpdatePaymentAttributeUseCase) Execute(ctx context.Context, req *paymentattributepb.UpdatePaymentAttributeRequest) (*paymentattributepb.UpdatePaymentAttributeResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		ports.EntityPaymentAttribute, ports.ActionUpdate); err != nil {
		return nil, err
	}

	// Check if transaction service is available and supports transactions
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes payment attribute update within a transaction
func (uc *UpdatePaymentAttributeUseCase) executeWithTransaction(ctx context.Context, req *paymentattributepb.UpdatePaymentAttributeRequest) (*paymentattributepb.UpdatePaymentAttributeResponse, error) {
	var result *paymentattributepb.UpdatePaymentAttributeResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return err
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
func (uc *UpdatePaymentAttributeUseCase) executeCore(ctx context.Context, req *paymentattributepb.UpdatePaymentAttributeRequest) (*paymentattributepb.UpdatePaymentAttributeResponse, error) {
	// Authorization check
	userID, err := contextutil.RequireUserIDFromContext(ctx)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_attribute.errors.authorization_failed", "Authorization failed for payment attributes [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	permission := ports.EntityPermission(ports.EntityPaymentAttribute, ports.ActionUpdate)
	hasPerm, err := uc.services.AuthorizationService.HasPermission(ctx, userID, permission)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_attribute.errors.authorization_failed", "Authorization failed for payment attributes [DEFAULT]")
		return nil, errors.New(translatedError)
	}
	if !hasPerm {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_attribute.errors.authorization_failed", "Authorization failed for payment attributes [DEFAULT]")
		return nil, errors.New(translatedError)
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Business logic and enrichment
	if err := uc.enrichPaymentAttributeData(req.Data); err != nil {
		return nil, err
	}

	// Entity reference validation
	if err := uc.validateEntityReferences(ctx, req.Data); err != nil {
		return nil, err
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		return nil, err
	}

	// Call repository
	resp, err := uc.repositories.PaymentAttribute.UpdatePaymentAttribute(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// validateInput validates the input request
func (uc *UpdatePaymentAttributeUseCase) validateInput(ctx context.Context, req *paymentattributepb.UpdatePaymentAttributeRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_attribute.validation.request_required", "Request is required [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_attribute.validation.data_required", "Payment attribute data is required [DEFAULT]"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "payment_attribute.validation.id_required", "Payment attribute ID is required"))
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
func (uc *UpdatePaymentAttributeUseCase) enrichPaymentAttributeData(paymentAttribute *paymentattributepb.PaymentAttribute) error {
	now := time.Now()

	// Update audit fields
	paymentAttribute.DateModified = &[]int64{now.UnixMilli()}[0]
	paymentAttribute.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return nil
}

// validateBusinessRules enforces business constraints for payment attributes
func (uc *UpdatePaymentAttributeUseCase) validateBusinessRules(ctx context.Context, paymentAttribute *paymentattributepb.PaymentAttribute) error {
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
func (uc *UpdatePaymentAttributeUseCase) validateEntityReferences(ctx context.Context, paymentAttribute *paymentattributepb.PaymentAttribute) error {
	// Validate Payment entity reference
	if paymentAttribute.PaymentId != "" {
		payment, err := uc.repositories.Payment.ReadPayment(ctx, &paymentpb.ReadPaymentRequest{
			Data: &paymentpb.Payment{Id: paymentAttribute.PaymentId},
		})
		if err != nil {
			return err
		}
		if payment == nil || payment.Data == nil || len(payment.Data) == 0 {
			translatedError := contextutil.GetTranslatedMessageWithContextAndTags(ctx, uc.services.TranslationService, "payment_attribute.errors.payment_not_found", map[string]interface{}{"paymentId": paymentAttribute.PaymentId}, "Referenced payment not found")
			return errors.New(translatedError)
		}
		if !payment.Data[0].Active {
			translatedError := contextutil.GetTranslatedMessageWithContextAndTags(ctx, uc.services.TranslationService, "payment_attribute.errors.payment_not_active", map[string]interface{}{"paymentId": paymentAttribute.PaymentId}, "Referenced payment not active")
			return errors.New(translatedError)
		}
	}

	// Validate Attribute entity reference
	if paymentAttribute.AttributeId != "" {
		attribute, err := uc.repositories.Attribute.ReadAttribute(ctx, &attributepb.ReadAttributeRequest{
			Data: &attributepb.Attribute{Id: paymentAttribute.AttributeId},
		})
		if err != nil {
			return err
		}
		if attribute == nil || attribute.Data == nil || len(attribute.Data) == 0 {
			translatedError := contextutil.GetTranslatedMessageWithContextAndTags(ctx, uc.services.TranslationService, "payment_attribute.errors.attribute_not_found", map[string]interface{}{"attributeId": paymentAttribute.AttributeId}, "Referenced attribute not found")
			return errors.New(translatedError)
		}
		if !attribute.Data[0].Active {
			translatedError := contextutil.GetTranslatedMessageWithContextAndTags(ctx, uc.services.TranslationService, "payment_attribute.errors.attribute_not_active", map[string]interface{}{"attributeId": paymentAttribute.AttributeId}, "Referenced attribute not active")
			return errors.New(translatedError)
		}
	}

	return nil
}
