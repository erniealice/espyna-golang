package withholding_certificate

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	withholdingcertificatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/withholding_certificate"
)

// UpdateWithholdingCertificateRepositories groups repository dependencies.
type UpdateWithholdingCertificateRepositories struct {
	WithholdingCertificate withholdingcertificatepb.WithholdingCertificateDomainServiceServer
}

// UpdateWithholdingCertificateServices groups service dependencies.
type UpdateWithholdingCertificateServices struct {
	Authorizer ports.Authorizer
	Translator ports.Translator
}

// UpdateWithholdingCertificateUseCase handles updating a withholding_certificate.
type UpdateWithholdingCertificateUseCase struct {
	repositories UpdateWithholdingCertificateRepositories
	services     UpdateWithholdingCertificateServices
}

// NewUpdateWithholdingCertificateUseCase creates a new UpdateWithholdingCertificateUseCase.
func NewUpdateWithholdingCertificateUseCase(repositories UpdateWithholdingCertificateRepositories, services UpdateWithholdingCertificateServices) *UpdateWithholdingCertificateUseCase {
	return &UpdateWithholdingCertificateUseCase{repositories: repositories, services: services}
}

// Execute performs the update withholding_certificate operation.
func (uc *UpdateWithholdingCertificateUseCase) Execute(ctx context.Context, req *withholdingcertificatepb.UpdateWithholdingCertificateRequest) (*withholdingcertificatepb.UpdateWithholdingCertificateResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityWithholdingCertificate, ports.ActionUpdate); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"withholding_certificate.validation.id_required", "Withholding Certificate ID is required [DEFAULT]"))
	}
	return uc.repositories.WithholdingCertificate.UpdateWithholdingCertificate(ctx, req)
}
