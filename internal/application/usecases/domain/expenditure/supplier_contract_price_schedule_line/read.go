package suppliercontractpriceschedulesline

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	scpslpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/supplier_contract_price_schedule_line"
)

// ReadSupplierContractPriceScheduleLineRepositories groups repository dependencies.
type ReadSupplierContractPriceScheduleLineRepositories struct {
	SupplierContractPriceScheduleLine scpslpb.SupplierContractPriceScheduleLineDomainServiceServer
}

// ReadSupplierContractPriceScheduleLineServices groups service dependencies.
type ReadSupplierContractPriceScheduleLineServices struct {
	Authorizer ports.Authorizer
	Translator ports.Translator
}

// ReadSupplierContractPriceScheduleLineUseCase handles reading a schedule-line.
type ReadSupplierContractPriceScheduleLineUseCase struct {
	repositories ReadSupplierContractPriceScheduleLineRepositories
	services     ReadSupplierContractPriceScheduleLineServices
}

// NewReadSupplierContractPriceScheduleLineUseCase creates a use case with grouped dependencies.
func NewReadSupplierContractPriceScheduleLineUseCase(
	repositories ReadSupplierContractPriceScheduleLineRepositories,
	services ReadSupplierContractPriceScheduleLineServices,
) *ReadSupplierContractPriceScheduleLineUseCase {
	return &ReadSupplierContractPriceScheduleLineUseCase{repositories: repositories, services: services}
}

// Execute performs the read schedule-line operation.
func (uc *ReadSupplierContractPriceScheduleLineUseCase) Execute(ctx context.Context, req *scpslpb.ReadSupplierContractPriceScheduleLineRequest) (*scpslpb.ReadSupplierContractPriceScheduleLineResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entitySupplierContractPriceScheduleLine, entityid.ActionRead); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"supplier_contract_price_schedule_line.validation.id_required", "Schedule line ID is required [DEFAULT]"))
	}
	return uc.repositories.SupplierContractPriceScheduleLine.ReadSupplierContractPriceScheduleLine(ctx, req)
}
