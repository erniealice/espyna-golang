package license

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	licensepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/license"
)

// GetLicenseItemPageDataRepositories groups all repository dependencies
type GetLicenseItemPageDataRepositories struct {
	License licensepb.LicenseDomainServiceServer
}

// GetLicenseItemPageDataServices groups all business service dependencies
type GetLicenseItemPageDataServices struct {
	TransactionService ports.TransactionService
	TranslationService ports.TranslationService
}

// GetLicenseItemPageDataUseCase handles the business logic for getting license item page data
type GetLicenseItemPageDataUseCase struct {
	repositories GetLicenseItemPageDataRepositories
	services     GetLicenseItemPageDataServices
}

// NewGetLicenseItemPageDataUseCase creates a new GetLicenseItemPageDataUseCase
func NewGetLicenseItemPageDataUseCase(
	repositories GetLicenseItemPageDataRepositories,
	services GetLicenseItemPageDataServices,
) *GetLicenseItemPageDataUseCase {
	return &GetLicenseItemPageDataUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the get license item page data operation
func (uc *GetLicenseItemPageDataUseCase) Execute(
	ctx context.Context,
	req *licensepb.GetLicenseItemPageDataRequest,
) (*licensepb.GetLicenseItemPageDataResponse, error) {
	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req.LicenseId); err != nil {
		return nil, err
	}

	// Use transaction service if available
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes license item page data retrieval within a transaction
func (uc *GetLicenseItemPageDataUseCase) executeWithTransaction(
	ctx context.Context,
	req *licensepb.GetLicenseItemPageDataRequest,
) (*licensepb.GetLicenseItemPageDataResponse, error) {
	var result *licensepb.GetLicenseItemPageDataResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
				txCtx,
				uc.services.TranslationService,
				"license.errors.item_page_data_failed",
				"license item page data retrieval failed: %w",
			), err)
		}
		result = res
		return nil
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// executeCore contains the core business logic for getting license item page data
func (uc *GetLicenseItemPageDataUseCase) executeCore(
	ctx context.Context,
	req *licensepb.GetLicenseItemPageDataRequest,
) (*licensepb.GetLicenseItemPageDataResponse, error) {
	// Create read request for the license
	readReq := &licensepb.ReadLicenseRequest{
		Data: &licensepb.License{
			Id: req.LicenseId,
		},
	}

	// Retrieve the license
	readResp, err := uc.repositories.License.ReadLicense(ctx, readReq)
	if err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"license.errors.read_failed",
			"failed to retrieve license: %w",
		), err)
	}

	if readResp == nil || len(readResp.Data) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"license.errors.not_found",
			"license not found",
		))
	}

	// Get the license (should be only one)
	license := readResp.Data[0]

	// Validate that we got the expected license
	if license.Id != req.LicenseId {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"license.errors.id_mismatch",
			"retrieved license ID does not match requested ID",
		))
	}

	return &licensepb.GetLicenseItemPageDataResponse{
		License: license,
		Success: true,
	}, nil
}

// validateInput validates the input request
func (uc *GetLicenseItemPageDataUseCase) validateInput(
	ctx context.Context,
	req *licensepb.GetLicenseItemPageDataRequest,
) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"license.validation.request_required",
			"request is required",
		))
	}

	if req.LicenseId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"license.validation.id_required",
			"license ID is required",
		))
	}

	return nil
}

// validateBusinessRules enforces business constraints for reading license item page data
func (uc *GetLicenseItemPageDataUseCase) validateBusinessRules(
	ctx context.Context,
	licenseId string,
) error {
	// Validate license ID format
	if len(licenseId) < 3 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(
			ctx,
			uc.services.TranslationService,
			"license.validation.id_too_short",
			"license ID is too short",
		))
	}

	return nil
}
