package suppliercontractline

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	suppliercontractlinepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/supplier_contract_line"
)

const entitySupplierContractLine = "supplier_contract_line"

// CreateSupplierContractLineRepositories groups repository dependencies.
type CreateSupplierContractLineRepositories struct {
	SupplierContractLine suppliercontractlinepb.SupplierContractLineDomainServiceServer
}

// CreateSupplierContractLineServices groups service dependencies.
type CreateSupplierContractLineServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	IDGenerator ports.IDGenerator
}

// CreateSupplierContractLineUseCase handles creating a supplier contract line.
type CreateSupplierContractLineUseCase struct {
	repositories CreateSupplierContractLineRepositories
	services     CreateSupplierContractLineServices
}

// NewCreateSupplierContractLineUseCase creates a use case with grouped dependencies.
func NewCreateSupplierContractLineUseCase(
	repositories CreateSupplierContractLineRepositories,
	services CreateSupplierContractLineServices,
) *CreateSupplierContractLineUseCase {
	return &CreateSupplierContractLineUseCase{repositories: repositories, services: services}
}

// Execute performs the create supplier contract line operation.
func (uc *CreateSupplierContractLineUseCase) Execute(ctx context.Context, req *suppliercontractlinepb.CreateSupplierContractLineRequest) (*suppliercontractlinepb.CreateSupplierContractLineResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entitySupplierContractLine, entityid.ActionCreate); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"supplier_contract_line.validation.data_required", "Supplier contract line data is required [DEFAULT]"))
	}
	if req.Data.SupplierContractId == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"supplier_contract_line.validation.contract_id_required", "Supplier contract ID is required [DEFAULT]"))
	}

	now := time.Now()
	if req.Data.Id == "" {
		req.Data.Id = uc.services.IDGenerator.GenerateID()
	}
	req.Data.DateCreated = &[]int64{now.UnixMilli()}[0]
	req.Data.DateModified = &[]int64{now.UnixMilli()}[0]
	req.Data.Active = true

	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		var result *suppliercontractlinepb.CreateSupplierContractLineResponse
		err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
			res, err := uc.repositories.SupplierContractLine.CreateSupplierContractLine(txCtx, req)
			if err != nil {
				return fmt.Errorf("supplier contract line creation failed: %w", err)
			}
			result = res
			return nil
		})
		if err != nil {
			return nil, err
		}
		return result, nil
	}
	return uc.repositories.SupplierContractLine.CreateSupplierContractLine(ctx, req)
}
