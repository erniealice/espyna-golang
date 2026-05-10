package withholding_certificate

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	withholdingcertificatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/withholding_certificate"
)

// CreateWithholdingCertificateRepositories groups repository dependencies.
type CreateWithholdingCertificateRepositories struct {
	WithholdingCertificate withholdingcertificatepb.WithholdingCertificateDomainServiceServer
}

// CreateWithholdingCertificateServices groups service dependencies.
type CreateWithholdingCertificateServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// CreateWithholdingCertificateUseCase handles creating a withholding_certificate.
type CreateWithholdingCertificateUseCase struct {
	repositories CreateWithholdingCertificateRepositories
	services     CreateWithholdingCertificateServices
}

// NewCreateWithholdingCertificateUseCase creates a new CreateWithholdingCertificateUseCase.
func NewCreateWithholdingCertificateUseCase(repositories CreateWithholdingCertificateRepositories, services CreateWithholdingCertificateServices) *CreateWithholdingCertificateUseCase {
	return &CreateWithholdingCertificateUseCase{repositories: repositories, services: services}
}

// Execute performs the create withholding_certificate operation.
func (uc *CreateWithholdingCertificateUseCase) Execute(ctx context.Context, req *withholdingcertificatepb.CreateWithholdingCertificateRequest) (*withholdingcertificatepb.CreateWithholdingCertificateResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityWithholdingCertificate, ports.ActionCreate); err != nil {
		return nil, err
	}

	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		var result *withholdingcertificatepb.CreateWithholdingCertificateResponse
		err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
			res, err := uc.executeCore(txCtx, req)
			if err != nil {
				return fmt.Errorf("withholding_certificate creation failed: %w", err)
			}
			result = res
			return nil
		})
		if err != nil {
			return nil, err
		}
		return result, nil
	}

	return uc.executeCore(ctx, req)
}

func (uc *CreateWithholdingCertificateUseCase) executeCore(ctx context.Context, req *withholdingcertificatepb.CreateWithholdingCertificateRequest) (*withholdingcertificatepb.CreateWithholdingCertificateResponse, error) {
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"withholding_certificate.validation.data_required", "Withholding Certificate data is required [DEFAULT]"))
	}

	// Enrich with ID and audit fields
	now := time.Now()
	if req.Data.Id == "" {
		req.Data.Id = uc.services.IDService.GenerateID()
	}
	req.Data.DateCreated = &[]int64{now.UnixMilli()}[0]
	req.Data.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	req.Data.DateModified = &[]int64{now.UnixMilli()}[0]
	req.Data.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	req.Data.Active = true

	return uc.repositories.WithholdingCertificate.CreateWithholdingCertificate(ctx, req)
}
