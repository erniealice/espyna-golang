package payment_term

import (
	"context"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	paymenttermpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/payment_term"
)

// GetPaymentTermItemPageDataRepositories groups repository dependencies for GetPaymentTermItemPageData use case
type GetPaymentTermItemPageDataRepositories struct {
	PaymentTerm paymenttermpb.PaymentTermDomainServiceServer
}

// GetPaymentTermItemPageDataServices groups service dependencies for GetPaymentTermItemPageData use case
type GetPaymentTermItemPageDataServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// GetPaymentTermItemPageDataUseCase handles getting individual payment term item data
type GetPaymentTermItemPageDataUseCase struct {
	paymenttermpb.UnimplementedPaymentTermDomainServiceServer
	repos    GetPaymentTermItemPageDataRepositories
	services GetPaymentTermItemPageDataServices
}

// NewGetPaymentTermItemPageDataUseCase creates a new GetPaymentTermItemPageData use case
func NewGetPaymentTermItemPageDataUseCase(
	repos GetPaymentTermItemPageDataRepositories,
	services GetPaymentTermItemPageDataServices,
) *GetPaymentTermItemPageDataUseCase {
	return &GetPaymentTermItemPageDataUseCase{
		repos:    repos,
		services: services,
	}
}

// Execute runs the GetPaymentTermItemPageData use case
func (uc *GetPaymentTermItemPageDataUseCase) Execute(
	ctx context.Context,
	req *paymenttermpb.GetPaymentTermItemPageDataRequest,
) (*paymenttermpb.GetPaymentTermItemPageDataResponse, error) {
	// Authorization check
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: "payment_term",
		Action: entityid.ActionRead,
	}); err != nil {
		return nil, err
	}

	// Delegate to the repository layer
	return uc.repos.PaymentTerm.GetPaymentTermItemPageData(ctx, req)
}

// Ensure the interface is implemented at compile time
var _ paymenttermpb.PaymentTermDomainServiceServer = (*GetPaymentTermItemPageDataUseCase)(nil)

// Required PaymentTermDomainServiceServer methods (delegated to avoid "method not implemented" errors)
func (uc *GetPaymentTermItemPageDataUseCase) CreatePaymentTerm(ctx context.Context, req *paymenttermpb.CreatePaymentTermRequest) (*paymenttermpb.CreatePaymentTermResponse, error) {
	return uc.repos.PaymentTerm.CreatePaymentTerm(ctx, req)
}

func (uc *GetPaymentTermItemPageDataUseCase) ReadPaymentTerm(ctx context.Context, req *paymenttermpb.ReadPaymentTermRequest) (*paymenttermpb.ReadPaymentTermResponse, error) {
	return uc.repos.PaymentTerm.ReadPaymentTerm(ctx, req)
}

func (uc *GetPaymentTermItemPageDataUseCase) UpdatePaymentTerm(ctx context.Context, req *paymenttermpb.UpdatePaymentTermRequest) (*paymenttermpb.UpdatePaymentTermResponse, error) {
	return uc.repos.PaymentTerm.UpdatePaymentTerm(ctx, req)
}

func (uc *GetPaymentTermItemPageDataUseCase) DeletePaymentTerm(ctx context.Context, req *paymenttermpb.DeletePaymentTermRequest) (*paymenttermpb.DeletePaymentTermResponse, error) {
	return uc.repos.PaymentTerm.DeletePaymentTerm(ctx, req)
}

func (uc *GetPaymentTermItemPageDataUseCase) ListPaymentTerms(ctx context.Context, req *paymenttermpb.ListPaymentTermsRequest) (*paymenttermpb.ListPaymentTermsResponse, error) {
	return uc.repos.PaymentTerm.ListPaymentTerms(ctx, req)
}

func (uc *GetPaymentTermItemPageDataUseCase) GetPaymentTermListPageData(ctx context.Context, req *paymenttermpb.GetPaymentTermListPageDataRequest) (*paymenttermpb.GetPaymentTermListPageDataResponse, error) {
	return uc.repos.PaymentTerm.GetPaymentTermListPageData(ctx, req)
}

func (uc *GetPaymentTermItemPageDataUseCase) GetPaymentTermItemPageData(ctx context.Context, req *paymenttermpb.GetPaymentTermItemPageDataRequest) (*paymenttermpb.GetPaymentTermItemPageDataResponse, error) {
	return uc.repos.PaymentTerm.GetPaymentTermItemPageData(ctx, req)
}
