package client

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	clientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client"
	userpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/user"
)

// CreateClientRepositories groups all repository dependencies
type CreateClientRepositories struct {
	Client clientpb.ClientDomainServiceServer // Primary entity repository
	User   userpb.UserDomainServiceServer     // User repository for embedded user data
	// Attribute overlay repos (Q-GSE-10): resolve/validate/persist client_attribute
	// rows in the SAME transaction as the client. Optional (nil-safe): unset =>
	// submitted attributes are rejected fail-closed.
	Attributes AttributeRepositories
}

// CreateClientServices groups all business service dependencies
type CreateClientServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator // Add this line
}

// CreateClientUseCase handles the business logic for creating clients
type CreateClientUseCase struct {
	repositories CreateClientRepositories
	services     CreateClientServices
}

// NewCreateClientUseCase creates use case with grouped dependencies
func NewCreateClientUseCase(
	repositories CreateClientRepositories,
	services CreateClientServices,
) *CreateClientUseCase {
	return &CreateClientUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewCreateClientUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewCreateClientUseCase with grouped parameters instead
func NewCreateClientUseCaseUngrouped(clientRepo clientpb.ClientDomainServiceServer) *CreateClientUseCase {
	repositories := CreateClientRepositories{
		Client: clientRepo,
	}

	services := CreateClientServices{
		Authorizer:  nil,
		Transactor:  ports.NewNoOpTransactor(),
		Translator:       ports.NewNoOpTranslator(),
		ActionGatekeeper: actiongate.NewActionGatekeeper(nil, ports.NewNoOpTranslator()),
		IDGenerator: ports.NewNoOpIDGenerator(), // Add this line
	}

	return NewCreateClientUseCase(repositories, services)
}

// Execute performs the create client operation
func (uc *CreateClientUseCase) Execute(ctx context.Context, req *clientpb.CreateClientRequest) (*clientpb.CreateClientResponse, error) {
	// Authorization check
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityid.Client,
		Action: entityid.ActionCreate,
	}); err != nil {
		return nil, err
	}

	// Input validation
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "client.validation.request_required", "Request is required for clients [DEFAULT]"))
	}

	// Business validation
	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		return nil, err
	}

	// Business enrichment
	enrichedClient := uc.applyBusinessLogic(req.Data)

	// The Attributes section is "present" when the drawer rendered it (marker) OR
	// any attribute value was submitted. Presence — not just a non-empty slice —
	// drives validation so a REQUIRED attribute omitted from an otherwise-empty
	// submission is still rejected (W3-MED-5); the required-coverage sweep lives in
	// resolveAndValidateAttributes and must run even for a zero-length submission.
	attrsPresent := req.GetAttributesPresent() || len(req.GetAttributes()) > 0

	// Reject a present Attributes section when the overlay pipeline is not wired —
	// a value the server cannot validate is never written (fail-closed, Q-GSE-10
	// rider #1); a required attribute that cannot be enforced is likewise refused.
	if attrsPresent && !uc.repositories.Attributes.enabled() {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "client.validation.attributes_unavailable", "Client attributes cannot be validated [DEFAULT]"))
	}

	// Use transaction service if available. Attributes are synced inside the SAME
	// transaction as the client so a validation/sync failure rolls the client back.
	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, enrichedClient, req.GetAttributes(), attrsPresent)
	}

	// Fallback to direct repository call
	return uc.executeCore(ctx, enrichedClient, req.GetAttributes(), attrsPresent)
}

// executeWithTransaction executes client creation within a transaction
func (uc *CreateClientUseCase) executeWithTransaction(ctx context.Context, enrichedClient *clientpb.Client, attrs []*commonpb.AttributeCodeValue, attrsPresent bool) (*clientpb.CreateClientResponse, error) {
	var result *clientpb.CreateClientResponse

	err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, enrichedClient, attrs, attrsPresent)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.Translator, "client.errors.creation_failed", "Client creation failed [DEFAULT]")
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

