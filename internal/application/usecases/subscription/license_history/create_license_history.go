package licensehistory

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	licensehistorypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/license_history"
)

// CreateLicenseHistoryRepositories groups all repository dependencies
type CreateLicenseHistoryRepositories struct {
	LicenseHistory licensehistorypb.LicenseHistoryDomainServiceServer // Primary entity repository
}

// CreateLicenseHistoryServices groups all business service dependencies
type CreateLicenseHistoryServices struct {
	AuthorizationService ports.AuthorizationService // RBAC and permissions
	TransactionService   ports.TransactionService   // Database transactions
	TranslationService   ports.TranslationService   // i18n error messages
	IDService            ports.IDService            // UUID generation
}

// CreateLicenseHistoryUseCase handles the business logic for creating license history entries
type CreateLicenseHistoryUseCase struct {
	repositories CreateLicenseHistoryRepositories
	services     CreateLicenseHistoryServices
}

// NewCreateLicenseHistoryUseCase creates a new CreateLicenseHistoryUseCase
func NewCreateLicenseHistoryUseCase(
	repositories CreateLicenseHistoryRepositories,
	services CreateLicenseHistoryServices,
) *CreateLicenseHistoryUseCase {
	return &CreateLicenseHistoryUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the create license history operation
func (uc *CreateLicenseHistoryUseCase) Execute(ctx context.Context, req *licensehistorypb.CreateLicenseHistoryRequest) (*licensehistorypb.CreateLicenseHistoryResponse, error) {
	// Check for transaction support and route accordingly
	if uc.services.TransactionService != nil {
		return uc.executeWithTransaction(ctx, req)
	}
	return uc.executeCore(ctx, req)
}

// executeWithTransaction performs the create license history operation within a transaction
func (uc *CreateLicenseHistoryUseCase) executeWithTransaction(ctx context.Context, req *licensehistorypb.CreateLicenseHistoryRequest) (*licensehistorypb.CreateLicenseHistoryResponse, error) {
	var result *licensehistorypb.CreateLicenseHistoryResponse
	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(ctx context.Context) error {
		var txErr error
		result, txErr = uc.executeCore(ctx, req)
		if txErr != nil {
			errMsg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "license_history.errors.creation_failed", "license history creation failed [DEFAULT]")
			return fmt.Errorf("%s: %w", errMsg, txErr)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// executeCore performs the core create license history operation
func (uc *CreateLicenseHistoryUseCase) executeCore(ctx context.Context, req *licensehistorypb.CreateLicenseHistoryRequest) (*licensehistorypb.CreateLicenseHistoryResponse, error) {
	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Business logic and enrichment
	if err := uc.enrichHistoryData(req.Data); err != nil {
		return nil, err
	}

	// Business rule validation
	if err := uc.validateBusinessRules(req.Data); err != nil {
		return nil, err
	}

	// Call repository
	resp, err := uc.repositories.LicenseHistory.CreateLicenseHistory(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *CreateLicenseHistoryUseCase) validateInput(ctx context.Context, req *licensehistorypb.CreateLicenseHistoryRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "license_history.validation.request_required", "request is required [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "license_history.validation.data_required", "license history data is required [DEFAULT]"))
	}
	if req.Data.LicenseId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "license_history.validation.license_id_required", "license ID is required [DEFAULT]"))
	}
	if req.Data.PerformedBy == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "license_history.validation.performed_by_required", "performed_by is required [DEFAULT]"))
	}
	return nil
}

// enrichHistoryData adds generated fields and audit information
func (uc *CreateLicenseHistoryUseCase) enrichHistoryData(history *licensehistorypb.LicenseHistory) error {
	now := time.Now()

	// Generate History ID if not provided
	if history.Id == "" {
		history.Id = uc.services.IDService.GenerateID()
	}

	// Set audit fields
	history.DateCreated = now.UnixMilli()
	history.DateCreatedString = now.Format(time.RFC3339)
	history.Active = true

	return nil
}

// validateBusinessRules enforces business constraints for license history
func (uc *CreateLicenseHistoryUseCase) validateBusinessRules(history *licensehistorypb.LicenseHistory) error {
	// Validate action is specified
	if history.Action == licensehistorypb.LicenseHistoryAction_LICENSE_HISTORY_ACTION_UNSPECIFIED {
		return errors.New("license history action must be specified")
	}

	// Validate license ID format
	if len(history.LicenseId) < 3 {
		return errors.New("license ID must be at least 3 characters long")
	}

	return nil
}
