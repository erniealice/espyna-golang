package license

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	licensehistory "github.com/erniealice/espyna-golang/internal/application/usecases/subscription/license_history"
	licensepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/license"
	licensehistorypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/license_history"
	subscriptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription"
)

// CreateLicenseRepositories groups all repository dependencies
type CreateLicenseRepositories struct {
	License      licensepb.LicenseDomainServiceServer           // Primary entity repository
	Subscription subscriptionpb.SubscriptionDomainServiceServer // For FK validation
}

// CreateLicenseServices groups all business service dependencies
type CreateLicenseServices struct {
	AuthorizationService ports.AuthorizationService // RBAC and permissions
	TransactionService   ports.TransactionService   // Database transactions
	TranslationService   ports.TranslationService   // i18n error messages
	IDService            ports.IDService            // UUID generation
}

// CreateLicenseUseCase handles the business logic for creating licenses
type CreateLicenseUseCase struct {
	repositories         CreateLicenseRepositories
	services             CreateLicenseServices
	createHistoryUseCase *licensehistory.CreateLicenseHistoryUseCase
}

// NewCreateLicenseUseCase creates a new CreateLicenseUseCase
func NewCreateLicenseUseCase(
	repositories CreateLicenseRepositories,
	services CreateLicenseServices,
	createHistoryUseCase *licensehistory.CreateLicenseHistoryUseCase,
) *CreateLicenseUseCase {
	return &CreateLicenseUseCase{
		repositories:         repositories,
		services:             services,
		createHistoryUseCase: createHistoryUseCase,
	}
}

// Execute performs the create license operation
func (uc *CreateLicenseUseCase) Execute(ctx context.Context, req *licensepb.CreateLicenseRequest) (*licensepb.CreateLicenseResponse, error) {
	// Check for transaction support and route accordingly
	if uc.services.TransactionService != nil {
		return uc.executeWithTransaction(ctx, req)
	}
	return uc.executeCore(ctx, req)
}

// executeWithTransaction performs the create license operation within a transaction
func (uc *CreateLicenseUseCase) executeWithTransaction(ctx context.Context, req *licensepb.CreateLicenseRequest) (*licensepb.CreateLicenseResponse, error) {
	var result *licensepb.CreateLicenseResponse
	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(ctx context.Context) error {
		var txErr error
		result, txErr = uc.executeCore(ctx, req)
		if txErr != nil {
			errMsg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "license.errors.creation_failed", "license creation failed [DEFAULT]")
			return fmt.Errorf("%s: %w", errMsg, txErr)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// executeCore performs the core create license operation
func (uc *CreateLicenseUseCase) executeCore(ctx context.Context, req *licensepb.CreateLicenseRequest) (*licensepb.CreateLicenseResponse, error) {
	// Authorization check
	if uc.services.AuthorizationService != nil && uc.services.AuthorizationService.IsEnabled() {
		userID, err := contextutil.RequireUserIDFromContext(ctx)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "license.errors.authorization_failed", "Authorization failed for license [DEFAULT]")
			return nil, errors.New(translatedError)
		}

		permission := ports.EntityPermission(ports.EntityLicense, ports.ActionCreate)
		hasPerm, err := uc.services.AuthorizationService.HasPermission(ctx, userID, permission)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "license.errors.authorization_failed", "Authorization failed for license [DEFAULT]")
			return nil, errors.New(translatedError)
		}
		if !hasPerm {
			translatedError := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "license.errors.authorization_failed", "Authorization failed for license [DEFAULT]")
			return nil, errors.New(translatedError)
		}
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Validate foreign key references
	if err := uc.validateEntityReferences(ctx, req.Data); err != nil {
		return nil, err
	}

	// Business logic and enrichment
	if err := uc.enrichLicenseData(req.Data); err != nil {
		return nil, err
	}

	// Business rule validation
	if err := uc.validateBusinessRules(req.Data); err != nil {
		return nil, err
	}

	// Call repository
	resp, err := uc.repositories.License.CreateLicense(ctx, req)
	if err != nil {
		return nil, err
	}

	// Create history entry for the created license
	if len(resp.Data) > 0 && uc.createHistoryUseCase != nil {
		createdLicense := resp.Data[0]

		// Get performed_by from context or use system
		performedBy := "system"
		if userID, err := contextutil.RequireUserIDFromContext(ctx); err == nil {
			performedBy = userID
		}

		historyReq := &licensehistorypb.CreateLicenseHistoryRequest{
			Data: &licensehistorypb.LicenseHistory{
				LicenseId:           createdLicense.Id,
				Action:              licensehistorypb.LicenseHistoryAction_LICENSE_HISTORY_ACTION_CREATED,
				PerformedBy:         performedBy,
				LicenseStatusBefore: licensepb.LicenseStatus_LICENSE_STATUS_UNSPECIFIED,
				LicenseStatusAfter:  createdLicense.Status,
			},
		}

		// If license was created with an assignment, include assignee info
		if createdLicense.AssigneeId != nil && *createdLicense.AssigneeId != "" {
			historyReq.Data.AssigneeId = createdLicense.AssigneeId
			historyReq.Data.AssigneeType = createdLicense.AssigneeType
			historyReq.Data.AssigneeName = createdLicense.AssigneeName
		}

		_, histErr := uc.createHistoryUseCase.Execute(ctx, historyReq)
		if histErr != nil {
			// Log the error but don't fail the main operation
			// History creation is secondary to the main license creation
		}
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *CreateLicenseUseCase) validateInput(ctx context.Context, req *licensepb.CreateLicenseRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "license.validation.request_required", "request is required [DEFAULT]"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "license.validation.data_required", "license data is required [DEFAULT]"))
	}
	if req.Data.SubscriptionId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "license.validation.subscription_id_required", "subscription ID is required [DEFAULT]"))
	}
	if req.Data.PlanId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "license.validation.plan_id_required", "plan ID is required [DEFAULT]"))
	}
	return nil
}

