package pettycash

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	pettycashfundpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/petty_cash_fund"
)

const entityPettyCashFund = "petty_cash_fund"

// CreatePettyCashFundRepositories groups all repository dependencies
type CreatePettyCashFundRepositories struct {
	PettyCashFund pettycashfundpb.PettyCashFundDomainServiceServer
}

// CreatePettyCashFundServices groups all business service dependencies
type CreatePettyCashFundServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// CreatePettyCashFundUseCase handles the business logic for creating petty cash funds
type CreatePettyCashFundUseCase struct {
	repositories CreatePettyCashFundRepositories
	services     CreatePettyCashFundServices
}

// NewCreatePettyCashFundUseCase creates use case with grouped dependencies
func NewCreatePettyCashFundUseCase(
	repositories CreatePettyCashFundRepositories,
	services CreatePettyCashFundServices,
) *CreatePettyCashFundUseCase {
	return &CreatePettyCashFundUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the create petty cash fund operation
func (uc *CreatePettyCashFundUseCase) Execute(ctx context.Context, req *pettycashfundpb.CreatePettyCashFundRequest) (*pettycashfundpb.CreatePettyCashFundResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityPettyCashFund, ports.ActionCreate); err != nil {
		return nil, err
	}

	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		var result *pettycashfundpb.CreatePettyCashFundResponse
		err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
			res, err := uc.executeCore(txCtx, req)
			if err != nil {
				translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "petty_cash_fund.errors.creation_failed", "Petty cash fund creation failed [DEFAULT]")
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

func (uc *CreatePettyCashFundUseCase) executeCore(ctx context.Context, req *pettycashfundpb.CreatePettyCashFundRequest) (*pettycashfundpb.CreatePettyCashFundResponse, error) {
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	if err := uc.enrichPettyCashFundData(req.Data); err != nil {
		return nil, err
	}

	if uc.repositories.PettyCashFund == nil {
		return nil, errors.New("petty cash fund repository is not available")
	}
	return uc.repositories.PettyCashFund.CreatePettyCashFund(ctx, req)
}

func (uc *CreatePettyCashFundUseCase) validateInput(ctx context.Context, req *pettycashfundpb.CreatePettyCashFundRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "petty_cash_fund.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "petty_cash_fund.validation.data_required", "[ERR-DEFAULT] Petty cash fund data is required"))
	}

	req.Data.Name = strings.TrimSpace(req.Data.Name)
	if req.Data.Name == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "petty_cash_fund.validation.name_required", "[ERR-DEFAULT] Fund name is required"))
	}
	if req.Data.AuthorizedAmount <= 0 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "petty_cash_fund.validation.authorized_amount_required", "[ERR-DEFAULT] Authorized amount must be greater than zero"))
	}
	return nil
}

func (uc *CreatePettyCashFundUseCase) enrichPettyCashFundData(pcf *pettycashfundpb.PettyCashFund) error {
	now := time.Now()
	if pcf.Id == "" {
		pcf.Id = uc.services.IDService.GenerateID()
	}
	pcf.DateCreated = &[]int64{now.UnixMilli()}[0]
	pcf.DateModified = &[]int64{now.UnixMilli()}[0]
	pcf.Active = true
	// Current balance starts equal to authorized amount
	if pcf.CurrentBalance == 0 {
		pcf.CurrentBalance = pcf.AuthorizedAmount
	}
	return nil
}
