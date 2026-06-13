package deferredrevenue

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	deferredrevenuepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/deferred_revenue"
)

const entityDeferredRevenue = "deferred_revenue"

// CreateDeferredRevenueRepositories groups all repository dependencies
type CreateDeferredRevenueRepositories struct {
	DeferredRevenue deferredrevenuepb.DeferredRevenueDomainServiceServer
}

// CreateDeferredRevenueServices groups all business service dependencies
type CreateDeferredRevenueServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
}

// CreateDeferredRevenueUseCase handles the business logic for creating deferred revenue
type CreateDeferredRevenueUseCase struct {
	repositories CreateDeferredRevenueRepositories
	services     CreateDeferredRevenueServices
}

// NewCreateDeferredRevenueUseCase creates use case with grouped dependencies
func NewCreateDeferredRevenueUseCase(
	repositories CreateDeferredRevenueRepositories,
	services CreateDeferredRevenueServices,
) *CreateDeferredRevenueUseCase {
	return &CreateDeferredRevenueUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the create deferred revenue operation
func (uc *CreateDeferredRevenueUseCase) Execute(ctx context.Context, req *deferredrevenuepb.CreateDeferredRevenueRequest) (*deferredrevenuepb.CreateDeferredRevenueResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityDeferredRevenue,
		Action: entityid.ActionCreate,
	}); err != nil {
		return nil, err
	}

	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		var result *deferredrevenuepb.CreateDeferredRevenueResponse
		err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
			res, err := uc.executeCore(txCtx, req)
			if err != nil {
				translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.Translator, "deferred_revenue.errors.creation_failed", "Deferred revenue creation failed [DEFAULT]")
				return fmt.Errorf("%s: %w", translatedError, err)
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

func (uc *CreateDeferredRevenueUseCase) executeCore(ctx context.Context, req *deferredrevenuepb.CreateDeferredRevenueRequest) (*deferredrevenuepb.CreateDeferredRevenueResponse, error) {
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	if err := uc.enrichDeferredRevenueData(req.Data); err != nil {
		return nil, err
	}

	if uc.repositories.DeferredRevenue == nil {
		return nil, errors.New("deferred revenue repository is not available")
	}
	return uc.repositories.DeferredRevenue.CreateDeferredRevenue(ctx, req)
}

func (uc *CreateDeferredRevenueUseCase) validateInput(ctx context.Context, req *deferredrevenuepb.CreateDeferredRevenueRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "deferred_revenue.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "deferred_revenue.validation.data_required", "[ERR-DEFAULT] Deferred revenue data is required"))
	}

	req.Data.Description = strings.TrimSpace(req.Data.Description)
	if req.Data.Description == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "deferred_revenue.validation.description_required", "[ERR-DEFAULT] Description is required"))
	}
	if req.Data.TotalAmount <= 0 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "deferred_revenue.validation.total_amount_required", "[ERR-DEFAULT] Total amount must be greater than zero"))
	}
	if req.Data.RecognitionMonths <= 0 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "deferred_revenue.validation.recognition_months_required", "[ERR-DEFAULT] Recognition months must be greater than zero"))
	}
	return nil
}

func (uc *CreateDeferredRevenueUseCase) enrichDeferredRevenueData(dr *deferredrevenuepb.DeferredRevenue) error {
	now := time.Now()
	if dr.Id == "" {
		dr.Id = uc.services.IDGenerator.GenerateID()
	}
	dr.DateCreated = &[]int64{now.UnixMilli()}[0]
	dr.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	dr.DateModified = &[]int64{now.UnixMilli()}[0]
	dr.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	dr.Active = true
	// Remaining amount starts equal to total; recognized starts at zero
	if dr.RemainingAmount == 0 {
		dr.RemainingAmount = dr.TotalAmount
	}
	return nil
}
