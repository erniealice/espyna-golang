package license

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"time"

	"leapfor.xyz/espyna/internal/application/ports"
	contextutil "leapfor.xyz/espyna/internal/application/shared/context"
	licensehistory "leapfor.xyz/espyna/internal/application/usecases/subscription/license_history"
	licensepb "leapfor.xyz/esqyma/golang/v1/domain/subscription/license"
	licensehistorypb "leapfor.xyz/esqyma/golang/v1/domain/subscription/license_history"
	planpb "leapfor.xyz/esqyma/golang/v1/domain/subscription/plan"
	subscriptionpb "leapfor.xyz/esqyma/golang/v1/domain/subscription/subscription"
)

// CreateLicensesFromPlanRepositories groups all repository dependencies
type CreateLicensesFromPlanRepositories struct {
	License      licensepb.LicenseDomainServiceServer           // Primary entity repository
	Subscription subscriptionpb.SubscriptionDomainServiceServer // For FK validation and assigned_count updates
	Plan         planpb.PlanDomainServiceServer                 // For plan-based entitlement creation
}

// CreateLicensesFromPlanServices groups all business service dependencies
type CreateLicensesFromPlanServices struct {
	AuthorizationService ports.AuthorizationService // RBAC and permissions
	TransactionService   ports.TransactionService   // Database transactions
	TranslationService   ports.TranslationService   // i18n error messages
	IDService            ports.IDService            // UUID generation
}

// CreateLicensesFromPlanUseCase handles the business logic for bulk license creation from a plan
type CreateLicensesFromPlanUseCase struct {
	repositories         CreateLicensesFromPlanRepositories
	services             CreateLicensesFromPlanServices
	createHistoryUseCase *licensehistory.CreateLicenseHistoryUseCase
}

// NewCreateLicensesFromPlanUseCase creates a new CreateLicensesFromPlanUseCase
func NewCreateLicensesFromPlanUseCase(
	repositories CreateLicensesFromPlanRepositories,
	services CreateLicensesFromPlanServices,
	createHistoryUseCase *licensehistory.CreateLicenseHistoryUseCase,
) *CreateLicensesFromPlanUseCase {
	return &CreateLicensesFromPlanUseCase{
		repositories:         repositories,
		services:             services,
		createHistoryUseCase: createHistoryUseCase,
	}
}

// Execute performs the create licenses from plan operation
func (uc *CreateLicensesFromPlanUseCase) Execute(ctx context.Context, req *licensepb.CreateLicensesFromPlanRequest) (*licensepb.CreateLicensesFromPlanResponse, error) {
	// Check for transaction support and route accordingly
	if uc.services.TransactionService != nil {
		return uc.executeWithTransaction(ctx, req)
	}
	return uc.executeCore(ctx, req)
}

