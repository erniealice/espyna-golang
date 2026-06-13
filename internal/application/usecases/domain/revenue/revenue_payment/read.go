package revenuepayment

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue_payment"
)

// ReadRevenuePaymentRepositories groups all repository dependencies
type ReadRevenuePaymentRepositories struct {
	RevenuePayment pb.RevenuePaymentDomainServiceServer
}

// ReadRevenuePaymentServices groups all business service dependencies
type ReadRevenuePaymentServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// ReadRevenuePaymentUseCase handles the business logic for reading revenue payments
type ReadRevenuePaymentUseCase struct {
	repositories ReadRevenuePaymentRepositories
	services     ReadRevenuePaymentServices
}

// NewReadRevenuePaymentUseCase creates use case with grouped dependencies
func NewReadRevenuePaymentUseCase(
	repositories ReadRevenuePaymentRepositories,
	services ReadRevenuePaymentServices,
) *ReadRevenuePaymentUseCase {
	return &ReadRevenuePaymentUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the read revenue payment operation
func (uc *ReadRevenuePaymentUseCase) Execute(ctx context.Context, req *pb.ReadRevenuePaymentRequest) (*pb.ReadRevenuePaymentResponse, error) {
	// Authorization check
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityRevenuePayment,
		Action: entityid.ActionRead,
	}); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Call repository
	resp, err := uc.repositories.RevenuePayment.ReadRevenuePayment(ctx, req)
	if err != nil {
		return nil, err
	}

	// Return response as-is (even if empty data for not found case)
	return resp, nil
}

// validateInput validates the input request
func (uc *ReadRevenuePaymentUseCase) validateInput(ctx context.Context, req *pb.ReadRevenuePaymentRequest) error {
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
