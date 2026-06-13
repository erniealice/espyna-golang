package attribute

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	attributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
)

// CreateAttributeUseCase handles the business logic for creating attributes
// CreateAttributeRepositories groups all repository dependencies
type CreateAttributeRepositories struct {
	Attribute attributepb.AttributeDomainServiceServer // Primary entity repository
}

// CreateAttributeServices groups all business service dependencies
type CreateAttributeServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
}

// CreateAttributeUseCase handles the business logic for creating attributes
type CreateAttributeUseCase struct {
	repositories CreateAttributeRepositories
	services     CreateAttributeServices
}

// NewCreateAttributeUseCase creates use case with grouped dependencies
func NewCreateAttributeUseCase(
	repositories CreateAttributeRepositories,
	services CreateAttributeServices,
) *CreateAttributeUseCase {
	return &CreateAttributeUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewCreateAttributeUseCaseUngrouped creates a new CreateAttributeUseCase
// Deprecated: Use NewCreateAttributeUseCase with grouped parameters instead
func NewCreateAttributeUseCaseUngrouped(attributeRepo attributepb.AttributeDomainServiceServer) *CreateAttributeUseCase {
	// Build grouped parameters internally for backward compatibility
	repositories := CreateAttributeRepositories{
		Attribute: attributeRepo,
	}

	services := CreateAttributeServices{
		Transactor:  ports.NewNoOpTransactor(),
		Translator:  ports.NewNoOpTranslator(),
		IDGenerator: ports.NewNoOpIDGenerator(),
	}

	return NewCreateAttributeUseCase(repositories, services)
}

// Execute performs the create attribute operation
func (uc *CreateAttributeUseCase) Execute(ctx context.Context, req *attributepb.CreateAttributeRequest) (*attributepb.CreateAttributeResponse, error) {
	// Authorization check
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: "attribute",
		Action: entityid.ActionCreate,
	}); err != nil {
		return nil, err
	}

	// Check if transaction service is available and supports transactions
	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, req)
	}

	// Fallback to non-transactional execution
	return uc.executeCore(ctx, req)
}

// executeWithTransaction executes attribute creation within a transaction
func (uc *CreateAttributeUseCase) executeWithTransaction(ctx context.Context, req *attributepb.CreateAttributeRequest) (*attributepb.CreateAttributeResponse, error) {
	var result *attributepb.CreateAttributeResponse

	err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, req)
		if err != nil {
			return fmt.Errorf("attribute creation failed: %w", err)
		}
		result = res
		return nil
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// executeCore contains the core business logic (moved from original Execute method)
func (uc *CreateAttributeUseCase) executeCore(ctx context.Context, req *attributepb.CreateAttributeRequest) (*attributepb.CreateAttributeResponse, error) {
	// Input validation
	if err := uc.validateInput(req); err != nil {
		return nil, fmt.Errorf("input validation failed: %w", err)
	}

	// Business logic and enrichment
	if err := uc.enrichAttributeData(req.Data); err != nil {
		return nil, fmt.Errorf("business logic enrichment failed: %w", err)
	}

	// Business rule validation
	if err := uc.validateBusinessRules(req.Data); err != nil {
		return nil, fmt.Errorf("business rule validation failed: %w", err)
	}

	// Call repository
	return uc.repositories.Attribute.CreateAttribute(ctx, req)
}

// validateInput validates the input request
func (uc *CreateAttributeUseCase) validateInput(req *attributepb.CreateAttributeRequest) error {
	if req == nil {
		return errors.New("request is required")
	}
	if req.Data == nil {
		return errors.New("attribute data is required")
	}
	if req.Data.Name == "" {
		return errors.New("attribute name is required")
	}
	return nil
}

// enrichAttributeData adds generated fields and audit information
func (uc *CreateAttributeUseCase) enrichAttributeData(attribute *attributepb.Attribute) error {
	now := time.Now()

	// Generate Attribute ID if not provided
	if attribute.Id == "" {
		attribute.Id = uc.services.IDGenerator.GenerateID()
	}

	// Set attribute audit fields
	attribute.DateCreated = &[]int64{now.Unix()}[0]
	attribute.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	attribute.DateModified = &[]int64{now.Unix()}[0]
	attribute.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	attribute.Active = true

	return nil
}

// validateBusinessRules enforces business constraints
func (uc *CreateAttributeUseCase) validateBusinessRules(attribute *attributepb.Attribute) error {
	// Validate name length
	if len(strings.TrimSpace(attribute.Name)) == 0 {
		return errors.New("attribute name cannot be empty")
	}

	if len(attribute.Name) < 2 {
		return errors.New("attribute name must be at least 2 characters long")
	}

	if len(attribute.Name) > 100 {
		return errors.New("attribute name cannot exceed 100 characters")
	}

	// Validate description length if provided
	if attribute.Description != "" && len(attribute.Description) > 500 {
		return errors.New("attribute description cannot exceed 500 characters")
	}

	return nil
}
