package withholding_certificate

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	withholdingcertificatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/withholding_certificate"
)

// DeleteWithholdingCertificateRepositories groups repository dependencies.
type DeleteWithholdingCertificateRepositories struct {
	WithholdingCertificate withholdingcertificatepb.WithholdingCertificateDomainServiceServer
}

// DeleteWithholdingCertificateServices groups service dependencies.
type DeleteWithholdingCertificateServices struct {
	AuthorizationService ports.AuthorizationService
	TranslationService   ports.TranslationService
}

// DeleteWithholdingCertificateUseCase handles deleting a withholding_certificate.
type DeleteWithholdingCertificateUseCase struct {
	repositories DeleteWithholdingCertificateRepositories
	services     DeleteWithholdingCertificateServices
}

// NewDeleteWithholdingCertificateUseCase creates a new DeleteWithholdingCertificateUseCase.
func NewDeleteWithholdingCertificateUseCase(repositories DeleteWithholdingCertificateRepositories, services DeleteWithholdingCertificateServices) *DeleteWithholdingCertificateUseCase {
	return &DeleteWithholdingCertificateUseCase{repositories: repositories, services: services}
}

// Execute performs the delete withholding_certificate operation.
func (uc *DeleteWithholdingCertificateUseCase) Execute(ctx context.Context, req *withholdingcertificatepb.DeleteWithholdingCertificateRequest) (*withholdingcertificatepb.DeleteWithholdingCertificateResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityWithholdingCertificate, ports.ActionDelete); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"withholding_certificate.validation.id_required", "Withholding Certificate ID is required [DEFAULT]"))
	}
	return uc.repositories.WithholdingCertificate.DeleteWithholdingCertificate(ctx, req)
}
