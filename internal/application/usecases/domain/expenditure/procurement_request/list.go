package procurementrequest

import (
	"context"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	procurementrequestpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/procurement_request"
)

// ListProcurementRequestsRepositories groups repository dependencies.
type ListProcurementRequestsRepositories struct {
	ProcurementRequest procurementrequestpb.ProcurementRequestDomainServiceServer
}

// ListProcurementRequestsServices groups service dependencies.
type ListProcurementRequestsServices struct {
	Authorizer ports.Authorizer
	Translator ports.Translator
}

// ListProcurementRequestsUseCase handles listing procurement requests.
type ListProcurementRequestsUseCase struct {
	repositories ListProcurementRequestsRepositories
	services     ListProcurementRequestsServices
}

// NewListProcurementRequestsUseCase creates a use case with grouped dependencies.
func NewListProcurementRequestsUseCase(
	repositories ListProcurementRequestsRepositories,
	services ListProcurementRequestsServices,
) *ListProcurementRequestsUseCase {
	return &ListProcurementRequestsUseCase{repositories: repositories, services: services}
}

// Execute performs the list procurement requests operation.
func (uc *ListProcurementRequestsUseCase) Execute(ctx context.Context, req *procurementrequestpb.ListProcurementRequestsRequest) (*procurementrequestpb.ListProcurementRequestsResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityProcurementRequest, ports.ActionList); err != nil {
		return nil, err
	}
	return uc.repositories.ProcurementRequest.ListProcurementRequests(ctx, req)
}
