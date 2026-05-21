package securitydeposit

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	securitydepositpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/security_deposit"
)

const entitySecurityDeposit = "security_deposit"

// CreateSecurityDepositRepositories groups all repository dependencies
type CreateSecurityDepositRepositories struct {
	SecurityDeposit securitydepositpb.SecurityDepositDomainServiceServer
}

// CreateSecurityDepositServices groups all business service dependencies
type CreateSecurityDepositServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	IDGenerator ports.IDGenerator
}

// CreateSecurityDepositUseCase handles the business logic for creating security deposits
type CreateSecurityDepositUseCase struct {
	repositories CreateSecurityDepositRepositories
	services     CreateSecurityDepositServices
}

// NewCreateSecurityDepositUseCase creates use case with grouped dependencies
func NewCreateSecurityDepositUseCase(
	repositories CreateSecurityDepositRepositories,
	services CreateSecurityDepositServices,
) *CreateSecurityDepositUseCase {
	return &CreateSecurityDepositUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the create security deposit operation
func (uc *CreateSecurityDepositUseCase) Execute(ctx context.Context, req *securitydepositpb.CreateSecurityDepositRequest) (*securitydepositpb.CreateSecurityDepositResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entitySecurityDeposit, ports.ActionCreate); err != nil {
		return nil, err
	}

	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		var result *securitydepositpb.CreateSecurityDepositResponse
		err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
			res, err := uc.executeCore(txCtx, req)
			if err != nil {
				translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.Translator, "security_deposit.errors.creation_failed", "Security deposit creation failed [DEFAULT]")
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

func (uc *CreateSecurityDepositUseCase) executeCore(ctx context.Context, req *securitydepositpb.CreateSecurityDepositRequest) (*securitydepositpb.CreateSecurityDepositResponse, error) {
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	if err := uc.enrichSecurityDepositData(req.Data); err != nil {
		return nil, err
	}

	if uc.repositories.SecurityDeposit == nil {
		return nil, errors.New("security deposit repository is not available")
	}
	return uc.repositories.SecurityDeposit.CreateSecurityDeposit(ctx, req)
}

func (uc *CreateSecurityDepositUseCase) validateInput(ctx context.Context, req *securitydepositpb.CreateSecurityDepositRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "security_deposit.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "security_deposit.validation.data_required", "[ERR-DEFAULT] Security deposit data is required"))
	}

	req.Data.CounterpartyName = strings.TrimSpace(req.Data.CounterpartyName)
	if req.Data.CounterpartyName == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "security_deposit.validation.counterparty_required", "[ERR-DEFAULT] Counterparty name is required"))
	}
	if req.Data.Amount <= 0 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "security_deposit.validation.amount_required", "[ERR-DEFAULT] Amount must be greater than zero"))
	}
	return nil
}

func (uc *CreateSecurityDepositUseCase) enrichSecurityDepositData(sd *securitydepositpb.SecurityDeposit) error {
	now := time.Now()
	if sd.Id == "" {
		sd.Id = uc.services.IDGenerator.GenerateID()
	}
	sd.DateCreated = &[]int64{now.UnixMilli()}[0]
	sd.DateModified = &[]int64{now.UnixMilli()}[0]
	sd.Active = true
	return nil
}
