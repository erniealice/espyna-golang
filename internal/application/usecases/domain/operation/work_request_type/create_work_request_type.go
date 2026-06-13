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
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	work_request_typepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/work_request_type"
)

// CreateWorkRequestTypeRepositories groups all repository dependencies
type CreateWorkRequestTypeRepositories struct {
	WorkRequestType work_request_typepb.WorkRequestTypeDomainServiceServer // Primary entity repository
}

// CreateWorkRequestTypeServices groups all business service dependencies
type CreateWorkRequestTypeServices struct {
	Transactor       ports.Transactor
	Translator       ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator      ports.IDGenerator
}

// CreateWorkRequestTypeUseCase handles the business logic for creating work request types
type CreateWorkRequestTypeUseCase struct {
	repositories CreateWorkRequestTypeRepositories
	services     CreateWorkRequestTypeServices
}

// NewCreateWorkRequestTypeUseCase creates use case with grouped dependencies
func NewCreateWorkRequestTypeUseCase(
	repositories CreateWorkRequestTypeRepositories,
	services CreateWorkRequestTypeServices,
) *CreateWorkRequestTypeUseCase {
	return &CreateWorkRequestTypeUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the create work request type operation
func (uc *CreateWorkRequestTypeUseCase) Execute(ctx context.Context, req *work_request_typepb.CreateWorkRequestTypeRequest) (*work_request_typepb.CreateWorkRequestTypeResponse, error) {
	// Authorization check
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.WorkRequestType,
		Action: entityid.ActionCreate,
	}); err != nil {
		return nil, err
	}

	// Input validation
	if err := uc.validateInput(ctx, req); err != nil {
		return nil, err
	}

	// Business rule validation
	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		return nil, err
	}

	// Business enrichment
	uc.applyBusinessLogic(req.Data)

	// Use transaction service if available
	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes creation within a transaction
func (uc *CreateWorkRequestTypeUseCase) executeWithTransaction(ctx context.Context, req *work_request_typepb.CreateWorkRequestTypeRequest) (*work_request_typepb.CreateWorkRequestTypeResponse, error) {
	var result *work_request_typepb.CreateWorkRequestTypeResponse
	err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return err
		}
		result = res
		return nil
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// executeCore contains the core business logic
func (uc *CreateWorkRequestTypeUseCase) executeCore(ctx context.Context, req *work_request_typepb.CreateWorkRequestTypeRequest) (*work_request_typepb.CreateWorkRequestTypeResponse, error) {
	// Code uniqueness check: list existing types in this workspace and verify code is unique
	if err := uc.checkCodeUniqueness(ctx, req.Data.WorkspaceId, req.Data.Code, ""); err != nil {
		return nil, err
	}

	// Call repository
	resp, err := uc.repositories.WorkRequestType.CreateWorkRequestType(ctx, req)
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "work_request_type.errors.creation_failed", "[ERR-DEFAULT] Work request type creation failed"))
	}
	return resp, nil
}

// checkCodeUniqueness verifies that the code is unique within the workspace
func (uc *CreateWorkRequestTypeUseCase) checkCodeUniqueness(ctx context.Context, workspaceID, code, excludeID string) error {
	// List existing types with a filter on code
	listReq := &work_request_typepb.ListWorkRequestTypesRequest{
		Filters: &commonpb.FilterRequest{
			Filters: []*commonpb.TypedFilter{
				{
					Field: "wrt.code",
					FilterType: &commonpb.TypedFilter_StringFilter{
						StringFilter: &commonpb.StringFilter{
							Value:    code,
							Operator: commonpb.StringOperator_STRING_EQUALS,
						},
					},
				},
			},
		},
	}

	resp, err := uc.repositories.WorkRequestType.ListWorkRequestTypes(ctx, listReq)
	if err != nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "work_request_type.errors.code_check_failed", "[ERR-DEFAULT] Failed to validate code uniqueness"))
	}

	for _, existing := range resp.GetData() {
		if excludeID != "" && existing.GetId() == excludeID {
			continue
		}
		if existing.GetCode() == code && existing.GetWorkspaceId() == workspaceID {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "work_request_type.validation.code_already_exists", "[ERR-DEFAULT] A work request type with this code already exists in this workspace"))
		}
	}

	return nil
}

