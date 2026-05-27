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

// UpdateDisbursementMethodRepositories groups all repository dependencies.
type UpdateDisbursementMethodRepositories struct {
	DisbursementMethod disbursementmethodpb.DisbursementMethodDomainServiceServer
}

// UpdateDisbursementMethodServices groups all business service dependencies.
type UpdateDisbursementMethodServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// UpdateDisbursementMethodUseCase handles the business logic for updating disbursement methods.
type UpdateDisbursementMethodUseCase struct {
	repositories UpdateDisbursementMethodRepositories
	services     UpdateDisbursementMethodServices
}

// NewUpdateDisbursementMethodUseCase creates use case with grouped dependencies.
func NewUpdateDisbursementMethodUseCase(
	repositories UpdateDisbursementMethodRepositories,
	services UpdateDisbursementMethodServices,
) *UpdateDisbursementMethodUseCase {
	return &UpdateDisbursementMethodUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the update disbursement method operation. Transaction-aware
// so lifecycle transitions can route their terminal write through this wrapper.
func (uc *UpdateDisbursementMethodUseCase) Execute(ctx context.Context, req *disbursementmethodpb.UpdateDisbursementMethodRequest) (*disbursementmethodpb.UpdateDisbursementMethodResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityDisbursementMethod, ports.ActionUpdate); err != nil {
		return nil, err
	}

	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		if uc.services.Transactor.IsTransactionActive(ctx) {
			return uc.executeCore(ctx, req)
		}
		var result *disbursementmethodpb.UpdateDisbursementMethodResponse
		err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
			res, err := uc.executeCore(txCtx, req)
			if err != nil {
				return fmt.Errorf("disbursement method update failed: %w", err)
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

func (uc *UpdateDisbursementMethodUseCase) executeCore(ctx context.Context, req *disbursementmethodpb.UpdateDisbursementMethodRequest) (*disbursementmethodpb.UpdateDisbursementMethodResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "disbursement_method.validation.id_required", "Disbursement method ID is required [DEFAULT]"))
	}

	req.Data.Name = strings.TrimSpace(req.Data.Name)
	if req.Data.Name == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "disbursement_method.validation.name_required", "[ERR-DEFAULT] Name is required"))
	}

	now := time.Now()
	req.Data.DateModified = &[]int64{now.UnixMilli()}[0]
	req.Data.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	if uc.repositories.DisbursementMethod == nil {
		return nil, errors.New("disbursement method repository is not available")
	}
	return uc.repositories.DisbursementMethod.UpdateDisbursementMethod(ctx, req)
}
