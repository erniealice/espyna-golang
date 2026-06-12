package revenuepayment

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue_payment"
)

// UpdateRevenuePaymentRepositories groups all repository dependencies
type UpdateRevenuePaymentRepositories struct {
	RevenuePayment pb.RevenuePaymentDomainServiceServer
}

// UpdateRevenuePaymentServices groups all business service dependencies
type UpdateRevenuePaymentServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// UpdateRevenuePaymentUseCase handles the business logic for updating revenue payments
type UpdateRevenuePaymentUseCase struct {
	repositories UpdateRevenuePaymentRepositories
	services     UpdateRevenuePaymentServices
}

// NewUpdateRevenuePaymentUseCase creates use case with grouped dependencies
func NewUpdateRevenuePaymentUseCase(
	repositories UpdateRevenuePaymentRepositories,
	services UpdateRevenuePaymentServices,
) *UpdateRevenuePaymentUseCase {
	return &UpdateRevenuePaymentUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the update revenue payment operation
func (uc *UpdateRevenuePaymentUseCase) Execute(ctx context.Context, req *pb.UpdateRevenuePaymentRequest) (*pb.UpdateRevenuePaymentResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityRevenuePayment, entityid.ActionUpdate); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "revenue_payment.errors.input_validation_failed", "[ERR-DEFAULT] Input validation failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Business logic and enrichment
	if err := uc.enrichRevenuePaymentData(req.Data); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "revenue_payment.errors.enrichment_failed", "[ERR-DEFAULT] Data enrichment failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	resp, err := uc.repositories.RevenuePayment.UpdateRevenuePayment(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "revenue_payment.errors.update_failed", "[ERR-DEFAULT] Revenue payment update failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *UpdateRevenuePaymentUseCase) validateInput(ctx context.Context, req *pb.UpdateRevenuePaymentRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "revenue_payment.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "revenue_payment.validation.data_required", "[ERR-DEFAULT] Revenue payment data is required"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "revenue_payment.validation.id_required", "[ERR-DEFAULT] Revenue payment ID is required"))
	}
	return nil
}

// enrichRevenuePaymentData refreshes the modification audit fields for updates
func (uc *UpdateRevenuePaymentUseCase) enrichRevenuePaymentData(payment *pb.RevenuePayment) error {
	now := time.Now()

	payment.DateModified = &[]int64{now.UnixMilli()}[0]
	payment.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return nil
}
