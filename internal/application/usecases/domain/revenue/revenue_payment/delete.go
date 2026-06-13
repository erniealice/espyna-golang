package revenuepayment

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue_payment"
)

// DeleteRevenuePaymentRepositories groups all repository dependencies
type DeleteRevenuePaymentRepositories struct {
	RevenuePayment pb.RevenuePaymentDomainServiceServer
}

// DeleteRevenuePaymentServices groups all business service dependencies
type DeleteRevenuePaymentServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// DeleteRevenuePaymentUseCase handles the business logic for deleting revenue payments
type DeleteRevenuePaymentUseCase struct {
	repositories DeleteRevenuePaymentRepositories
	services     DeleteRevenuePaymentServices
}

// NewDeleteRevenuePaymentUseCase creates use case with grouped dependencies
func NewDeleteRevenuePaymentUseCase(
	repositories DeleteRevenuePaymentRepositories,
	services DeleteRevenuePaymentServices,
) *DeleteRevenuePaymentUseCase {
	return &DeleteRevenuePaymentUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the delete revenue payment operation
func (uc *DeleteRevenuePaymentUseCase) Execute(ctx context.Context, req *pb.DeleteRevenuePaymentRequest) (*pb.DeleteRevenuePaymentResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityRevenuePayment, entityid.ActionDelete); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "revenue_payment.errors.input_validation_failed", "[ERR-DEFAULT] Input validation failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	resp, err := uc.repositories.RevenuePayment.DeleteRevenuePayment(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "revenue_payment.errors.deletion_failed", "[ERR-DEFAULT] Revenue payment deletion failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *DeleteRevenuePaymentUseCase) validateInput(ctx context.Context, req *pb.DeleteRevenuePaymentRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "revenue_payment.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "revenue_payment.validation.data_required", "[ERR-DEFAULT] Data is required"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "revenue_payment.validation.id_required", "[ERR-DEFAULT] ID is required"))
	}
	return nil
}
