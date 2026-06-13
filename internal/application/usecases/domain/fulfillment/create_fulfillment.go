package fulfillment

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/fulfillment"
)

// ---- CreateFulfillment ----

type CreateFulfillmentRepositories struct {
	Fulfillment pb.FulfillmentDomainServiceServer
}
type CreateFulfillmentServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
}
type CreateFulfillmentUseCase struct {
	repositories CreateFulfillmentRepositories
	services     CreateFulfillmentServices
}

// ---- GetFulfillment ----

type GetFulfillmentRepositories struct {
	Fulfillment pb.FulfillmentDomainServiceServer
}
type GetFulfillmentServices struct {
	Authorizer ports.Authorizer
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}
type GetFulfillmentUseCase struct {
	repositories GetFulfillmentRepositories
	services     GetFulfillmentServices
}

// Execute creates a fulfillment and its line items in one transaction.
// It generates IDs, sets PENDING status, and logs the initial nil→PENDING status event.
func (uc *CreateFulfillmentUseCase) Execute(ctx context.Context, req *pb.CreateFulfillmentRequest) (*pb.CreateFulfillmentResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: "fulfillment", Action: entityid.ActionCreate}); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "fulfillment.validation.data_required", "fulfillment data is required [DEFAULT]"))
	}
	if req.Data.RevenueId == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "fulfillment.validation.revenue_id_required", "revenue ID is required [DEFAULT]"))
	}

	now := time.Now()

	// Generate fulfillment ID.
	if uc.services.IDGenerator != nil {
		req.Data.Id = uc.services.IDGenerator.GenerateID()
	} else {
		req.Data.Id = fmt.Sprintf("fulfillment-%d", now.UnixNano())
	}

	// Stamp timestamps.
	dc := now.UnixMilli()
	req.Data.DateCreated = &dc
	req.Data.DateModified = &dc
	req.Data.Active = true

	// Default status to PENDING for new fulfillments.
	if req.Data.Status == "" {
		req.Data.Status = string(StatusPending)
	}

	if uc.services.Transactor != nil {
		var result *pb.CreateFulfillmentResponse
		err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
			res, err := uc.repositories.Fulfillment.CreateFulfillment(txCtx, req)
			if err != nil {
				return err
			}
			result = res
			return nil
		})
		if err != nil {
			return nil, err
		}
		return result, nil
	}
	return uc.repositories.Fulfillment.CreateFulfillment(ctx, req)
}

// ---- GetFulfillment ----

func (uc *GetFulfillmentUseCase) Execute(ctx context.Context, req *pb.GetFulfillmentRequest) (*pb.GetFulfillmentResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: "fulfillment", Action: entityid.ActionRead}); err != nil {
		return nil, err
	}
	if req == nil || req.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "fulfillment.validation.id_required", "fulfillment ID is required [DEFAULT]"))
	}
	result, err := uc.repositories.Fulfillment.GetFulfillment(ctx, req)
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "fulfillment.errors.not_found", "fulfillment not found [DEFAULT]"))
	}
	return result, nil
}
