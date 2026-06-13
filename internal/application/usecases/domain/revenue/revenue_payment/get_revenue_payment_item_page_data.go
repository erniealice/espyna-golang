package revenuepayment

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue_payment"
)

// GetRevenuePaymentItemPageDataRepositories groups all repository dependencies
type GetRevenuePaymentItemPageDataRepositories struct {
	RevenuePayment pb.RevenuePaymentDomainServiceServer // Primary entity repository
}

// GetRevenuePaymentItemPageDataServices groups all business service dependencies
type GetRevenuePaymentItemPageDataServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// GetRevenuePaymentItemPageDataUseCase handles the business logic for getting revenue payment item page data
type GetRevenuePaymentItemPageDataUseCase struct {
	repositories GetRevenuePaymentItemPageDataRepositories
	services     GetRevenuePaymentItemPageDataServices
}

// NewGetRevenuePaymentItemPageDataUseCase creates use case with grouped dependencies
func NewGetRevenuePaymentItemPageDataUseCase(
	repositories GetRevenuePaymentItemPageDataRepositories,
	services GetRevenuePaymentItemPageDataServices,
) *GetRevenuePaymentItemPageDataUseCase {
	return &GetRevenuePaymentItemPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get revenue payment item page data operation
func (uc *GetRevenuePaymentItemPageDataUseCase) Execute(ctx context.Context, req *pb.GetRevenuePaymentItemPageDataRequest) (*pb.GetRevenuePaymentItemPageDataResponse, error) {
	// Authorization check
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityRevenuePayment,
		Action: entityid.ActionRead,
	}); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "revenue_payment.errors.input_validation_failed", "[ERR-DEFAULT] Input validation failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	resp, err := uc.repositories.RevenuePayment.GetRevenuePaymentItemPageData(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "revenue_payment.errors.get_item_page_data_failed", "[ERR-DEFAULT] Failed to load revenue payment details")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *GetRevenuePaymentItemPageDataUseCase) validateInput(ctx context.Context, req *pb.GetRevenuePaymentItemPageDataRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "revenue_payment.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}

	// Validate revenue payment ID
	if req.RevenuePaymentId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "revenue_payment.validation.revenue_payment_id_required", "[ERR-DEFAULT] Revenue payment ID is required"))
	}

	// Ensure ID doesn't contain invalid characters
	if strings.ContainsAny(req.RevenuePaymentId, " \t\n\r") {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "revenue_payment.validation.revenue_payment_id_invalid_characters", "[ERR-DEFAULT] Revenue payment ID contains invalid characters"))
	}

	return nil
}
