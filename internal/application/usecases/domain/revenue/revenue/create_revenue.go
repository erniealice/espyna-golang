package revenue

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	paymenttermpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/payment_term"
	revenuepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue"
)

const entityRevenue = "revenue"

// CreateRevenueRepositories groups all repository dependencies
type CreateRevenueRepositories struct {
	Revenue     revenuepb.RevenueDomainServiceServer
	PaymentTerm paymenttermpb.PaymentTermDomainServiceServer
}

// CreateRevenueServices groups all business service dependencies
type CreateRevenueServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService

	// ComputeTaxes wires the post-create tax-compute hook
	// (tax-integration plan §4 Phase D). Optional — when nil the tax-compute
	// step is skipped entirely (no error). Failure is non-fatal; the created
	// Revenue is always returned even if tax-line creation fails.
	ComputeTaxes ComputeTaxesForRevenueInvoker
}

// CreateRevenueUseCase handles the business logic for creating revenues
type CreateRevenueUseCase struct {
	repositories CreateRevenueRepositories
	services     CreateRevenueServices
}

// NewCreateRevenueUseCase creates use case with grouped dependencies
func NewCreateRevenueUseCase(
	repositories CreateRevenueRepositories,
	services CreateRevenueServices,
) *CreateRevenueUseCase {
	return &CreateRevenueUseCase{
		repositories: repositories,
		services:     services,
	}
}

// SetComputeTaxes installs the post-create tax-compute invoker after
// construction. Used by the composition layer (tax-integration plan §4 Phase D)
// to break the initialization ordering cycle without threading the tax use case
// through the entire revenue.NewUseCases signature.
//
// Safe to call with nil — disables the tax-compute hook (no warning, no lines).
func (uc *CreateRevenueUseCase) SetComputeTaxes(invoker ComputeTaxesForRevenueInvoker) {
	if uc == nil {
		return
	}
	uc.services.ComputeTaxes = invoker
}

// Execute performs the create revenue operation
func (uc *CreateRevenueUseCase) Execute(ctx context.Context, req *revenuepb.CreateRevenueRequest) (*revenuepb.CreateRevenueResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityRevenue, ports.ActionCreate); err != nil {
		return nil, err
	}

	var result *revenuepb.CreateRevenueResponse

	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
			res, err := uc.executeCore(txCtx, req)
			if err != nil {
				return fmt.Errorf("revenue creation failed: %w", err)
			}
			result = res
			return nil
		})
		if err != nil {
			return nil, err
		}
	} else {
		res, err := uc.executeCore(ctx, req)
		if err != nil {
			return nil, err
		}
		result = res
	}

	// Phase 4 H1 fix: Do NOT fire ComputeTaxes here.
	// CreateRevenueUseCase handles the plain "create header only" path — line items
	// do not exist yet at this point. Firing compute with zero lines would produce
	// empty RevenueTaxLine rows and incorrect denorm values.
	//
	// Compute is triggered by:
	//   1. RecognizeRevenueFromSubscription — fires after all lines are persisted.
	//   2. RecomputeTaxes admin action — used after manually adding lines to a revenue.
	//
	// The ComputeTaxes field and SetComputeTaxes setter are retained for potential
	// future use (e.g. if a CreateRevenueWithLineItems variant is wired here).

	return result, nil
}

func (uc *CreateRevenueUseCase) executeCore(ctx context.Context, req *revenuepb.CreateRevenueRequest) (*revenuepb.CreateRevenueResponse, error) {
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "revenue.validation.data_required", "Revenue data is required [DEFAULT]"))
	}

	// Enrich with ID and audit fields
	now := time.Now()
	if req.Data.Id == "" {
		req.Data.Id = uc.services.IDService.GenerateID()
	}
	req.Data.DateCreated = &[]int64{now.UnixMilli()}[0]
	req.Data.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	req.Data.DateModified = &[]int64{now.UnixMilli()}[0]
	req.Data.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	req.Data.Active = true

	// Compute due date from payment term if provided
	if req.Data.PaymentTermId != nil && *req.Data.PaymentTermId != "" && uc.repositories.PaymentTerm != nil {
		ptResp, err := uc.repositories.PaymentTerm.ReadPaymentTerm(ctx, &paymenttermpb.ReadPaymentTermRequest{
			Data: &paymenttermpb.PaymentTerm{Id: *req.Data.PaymentTermId},
		})
		if err == nil && len(ptResp.Data) > 0 {
			pt := ptResp.Data[0]
			baseDateStr := req.Data.GetRevenueDate()
			baseDate, parseErr := time.Parse("2006-01-02", baseDateStr)
			if parseErr == nil {
				ptType := strings.ToLower(pt.Type)
				var dueDateStr string
				switch ptType {
				case "net":
					dueDateStr = baseDate.AddDate(0, 0, int(pt.NetDays)).Format("2006-01-02")
				case "due_on_receipt", "cod":
					dueDateStr = baseDateStr
				case "proximate":
					if day := int(pt.GetProximateDay()); day >= 1 && day <= 28 {
						next := time.Date(baseDate.Year(), baseDate.Month()+1, day, 0, 0, 0, 0, time.UTC)
						dueDateStr = next.Format("2006-01-02")
					}
				}
				if dueDateStr != "" {
					req.Data.DueDate = &dueDateStr
				}
			}
		}
	}

	return uc.repositories.Revenue.CreateRevenue(ctx, req)
}