// executeWithTransaction performs the bulk license creation within a transaction
func (uc *CreateLicensesFromPlanUseCase) executeWithTransaction(ctx context.Context, req *licensepb.CreateLicensesFromPlanRequest) (*licensepb.CreateLicensesFromPlanResponse, error) {
	var result *licensepb.CreateLicensesFromPlanResponse
	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(ctx context.Context) error {
		var txErr error
		result, txErr = uc.executeCore(ctx, req)
		if txErr != nil {
			errMsg := contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "license.errors.bulk_creation_failed", "bulk license creation failed [DEFAULT]")
			return fmt.Errorf("%s: %w", errMsg, txErr)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// executeCore performs the core bulk license creation operation
func (uc *CreateLicensesFromPlanUseCase) executeCore(ctx context.Context, req *licensepb.CreateLicensesFromPlanRequest) (*licensepb.CreateLicensesFromPlanResponse, error) {
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

	// Validate subscription exists and get subscription data
	subscription, err := uc.validateAndGetSubscription(ctx, req.SubscriptionId)
	if err != nil {
		return nil, err
	}

	// Validate plan exists
	if err := uc.validatePlanExists(ctx, req.PlanId); err != nil {
		return nil, err
	}

	// Determine license type
	licenseType := licensepb.LicenseType_LICENSE_TYPE_USER
	if req.DefaultLicenseType != nil && *req.DefaultLicenseType != "" {
		switch *req.DefaultLicenseType {
		case "user":
			licenseType = licensepb.LicenseType_LICENSE_TYPE_USER
		case "device":
			licenseType = licensepb.LicenseType_LICENSE_TYPE_DEVICE
		case "tenant":
			licenseType = licensepb.LicenseType_LICENSE_TYPE_TENANT
		case "floating":
			licenseType = licensepb.LicenseType_LICENSE_TYPE_FLOATING
		}
	}

	// Get performed_by from context or use system
	performedBy := "system"
	if userID, err := contextutil.RequireUserIDFromContext(ctx); err == nil {
		performedBy = userID
	}

	// Create N licenses
	createdLicenses := make([]*licensepb.License, 0, req.Quantity)
	now := time.Now()

	for i := int32(1); i <= req.Quantity; i++ {
		license := &licensepb.License{
			Id:                 uc.services.IDService.GenerateID(),
			SubscriptionId:     req.SubscriptionId,
			PlanId:             req.PlanId,
			LicenseKey:         uc.generateLicenseKey(),
			LicenseType:        licenseType,
			Status:             licensepb.LicenseStatus_LICENSE_STATUS_PENDING,
			SequenceNumber:     &i,
			Active:             true,
			DateCreated:        &[]int64{now.UnixMilli()}[0],
			DateCreatedString:  &[]string{now.Format(time.RFC3339)}[0],
			DateModified:       &[]int64{now.UnixMilli()}[0],
			DateModifiedString: &[]string{now.Format(time.RFC3339)}[0],
		}

		// Create the license
		createResp, err := uc.repositories.License.CreateLicense(ctx, &licensepb.CreateLicenseRequest{
			Data: license,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create license %d of %d: %w", i, req.Quantity, err)
		}

		if len(createResp.Data) > 0 {
			createdLicense := createResp.Data[0]
			createdLicenses = append(createdLicenses, createdLicense)

			// Create history entry for the created license
			if uc.createHistoryUseCase != nil {
				historyReq := &licensehistorypb.CreateLicenseHistoryRequest{
					Data: &licensehistorypb.LicenseHistory{
						LicenseId:           createdLicense.Id,
						Action:              licensehistorypb.LicenseHistoryAction_LICENSE_HISTORY_ACTION_CREATED,
						PerformedBy:         performedBy,
						LicenseStatusBefore: licensepb.LicenseStatus_LICENSE_STATUS_UNSPECIFIED,
						LicenseStatusAfter:  createdLicense.Status,
					},
				}
				_, _ = uc.createHistoryUseCase.Execute(ctx, historyReq)
			}
		}
	}

	// Auto-assign first license to purchaser if requested
	if req.AutoAssignToPurchaser != nil && *req.AutoAssignToPurchaser && len(createdLicenses) > 0 {
		firstLicense := createdLicenses[0]

		// Get client ID from subscription
		if subscription.ClientId != "" {
			firstLicense.AssigneeId = &subscription.ClientId
			assigneeType := "client"
			firstLicense.AssigneeType = &assigneeType
			firstLicense.AssignedBy = &performedBy
			firstLicense.DateAssigned = &[]int64{now.UnixMilli()}[0]
			dateAssignedStr := now.Format(time.RFC3339)
			firstLicense.DateAssignedString = &dateAssignedStr
			firstLicense.Status = licensepb.LicenseStatus_LICENSE_STATUS_ACTIVE

			// Update the license
			_, err := uc.repositories.License.UpdateLicense(ctx, &licensepb.UpdateLicenseRequest{
				Data: firstLicense,
			})
			if err == nil {
				// Create assignment history
				if uc.createHistoryUseCase != nil {
					historyReq := &licensehistorypb.CreateLicenseHistoryRequest{
						Data: &licensehistorypb.LicenseHistory{
							LicenseId:           firstLicense.Id,
							Action:              licensehistorypb.LicenseHistoryAction_LICENSE_HISTORY_ACTION_ASSIGNED,
							AssigneeId:          firstLicense.AssigneeId,
							AssigneeType:        firstLicense.AssigneeType,
							PerformedBy:         performedBy,
							LicenseStatusBefore: licensepb.LicenseStatus_LICENSE_STATUS_PENDING,
							LicenseStatusAfter:  licensepb.LicenseStatus_LICENSE_STATUS_ACTIVE,
						},
					}
					_, _ = uc.createHistoryUseCase.Execute(ctx, historyReq)
				}

				// Update subscription assigned count
				uc.updateSubscriptionAssignedCount(ctx, req.SubscriptionId, 1)
			}
		}
	}

	return &licensepb.CreateLicensesFromPlanResponse{
		Licenses:     createdLicenses,
		CreatedCount: int32(len(createdLicenses)),
		Success:      true,
	}, nil
}

// generateLicenseKey generates a human-readable license key
func (uc *CreateLicensesFromPlanUseCase) generateLicenseKey() string {
	year := time.Now().Year()

	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	randomPart := make([]byte, 8)
	for i := range randomPart {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			randomPart[i] = charset[i%len(charset)]
		} else {
			randomPart[i] = charset[n.Int64()]
		}
	}

	return fmt.Sprintf("LIC-%d-%s", year, string(randomPart))
}

// validateInput validates the input request
func (uc *CreateLicensesFromPlanUseCase) validateInput(ctx context.Context, req *licensepb.CreateLicensesFromPlanRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "license.validation.request_required", "request is required [DEFAULT]"))
	}
	if req.SubscriptionId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "license.validation.subscription_id_required", "subscription ID is required [DEFAULT]"))
	}
	if req.PlanId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "license.validation.plan_id_required", "plan ID is required [DEFAULT]"))
	}
	if req.Quantity <= 0 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "license.validation.quantity_required", "quantity must be greater than 0 [DEFAULT]"))
	}
	if req.Quantity > 1000 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "license.validation.quantity_too_large", "quantity cannot exceed 1000 [DEFAULT]"))
	}
	return nil
}

