package disbursementmethod

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	disbursementmethodpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/disbursement_method"
)

// entityDisbursementMethod is the permission namespace + translation key root.
const entityDisbursementMethod = "disbursement_method"

// CreateDisbursementMethodRepositories groups all repository dependencies.
type CreateDisbursementMethodRepositories struct {
	DisbursementMethod disbursementmethodpb.DisbursementMethodDomainServiceServer
}

// CreateDisbursementMethodServices groups all business service dependencies.
type CreateDisbursementMethodServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	IDGenerator ports.IDGenerator
}

// CreateDisbursementMethodUseCase handles the business logic for creating disbursement methods.
type CreateDisbursementMethodUseCase struct {
	repositories CreateDisbursementMethodRepositories
	services     CreateDisbursementMethodServices
}

// NewCreateDisbursementMethodUseCase creates use case with grouped dependencies.
func NewCreateDisbursementMethodUseCase(
	repositories CreateDisbursementMethodRepositories,
	services CreateDisbursementMethodServices,
) *CreateDisbursementMethodUseCase {
	return &CreateDisbursementMethodUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the create disbursement method operation.
func (uc *CreateDisbursementMethodUseCase) Execute(ctx context.Context, req *disbursementmethodpb.CreateDisbursementMethodRequest) (*disbursementmethodpb.CreateDisbursementMethodResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityDisbursementMethod, ports.ActionCreate); err != nil {
		return nil, err
	}

	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		var result *disbursementmethodpb.CreateDisbursementMethodResponse
		err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
			res, err := uc.executeCore(txCtx, req)
			if err != nil {
				translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.Translator, "disbursement_method.errors.creation_failed", "Disbursement method creation failed [DEFAULT]")
				return fmt.Errorf("%s: %w", translatedError, err)
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

func (uc *CreateDisbursementMethodUseCase) executeCore(ctx context.Context, req *disbursementmethodpb.CreateDisbursementMethodRequest) (*disbursementmethodpb.CreateDisbursementMethodResponse, error) {
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	if err := uc.enrichData(req.Data); err != nil {
		return nil, err
	}

	if uc.repositories.DisbursementMethod == nil {
		return nil, errors.New("disbursement method repository is not available")
	}
	return uc.repositories.DisbursementMethod.CreateDisbursementMethod(ctx, req)
}

func (uc *CreateDisbursementMethodUseCase) validateInput(ctx context.Context, req *disbursementmethodpb.CreateDisbursementMethodRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "disbursement_method.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "disbursement_method.validation.data_required", "[ERR-DEFAULT] Disbursement method data is required"))
	}

	req.Data.Name = strings.TrimSpace(req.Data.Name)
	if req.Data.Name == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "disbursement_method.validation.name_required", "[ERR-DEFAULT] Name is required"))
	}

	// D-1.5 Q3 STRICT: tax_effect_kind is REQUIRED on the template.
	if req.Data.GetTaxEffectKind() == disbursementmethodpb.DisbursementMethodTaxEffectKind_DISBURSEMENT_METHOD_TAX_EFFECT_KIND_UNSPECIFIED {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "disbursement_method.validation.tax_effect_kind_required", "[ERR-DEFAULT] Tax effect kind is required"))
	}

	return nil
}

func (uc *CreateDisbursementMethodUseCase) enrichData(dm *disbursementmethodpb.DisbursementMethod) error {
	now := time.Now()
	if dm.Id == "" {
		dm.Id = uc.services.IDGenerator.GenerateID()
	}
	dm.DateCreated = &[]int64{now.UnixMilli()}[0]
	dm.DateModified = &[]int64{now.UnixMilli()}[0]
	dm.Active = true

	// New templates begin life as DRAFT (D-1.8 lifecycle).
	if dm.GetLifecycle() == disbursementmethodpb.DisbursementMethodLifecycle_DISBURSEMENT_METHOD_LIFECYCLE_UNSPECIFIED {
		dm.Lifecycle = disbursementmethodpb.DisbursementMethodLifecycle_DISBURSEMENT_METHOD_LIFECYCLE_DRAFT
	}
	if dm.GetVersionStatus() == disbursementmethodpb.DisbursementMethodVersionStatus_DISBURSEMENT_METHOD_VERSION_STATUS_UNSPECIFIED {
		dm.VersionStatus = disbursementmethodpb.DisbursementMethodVersionStatus_DISBURSEMENT_METHOD_VERSION_STATUS_DRAFT
	}
	if dm.Revision == 0 {
		dm.Revision = 1
	}
	return nil
}
