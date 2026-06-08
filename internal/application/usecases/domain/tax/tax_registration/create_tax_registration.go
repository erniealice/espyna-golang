package tax_registration

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	taxregistrationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/tax/tax_registration"
	taxregistrationkindpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/tax/tax_registration_kind"
)

// CreateTaxRegistrationRepositories groups repository dependencies.
type CreateTaxRegistrationRepositories struct {
	TaxRegistration     taxregistrationpb.TaxRegistrationDomainServiceServer
	TaxRegistrationKind taxregistrationkindpb.TaxRegistrationKindDomainServiceServer
}

// CreateTaxRegistrationServices groups service dependencies.
type CreateTaxRegistrationServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	IDGenerator ports.IDGenerator
}

// CreateTaxRegistrationUseCase handles creating a tax_registration.
type CreateTaxRegistrationUseCase struct {
	repositories CreateTaxRegistrationRepositories
	services     CreateTaxRegistrationServices
}

// NewCreateTaxRegistrationUseCase creates a new CreateTaxRegistrationUseCase.
func NewCreateTaxRegistrationUseCase(repositories CreateTaxRegistrationRepositories, services CreateTaxRegistrationServices) *CreateTaxRegistrationUseCase {
	return &CreateTaxRegistrationUseCase{repositories: repositories, services: services}
}

// Execute performs the create tax_registration operation.
// It copies compute_path and party_role from the registration kind as a snapshot denorm.
func (uc *CreateTaxRegistrationUseCase) Execute(ctx context.Context, req *taxregistrationpb.CreateTaxRegistrationRequest) (*taxregistrationpb.CreateTaxRegistrationResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityTaxRegistration, entityid.ActionCreate); err != nil {
		return nil, err
	}

	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		var result *taxregistrationpb.CreateTaxRegistrationResponse
		err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
			res, err := uc.executeCore(txCtx, req)
			if err != nil {
				return fmt.Errorf("tax_registration creation failed: %w", err)
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

func (uc *CreateTaxRegistrationUseCase) executeCore(ctx context.Context, req *taxregistrationpb.CreateTaxRegistrationRequest) (*taxregistrationpb.CreateTaxRegistrationResponse, error) {
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"tax_registration.validation.data_required", "Tax Registration data is required [DEFAULT]"))
	}
	if req.Data.TaxRegistrationKindId == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"tax_registration.validation.kind_id_required", "Tax Registration Kind is required [DEFAULT]"))
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

	// Denorm: copy compute_path + party_role from the registration kind as a snapshot.
	// The snapshot enum ordinal values match those of the kind's enum by design.
	kindResp, err := uc.repositories.TaxRegistrationKind.ReadTaxRegistrationKind(ctx,
		&taxregistrationkindpb.ReadTaxRegistrationKindRequest{
			Data: &taxregistrationkindpb.TaxRegistrationKind{Id: req.Data.TaxRegistrationKindId},
		})
	if err != nil {
		return nil, fmt.Errorf("failed to read tax_registration_kind for snapshot: %w", err)
	}
	if len(kindResp.Data) > 0 {
		kind := kindResp.Data[0]
		req.Data.ComputePathSnapshot = taxregistrationpb.TaxRegistrationComputePathSnapshot(kind.GetComputePath().Number())
		req.Data.PartyRoleSnapshot = taxregistrationpb.TaxRegistrationPartyRoleSnapshot(kind.GetPartyRole().Number())
	}

	return uc.repositories.TaxRegistration.CreateTaxRegistration(ctx, req)
}