// validateEntityReferences validates that all referenced entities exist
func (uc *CreateLicenseUseCase) validateEntityReferences(ctx context.Context, license *licensepb.License) error {
	// Validate Subscription exists
	if license.SubscriptionId != "" {
		subResp, err := uc.repositories.Subscription.ReadSubscription(ctx, &subscriptionpb.ReadSubscriptionRequest{
			Data: &subscriptionpb.Subscription{Id: license.SubscriptionId},
		})
		if err != nil || subResp == nil || len(subResp.Data) == 0 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "license.errors.subscription_not_found", "subscription not found [DEFAULT]"))
		}
		if !subResp.Data[0].Active {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "license.errors.subscription_not_active", "subscription is not active [DEFAULT]"))
		}
	}

	return nil
}

// enrichLicenseData adds generated fields and audit information
func (uc *CreateLicenseUseCase) enrichLicenseData(license *licensepb.License) error {
	now := time.Now()

	// Generate License ID if not provided
	if license.Id == "" {
		license.Id = uc.services.IDService.GenerateID()
	}

	// Generate license key if not provided (format: LIC-{YYYY}-{RANDOM})
	if license.LicenseKey == "" {
		license.LicenseKey = uc.generateLicenseKey()
	}

	// Set default status to PENDING if not specified
	if license.Status == licensepb.LicenseStatus_LICENSE_STATUS_UNSPECIFIED {
		license.Status = licensepb.LicenseStatus_LICENSE_STATUS_PENDING
	}

	// Set default license type if not specified
	if license.LicenseType == licensepb.LicenseType_LICENSE_TYPE_UNSPECIFIED {
		license.LicenseType = licensepb.LicenseType_LICENSE_TYPE_USER
	}

	// Set audit fields
	license.DateCreated = &[]int64{now.UnixMilli()}[0]
	license.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	license.DateModified = &[]int64{now.UnixMilli()}[0]
	license.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	license.Active = true

	return nil
}

// generateLicenseKey generates a human-readable license key
func (uc *CreateLicenseUseCase) generateLicenseKey() string {
	year := time.Now().Year()

	// Generate random alphanumeric string
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	randomPart := make([]byte, 8)
	for i := range randomPart {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			// Fallback to predictable but unique value
			randomPart[i] = charset[i%len(charset)]
		} else {
			randomPart[i] = charset[n.Int64()]
		}
	}

	return fmt.Sprintf("LIC-%d-%s", year, string(randomPart))
}

// validateBusinessRules enforces business constraints for licenses
func (uc *CreateLicenseUseCase) validateBusinessRules(license *licensepb.License) error {
	// Validate subscription ID format
	if len(license.SubscriptionId) < 3 {
		return errors.New("subscription ID must be at least 3 characters long")
	}

	// Validate plan ID format
	if len(license.PlanId) < 3 {
		return errors.New("plan ID must be at least 3 characters long")
	}

	return nil
}
