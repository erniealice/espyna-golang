package procurementrequestline

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	procurementrequestlinepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/procurement_request_line"
)

const entityProcurementRequestLine = "procurement_request_line"

// CreateProcurementRequestLineRepositories groups repository dependencies.
type CreateProcurementRequestLineRepositories struct {
	ProcurementRequestLine procurementrequestlinepb.ProcurementRequestLineDomainServiceServer
}

// CreateProcurementRequestLineServices groups service dependencies.
type CreateProcurementRequestLineServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
}

// CreateProcurementRequestLineUseCase handles creating a procurement request line.
type CreateProcurementRequestLineUseCase struct {
	repositories CreateProcurementRequestLineRepositories
	services     CreateProcurementRequestLineServices
}

// NewCreateProcurementRequestLineUseCase creates a use case with grouped dependencies.
func NewCreateProcurementRequestLineUseCase(
	repositories CreateProcurementRequestLineRepositories,
	services CreateProcurementRequestLineServices,
) *CreateProcurementRequestLineUseCase {
	return &CreateProcurementRequestLineUseCase{repositories: repositories, services: services}
}

// Execute performs the create procurement request line operation.
func (uc *CreateProcurementRequestLineUseCase) Execute(ctx context.Context, req *procurementrequestlinepb.CreateProcurementRequestLineRequest) (*procurementrequestlinepb.CreateProcurementRequestLineResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityProcurementRequestLine,
		Action: entityid.ActionCreate,
	}); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"procurement_request_line.validation.data_required", "Procurement request line data is required [DEFAULT]"))
	}
	if req.Data.ProcurementRequestId == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"procurement_request_line.validation.request_id_required", "Procurement request ID is required [DEFAULT]"))
	}

	now := time.Now()
	if req.Data.Id == "" {
		req.Data.Id = uc.services.IDGenerator.GenerateID()
	}
	req.Data.DateCreated = &[]int64{now.UnixMilli()}[0]
	req.Data.DateModified = &[]int64{now.UnixMilli()}[0]
	req.Data.Active = true

	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		var result *procurementrequestlinepb.CreateProcurementRequestLineResponse
		err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
			res, err := uc.repositories.ProcurementRequestLine.CreateProcurementRequestLine(txCtx, req)
			if err != nil {
				return fmt.Errorf("procurement request line creation failed: %w", err)
			}
			result = res
			return nil
		})
		if err != nil {
			return nil, err
		}
		return result, nil
	}
	return uc.repositories.ProcurementRequestLine.CreateProcurementRequestLine(ctx, req)
}
