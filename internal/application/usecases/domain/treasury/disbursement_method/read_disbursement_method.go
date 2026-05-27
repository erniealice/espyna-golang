package disbursementmethod

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	disbursementmethodpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/disbursement_method"
)

// ReadDisbursementMethodRepositories groups all repository dependencies.
type ReadDisbursementMethodRepositories struct {
	DisbursementMethod disbursementmethodpb.DisbursementMethodDomainServiceServer
}

// ReadDisbursementMethodServices groups all business service dependencies.
type ReadDisbursementMethodServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// ReadDisbursementMethodUseCase handles the business logic for reading a disbursement method.
type ReadDisbursementMethodUseCase struct {
	repositories ReadDisbursementMethodRepositories
	services     ReadDisbursementMethodServices
}

// NewReadDisbursementMethodUseCase creates use case with grouped dependencies.
func NewReadDisbursementMethodUseCase(
	repositories ReadDisbursementMethodRepositories,
	services ReadDisbursementMethodServices,
) *ReadDisbursementMethodUseCase {
	return &ReadDisbursementMethodUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the read disbursement method operation.
func (uc *ReadDisbursementMethodUseCase) Execute(ctx context.Context, req *disbursementmethodpb.ReadDisbursementMethodRequest) (*disbursementmethodpb.ReadDisbursementMethodResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityDisbursementMethod, ports.ActionRead); err != nil {
		return nil, err
	}

	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "disbursement_method.validation.id_required", "Disbursement method ID is required [DEFAULT]"))
	}

	if uc.repositories.DisbursementMethod == nil {
		return nil, errors.New("disbursement method repository is not available")
	}
	return uc.repositories.DisbursementMethod.ReadDisbursementMethod(ctx, req)
}
