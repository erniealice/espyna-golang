package withholding_certificate

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	withholdingcertificatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/withholding_certificate"
)

// CreateWithholdingCertificateRepositories groups repository dependencies.
type CreateWithholdingCertificateRepositories struct {
	WithholdingCertificate withholdingcertificatepb.WithholdingCertificateDomainServiceServer
}

// CreateWithholdingCertificateServices groups service dependencies.
type CreateWithholdingCertificateServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	IDGenerator ports.IDGenerator
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
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityWithholdingCertificate, entityid.ActionCreate); err != nil {
		return nil, err
	}

	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		var result *withholdingcertificatepb.CreateWithholdingCertificateResponse
		err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
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
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"withholding_certificate.validation.data_required", "Withholding Certificate data is required [DEFAULT]"))
	}

	// Enrich with ID and audit fields
	now := time.Now()
	if req.Data.Id == "" {
		req.Data.Id = uc.services.IDGenerator.GenerateID()
	}
	req.Data.DateCreated = &[]int64{now.UnixMilli()}[0]
	req.Data.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	req.Data.DateModified = &[]int64{now.UnixMilli()}[0]
	req.Data.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	req.Data.Active = true

	return uc.repositories.WithholdingCertificate.CreateWithholdingCertificate(ctx, req)
}
