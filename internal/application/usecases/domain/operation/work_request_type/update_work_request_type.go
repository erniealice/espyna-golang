package work_request_type

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	work_request_typepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/work_request_type"
)

// UpdateWorkRequestTypeRepositories groups all repository dependencies
type UpdateWorkRequestTypeRepositories struct {
	WorkRequestType work_request_typepb.WorkRequestTypeDomainServiceServer // Primary entity repository
}

// UpdateWorkRequestTypeServices groups all business service dependencies
type UpdateWorkRequestTypeServices struct {
	Transactor       ports.Transactor
	Translator       ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// UpdateWorkRequestTypeUseCase handles the business logic for updating work request types
type UpdateWorkRequestTypeUseCase struct {
	repositories UpdateWorkRequestTypeRepositories
	services     UpdateWorkRequestTypeServices
}

// NewUpdateWorkRequestTypeUseCase creates use case with grouped dependencies
func NewUpdateWorkRequestTypeUseCase(
	repositories UpdateWorkRequestTypeRepositories,
	services UpdateWorkRequestTypeServices,
) *UpdateWorkRequestTypeUseCase {
	return &UpdateWorkRequestTypeUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the update work request type operation
func (uc *UpdateWorkRequestTypeUseCase) Execute(ctx context.Context, req *work_request_typepb.UpdateWorkRequestTypeRequest) (*work_request_typepb.UpdateWorkRequestTypeResponse, error) {
	// Authorization check
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.WorkRequestType,
		Action: entityid.ActionUpdate,
	}); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Business enrichment
	uc.enrichData(req.Data)

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		return nil, err
	}

	// Call repository
	resp, err := uc.repositories.WorkRequestType.UpdateWorkRequestType(ctx, req)
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "work_request_type.errors.update_failed", "[ERR-DEFAULT] Work request type update failed"))
	}

	return resp, nil
}

// validateInput validates the input request
func (uc *UpdateWorkRequestTypeUseCase) validateInput(ctx context.Context, req *work_request_typepb.UpdateWorkRequestTypeRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "work_request_type.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "work_request_type.validation.data_required", "[ERR-DEFAULT] Work request type data is required"))
	}

	// Trim whitespace
	req.Data.LabelKey = strings.TrimSpace(req.Data.LabelKey)
	req.Data.DescriptionKey = strings.TrimSpace(req.Data.DescriptionKey)
	req.Data.IconKey = strings.TrimSpace(req.Data.IconKey)

	if req.Data.Id == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "work_request_type.validation.id_required", "[ERR-DEFAULT] Work request type ID is required"))
	}

	return nil
}

// enrichData applies business enrichment for updates
func (uc *UpdateWorkRequestTypeUseCase) enrichData(data *work_request_typepb.WorkRequestType) {
	now := time.Now()

	// Derive active from status (active = status is ACTIVE)
	data.Active = data.Status == work_request_typepb.WorkRequestTypeStatus_WORK_REQUEST_TYPE_STATUS_ACTIVE

	// Set modification timestamp
	data.DateModified = &[]int64{now.UnixMilli()}[0]
	data.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
}

// validateBusinessRules enforces business constraints
func (uc *UpdateWorkRequestTypeUseCase) validateBusinessRules(ctx context.Context, data *work_request_typepb.WorkRequestType) error {
	// Validate label key length if provided
	if data.LabelKey != "" && len(data.LabelKey) > 200 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "work_request_type.validation.label_key_too_long", "[ERR-DEFAULT] Label key must not exceed 200 characters"))
	}

	// Validate description key length if provided
	if data.DescriptionKey != "" && len(data.DescriptionKey) > 200 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "work_request_type.validation.description_key_too_long", "[ERR-DEFAULT] Description key must not exceed 200 characters"))
	}

	// Validate SLA hours is non-negative
	if data.DefaultSlaHours < 0 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "work_request_type.validation.sla_hours_negative", "[ERR-DEFAULT] Default SLA hours must not be negative"))
	}

	// Validate code format if code is being updated (lowercase alphanumeric + underscores)
	if data.Code != "" {
		if len(data.Code) < 2 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "work_request_type.validation.code_too_short", "[ERR-DEFAULT] Code must be at least 2 characters"))
		}
		if len(data.Code) > 50 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "work_request_type.validation.code_too_long", "[ERR-DEFAULT] Code must not exceed 50 characters"))
		}
		for _, ch := range data.Code {
			if !((ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '_') {
				return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "work_request_type.validation.code_format_invalid", fmt.Sprintf("[ERR-DEFAULT] Code must contain only lowercase letters, digits, and underscores")))
			}
		}
	}

	return nil
}
