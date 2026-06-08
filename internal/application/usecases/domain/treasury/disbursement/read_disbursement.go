package disbursement

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	disbursementpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/disbursement"
)

// ReadDisbursementRepositories groups all repository dependencies
type ReadDisbursementRepositories struct {
	Disbursement disbursementpb.DisbursementDomainServiceServer
}

// ReadDisbursementServices groups all business service dependencies
type ReadDisbursementServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// ReadDisbursementUseCase handles the business logic for reading a disbursement
type ReadDisbursementUseCase struct {
	repositories ReadDisbursementRepositories
	services     ReadDisbursementServices
}

// NewReadDisbursementUseCase creates use case with grouped dependencies
func NewReadDisbursementUseCase(
	repositories ReadDisbursementRepositories,
	services ReadDisbursementServices,
) *ReadDisbursementUseCase {
	return &ReadDisbursementUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the read disbursement operation
func (uc *ReadDisbursementUseCase) Execute(ctx context.Context, req *disbursementpb.ReadDisbursementRequest) (*disbursementpb.ReadDisbursementResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityDisbursement, entityid.ActionRead); err != nil {
		return nil, err
	}

	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "disbursement.validation.id_required", "Disbursement ID is required [DEFAULT]"))
	}

	return uc.repositories.Disbursement.ReadDisbursement(ctx, req)
}
