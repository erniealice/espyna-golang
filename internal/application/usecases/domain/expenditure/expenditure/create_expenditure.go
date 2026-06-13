package expenditure

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
	paymenttermpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/payment_term"
	expenditurepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expenditure"
)

const entityExpenditure = "expenditure"

// CreateExpenditureRepositories groups all repository dependencies
type CreateExpenditureRepositories struct {
	Expenditure expenditurepb.ExpenditureDomainServiceServer
	PaymentTerm paymenttermpb.PaymentTermDomainServiceServer
}

// CreateExpenditureServices groups all business service dependencies
type CreateExpenditureServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
}

// CreateExpenditureUseCase handles the business logic for creating expenditures
type CreateExpenditureUseCase struct {
	repositories CreateExpenditureRepositories
	services     CreateExpenditureServices
}

// NewCreateExpenditureUseCase creates use case with grouped dependencies
func NewCreateExpenditureUseCase(
	repositories CreateExpenditureRepositories,
	services CreateExpenditureServices,
) *CreateExpenditureUseCase {
	return &CreateExpenditureUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the create expenditure operation
func (uc *CreateExpenditureUseCase) Execute(ctx context.Context, req *expenditurepb.CreateExpenditureRequest) (*expenditurepb.CreateExpenditureResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityExpenditure,
		Action: entityid.ActionCreate,
	}); err != nil {
		return nil, err
	}

	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		var result *expenditurepb.CreateExpenditureResponse
		err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
			res, err := uc.executeCore(txCtx, req)
			if err != nil {
				return fmt.Errorf("expenditure creation failed: %w", err)
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

func (uc *CreateExpenditureUseCase) executeCore(ctx context.Context, req *expenditurepb.CreateExpenditureRequest) (*expenditurepb.CreateExpenditureResponse, error) {
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "expenditure.validation.data_required", "Expenditure data is required [DEFAULT]"))
	}

	// Enrich with ID and audit fields
	now := time.Now()
	if req.Data.Id == "" {
		req.Data.Id = uc.services.IDGenerator.GenerateID()
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
			baseDate := req.Data.GetExpenditureDate()
			ptType := strings.ToLower(pt.Type)
			var dueDate int64
			switch ptType {
			case "net":
				dueDate = baseDate + int64(pt.NetDays)*86400000
			case "due_on_receipt", "cod":
				dueDate = baseDate
			case "proximate":
				if day := int(pt.GetProximateDay()); day >= 1 && day <= 28 {
					base := time.UnixMilli(baseDate).UTC()
					next := time.Date(base.Year(), base.Month()+1, day, 0, 0, 0, 0, time.UTC)
					dueDate = next.UnixMilli()
				}
			}
			if dueDate > 0 {
				dueDateStr := time.UnixMilli(dueDate).UTC().Format("2006-01-02")
				req.Data.DueDate = &dueDateStr
			}
		}
	}

	return uc.repositories.Expenditure.CreateExpenditure(ctx, req)
}
