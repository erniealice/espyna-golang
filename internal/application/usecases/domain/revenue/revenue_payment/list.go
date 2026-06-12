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

// ListRevenuePaymentsRepositories groups all repository dependencies
type ListRevenuePaymentsRepositories struct {
	RevenuePayment pb.RevenuePaymentDomainServiceServer
}

// ListRevenuePaymentsServices groups all business service dependencies
type ListRevenuePaymentsServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// ListRevenuePaymentsUseCase handles the business logic for listing revenue payments
type ListRevenuePaymentsUseCase struct {
	repositories ListRevenuePaymentsRepositories
	services     ListRevenuePaymentsServices
}

// NewListRevenuePaymentsUseCase creates use case with grouped dependencies
func NewListRevenuePaymentsUseCase(
	repositories ListRevenuePaymentsRepositories,
	services ListRevenuePaymentsServices,
) *ListRevenuePaymentsUseCase {
	return &ListRevenuePaymentsUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the list revenue payments operation.
//
// ListRevenuePaymentsRequest carries an optional FilterRequest (filters=2) which the
// W4 adapter honors as a server-side revenue_id filter (design doc §4 / §5.3). The
// usecase passes filters through unchanged.
func (uc *ListRevenuePaymentsUseCase) Execute(ctx context.Context, req *pb.ListRevenuePaymentsRequest) (*pb.ListRevenuePaymentsResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityRevenuePayment, entityid.ActionList); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "revenue_payment.errors.input_validation_failed", "[ERR-DEFAULT] Input validation failed")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	// Call repository
	resp, err := uc.repositories.RevenuePayment.ListRevenuePayments(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "revenue_payment.errors.list_failed", "[ERR-DEFAULT] Failed to list revenue payments")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *ListRevenuePaymentsUseCase) validateInput(ctx context.Context, req *pb.ListRevenuePaymentsRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "revenue_payment.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}
	return nil
}
