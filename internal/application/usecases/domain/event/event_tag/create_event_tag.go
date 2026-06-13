package eventtag

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	eventtagpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event_tag"
)

// CreateEventTagRepositories groups all repository dependencies
type CreateEventTagRepositories struct {
	EventTag eventtagpb.EventTagDomainServiceServer // Primary entity repository
}

// CreateEventTagServices groups all business service dependencies
type CreateEventTagServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
}

// CreateEventTagUseCase handles the business logic for creating event_tag records
type CreateEventTagUseCase struct {
	repositories CreateEventTagRepositories
	services     CreateEventTagServices
}

// NewCreateEventTagUseCase creates use case with grouped dependencies
func NewCreateEventTagUseCase(
	repositories CreateEventTagRepositories,
	services CreateEventTagServices,
) *CreateEventTagUseCase {
	return &CreateEventTagUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs the create event_tag operation
func (uc *CreateEventTagUseCase) Execute(ctx context.Context, req *eventtagpb.CreateEventTagRequest) (*eventtagpb.CreateEventTagResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.EventTag,
		Action: entityid.ActionCreate,
	}); err != nil {
		return nil, err
	}

	if err := uc.validateInput(req); err != nil {
		return nil, fmt.Errorf("input validation failed: %w", err)
	}

	if err := uc.enrichEventTagData(req.Data); err != nil {
		return nil, fmt.Errorf("business logic enrichment failed: %w", err)
	}

	if uc.shouldUseTransaction(ctx) {
		return uc.executeWithTransaction(ctx, req)
	}

	return uc.executeWithoutTransaction(ctx, req)
}

func (uc *CreateEventTagUseCase) shouldUseTransaction(ctx context.Context) bool {
	if uc.services.Transactor == nil || !uc.services.Transactor.SupportsTransactions() {
		return false
	}
	if uc.services.Transactor.IsTransactionActive(ctx) {
		return false
	}
	return true
}

func (uc *CreateEventTagUseCase) executeWithTransaction(ctx context.Context, req *eventtagpb.CreateEventTagRequest) (*eventtagpb.CreateEventTagResponse, error) {
	var response *eventtagpb.CreateEventTagResponse

	err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		if err := uc.validateBusinessRules(req.Data); err != nil {
			return err
		}

		createResponse, err := uc.repositories.EventTag.CreateEventTag(txCtx, req)
		if err != nil {
			return fmt.Errorf("failed to create event_tag: %w", err)
		}

		response = createResponse
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("transaction execution failed: %w", err)
	}

	return response, nil
}

func (uc *CreateEventTagUseCase) executeWithoutTransaction(ctx context.Context, req *eventtagpb.CreateEventTagRequest) (*eventtagpb.CreateEventTagResponse, error) {
	if err := uc.validateBusinessRules(req.Data); err != nil {
		return nil, fmt.Errorf("business rule validation failed: %w", err)
	}

	return uc.repositories.EventTag.CreateEventTag(ctx, req)
}

func (uc *CreateEventTagUseCase) validateInput(req *eventtagpb.CreateEventTagRequest) error {
	if req == nil {
		return errors.New("request is required")
	}
	if req.Data == nil {
		return errors.New("event_tag data is required")
	}
	if req.Data.Name == "" {
		return errors.New("event_tag name is required")
	}
	return nil
}

func (uc *CreateEventTagUseCase) enrichEventTagData(eventTag *eventtagpb.EventTag) error {
	now := time.Now()

	if eventTag.Id == "" {
		eventTag.Id = uc.services.IDGenerator.GenerateID()
	}

	eventTag.DateCreated = &[]int64{now.UnixMilli()}[0]
	eventTag.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	eventTag.DateModified = &[]int64{now.UnixMilli()}[0]
	eventTag.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	eventTag.Active = true

	return nil
}

func (uc *CreateEventTagUseCase) validateBusinessRules(eventTag *eventtagpb.EventTag) error {
	if eventTag.WorkspaceId == "" {
		return errors.New("workspace_id is required for event_tag")
	}
	return nil
}
