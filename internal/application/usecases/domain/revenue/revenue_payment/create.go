package revenuepayment

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue_payment"
)

// CreateRevenuePaymentRepositories groups all repository dependencies
type CreateRevenuePaymentRepositories struct {
	RevenuePayment pb.RevenuePaymentDomainServiceServer
}

// CreateRevenuePaymentServices groups all business service dependencies
type CreateRevenuePaymentServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
}

// CreateRevenuePaymentUseCase handles the business logic for creating revenue payments
type CreateRevenuePaymentUseCase struct {
	repositories CreateRevenuePaymentRepositories
	services     CreateRevenuePaymentServices
}

// NewCreateRevenuePaymentUseCase creates use case with grouped dependencies
func NewCreateRevenuePaymentUseCase(
	repositories CreateRevenuePaymentRepositories,
	services CreateRevenuePaymentServices,
) *CreateRevenuePaymentUseCase {
	return &CreateRevenuePaymentUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the create revenue payment operation
func (uc *CreateRevenuePaymentUseCase) Execute(ctx context.Context, req *pb.CreateRevenuePaymentRequest) (*pb.CreateRevenuePaymentResponse, error) {
	// Authorization check
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityRevenuePayment,
		Action: entityid.ActionCreate,
	}); err != nil {
		return nil, err
	}

	// Check if transaction service is available and supports transactions
	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes revenue payment creation within a transaction
func (uc *CreateRevenuePaymentUseCase) executeWithTransaction(ctx context.Context, req *pb.CreateRevenuePaymentRequest) (*pb.CreateRevenuePaymentResponse, error) {
	var result *pb.CreateRevenuePaymentResponse

	err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.Translator, "revenue_payment.errors.creation_failed", "Revenue payment creation failed [DEFAULT]")
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

// executeCore contains the core business logic
func (uc *CreateRevenuePaymentUseCase) executeCore(ctx context.Context, req *pb.CreateRevenuePaymentRequest) (*pb.CreateRevenuePaymentResponse, error) {
	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Business logic and enrichment
	if err := uc.enrichRevenuePaymentData(req.Data); err != nil {
		return nil, err
	}

	// Call repository
	return uc.repositories.RevenuePayment.CreateRevenuePayment(ctx, req)
}

// validateInput validates the input request
func (uc *CreateRevenuePaymentUseCase) validateInput(ctx context.Context, req *pb.CreateRevenuePaymentRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "revenue_payment.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "revenue_payment.validation.data_required", "[ERR-DEFAULT] Revenue payment data is required"))
	}
	if req.Data.RevenueId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "revenue_payment.validation.revenue_id_required", "[ERR-DEFAULT] Revenue ID is required"))
	}
	return nil
}

// enrichRevenuePaymentData adds generated fields, audit information, and status defaults.
//
// Per entity-status-conventions, a transactional row MUST carry a non-empty status
// alongside active=true or it vanishes from status-filtered lists. The form defaults
// (collection_type="sale", status="completed") that today live as string literals in
// the centymo write path move here as usecase defaults.
func (uc *CreateRevenuePaymentUseCase) enrichRevenuePaymentData(payment *pb.RevenuePayment) error {
	now := time.Now()

	// Generate ID if not provided
	if payment.Id == "" {
		payment.Id = uc.services.IDGenerator.GenerateID()
	}

	// Status / lifecycle defaults
	if payment.Status == nil || *payment.Status == "" {
		completed := "completed"
		payment.Status = &completed
	}
	if payment.CollectionType == nil || *payment.CollectionType == "" {
		sale := "sale"
		payment.CollectionType = &sale
	}
	if payment.Currency == "" {
		payment.Currency = "PHP"
	}
	payment.Active = true

	// Audit fields
	payment.DateCreated = &[]int64{now.UnixMilli()}[0]
	payment.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	payment.DateModified = &[]int64{now.UnixMilli()}[0]
	payment.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return nil
}