// validateInput validates the input request
func (uc *CreateWorkRequestTypeUseCase) validateInput(ctx context.Context, req *work_request_typepb.CreateWorkRequestTypeRequest) error {
	if req == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "work_request_type.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}
	if req.Data == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "work_request_type.validation.data_required", "[ERR-DEFAULT] Work request type data is required"))
	}

	// Trim whitespace
	req.Data.Code = strings.TrimSpace(req.Data.Code)
	req.Data.LabelKey = strings.TrimSpace(req.Data.LabelKey)
	req.Data.DescriptionKey = strings.TrimSpace(req.Data.DescriptionKey)
	req.Data.IconKey = strings.TrimSpace(req.Data.IconKey)

	if req.Data.WorkspaceId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "work_request_type.validation.workspace_id_required", "[ERR-DEFAULT] Workspace ID is required"))
	}
	if req.Data.Code == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "work_request_type.validation.code_required", "[ERR-DEFAULT] Code is required"))
	}
	if req.Data.LabelKey == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "work_request_type.validation.label_key_required", "[ERR-DEFAULT] Label key is required"))
	}

	return nil
}

// validateBusinessRules enforces business constraints
func (uc *CreateWorkRequestTypeUseCase) validateBusinessRules(ctx context.Context, data *work_request_typepb.WorkRequestType) error {
	// Validate code length
	if len(data.Code) < 2 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "work_request_type.validation.code_too_short", "[ERR-DEFAULT] Code must be at least 2 characters"))
	}
	if len(data.Code) > 50 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "work_request_type.validation.code_too_long", "[ERR-DEFAULT] Code must not exceed 50 characters"))
	}

	// Validate label key length
	if len(data.LabelKey) > 200 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "work_request_type.validation.label_key_too_long", "[ERR-DEFAULT] Label key must not exceed 200 characters"))
	}

	// Validate description key length
	if len(data.DescriptionKey) > 200 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "work_request_type.validation.description_key_too_long", "[ERR-DEFAULT] Description key must not exceed 200 characters"))
	}

	// Validate category is set
	if data.Category == work_request_typepb.WorkRequestTypeCategory_WORK_REQUEST_TYPE_CATEGORY_UNSPECIFIED {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "work_request_type.validation.category_required", "[ERR-DEFAULT] Category is required"))
	}

	// Validate SLA hours is non-negative
	if data.DefaultSlaHours < 0 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "work_request_type.validation.sla_hours_negative", "[ERR-DEFAULT] Default SLA hours must not be negative"))
	}

	// Validate code format: lowercase alphanumeric + underscores
	for _, ch := range data.Code {
		if !((ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '_') {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "work_request_type.validation.code_format_invalid", fmt.Sprintf("[ERR-DEFAULT] Code must contain only lowercase letters, digits, and underscores")))
		}
	}

	return nil
}

// applyBusinessLogic applies business rules and enrichment
func (uc *CreateWorkRequestTypeUseCase) applyBusinessLogic(data *work_request_typepb.WorkRequestType) {
	now := time.Now()

	// Generate ID if not provided
	if data.Id == "" {
		data.Id = uc.services.IDGenerator.GenerateID()
	}

	// Set status to ACTIVE by default for new types
	if data.Status == work_request_typepb.WorkRequestTypeStatus_WORK_REQUEST_TYPE_STATUS_UNSPECIFIED {
		data.Status = work_request_typepb.WorkRequestTypeStatus_WORK_REQUEST_TYPE_STATUS_ACTIVE
	}

	// Derive active from status (active = status is ACTIVE)
	data.Active = data.Status == work_request_typepb.WorkRequestTypeStatus_WORK_REQUEST_TYPE_STATUS_ACTIVE

	// Set audit timestamps
	data.DateCreated = &[]int64{now.UnixMilli()}[0]
	data.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	data.DateModified = &[]int64{now.UnixMilli()}[0]
	data.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
}
