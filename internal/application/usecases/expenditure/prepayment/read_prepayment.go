package prepayment

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	prepaymentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/prepayment"
)

// ReadPrepaymentRepositories groups all repository dependencies
type ReadPrepaymentRepositories struct {
	Prepayment prepaymentpb.PrepaymentDomainServiceServer
}

// ReadPrepaymentServices groups all business service dependencies
type ReadPrepaymentServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ReadPrepaymentUseCase handles the business logic for reading a prepayment
type ReadPrepaymentUseCase struct {
	repositories ReadPrepaymentRepositories
	services     ReadPrepaymentServices
}

// NewReadPrepaymentUseCase creates use case with grouped dependencies
func NewReadPrepaymentUseCase(
	repositories ReadPrepaymentRepositories,
	services ReadPrepaymentServices,
) *ReadPrepaymentUseCase {
	return &ReadPrepaymentUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the read prepayment operation
func (uc *ReadPrepaymentUseCase) Execute(ctx context.Context, req *prepaymentpb.ReadPrepaymentRequest) (*prepaymentpb.ReadPrepaymentResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityPrepayment, ports.ActionRead); err != nil {
		return nil, err
	}

	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	if uc.repositories.Prepayment == nil {
		return nil, errors.New("prepayment repository is not available")
	}
	return uc.repositories.Prepayment.ReadPrepayment(ctx, req)
}

func (uc *ReadPrepaymentUseCase) validateInput(ctx context.Context, req *prepaymentpb.ReadPrepaymentRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "prepayment.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "prepayment.validation.data_required", "[ERR-DEFAULT] Data is required"))
	}
	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "prepayment.validation.id_required", "[ERR-DEFAULT] ID is required"))
	}
	return nil
}
