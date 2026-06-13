package procurementrequestline

import (
	"context"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	procurementrequestlinepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/procurement_request_line"
)

// ListProcurementRequestLinesRepositories groups repository dependencies.
type ListProcurementRequestLinesRepositories struct {
	ProcurementRequestLine procurementrequestlinepb.ProcurementRequestLineDomainServiceServer
}

// ListProcurementRequestLinesServices groups service dependencies.
type ListProcurementRequestLinesServices struct {
	Authorizer ports.Authorizer
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// ListProcurementRequestLinesUseCase handles listing procurement request lines.
type ListProcurementRequestLinesUseCase struct {
	repositories ListProcurementRequestLinesRepositories
	services     ListProcurementRequestLinesServices
}

// NewListProcurementRequestLinesUseCase creates a use case with grouped dependencies.
func NewListProcurementRequestLinesUseCase(
	repositories ListProcurementRequestLinesRepositories,
	services ListProcurementRequestLinesServices,
) *ListProcurementRequestLinesUseCase {
	return &ListProcurementRequestLinesUseCase{repositories: repositories, services: services}
}

// Execute performs the list procurement request lines operation.
func (uc *ListProcurementRequestLinesUseCase) Execute(ctx context.Context, req *procurementrequestlinepb.ListProcurementRequestLinesRequest) (*procurementrequestlinepb.ListProcurementRequestLinesResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityProcurementRequestLine,
		Action: entityid.ActionList,
	}); err != nil {
		return nil, err
	}
	return uc.repositories.ProcurementRequestLine.ListProcurementRequestLines(ctx, req)
}
