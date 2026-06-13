package payment_term

import (
	"context"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	paymenttermpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/payment_term"
)

// ListPaymentTermsRepositories groups all repository dependencies
type ListPaymentTermsRepositories struct {
	PaymentTerm paymenttermpb.PaymentTermDomainServiceServer // Primary entity repository
}

// ListPaymentTermsServices groups all business service dependencies
type ListPaymentTermsServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// ListPaymentTermsUseCase handles the business logic for listing payment terms
type ListPaymentTermsUseCase struct {
	repositories ListPaymentTermsRepositories
	services     ListPaymentTermsServices
}

// NewListPaymentTermsUseCase creates use case with grouped dependencies
func NewListPaymentTermsUseCase(
	repositories ListPaymentTermsRepositories,
	services ListPaymentTermsServices,
) *ListPaymentTermsUseCase {
	return &ListPaymentTermsUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewListPaymentTermsUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewListPaymentTermsUseCase with grouped parameters instead
func NewListPaymentTermsUseCaseUngrouped(paymentTermRepo paymenttermpb.PaymentTermDomainServiceServer) *ListPaymentTermsUseCase {
	repositories := ListPaymentTermsRepositories{
		PaymentTerm: paymentTermRepo,
	}

	services := ListPaymentTermsServices{
		Authorizer: nil,
		Transactor: ports.NewNoOpTransactor(),
		Translator: ports.NewNoOpTranslator(),
	}

	return NewListPaymentTermsUseCase(repositories, services)
}

// Execute performs the list payment terms operation
func (uc *ListPaymentTermsUseCase) Execute(ctx context.Context, req *paymenttermpb.ListPaymentTermsRequest) (*paymenttermpb.ListPaymentTermsResponse, error) {
	// Authorization check
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: "payment_term",
		Action: entityid.ActionList,
	}); err != nil {
		return nil, err
	}

	// Input validation
	if req == nil {
		req = &paymenttermpb.ListPaymentTermsRequest{}
	}

	// Call repository
	resp, err := uc.repositories.PaymentTerm.ListPaymentTerms(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "payment_term.errors.list_failed", "Failed to retrieve payment terms [DEFAULT]")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}

	return resp, nil
}
