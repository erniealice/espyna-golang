package procurementrequest

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	procurementrequestpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/procurement_request"
)

// SpawnPurchaseOrderRepositories groups repository dependencies.
type SpawnPurchaseOrderRepositories struct {
	ProcurementRequest procurementrequestpb.ProcurementRequestDomainServiceServer
}

// SpawnPurchaseOrderServices groups service dependencies.
type SpawnPurchaseOrderServices struct {
	Authorizer ports.Authorizer
	Transactor ports.Transactor
	Translator ports.Translator
}

// SpawnPurchaseOrderUseCase creates a PurchaseOrder from an APPROVED procurement request.
// It copies header fields, spawns line items from the request lines, sets the back-FK
// procurement_request_id on the new PO, and records the PO ID on the request.
// The heavy transactional logic lives in the adapter (SpawnPurchaseOrder on the repository)
// to keep the SQL in one place and allow adapter-level UUID generation.
type SpawnPurchaseOrderUseCase struct {
	repositories SpawnPurchaseOrderRepositories
	services     SpawnPurchaseOrderServices
}

// NewSpawnPurchaseOrderUseCase creates a use case with grouped dependencies.
func NewSpawnPurchaseOrderUseCase(
	repositories SpawnPurchaseOrderRepositories,
	services SpawnPurchaseOrderServices,
) *SpawnPurchaseOrderUseCase {
	return &SpawnPurchaseOrderUseCase{repositories: repositories, services: services}
}

// Execute performs the spawn purchase order operation.
func (uc *SpawnPurchaseOrderUseCase) Execute(ctx context.Context, req *procurementrequestpb.SpawnPurchaseOrderRequest) (*procurementrequestpb.SpawnPurchaseOrderResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityProcurementRequest, ports.ActionUpdate); err != nil {
		return nil, err
	}
	if req == nil || req.GetProcurementRequestId() == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"procurement_request.validation.id_required", "Procurement request ID is required [DEFAULT]"))
	}

	resp, err := uc.repositories.ProcurementRequest.SpawnPurchaseOrder(ctx, req)
	if err != nil {
		translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"procurement_request.errors.spawn_po_failed", "[ERR-DEFAULT] Failed to spawn purchase order from request")
		return nil, fmt.Errorf("%s: %w", translatedError, err)
	}
	return resp, nil
}
