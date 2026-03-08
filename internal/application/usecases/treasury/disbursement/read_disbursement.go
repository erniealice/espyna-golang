package disbursement

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	disbursementpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/disbursement"
)

// ReadDisbursementRepositories groups all repository dependencies
type ReadDisbursementRepositories struct {
	Disbursement disbursementpb.DisbursementDomainServiceServer
}

// ReadDisbursementServices groups all business service dependencies
type ReadDisbursementServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
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
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityDisbursement, ports.ActionRead); err != nil {
		return nil, err
	}

	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "disbursement.validation.id_required", "Disbursement ID is required [DEFAULT]"))
	}

	return uc.repositories.Disbursement.ReadDisbursement(ctx, req)
}
