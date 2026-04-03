package revenue

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
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

// Execute performs the create revenue operation
func (uc *CreateRevenueUseCase) Execute(ctx context.Context, req *revenuepb.CreateRevenueRequest) (*revenuepb.CreateRevenueResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityRevenue, ports.ActionCreate); err != nil {
		return nil, err
	}

	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		var result *revenuepb.CreateRevenueResponse
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
		return result, nil
	}

	return uc.executeCore(ctx, req)
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