// executeCore contains the core business logic for creating a client
func (uc *CreateClientUseCase) executeCore(ctx context.Context, enrichedClient *clientpb.Client, attrs []*commonpb.AttributeCodeValue, attrsPresent bool) (*clientpb.CreateClientResponse, error) {
	// Validate attributes BEFORE any write so a bad value never even creates the
	// user/client (the transaction would roll it back, but failing early is cleaner).
	// Resolution runs whenever the section is present — even with zero submitted
	// values — so an omitted REQUIRED attribute is rejected (W3-MED-5).
	var resolvedAttrs []resolvedAttribute
	if attrsPresent && uc.repositories.Attributes.enabled() {
		var err error
		resolvedAttrs, err = resolveAndValidateAttributes(ctx, uc.repositories.Attributes, attrs)
		if err != nil {
			return nil, err
		}
	}
	// Step 1: Find or create User record (if User repository is available and user data exists)
	if uc.repositories.User != nil && enrichedClient.User != nil {
		user, err := uc.findOrCreateUser(ctx, enrichedClient.User)
		if err != nil {
			return nil, fmt.Errorf("failed to find or create user record: %w", err)
		}

		// Update the client's UserId reference with the user's ID
		enrichedClient.UserId = user.Id
		enrichedClient.User = user
	}

	// Step 2: Create Client record (with reference to the User)
	resp, err := uc.repositories.Client.CreateClient(ctx, &clientpb.CreateClientRequest{
		Data: enrichedClient,
	})
	if err != nil {
		return nil, err
	}

	// Step 3: Persist validated client_attribute rows in the same transaction.
	if len(resolvedAttrs) > 0 {
		clientID := enrichedClient.GetId()
		if data := resp.GetData(); len(data) > 0 && data[0].GetId() != "" {
			clientID = data[0].GetId()
		}
		if err := syncClientAttributes(ctx, uc.repositories.Attributes, uc.services.IDGenerator, clientID, resolvedAttrs); err != nil {
			return nil, err
		}
	}

	return resp, nil
}

