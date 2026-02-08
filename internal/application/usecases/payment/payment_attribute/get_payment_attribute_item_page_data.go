package payment_attribute

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	paymentattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/payment/payment_attribute"
)

type GetPaymentAttributeItemPageDataRepositories struct {
	PaymentAttribute paymentattributepb.PaymentAttributeDomainServiceServer
}

type GetPaymentAttributeItemPageDataServices struct {
	TransactionService ports.TransactionService
	TranslationService ports.TranslationService
}

// GetPaymentAttributeItemPageDataUseCase handles the business logic for getting payment attribute item page data
type GetPaymentAttributeItemPageDataUseCase struct {
	repositories GetPaymentAttributeItemPageDataRepositories
	services     GetPaymentAttributeItemPageDataServices
}

// NewGetPaymentAttributeItemPageDataUseCase creates a new GetPaymentAttributeItemPageDataUseCase
func NewGetPaymentAttributeItemPageDataUseCase(
	repositories GetPaymentAttributeItemPageDataRepositories,
	services GetPaymentAttributeItemPageDataServices,
) *GetPaymentAttributeItemPageDataUseCase {
	return &GetPaymentAttributeItemPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get payment attribute item page data operation
func (uc *GetPaymentAttributeItemPageDataUseCase) Execute(
	ctx context.Context,
	req *paymentattributepb.GetPaymentAttributeItemPageDataRequest,
) (*paymentattributepb.GetPaymentAttributeItemPageDataResponse, error) {
	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req.PaymentAttributeId); err != nil {
		return nil, err
	}

	// Use transaction service if available
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes payment attribute item page data retrieval within a transaction
func (uc *GetPaymentAttributeItemPageDataUseCase) executeWithTransaction(
	ctx context.Context,
	req *paymentattributepb.GetPaymentAttributeItemPageDataRequest,
) (*paymentattributepb.GetPaymentAttributeItemPageDataResponse, error) {
	var result *paymentattributepb.GetPaymentAttributeItemPageDataResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
				txCtx,
				uc.services.TranslationService,
				"payment_attribute.errors.item_page_data_failed",
				"payment attribute item page data retrieval failed: %w",
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

// executeCore contains the core business logic for getting payment attribute item page data
func (uc *GetPaymentAttributeItemPageDataUseCase) executeCore(
	ctx context.Context,
	req *paymentattributepb.GetPaymentAttributeItemPageDataRequest,
) (*paymentattributepb.GetPaymentAttributeItemPageDataResponse, error) {
	// Create read request for the payment attribute
	readReq := &paymentattributepb.ReadPaymentAttributeRequest{
		Data: &paymentattributepb.PaymentAttribute{
			Id: req.PaymentAttributeId,
		},
	}

	// Retrieve the payment attribute
	readResp, err := uc.repositories.PaymentAttribute.ReadPaymentAttribute(ctx, readReq)
	if err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"payment_attribute.errors.read_failed",
			"failed to retrieve payment attribute: %w",
		), err)
	}

	if readResp == nil || len(readResp.Data) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"payment_attribute.errors.not_found",
			"payment attribute not found",
		))
	}

	// Get the payment attribute (should be only one)
	paymentAttribute := readResp.Data[0]

	// Validate that we got the expected payment attribute
	if paymentAttribute.Id != req.PaymentAttributeId {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"payment_attribute.errors.id_mismatch",
			"retrieved payment attribute ID does not match requested ID",
		))
	}

	// For now, return the payment attribute as-is
	return &paymentattributepb.GetPaymentAttributeItemPageDataResponse{
		PaymentAttribute: paymentAttribute,
		Success:          true,
	}, nil
}

// validateInput validates the input request
func (uc *GetPaymentAttributeItemPageDataUseCase) validateInput(
	ctx context.Context,
	req *paymentattributepb.GetPaymentAttributeItemPageDataRequest,
) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"payment_attribute.validation.request_required",
			"Request is required for payment attributes [DEFAULT]",
		))
	}

	// Validate payment attribute ID - uses direct field NOT nested Data
	if strings.TrimSpace(req.PaymentAttributeId) == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"payment_attribute.validation.id_required",
			"Payment attribute ID is required [DEFAULT]",
		))
	}

	// Basic ID format validation
	if len(req.PaymentAttributeId) < 3 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"payment_attribute.validation.id_too_short",
			"Payment attribute ID must be at least 3 characters [DEFAULT]",
		))
	}

	return nil
}

// validateBusinessRules enforces business constraints for reading payment attribute item page data
func (uc *GetPaymentAttributeItemPageDataUseCase) validateBusinessRules(
	ctx context.Context,
	paymentAttributeId string,
) error {
	// Validate payment attribute ID format
	if len(paymentAttributeId) < 3 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"payment_attribute.validation.id_too_short",
			"payment attribute ID is too short",
		))
	}

	return nil
}
