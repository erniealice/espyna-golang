package withholding_certificate

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	withholdingcertificatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/withholding_certificate"
)

// ReadWithholdingCertificateRepositories groups repository dependencies.
type ReadWithholdingCertificateRepositories struct {
	WithholdingCertificate withholdingcertificatepb.WithholdingCertificateDomainServiceServer
}

// ReadWithholdingCertificateServices groups service dependencies.
type ReadWithholdingCertificateServices struct {
	Authorizer ports.Authorizer
	Translator ports.Translator
}

// ReadWithholdingCertificateUseCase handles reading a withholding_certificate.
type ReadWithholdingCertificateUseCase struct {
	repositories ReadWithholdingCertificateRepositories
	services     ReadWithholdingCertificateServices
}

// NewReadWithholdingCertificateUseCase creates a new ReadWithholdingCertificateUseCase.
func NewReadWithholdingCertificateUseCase(repositories ReadWithholdingCertificateRepositories, services ReadWithholdingCertificateServices) *ReadWithholdingCertificateUseCase {
	return &ReadWithholdingCertificateUseCase{repositories: repositories, services: services}
}

// Execute performs the read withholding_certificate operation.
func (uc *ReadWithholdingCertificateUseCase) Execute(ctx context.Context, req *withholdingcertificatepb.ReadWithholdingCertificateRequest) (*withholdingcertificatepb.ReadWithholdingCertificateResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityWithholdingCertificate, ports.ActionRead); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"withholding_certificate.validation.id_required", "Withholding Certificate ID is required [DEFAULT]"))
	}
	return uc.repositories.WithholdingCertificate.ReadWithholdingCertificate(ctx, req)
}
