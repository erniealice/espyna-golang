package disbursement

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	disbursementpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/disbursement"
)

const entityDisbursement = "disbursement"

// CreateDisbursementRepositories groups all repository dependencies
type CreateDisbursementRepositories struct {
	Disbursement disbursementpb.DisbursementDomainServiceServer
}

// CreateDisbursementServices groups all business service dependencies
type CreateDisbursementServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	IDGenerator ports.IDGenerator
}

// CreateDisbursementUseCase handles the business logic for creating disbursements
type CreateDisbursementUseCase struct {
	repositories CreateDisbursementRepositories
	services     CreateDisbursementServices
}

// NewCreateDisbursementUseCase creates use case with grouped dependencies
func NewCreateDisbursementUseCase(
	repositories CreateDisbursementRepositories,
	services CreateDisbursementServices,
) *CreateDisbursementUseCase {
	return &CreateDisbursementUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the create disbursement operation.
//
// 20260518-hexagonal-strict-adherence Phase 1.C-iv — BURN_DOWN guard relocated
// from the postgres adapter (F4 layer-violation fix).
func (uc *CreateDisbursementUseCase) Execute(ctx context.Context, req *disbursementpb.CreateDisbursementRequest) (*disbursementpb.CreateDisbursementResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityDisbursement, ports.ActionCreate); err != nil {
		return nil, err
	}

	// Plan B Phase 0 hard rule: ADVANCE_KIND_BURN_DOWN is reserved for v2.
	if req != nil && req.Data != nil {
		if err := validateAdvanceKindNotBurnDown(req.Data.GetAdvanceKind()); err != nil {
			return nil, err
		}
	}

	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		var result *disbursementpb.CreateDisbursementResponse
		err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
			res, err := uc.executeCore(txCtx, req)
			if err != nil {
				return fmt.Errorf("disbursement creation failed: %w", err)
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

func (uc *CreateDisbursementUseCase) executeCore(ctx context.Context, req *disbursementpb.CreateDisbursementRequest) (*disbursementpb.CreateDisbursementResponse, error) {
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "disbursement.validation.data_required", "Disbursement data is required [DEFAULT]"))
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

	return uc.repositories.Disbursement.CreateDisbursement(ctx, req)
}
