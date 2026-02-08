package client

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	clientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client"
	userpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/user"
)

// CreateClientRepositories groups all repository dependencies
type CreateClientRepositories struct {
	Client clientpb.ClientDomainServiceServer // Primary entity repository
	User   userpb.UserDomainServiceServer     // User repository for embedded user data
}

// CreateClientServices groups all business service dependencies
type CreateClientServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService // Add this line
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
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
		IDService:            ports.NewNoOpIDService(), // Add this line
	}

	return NewCreateClientUseCase(repositories, services)
}

// Execute performs the create client operation
func (uc *CreateClientUseCase) Execute(ctx context.Context, req *clientpb.CreateClientRequest) (*clientpb.CreateClientResponse, error) {
	// Input validation
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "client.validation.request_required", "Request is required for clients [DEFAULT]"))
	}

	// Business validation
	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		return nil, err
	}

	// Business enrichment
	enrichedClient := uc.applyBusinessLogic(req.Data)

	// Use transaction service if available
	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		return uc.executeWithTransaction(ctx, enrichedClient)
	}

	// Fallback to direct repository call
	return uc.executeCore(ctx, enrichedClient)
}

// executeWithTransaction executes client creation within a transaction
func (uc *CreateClientUseCase) executeWithTransaction(ctx context.Context, enrichedClient *clientpb.Client) (*clientpb.CreateClientResponse, error) {
	var result *clientpb.CreateClientResponse

	err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		res, err := uc.executeCore(txCtx, enrichedClient)
		if err != nil {
			translatedError := contextutil.GetTranslatedMessageWithContext(txCtx, uc.services.TranslationService, "client.errors.creation_failed", "Client creation failed [DEFAULT]")
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
func (uc *CreateClientUseCase) executeCore(ctx context.Context, enrichedClient *clientpb.Client) (*clientpb.CreateClientResponse, error) {
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
	return uc.repositories.Client.CreateClient(ctx, &clientpb.CreateClientRequest{
		Data: enrichedClient,
	})
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
		if uc.services.IDService != nil {
			client.Id = uc.services.IDService.GenerateID()
		} else {
			// Fallback to timestamp-based ID for defensive programming
			client.Id = fmt.Sprintf("client-%d", now.UnixNano())
		}
	}

	// Business logic: Generate User ID if not provided
	if client.User != nil {
		if client.User.Id == "" {
			if uc.services.IDService != nil {
				client.User.Id = uc.services.IDService.GenerateID()
			} else {
				// Fallback to timestamp-based ID for defensive programming
				client.User.Id = fmt.Sprintf("user-%d", now.UnixNano())
			}
		}
	}

	// Business logic: Generate internal_id if not provided
	if client.InternalId == "" {
		if uc.services.IDService != nil {
			client.InternalId = uc.services.IDService.GenerateID()
		} else {
			// Fallback to timestamp-based ID for defensive programming
			client.InternalId = fmt.Sprintf("internal-%d", now.UnixNano())
		}
	}

	// Business logic: Set active status for new clients
	client.Active = true

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
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "client.validation.data_required", "Client data is required [DEFAULT]"))
	}
	if client.User == nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "client.validation.user_data_required", "Client user data is required [DEFAULT]"))
	}
	if client.User.FirstName == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "client.validation.first_name_required", "Client first name is required [DEFAULT]"))
	}
	if client.User.LastName == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "client.validation.last_name_required", "Client last name is required [DEFAULT]"))
	}
	if client.User.EmailAddress == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "client.validation.email_required", "Client email address is required [DEFAULT]"))
	}

	// Business rule: Email format validation
	if err := uc.validateEmail(client.User.EmailAddress); err != nil {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "client.validation.email_invalid", "Invalid email format [DEFAULT]"))
	}

	// Business rule: Name length constraints
	fullName := client.User.FirstName + " " + client.User.LastName
	if len(fullName) <= 3 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "client.validation.full_name_too_short", "Client full name must be at least 3 characters long [DEFAULT]"))
	}

	if len(fullName) > 100 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "client.validation.full_name_too_long", "Client full name cannot exceed 100 characters [DEFAULT]"))
	}

	// Business rule: Individual name part validation
	if len(client.User.FirstName) < 1 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "client.validation.first_name_too_short", "First name must be at least 1 character long [DEFAULT]"))
	}

	if len(client.User.LastName) < 1 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "client.validation.last_name_too_short", "Last name must be at least 1 character long [DEFAULT]"))
	}

	// Business rule: Internal ID format validation
	if client.InternalId != "" {
		if len(client.InternalId) < 3 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "client.validation.internal_id_too_short", "Internal ID must be at least 3 characters long [DEFAULT]"))
		}
		if len(client.InternalId) > 50 {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "client.validation.internal_id_too_long", "Internal ID cannot exceed 50 characters [DEFAULT]"))
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