// validateAndGetSubscription validates that the subscription exists and returns it
func (uc *CreateLicensesFromPlanUseCase) validateAndGetSubscription(ctx context.Context, subscriptionID string) (*subscriptionpb.Subscription, error) {
	subResp, err := uc.repositories.Subscription.ReadSubscription(ctx, &subscriptionpb.ReadSubscriptionRequest{
		Data: &subscriptionpb.Subscription{Id: subscriptionID},
	})
	if err != nil || subResp == nil || len(subResp.Data) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "license.errors.subscription_not_found", "subscription not found [DEFAULT]"))
	}
	if !subResp.Data[0].Active {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "license.errors.subscription_not_active", "subscription is not active [DEFAULT]"))
	}
	return subResp.Data[0], nil
}

// validatePlanExists validates that the plan exists
func (uc *CreateLicensesFromPlanUseCase) validatePlanExists(ctx context.Context, planID string) error {
	planResp, err := uc.repositories.Plan.ReadPlan(ctx, &planpb.ReadPlanRequest{
		Data: &planpb.Plan{Id: &planID},
	})
	if err != nil || planResp == nil || len(planResp.Data) == 0 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "license.errors.plan_not_found", "plan not found [DEFAULT]"))
	}
	if !planResp.Data[0].Active {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "license.errors.plan_not_active", "plan is not active [DEFAULT]"))
	}
	return nil
}

// updateSubscriptionAssignedCount updates the subscription's assigned_count
func (uc *CreateLicensesFromPlanUseCase) updateSubscriptionAssignedCount(ctx context.Context, subscriptionID string, delta int32) {
	subResp, err := uc.repositories.Subscription.ReadSubscription(ctx, &subscriptionpb.ReadSubscriptionRequest{
		Data: &subscriptionpb.Subscription{Id: subscriptionID},
	})
	if err != nil || subResp == nil || len(subResp.Data) == 0 {
		return
	}

	sub := subResp.Data[0]
	var currentCount int32
	if sub.AssignedCount != nil {
		currentCount = *sub.AssignedCount
	}
	newCount := currentCount + delta
	sub.AssignedCount = &newCount

	// Calculate available count
	if sub.Quantity != nil {
		availableCount := *sub.Quantity - newCount
		sub.AvailableCount = &availableCount
	}

	// Update subscription
	_, _ = uc.repositories.Subscription.UpdateSubscription(ctx, &subscriptionpb.UpdateSubscriptionRequest{
		Data: sub,
	})
}
