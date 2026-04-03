package payment_term

import (
	"context"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	paymenttermpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/payment_term"
)

// GetPaymentTermListPageDataRepositories groups repository dependencies for GetPaymentTermListPageData use case
type GetPaymentTermListPageDataRepositories struct {
	PaymentTerm paymenttermpb.PaymentTermDomainServiceServer
}

// GetPaymentTermListPageDataServices groups service dependencies for GetPaymentTermListPageData use case
type GetPaymentTermListPageDataServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// GetPaymentTermListPageDataUseCase handles getting paginated payment term list data with search, filtering, and sorting
type GetPaymentTermListPageDataUseCase struct {
	paymenttermpb.UnimplementedPaymentTermDomainServiceServer
	repos    GetPaymentTermListPageDataRepositories
	services GetPaymentTermListPageDataServices
}

// NewGetPaymentTermListPageDataUseCase creates a new GetPaymentTermListPageData use case
func NewGetPaymentTermListPageDataUseCase(
	repos GetPaymentTermListPageDataRepositories,
	services GetPaymentTermListPageDataServices,
) *GetPaymentTermListPageDataUseCase {
	return &GetPaymentTermListPageDataUseCase{
		repos:    repos,
		services: services,
	}
}

// Execute runs the GetPaymentTermListPageData use case
func (uc *GetPaymentTermListPageDataUseCase) Execute(
	ctx context.Context,
	req *paymenttermpb.GetPaymentTermListPageDataRequest,
) (*paymenttermpb.GetPaymentTermListPageDataResponse, error) {
	// Authorization check
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		"payment_term", ports.ActionList); err != nil {
		return nil, err
	}

	// Delegate to the repository layer
	return uc.repos.PaymentTerm.GetPaymentTermListPageData(ctx, req)
}

// Ensure the interface is implemented at compile time
var _ paymenttermpb.PaymentTermDomainServiceServer = (*GetPaymentTermListPageDataUseCase)(nil)

// Required PaymentTermDomainServiceServer methods (delegated to avoid "method not implemented" errors)
func (uc *GetPaymentTermListPageDataUseCase) CreatePaymentTerm(ctx context.Context, req *paymenttermpb.CreatePaymentTermRequest) (*paymenttermpb.CreatePaymentTermResponse, error) {
	return uc.repos.PaymentTerm.CreatePaymentTerm(ctx, req)
}

func (uc *GetPaymentTermListPageDataUseCase) ReadPaymentTerm(ctx context.Context, req *paymenttermpb.ReadPaymentTermRequest) (*paymenttermpb.ReadPaymentTermResponse, error) {
	return uc.repos.PaymentTerm.ReadPaymentTerm(ctx, req)
}

func (uc *GetPaymentTermListPageDataUseCase) UpdatePaymentTerm(ctx context.Context, req *paymenttermpb.UpdatePaymentTermRequest) (*paymenttermpb.UpdatePaymentTermResponse, error) {
	return uc.repos.PaymentTerm.UpdatePaymentTerm(ctx, req)
}

func (uc *GetPaymentTermListPageDataUseCase) DeletePaymentTerm(ctx context.Context, req *paymenttermpb.DeletePaymentTermRequest) (*paymenttermpb.DeletePaymentTermResponse, error) {
	return uc.repos.PaymentTerm.DeletePaymentTerm(ctx, req)
}

func (uc *GetPaymentTermListPageDataUseCase) ListPaymentTerms(ctx context.Context, req *paymenttermpb.ListPaymentTermsRequest) (*paymenttermpb.ListPaymentTermsResponse, error) {
	return uc.repos.PaymentTerm.ListPaymentTerms(ctx, req)
}

func (uc *GetPaymentTermListPageDataUseCase) GetPaymentTermItemPageData(ctx context.Context, req *paymenttermpb.GetPaymentTermItemPageDataRequest) (*paymenttermpb.GetPaymentTermItemPageDataResponse, error) {
	return uc.repos.PaymentTerm.GetPaymentTermItemPageData(ctx, req)
}

func (uc *GetPaymentTermListPageDataUseCase) GetPaymentTermListPageData(ctx context.Context, req *paymenttermpb.GetPaymentTermListPageDataRequest) (*paymenttermpb.GetPaymentTermListPageDataResponse, error) {
	return uc.repos.PaymentTerm.GetPaymentTermListPageData(ctx, req)
}
