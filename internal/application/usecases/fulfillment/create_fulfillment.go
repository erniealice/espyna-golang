package fulfillment

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/fulfillment"
)

// ---- CreateFulfillment ----

type CreateFulfillmentRepositories struct {
	Fulfillment pb.FulfillmentDomainServiceServer
}
type CreateFulfillmentServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
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
	AuthorizationService ports.AuthorizationService
	TranslationService   ports.TranslationService
}
type GetFulfillmentUseCase struct {
	repositories GetFulfillmentRepositories
	services     GetFulfillmentServices
}

// Execute creates a fulfillment and its line items in one transaction.
// It generates IDs, sets PENDING status, and logs the initial nil→PENDING status event.
func (uc *CreateFulfillmentUseCase) Execute(ctx context.Context, req *pb.CreateFulfillmentRequest) (*pb.CreateFulfillmentResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService, "fulfillment", ports.ActionCreate); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "fulfillment.validation.data_required", "fulfillment data is required [DEFAULT]"))
	}
	if req.Data.RevenueId == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "fulfillment.validation.revenue_id_required", "revenue ID is required [DEFAULT]"))
	}

	now := time.Now()

	// Generate fulfillment ID.
	if uc.services.IDService != nil {
		req.Data.Id = uc.services.IDService.GenerateID()
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

	if uc.services.TransactionService != nil {
		var result *pb.CreateFulfillmentResponse
		err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
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
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService, "fulfillment", ports.ActionRead); err != nil {
		return nil, err
	}
	if req == nil || req.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "fulfillment.validation.id_required", "fulfillment ID is required [DEFAULT]"))
	}
	result, err := uc.repositories.Fulfillment.GetFulfillment(ctx, req)
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "fulfillment.errors.not_found", "fulfillment not found [DEFAULT]"))
	}
	return result, nil
}
