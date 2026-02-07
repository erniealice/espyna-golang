package license

import (
	"context"
	"errors"
	"fmt"
	"time"

	"leapfor.xyz/espyna/internal/application/ports"
	contextutil "leapfor.xyz/espyna/internal/application/shared/context"
	licensepb "leapfor.xyz/esqyma/golang/v1/domain/subscription/license"
)

// ValidateLicenseAccessRepositories groups all repository dependencies
type ValidateLicenseAccessRepositories struct {
	License licensepb.LicenseDomainServiceServer // Primary entity repository
}

// ValidateLicenseAccessServices groups all business service dependencies
type ValidateLicenseAccessServices struct {
	TransactionService ports.TransactionService // Database transactions
	TranslationService ports.TranslationService // i18n error messages
}

// ValidateLicenseAccessUseCase handles the business logic for validating license access
type ValidateLicenseAccessUseCase struct {
	repositories ValidateLicenseAccessRepositories
	services     ValidateLicenseAccessServices
}

// NewValidateLicenseAccessUseCase creates a new ValidateLicenseAccessUseCase
func NewValidateLicenseAccessUseCase(
	repositories ValidateLicenseAccessRepositories,
	services ValidateLicenseAccessServices,
) *ValidateLicenseAccessUseCase {
	return &ValidateLicenseAccessUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the validate license access operation
func (uc *ValidateLicenseAccessUseCase) Execute(ctx context.Context, req *licensepb.ValidateLicenseAccessRequest) (*licensepb.ValidateLicenseAccessResponse, error) {
	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Read the license
	readResp, err := uc.repositories.License.ReadLicense(ctx, &licensepb.ReadLicenseRequest{
		Data: &licensepb.License{Id: req.LicenseId},
	})
	if err != nil || readResp == nil || len(readResp.Data) == 0 {
		return &licensepb.ValidateLicenseAccessResponse{
			IsValid:           false,
			ValidationMessage: ptr(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "license.validation.not_found", "license not found")),
			Success:           true,
		}, nil
	}

	license := readResp.Data[0]

	// Check if license is active (not soft-deleted)
	if !license.Active {
		return &licensepb.ValidateLicenseAccessResponse{
			IsValid:           false,
			ValidationMessage: ptr("license has been deleted"),
			Success:           true,
		}, nil
	}

	// Check license status
	switch license.Status {
	case licensepb.LicenseStatus_LICENSE_STATUS_REVOKED:
		return &licensepb.ValidateLicenseAccessResponse{
			IsValid:           false,
			ValidationMessage: ptr("license has been revoked"),
			Success:           true,
		}, nil
	case licensepb.LicenseStatus_LICENSE_STATUS_EXPIRED:
		return &licensepb.ValidateLicenseAccessResponse{
			IsValid:           false,
			ValidationMessage: ptr("license has expired"),
			Success:           true,
		}, nil
	case licensepb.LicenseStatus_LICENSE_STATUS_SUSPENDED:
		return &licensepb.ValidateLicenseAccessResponse{
			IsValid:           false,
			ValidationMessage: ptr("license is currently suspended"),
			Success:           true,
		}, nil
	case licensepb.LicenseStatus_LICENSE_STATUS_PENDING:
		return &licensepb.ValidateLicenseAccessResponse{
			IsValid:           false,
			ValidationMessage: ptr("license is pending activation"),
			Success:           true,
		}, nil
	}

	// Check validity dates
	now := time.Now().UnixMilli()

	// Check if license is within valid date range (date_valid_from)
	if license.DateValidFrom != nil && *license.DateValidFrom > 0 {
		if now < *license.DateValidFrom {
			return &licensepb.ValidateLicenseAccessResponse{
				IsValid:           false,
				ValidationMessage: ptr("license is not yet valid"),
				Success:           true,
			}, nil
		}
	}

	// Check if license has expired (date_valid_until)
	if license.DateValidUntil != nil && *license.DateValidUntil > 0 {
		if now > *license.DateValidUntil {
			return &licensepb.ValidateLicenseAccessResponse{
				IsValid:           false,
				ValidationMessage: ptr("license validity period has ended"),
				Success:           true,
			}, nil
		}
	}

	// If assignee validation is requested
	if req.AssigneeId != nil && *req.AssigneeId != "" {
		// Check if license is assigned
		if license.AssigneeId == nil || *license.AssigneeId == "" {
			return &licensepb.ValidateLicenseAccessResponse{
				IsValid:           false,
				ValidationMessage: ptr("license is not assigned to any user"),
				Success:           true,
			}, nil
		}

		// Check if license is assigned to the requesting user
		if *license.AssigneeId != *req.AssigneeId {
			return &licensepb.ValidateLicenseAccessResponse{
				IsValid:           false,
				ValidationMessage: ptr("license is assigned to a different user"),
				Success:           true,
			}, nil
		}

		// If assignee type is specified, validate it too
		if req.AssigneeType != nil && *req.AssigneeType != "" {
			if license.AssigneeType == nil || *license.AssigneeType != *req.AssigneeType {
				return &licensepb.ValidateLicenseAccessResponse{
					IsValid:           false,
					ValidationMessage: ptr("license assignee type does not match"),
					Success:           true,
				}, nil
			}
		}
	}

	// All validations passed
	return &licensepb.ValidateLicenseAccessResponse{
		IsValid:           true,
		License:           license,
		ValidationMessage: ptr("license is valid"),
		Success:           true,
	}, nil
}

// ptr is a helper function to create a pointer to a string
func ptr(s string) *string {
	return &s
}

// validateInput validates the input request
func (uc *ValidateLicenseAccessUseCase) validateInput(ctx context.Context, req *licensepb.ValidateLicenseAccessRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "license.validation.request_required", "request is required [DEFAULT]"))
	}
	if req.LicenseId == "" {
		// Check if license_key is provided instead
		if req.LicenseKey == nil || *req.LicenseKey == "" {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "license.validation.id_or_key_required", "license ID or license key is required [DEFAULT]"))
		}
	}
	return nil
}

// ValidateLicenseByKey validates access by license key instead of ID
func (uc *ValidateLicenseAccessUseCase) ValidateLicenseByKey(ctx context.Context, req *licensepb.ValidateLicenseAccessRequest) (*licensepb.ValidateLicenseAccessResponse, error) {
	if req.LicenseKey == nil || *req.LicenseKey == "" {
		return nil, errors.New("license key is required")
	}

	// List all licenses and find by key
	// Note: In production, this should use a more efficient lookup (e.g., indexed query)
	listResp, err := uc.repositories.License.ListLicenses(ctx, &licensepb.ListLicensesRequest{})
	if err != nil {
		return nil, fmt.Errorf("failed to search for license: %w", err)
	}

	var matchingLicense *licensepb.License
	for _, license := range listResp.Data {
		if license.LicenseKey == *req.LicenseKey {
			matchingLicense = license
			break
		}
	}

	if matchingLicense == nil {
		return &licensepb.ValidateLicenseAccessResponse{
			IsValid:           false,
			ValidationMessage: ptr("license key not found"),
			Success:           true,
		}, nil
	}

	// Modify request to use the found license ID
	req.LicenseId = matchingLicense.Id

	// Call the standard validation
	return uc.Execute(ctx, req)
}
