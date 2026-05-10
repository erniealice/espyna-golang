package withholding_certificate

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	withholdingcertificatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/withholding_certificate"
)

// ListWithholdingCertificatesRepositories groups repository dependencies.
type ListWithholdingCertificatesRepositories struct {
	WithholdingCertificate withholdingcertificatepb.WithholdingCertificateDomainServiceServer
}

// ListWithholdingCertificatesServices groups service dependencies.
type ListWithholdingCertificatesServices struct {
	AuthorizationService ports.AuthorizationService
	TranslationService   ports.TranslationService
}

// ListWithholdingCertificatesUseCase handles listing withholding certificates.
type ListWithholdingCertificatesUseCase struct {
	repositories ListWithholdingCertificatesRepositories
	services     ListWithholdingCertificatesServices
}

// NewListWithholdingCertificatesUseCase creates a new ListWithholdingCertificatesUseCase.
func NewListWithholdingCertificatesUseCase(repositories ListWithholdingCertificatesRepositories, services ListWithholdingCertificatesServices) *ListWithholdingCertificatesUseCase {
	return &ListWithholdingCertificatesUseCase{repositories: repositories, services: services}
}

// Execute performs the list withholding_certificates operation.
func (uc *ListWithholdingCertificatesUseCase) Execute(ctx context.Context, req *withholdingcertificatepb.ListWithholdingCertificatesRequest) (*withholdingcertificatepb.ListWithholdingCertificatesResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityWithholdingCertificate, ports.ActionList); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"withholding_certificate.validation.request_required", "Request is required [DEFAULT]"))
	}
	return uc.repositories.WithholdingCertificate.ListWithholdingCertificates(ctx, req)
}
