package procurementrequest

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	procurementrequestpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/procurement_request"
)

const entityProcurementRequest = "procurement_request"

// CreateProcurementRequestRepositories groups repository dependencies.
type CreateProcurementRequestRepositories struct {
	ProcurementRequest procurementrequestpb.ProcurementRequestDomainServiceServer
}

// CreateProcurementRequestServices groups service dependencies.
type CreateProcurementRequestServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// CreateProcurementRequestUseCase handles creating a procurement request.
type CreateProcurementRequestUseCase struct {
	repositories CreateProcurementRequestRepositories
	services     CreateProcurementRequestServices
}

// NewCreateProcurementRequestUseCase creates a use case with grouped dependencies.
func NewCreateProcurementRequestUseCase(
	repositories CreateProcurementRequestRepositories,
	services CreateProcurementRequestServices,
) *CreateProcurementRequestUseCase {
	return &CreateProcurementRequestUseCase{repositories: repositories, services: services}
}

// Execute performs the create procurement request operation.
func (uc *CreateProcurementRequestUseCase) Execute(ctx context.Context, req *procurementrequestpb.CreateProcurementRequestRequest) (*procurementrequestpb.CreateProcurementRequestResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityProcurementRequest, ports.ActionCreate); err != nil {
		return nil, err
	}

	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		var result *procurementrequestpb.CreateProcurementRequestResponse
		err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
			res, err := uc.executeCore(txCtx, req)
			if err != nil {
				return fmt.Errorf("procurement request creation failed: %w", err)
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

func (uc *CreateProcurementRequestUseCase) executeCore(ctx context.Context, req *procurementrequestpb.CreateProcurementRequestRequest) (*procurementrequestpb.CreateProcurementRequestResponse, error) {
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"procurement_request.validation.data_required", "Procurement request data is required [DEFAULT]"))
	}

	now := time.Now()
	if req.Data.Id == "" {
		req.Data.Id = uc.services.IDService.GenerateID()
	}
	req.Data.DateCreated = &[]int64{now.UnixMilli()}[0]
	req.Data.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	req.Data.DateModified = &[]int64{now.UnixMilli()}[0]
	req.Data.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	req.Data.Active = true

	// Default status: DRAFT per entity-status-conventions (create use case must default status).
	if req.Data.Status == procurementrequestpb.ProcurementRequestStatus_PROCUREMENT_REQUEST_STATUS_UNSPECIFIED {
		req.Data.Status = procurementrequestpb.ProcurementRequestStatus_PROCUREMENT_REQUEST_STATUS_DRAFT
	}

	return uc.repositories.ProcurementRequest.CreateProcurementRequest(ctx, req)
}