// findOrCreateUser finds an existing user by email or creates a new one
// This implements the "find or create" pattern at the use case level
func (uc *CreateClientUseCase) findOrCreateUser(ctx context.Context, userData *userpb.User) (*userpb.User, error) {
	if userData.EmailAddress == "" {
		return nil, fmt.Errorf("email address is required to find or create user")
	}

	// Step 1: Try to find existing user by email
	listResp, err := uc.repositories.User.ListUsers(ctx, &userpb.ListUsersRequest{
		Filters: &commonpb.FilterRequest{
			Filters: []*commonpb.TypedFilter{
				{
					Field: "email_address",
					FilterType: &commonpb.TypedFilter_StringFilter{
						StringFilter: &commonpb.StringFilter{
							Value:    userData.EmailAddress,
							Operator: commonpb.StringOperator_STRING_EQUALS,
						},
					},
				},
			},
		},
	})

	// If we found an existing user, return it
	if err == nil && listResp != nil && len(listResp.Data) > 0 {
		return listResp.Data[0], nil
	}

	// Step 2: No existing user found - create a new one
	createResp, err := uc.repositories.User.CreateUser(ctx, &userpb.CreateUserRequest{
		Data: userData,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	if createResp == nil || len(createResp.Data) == 0 {
		return nil, fmt.Errorf("user creation returned no data")
	}

	return createResp.Data[0], nil
}

// applyBusinessLogic applies business rules and returns enriched client
func (uc *CreateClientUseCase) applyBusinessLogic(client *clientpb.Client) *clientpb.Client {
	now := time.Now()

	// Business logic: Generate Client ID if not provided
	if client.Id == "" {
		if uc.services.IDGenerator != nil {
			client.Id = uc.services.IDGenerator.GenerateID()
		} else {
			// Fallback to timestamp-based ID for defensive programming
			client.Id = fmt.Sprintf("client-%d", now.UnixNano())
		}
	}

	// Business logic: Generate User ID if not provided
	if client.User != nil {
		if client.User.Id == "" {
			if uc.services.IDGenerator != nil {
				client.User.Id = uc.services.IDGenerator.GenerateID()
			} else {
				// Fallback to timestamp-based ID for defensive programming
				client.User.Id = fmt.Sprintf("user-%d", now.UnixNano())
			}
		}
	}

	// Business logic: Generate internal_id if not provided
	if client.InternalId == "" {
		if uc.services.IDGenerator != nil {
			client.InternalId = uc.services.IDGenerator.GenerateID()
		} else {
			// Fallback to timestamp-based ID for defensive programming
			client.InternalId = fmt.Sprintf("internal-%d", now.UnixNano())
		}
	}

	// Business logic: Set active status for new clients
	client.Active = true
	if client.Status == nil || *client.Status == "" {
		active := "active"
		client.Status = &active
	}

	// Business logic: Set creation audit fields
	client.DateCreated = &[]int64{now.UnixMilli()}[0]
	client.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	client.DateModified = &[]int64{now.UnixMilli()}[0]
	client.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	// Business logic: Set user audit fields
	if client.User != nil {
		client.User.DateCreated = &[]int64{now.UnixMilli()}[0]
		client.User.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
		client.User.DateModified = &[]int64{now.UnixMilli()}[0]
		client.User.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
		client.User.Active = true

		// Business logic: Set the UserId reference
		client.UserId = client.User.Id
	}

	return client
}

// validateBusinessRules enforces business constraints
func (uc *CreateClientUseCase) validateBusinessRules(ctx context.Context, client *clientpb.Client) error {
	// Business rule: Required data validation
	if client == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "client.validation.data_required", "Client data is required [DEFAULT]"))
	}
	if client.User == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "client.validation.user_data_required", "Client user data is required [DEFAULT]"))
	}
	if client.User.FirstName == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "client.validation.first_name_required", "Client first name is required [DEFAULT]"))
	}
	if client.User.LastName == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "client.validation.last_name_required", "Client last name is required [DEFAULT]"))
	}
	if client.User.EmailAddress == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "client.validation.email_required", "Client email address is required [DEFAULT]"))
	}

	// Business rule: Email format validation
	if err := uc.validateEmail(client.User.EmailAddress); err != nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "client.validation.email_invalid", "Invalid email format [DEFAULT]"))
	}

	// Business rule: Name length constraints
	fullName := client.User.FirstName + " " + client.User.LastName
	if len(fullName) < 3 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "client.validation.full_name_too_short", "Client full name must be at least 3 characters long [DEFAULT]"))
	}

	if len(fullName) > 100 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "client.validation.full_name_too_long", "Client full name cannot exceed 100 characters [DEFAULT]"))
	}

	// Business rule: Individual name part validation
	if len(client.User.FirstName) < 1 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "client.validation.first_name_too_short", "First name must be at least 1 character long [DEFAULT]"))
	}

	if len(client.User.LastName) < 1 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "client.validation.last_name_too_short", "Last name must be at least 1 character long [DEFAULT]"))
	}

	// Business rule: Internal ID format validation
	if client.InternalId != "" {
		if len(client.InternalId) < 3 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "client.validation.internal_id_too_short", "Internal ID must be at least 3 characters long [DEFAULT]"))
		}
		if len(client.InternalId) > 50 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "client.validation.internal_id_too_long", "Internal ID cannot exceed 50 characters [DEFAULT]"))
		}
	}

	return nil
}

// validateEmail validates email format
func (uc *CreateClientUseCase) validateEmail(email string) error {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(email) {
		return errors.New("invalid email format")
	}
	return nil
}

// Additional validation methods can be added here as needed
